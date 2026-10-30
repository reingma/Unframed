package snapshot_test

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/reingma/Unframed/backend/internal/build"
	"github.com/reingma/Unframed/backend/internal/content/contenttest"
	"github.com/reingma/Unframed/backend/internal/document"
	"github.com/reingma/Unframed/backend/internal/publish"
	"github.com/reingma/Unframed/backend/internal/snapshot"
)

var tree = contenttest.MemSource{
	".configuration":              "title = \"Unframed\"\n[theme]\naccent = \"#336699\"\nbackground = \"#FFFFFF\"\ntext = \"#111111\"\n",
	"index.md":                    "Home.\n",
	"Essays/Hello-World.md":       "+++\naliases = [\"/essays/hello\"]\n+++\n# Hello\n",
	"Essays/.configuration":       "aliases = [\"/writing\"]\n",
	"Essays/Hello-World.draft.md": "Draft.\n",
	"Essays/Broken.md":            "+++\n[theme]\naccent = \"red\"\n+++\nNever rendered.\n",
	"Bad-Dir/.configuration":      "order = \"first\"\n",
	"Bad-Dir/Inside.md":           "Hidden by its directory.\n",
}

func buildResult(t *testing.T) *build.Result {
	t.Helper()
	result, err := build.Build(context.Background(), tree, publish.NewLocal(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestNewEncodesDocumentsThatDecodeBack(t *testing.T) {
	result := buildResult(t)
	s, err := snapshot.New(result, time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	if s.BuildID() != result.BuildID || s.Envelopes() != len(result.Envelopes) {
		t.Fatalf("snapshot %s with %d envelopes, want %s with %d", s.BuildID(), s.Envelopes(), result.BuildID, len(result.Envelopes))
	}
	manifest, err := document.DecodeManifest(s.Manifest().JSON)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Site.Title != result.Manifest.Site.Title || len(manifest.Nav.Children) != len(result.Manifest.Nav.Children) {
		t.Fatalf("decoded manifest = %+v, want %+v", manifest, result.Manifest)
	}
	for key, want := range result.Envelopes {
		doc, ok := s.Envelope(key.URLPath, key.Draft)
		if !ok {
			t.Fatalf("missing envelope %+v", key)
		}
		got, err := document.DecodeEnvelope(doc.JSON)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("envelope %+v decodes to %+v, want %+v", key, got, want)
		}
		if len(doc.Hash) != 64 {
			t.Fatalf("hash %q is not a SHA-256 hex digest", doc.Hash)
		}
	}
	draft, _ := s.Envelope("/essays/hello-world", true)
	published, _ := s.Envelope("/essays/hello-world", false)
	if draft.Hash == published.Hash {
		t.Fatal("different documents share a hash")
	}
}

func TestUnknownSchemaVersionIsExcludedAndReported(t *testing.T) {
	result := buildResult(t)
	key := build.Key{URLPath: "/essays/hello-world"}
	envelope := result.Envelopes[key]
	envelope.SchemaVersion = document.SchemaVersion + 1
	result.Envelopes[key] = envelope

	s, err := snapshot.New(result, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Envelope(key.URLPath, false); ok {
		t.Fatal("envelope with an unknown schemaVersion is served")
	}
	if _, ok := s.Envelope(key.URLPath, true); !ok {
		t.Fatal("the draft at the same URL was dropped too")
	}
	found := slices.ContainsFunc(s.DiagnosticsFor(key.URLPath), func(d snapshot.Diagnostic) bool {
		return d.Path == "Essays/Hello-World.md" && strings.Contains(d.Message, "schema version")
	})
	if !found {
		t.Fatalf("diagnostics = %+v, want the schema version failure", s.Diagnostics())
	}
}

func TestDiagnosticsFor(t *testing.T) {
	s, err := snapshot.New(buildResult(t), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	paths := func(urlPath string) []string {
		var got []string
		for _, d := range s.DiagnosticsFor(urlPath) {
			got = append(got, d.Path)
		}
		return got
	}
	if got := paths("/essays/broken"); !slices.Equal(got, []string{"Essays/Broken.md"}) {
		t.Fatalf("/essays/broken diagnostics = %v", got)
	}
	if got := paths("/bad-dir/inside"); len(got) != 1 || !strings.HasPrefix(got[0], "Bad-Dir") {
		t.Fatalf("an excluded directory's error is not reported for its children: %v", got)
	}
	if got := paths("/essays/hello-world"); len(got) != 0 {
		t.Fatalf("valid article has diagnostics %v", got)
	}
	if got := paths("/essays-other"); len(got) != 0 {
		t.Fatalf("a sibling with a shared prefix matched %v", got)
	}
}

func TestDiagnosticsHonorAncestorSlugOverrides(t *testing.T) {
	files := contenttest.MemSource{
		".configuration":           tree[".configuration"],
		"Long-Name/.configuration": "slug = \"short\"\n",
		"Long-Name/Ok.md":          "Ok.\n",
		"Long-Name/Broken.md":      tree["Essays/Broken.md"],
	}
	result, err := build.Build(context.Background(), files, publish.NewLocal(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	s, err := snapshot.New(result, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if got := s.DiagnosticsFor("/short/broken"); len(got) != 1 || got[0].Path != "Long-Name/Broken.md" {
		t.Fatalf("diagnostics for /short/broken = %+v, all = %+v", got, s.Diagnostics())
	}
}

// Every read through the store sees documents of a single build while
// rebuilds swap snapshots concurrently.
func TestStoreReadsAreConsistentDuringSwaps(t *testing.T) {
	marked := func(n int) *snapshot.Snapshot {
		result := buildResult(t)
		mark := fmt.Sprintf("Build %d", n)
		result.Manifest.Site.Title = mark
		for key, envelope := range result.Envelopes {
			envelope.Metadata.Title = mark
			result.Envelopes[key] = envelope
		}
		s, err := snapshot.New(result, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	snapshots := make([]*snapshot.Snapshot, 20)
	for i := range snapshots {
		snapshots[i] = marked(i)
	}
	var store snapshot.Store
	store.Swap(snapshots[0])

	var wg sync.WaitGroup
	done := make(chan struct{})
	for range 8 {
		wg.Go(func() {
			for {
				select {
				case <-done:
					return
				default:
				}
				s := store.Current()
				manifest, err := document.DecodeManifest(s.Manifest().JSON)
				if err != nil {
					t.Error(err)
					return
				}
				doc, _ := s.Envelope("/essays/hello-world", false)
				envelope, err := document.DecodeEnvelope(doc.JSON)
				if err != nil {
					t.Error(err)
					return
				}
				if envelope.Metadata.Title != manifest.Site.Title {
					t.Errorf("read mixed builds: %q and %q", manifest.Site.Title, envelope.Metadata.Title)
					return
				}
			}
		})
	}
	for i := range 2000 {
		store.Swap(snapshots[i%len(snapshots)])
	}
	close(done)
	wg.Wait()
}

// Whichever published envelope fails to decode, every link the manifest
// still has resolves to a served envelope.
func TestManifestLinksOnlyServedEnvelopes(t *testing.T) {
	for key := range buildResult(t).Envelopes {
		if key.Draft {
			continue
		}
		t.Run(key.URLPath, func(t *testing.T) {
			result := buildResult(t)
			envelope := result.Envelopes[key]
			envelope.SchemaVersion = document.SchemaVersion + 1
			result.Envelopes[key] = envelope
			s, err := snapshot.New(result, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			manifest, err := document.DecodeManifest(s.Manifest().JSON)
			if err != nil {
				t.Fatal(err)
			}
			var walk func(document.NavNode)
			walk = func(n document.NavNode) {
				if n.Kind == document.NavArticle || n.HasIndex {
					if _, ok := s.Envelope(n.URLPath, false); !ok {
						t.Fatalf("manifest links %s, which is not served", n.URLPath)
					}
				}
				for _, child := range n.Children {
					walk(child)
				}
			}
			walk(manifest.Nav)
			for _, r := range manifest.Redirects {
				if _, ok := s.Envelope(r.To, false); !r.Prefix && !ok {
					t.Fatalf("manifest redirects to %s, which is not served", r.To)
				}
			}
			if len(result.Manifest.Redirects) != 3 {
				t.Fatalf("build redirects = %v, want both aliases and their chain", result.Manifest.Redirects)
			}
			if _, ok := result.Envelopes[key]; !ok || len(result.Manifest.Nav.Children) == 0 {
				t.Fatal("New modified the build result")
			}
		})
	}
}
