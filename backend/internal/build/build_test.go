package build

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json/v2"
	"errors"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/reingma/Unframed/backend/internal/content"
	"github.com/reingma/Unframed/backend/internal/content/contenttest"
	"github.com/reingma/Unframed/backend/internal/content/scan"
	"github.com/reingma/Unframed/backend/internal/document"
	"github.com/reingma/Unframed/backend/internal/event"
)

const validRoot = "title = \"Unframed\"\n[theme]\naccent = \"#336699\"\nbackground = \"#FFFFFF\"\ntext = \"#111111\"\n"

var tree = map[string]string{
	".configuration":              "title = \"Unframed\"\ndescription = \"A blog\"\n[theme]\naccent = \"#336699\"\nbackground = \"#FFFFFF\"\ntext = \"#111111\"\n",
	"Essays/index.md":             "+++\ntitle = \"Essays\"\n+++\nAll essays.\n",
	"Essays/Hello-World.md":       "+++\ntitle = \"Hello World\"\n[theme]\naccent = \"#228B22\"\n+++\n# Hello\n\n<script>x</script>\n",
	"Essays/Hello-World.draft.md": "+++\ntitle = \"Hello Draft\"\n+++\nDraft text.\n",
	"Essays/Broken.md":            "+++\n[theme]\naccent = \"red\"\n+++\nNever rendered.\n",
	"Ideas/Upcoming.draft.md":     "Only a draft.\n",
	"favicon.ico":                 "ico",
}

type recorder struct {
	batches []event.Batch
	err     error
}

func (r *recorder) Publish(_ context.Context, batch event.Batch) error {
	r.batches = append(r.batches, batch)
	return r.err
}

func quiet() *slog.Logger { return slog.New(slog.DiscardHandler) }

func TestBuild(t *testing.T) {
	for name, source := range contenttest.Sources(t, tree) {
		t.Run(name, func(t *testing.T) {
			publisher := &recorder{}
			result, err := Build(context.Background(), source, publisher, quiet())
			if err != nil {
				t.Fatal(err)
			}

			keys := slices.SortedFunc(maps.Keys(result.Envelopes), func(a, b Key) int {
				return cmp.Or(cmp.Compare(a.URLPath, b.URLPath), boolCompare(a.Draft, b.Draft))
			})
			want := []Key{{"/essays", false}, {"/essays/hello-world", false}, {"/essays/hello-world", true}, {"/ideas/upcoming", true}}
			if !slices.Equal(keys, want) {
				t.Fatalf("envelopes = %v, want %v", keys, want)
			}
			hello := result.Envelopes[Key{"/essays/hello-world", false}]
			if hello.SchemaVersion != document.SchemaVersion || hello.Body.Kind != document.BodyHTML ||
				!strings.HasPrefix(hello.Body.Content, "<h1>Hello</h1>") || strings.Contains(hello.Body.Content, "<script") {
				t.Fatalf("rendered envelope = %+v", hello)
			}
			if hello.ResolvedConfig.Theme.Accent != "#228b22" || hello.ResolvedConfig.Theme.Background != "#ffffff" {
				t.Fatalf("resolved theme = %+v", hello.ResolvedConfig.Theme)
			}
			if draft := result.Envelopes[Key{"/essays/hello-world", true}]; !draft.Metadata.Draft || draft.Body.Content != "<p>Draft text.</p>\n" {
				t.Fatalf("draft envelope = %+v", draft)
			}

			if len(publisher.batches) != 1 {
				t.Fatalf("published %d batches, want 1", len(publisher.batches))
			}
			batch := publisher.batches[0]
			if batch.BuildID == "" || batch.BuildID != result.BuildID {
				t.Fatalf("batch build ID %q, result build ID %q", batch.BuildID, result.BuildID)
			}
			var published, failed []string
			for _, e := range batch.Events {
				switch e.Kind {
				case event.KindArticlePublished:
					published = append(published, e.URLPath)
				case event.KindValidationFailed:
					failed = append(failed, e.Path)
				}
			}
			if !slices.Equal(published, []string{"/essays/hello-world", "/essays"}) {
				t.Fatalf("published events = %v", published)
			}
			var wantFailed []string
			for _, e := range result.Content.Errors {
				wantFailed = append(wantFailed, e.Path)
			}
			if len(failed) == 0 || !slices.Equal(failed, wantFailed) || failed[0] != "Essays/Broken.md" {
				t.Fatalf("validation events = %v, scan errors = %v", failed, wantFailed)
			}

			public, err := json.Marshal(result.Manifest)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(public), "Draft") || strings.Contains(string(public), "upcoming") || strings.Contains(string(public), "broken") {
				t.Fatalf("public manifest exposes drafts or invalid items: %s", public)
			}
			if _, err := document.DecodeManifest(public); err != nil {
				t.Fatalf("public manifest does not decode: %v", err)
			}
		})
	}
}

