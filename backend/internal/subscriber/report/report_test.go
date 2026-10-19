package report

import (
	"context"
	"encoding/json/v2"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/reingma/Unframed/backend/internal/event"
)

var events = []event.Event{
	{SchemaVersion: event.SchemaVersion, ID: "BUILD1-1", BuildID: "BUILD1", Time: time.Date(2026, 10, 18, 3, 0, 0, 0, time.UTC),
		Kind: event.KindArticlePublished, Path: "Essays/Hello-World.md", URLPath: "/essays/hello-world"},
	{SchemaVersion: event.SchemaVersion, ID: "BUILD1-2", BuildID: "BUILD1", Time: time.Date(2026, 10, 18, 3, 0, 0, 0, time.UTC),
		Kind: event.KindValidationFailed, Path: "Bad Name.md", Message: `invalid name "Bad Name"`},
}

func TestSubscriberWritesReportOutsideContent(t *testing.T) {
	base := t.TempDir()
	contentRoot := filepath.Join(base, "content")
	if err := os.Mkdir(contentRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	reports := filepath.Join(base, "reports")
	subscriber, err := New(reports, contentRoot)
	if err != nil {
		t.Fatal(err)
	}
	if err := subscriber.Receive(context.Background(), events); err != nil {
		t.Fatal(err)
	}
	if err := subscriber.Receive(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	files, err := os.ReadDir(reports)
	if err != nil || len(files) != 1 {
		t.Fatalf("report directory holds %v (%v), want one report", files, err)
	}
	if name := files[0].Name(); !strings.HasPrefix(name, "build-") || !strings.HasSuffix(name, "-BUILD1.json") {
		t.Fatalf("report name = %q", name)
	}
	data, err := os.ReadFile(filepath.Join(reports, files[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	var got []event.Event
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if !slices.EqualFunc(got, events, func(a, b event.Event) bool { return a.Time.Equal(b.Time) && a.ID == b.ID && a.Message == b.Message }) {
		t.Fatalf("report = %+v", got)
	}
	if entries, _ := os.ReadDir(contentRoot); len(entries) != 0 {
		t.Fatalf("content root changed: %v", entries)
	}
}

func TestNewRejectsDirectoryInsideContent(t *testing.T) {
	base := t.TempDir()
	contentRoot := filepath.Join(base, "content")
	if err := os.MkdirAll(filepath.Join(contentRoot, "Essays"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "reports-link")
	if err := os.Symlink(filepath.Join(contentRoot, "Essays"), link); err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"root":                  contentRoot,
		"nested":                filepath.Join(contentRoot, "reports"),
		"hidden":                filepath.Join(contentRoot, ".reports"),
		"relative":              filepath.Join(contentRoot, "Essays", "..", "out"),
		"symlink inside":        link,
		"missing under symlink": filepath.Join(link, "new", "reports"),
	}
	for name, dir := range cases {
		t.Run(name, func(t *testing.T) {
			_, statErr := os.Stat(dir)
			existed := statErr == nil
			if _, err := New(dir, contentRoot); !errors.Is(err, ErrInsideContent) {
				t.Fatalf("New(%s) error = %v, want ErrInsideContent", dir, err)
			}
			if _, err := os.Stat(dir); !existed && !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("rejected report directory %s was created (stat error %v)", dir, err)
			}
		})
	}
	sibling := filepath.Join(base, "content-reports")
	if _, err := New(sibling, contentRoot); err != nil {
		t.Fatalf("a sibling sharing the root's name prefix was rejected: %v", err)
	}
}

func TestSubscriberRejectsUnsafeBuildID(t *testing.T) {
	base := t.TempDir()
	subscriber, err := New(filepath.Join(base, "a", "reports"), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"", "../../x", "a/b", `a\b`, "..", "id.json"} {
		unsafe := slices.Clone(events)
		unsafe[0].BuildID = id
		if err := subscriber.Receive(context.Background(), unsafe); !errors.Is(err, ErrInvalidBuildID) {
			t.Errorf("Receive(BuildID %q) error = %v, want ErrInvalidBuildID", id, err)
		}
	}
	mixed := slices.Clone(events)
	mixed[1].BuildID = "../../x"
	if err := subscriber.Receive(context.Background(), mixed); !errors.Is(err, ErrInvalidBuildID) {
		t.Errorf("Receive(mixed builds) error = %v, want ErrInvalidBuildID", err)
	}
	var written []string
	err = filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			written = append(written, path)
		}
		return err
	})
	if err != nil || len(written) != 0 {
		t.Fatalf("files written for rejected build IDs: %v (walk error %v)", written, err)
	}
}

func TestSubscriberHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	subscriber, err := New(filepath.Join(t.TempDir(), "reports"), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := subscriber.Receive(ctx, events); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}
