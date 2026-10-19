package event

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var batch = Batch{
	BuildID: "BUILD1",
	Time:    time.Date(2026, 10, 18, 3, 0, 0, 0, time.UTC),
	Events: []Event{
		{Kind: KindArticlePublished, Path: "Essays/Hello-World.md", URLPath: "/essays/hello-world"},
		{Kind: KindValidationFailed, Path: "Bad Name.md", Message: `invalid name "Bad Name"`},
	},
}

func TestReportPublisherWritesBatchOutsideContent(t *testing.T) {
	base := t.TempDir()
	contentRoot := filepath.Join(base, "content")
	if err := os.Mkdir(contentRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	reports := filepath.Join(base, "reports")
	publisher, err := NewReportPublisher(reports, contentRoot)
	if err != nil {
		t.Fatal(err)
	}
	if err := publisher.Publish(context.Background(), batch); err != nil {
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
	var got Batch
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.BuildID != batch.BuildID || !got.Time.Equal(batch.Time) || len(got.Events) != 2 || got.Events[1] != batch.Events[1] {
		t.Fatalf("report = %+v", got)
	}
	if entries, _ := os.ReadDir(contentRoot); len(entries) != 0 {
		t.Fatalf("content root changed: %v", entries)
	}
}

func TestReportPublisherRejectsDirectoryInsideContent(t *testing.T) {
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
			if _, err := NewReportPublisher(dir, contentRoot); !errors.Is(err, ErrReportInsideContent) {
				t.Fatalf("NewReportPublisher(%s) error = %v, want ErrReportInsideContent", dir, err)
			}
			if _, err := os.Stat(dir); !existed && !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("rejected report directory %s was created (stat error %v)", dir, err)
			}
		})
	}
	sibling := filepath.Join(base, "content-reports")
	if _, err := NewReportPublisher(sibling, contentRoot); err != nil {
		t.Fatalf("a sibling sharing the root's name prefix was rejected: %v", err)
	}
}

func TestReportPublisherRejectsUnsafeBuildID(t *testing.T) {
	base := t.TempDir()
	reports := filepath.Join(base, "a", "reports")
	publisher, err := NewReportPublisher(reports, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"", "../../x", "a/b", `a\b`, "..", "id.json"} {
		unsafe := batch
		unsafe.BuildID = id
		if err := publisher.Publish(context.Background(), unsafe); !errors.Is(err, ErrInvalidBuildID) {
			t.Errorf("Publish(BuildID %q) error = %v, want ErrInvalidBuildID", id, err)
		}
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

func TestLogPublisherCorrelatesEvents(t *testing.T) {
	var out bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&out, &slog.HandlerOptions{Level: slog.LevelDebug}))
	if err := (LogPublisher{Logger: logger}).Publish(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != len(batch.Events) {
		t.Fatalf("got %d records, want %d:\n%s", len(lines), len(batch.Events), out.String())
	}
	for i, line := range lines {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatal(err)
		}
		if record["build_id"] != "BUILD1" || record["kind"] != string(batch.Events[i].Kind) || record["path"] != batch.Events[i].Path {
			t.Errorf("record %d = %v", i, record)
		}
	}
}

type recorder struct {
	batches []Batch
	err     error
}

func (r *recorder) Publish(_ context.Context, b Batch) error {
	r.batches = append(r.batches, b)
	return r.err
}

func TestFanoutDeliversToEveryPublisher(t *testing.T) {
	failed := errors.New("unreachable")
	first, second := &recorder{err: failed}, &recorder{}
	err := Fanout(first, second).Publish(context.Background(), batch)
	if !errors.Is(err, failed) {
		t.Fatalf("error = %v, want the failing publisher's error", err)
	}
	if len(first.batches) != 1 || len(second.batches) != 1 {
		t.Fatalf("deliveries = %d, %d; want 1, 1", len(first.batches), len(second.batches))
	}
}

func TestPublishersHonorCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	dir := t.TempDir()
	report, err := NewReportPublisher(filepath.Join(dir, "reports"), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for name, publisher := range map[string]Publisher{"report": report, "log": LogPublisher{Logger: slog.New(slog.DiscardHandler)}} {
		if err := publisher.Publish(ctx, batch); !errors.Is(err, context.Canceled) {
			t.Errorf("%s: error = %v, want context.Canceled", name, err)
		}
	}
}