func boolCompare(a, b bool) int {
	switch {
	case a == b:
		return 0
	case b:
		return -1
	}
	return 1
}

func TestBuildPublishesFatalRootConfig(t *testing.T) {
	publisher := &recorder{}
	source := contenttest.MemSource{".configuration": "title = \"\"\n", "A.md": "a"}
	result, err := Build(context.Background(), source, publisher, quiet())
	if result != nil || !errors.Is(err, scan.ErrInvalidRootConfig) {
		t.Fatalf("Build = %v, %v; want ErrInvalidRootConfig", result, err)
	}
	if len(publisher.batches) != 1 {
		t.Fatalf("published %d batches, want 1", len(publisher.batches))
	}
	events := publisher.batches[0].Events
	if len(events) != 1 || events[0].Kind != event.KindValidationFailed || events[0].Path != content.ConfigurationFile || events[0].Message == "" {
		t.Fatalf("events = %+v", events)
	}
}

func TestBuildKeepsResultWhenPublishFails(t *testing.T) {
	unreachable := errors.New("unreachable")
	result, err := Build(context.Background(), contenttest.MemSource(tree), &recorder{err: unreachable}, quiet())
	if !errors.Is(err, ErrPublish) || !errors.Is(err, unreachable) {
		t.Fatalf("error = %v, want ErrPublish wrapping the publisher error", err)
	}
	if result == nil || len(result.Envelopes) != 4 {
		t.Fatalf("result = %+v, want the complete build", result)
	}
}

func TestBuildCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	publisher := &recorder{}
	if _, err := Build(ctx, contenttest.MemSource(tree), publisher, quiet()); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if len(publisher.batches) != 0 {
		t.Fatalf("a canceled build published %d batches", len(publisher.batches))
	}
}

func TestBuildLogsCarryBuildID(t *testing.T) {
	var out bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&out, &slog.HandlerOptions{Level: slog.LevelDebug}))
	result, err := Build(context.Background(), contenttest.MemSource(tree), event.LogPublisher{Logger: logger}, logger)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected scan, event and build records:\n%s", out.String())
	}
	for _, line := range lines {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatal(err)
		}
		if record["build_id"] != result.BuildID {
			t.Errorf("record without the build ID: %s", line)
		}
		if strings.Contains(line, "Never rendered") || strings.Contains(line, "Draft text") {
			t.Errorf("record logs article content: %s", line)
		}
	}
}

// TestBuildReportStaysOutsideContent rebuilds a filesystem tree with a report
// publisher and checks that reports never become content.
func TestBuildReportStaysOutsideContent(t *testing.T) {
	root := contenttest.WriteTree(t, tree)
	reports := filepath.Join(t.TempDir(), "reports")
	publisher, err := event.NewReportPublisher(reports, root)
	if err != nil {
		t.Fatal(err)
	}
	source := contenttest.OpenFS(t, root)
	first, err := Build(context.Background(), source, publisher, quiet())
	if err != nil {
		t.Fatal(err)
	}
	second, err := Build(context.Background(), source, publisher, quiet())
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Content.Articles) != len(first.Content.Articles) || len(second.Content.Assets) != len(first.Content.Assets) {
		t.Fatalf("rebuild changed the content: %d/%d articles, %d/%d assets",
			len(first.Content.Articles), len(second.Content.Articles), len(first.Content.Assets), len(second.Content.Assets))
	}
	entries, err := os.ReadDir(reports)
	if err != nil || len(entries) != 2 {
		t.Fatalf("reports = %v (%v), want one per build", entries, err)
	}
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(reports, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		var batch event.Batch
		if err := json.Unmarshal(data, &batch); err != nil || (batch.BuildID != first.BuildID && batch.BuildID != second.BuildID) {
			t.Fatalf("report %s = %+v (%v)", entry.Name(), batch, err)
		}
	}
}
