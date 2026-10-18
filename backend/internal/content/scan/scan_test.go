package scan

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/reingma/Unframed/backend/internal/content"
	"github.com/reingma/Unframed/backend/internal/content/contenttest"
)

func quietLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

var validTree = map[string]string{
	".configuration":                                  "title = \"Unframed\"\n[theme]\naccent = \"blue\"\nfont = \"serif\"\n",
	"Mathematics/.configuration":                      "order = 1\n[theme]\naccent = \"green\"\n",
	"Mathematics/index.md":                            "+++\nsummary = \"Notes on mathematics\"\n+++\nWelcome\n",
	"Mathematics/Probability/On-Martingales.md":       "+++\ntitle = \"On Martingales and Stopping\"\n[theme]\nfont = \"mono\"\n+++\n# Body\n",
	"Mathematics/Probability/On-Martingales.draft.md": "Draft rewrite\n",
	"Mathematics/Probability/figure.png":              "png",
	"Mathematics/Linear-Algebra.md":                   "No frontmatter\n",
	"Essays/.configuration":                           "slug = \"writing\"\n",
	"Essays/Hello-World.md":                           "+++\norder = 2\n+++\nHi\n",
	"Essays/About-This-Blog.md":                       "+++\norder = 1\n+++\nAbout\n",
	"Essays/Zeta.md":                                  "",
	"Essays/alpha.md":                                 "",
	"Ideas/Upcoming.draft.md":                         "secret",
	"Ideas/cover.png":                                 "png",
	"favicon.ico":                                     "ico",
}

// navEntry is a stable projection of the nav tree for comparisons.
type navEntry struct {
	URL, Title string
	Children   []navEntry
	Index      string
}

func project(node *content.Node) navEntry {
	entry := navEntry{URL: node.URLPath, Title: node.Title}
	if node.Index != nil {
		entry.Index = node.Index.SourcePath
	}
	for _, child := range node.Children {
		entry.Children = append(entry.Children, project(child))
	}
	return entry
}

func TestScanValidTree(t *testing.T) {
	for name, source := range contenttest.Sources(t, validTree) {
		t.Run(name, func(t *testing.T) {
			manifest, err := Scan(context.Background(), source, quietLogger())
			if err != nil {
				t.Fatal(err)
			}
			if len(manifest.Errors) != 0 {
				t.Fatalf("unexpected errors: %v", manifest.Errors)
			}
			want := navEntry{URL: "/", Title: "Unframed", Children: []navEntry{
				{URL: "/mathematics", Title: "Mathematics", Index: "Mathematics/index.md", Children: []navEntry{
					{URL: "/mathematics/linear-algebra", Title: "Linear Algebra"},
					{URL: "/mathematics/probability", Title: "Probability", Children: []navEntry{
						{URL: "/mathematics/probability/on-martingales", Title: "On Martingales and Stopping"},
					}},
				}},
				{URL: "/writing", Title: "Essays", Children: []navEntry{
					{URL: "/writing/about-this-blog", Title: "About This Blog"},
					{URL: "/writing/hello-world", Title: "Hello World"},
					{URL: "/writing/alpha", Title: "alpha"},
					{URL: "/writing/zeta", Title: "Zeta"},
				}},
			}}
			if got := project(manifest.Root); !reflect.DeepEqual(got, want) {
				t.Fatalf("nav tree\n got %+v\nwant %+v", got, want)
			}

			var assets []string
			for _, asset := range manifest.Assets {
				assets = append(assets, asset.Path)
			}
			wantAssets := []string{"Ideas/cover.png", "Mathematics/Probability/figure.png", "favicon.ico"}
			if !slices.Equal(assets, wantAssets) {
				t.Fatalf("assets = %v, want %v", assets, wantAssets)
			}
			if asset := manifest.Assets[2]; asset.Size != 3 || asset.ModTime.IsZero() {
				t.Fatalf("asset metadata = %+v", asset)
			}

			articles := map[string]*content.Node{}
			for _, article := range manifest.Articles {
				articles[article.SourcePath] = article
			}
			if len(articles) != 9 {
				t.Fatalf("article index has %d entries: %v", len(articles), articles)
			}
			martingales := articles["Mathematics/Probability/On-Martingales.md"]
			wantConfig := content.Config{"title": "On Martingales and Stopping", "theme": map[string]any{"accent": "green", "font": "mono"}}
			if !reflect.DeepEqual(martingales.Config, wantConfig) {
				t.Fatalf("resolved config = %#v, want %#v", martingales.Config, wantConfig)
			}
			draft := articles["Mathematics/Probability/On-Martingales.draft.md"]
			if !draft.Draft || draft.URLPath != martingales.URLPath || draft.Title != "On Martingales" {
				t.Fatalf("draft = %+v", draft)
			}
			if idea := articles["Ideas/Upcoming.draft.md"]; idea == nil || !idea.Draft {
				t.Fatalf("draft in an otherwise empty directory is not indexed: %+v", idea)
			}
			landing := articles["Mathematics/index.md"]
			if !landing.Landing || landing.URLPath != "/mathematics" || landing.Title != "Mathematics" || landing.Config["summary"] != "Notes on mathematics" {
				t.Fatalf("landing = %+v", landing)
			}
		})
	}
}

func TestScanNeverListsDrafts(t *testing.T) {
	manifest, err := Scan(context.Background(), contenttest.MemSource(validTree), quietLogger())
	if err != nil {
		t.Fatal(err)
	}
	var walk func(*content.Node)
	walk = func(node *content.Node) {
		if node.Draft || (node.Index != nil && node.Index.Draft) {
			t.Fatalf("draft %s is in the nav tree", node.SourcePath)
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	walk(manifest.Root)
}

func TestScanPerItemErrors(t *testing.T) {
	tree := map[string]string{
		".configuration":            "",
		"Valid.md":                  "ok",
		"Bad_Directory/Article.md":  "excluded with its directory",
		"Bad-Config/.configuration": "order = \n",
		"Bad-Config/Article.md":     "excluded with its directory",
		"Wrong-Type/.configuration": "title = 3\n",
		"Wrong Name.md":             "",
		"Unterminated.md":           "+++\ntitle = \"A\"\n",
		"Bad-Frontmatter.md":        "+++\ntitle = \n+++\n",
		"Bad-Slug.md":               "+++\nslug = \"Not-Lower\"\n+++\n",
		"Twin.md":                   "",
		"twin.md":                   "",
		"Shared.md":                 "",
		"Shared/Article.md":         "",
		"Override.md":               "+++\nslug = \"valid\"\n+++\n",
		"Section/index.md":          "+++\nslug = \"other\"\n+++\n",
		"Section/Kept.md":           "",
		"Section/Twin.draft.md":     "drafts collide separately",
		"Section/twin.draft.md":     "",
		"Section/Twin.md":           "a draft and its published article do not collide",
		"Section/photo.jpg":         "kept with its directory",
		"Bad_Directory/photo.jpg":   "excluded with its directory",
		"Bad-Config/photo.jpg":      "excluded with its directory",
		"Shared/photo.jpg":          "excluded with its directory",
	}
	manifest, err := Scan(context.Background(), contenttest.MemSource(tree), quietLogger())
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]error{
		"Bad_Directory":             content.ErrInvalidName,
		"Bad-Config/.configuration": content.ErrInvalidConfig,
		"Wrong-Type/.configuration": content.ErrInvalidConfig,
		"Wrong Name.md":             content.ErrInvalidName,
		"Unterminated.md":           content.ErrInvalidFrontmatter,
		"Bad-Frontmatter.md":        content.ErrInvalidConfig,
		"Bad-Slug.md":               content.ErrInvalidConfig,
		"Twin.md":                   ErrSlugCollision,
		"twin.md":                   ErrSlugCollision,
		"Shared.md":                 ErrSlugCollision,
		"Shared":                    ErrSlugCollision,
		"Valid.md":                  ErrSlugCollision,
		"Override.md":               ErrSlugCollision,
		"Section/index.md":          content.ErrInvalidConfig,
		"Section/Twin.draft.md":     ErrSlugCollision,
		"Section/twin.draft.md":     ErrSlugCollision,
	}
	got := map[string]error{}
	for _, validation := range manifest.Errors {
		if _, duplicate := got[validation.Path]; duplicate {
			t.Fatalf("%s reported twice", validation.Path)
		}
		got[validation.Path] = validation
	}
	for itemPath, sentinel := range want {
		if !errors.Is(got[itemPath], sentinel) {
			t.Errorf("%s: error %v, want %v", itemPath, got[itemPath], sentinel)
		}
	}
	if len(got) != len(want) {
		t.Errorf("errors = %v", manifest.Errors)
	}
	indexed := map[string]bool{}
	for _, article := range manifest.Articles {
		indexed[article.SourcePath] = true
	}
	for itemPath := range want {
		if indexed[itemPath] {
			t.Errorf("invalid %s is in the article index", itemPath)
		}
	}
	if len(manifest.Assets) != 1 || manifest.Assets[0].Path != "Section/photo.jpg" {
		t.Errorf("assets = %+v, want only Section/photo.jpg", manifest.Assets)
	}
	if !indexed["Section/Kept.md"] || !indexed["Section/Twin.md"] {
		t.Errorf("valid siblings were excluded: %v", indexed)
	}
	if got := project(manifest.Root); len(got.Children) != 1 || got.Children[0].URL != "/section" {
		t.Errorf("nav tree = %+v", got)
	}
}

func TestScanFatalRootConfig(t *testing.T) {
	for name, root := range map[string]*string{
		"missing":     nil,
		"unparseable": ptr("title = \n"),
		"wrong type":  ptr("title = []\n"),
		"root slug":   ptr("slug = \"home\"\n"),
		"root order":  ptr("order = 1\n"),
	} {
		t.Run(name, func(t *testing.T) {
			tree := contenttest.MemSource{"Article.md": "valid"}
			if root != nil {
				tree[".configuration"] = *root
			}
			manifest, err := Scan(context.Background(), tree, quietLogger())
			if !errors.Is(err, ErrInvalidRootConfig) || manifest != nil {
				t.Fatalf("Scan = %v, %v; want ErrInvalidRootConfig", manifest, err)
			}
		})
	}
}

func ptr(s string) *string { return &s }

// failingSource fails reads of one path, or every call once ctx is canceled.
type failingSource struct {
	contenttest.MemSource
	fail string
}

func (f failingSource) ReadFile(ctx context.Context, name string) ([]byte, error) {
	if name == f.fail {
		return nil, errors.New("permission denied")
	}
	return f.MemSource.ReadFile(ctx, name)
}

func TestScanUnreadableAndCanceled(t *testing.T) {
	source := failingSource{MemSource: contenttest.MemSource(validTree), fail: "Essays/Zeta.md"}
	manifest, err := Scan(context.Background(), source, quietLogger())
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Errors) != 1 || manifest.Errors[0].Path != "Essays/Zeta.md" || !errors.Is(manifest.Errors[0], ErrUnreadable) {
		t.Fatalf("errors = %v", manifest.Errors)
	}

	source.fail = ".configuration"
	if _, err := Scan(context.Background(), source, quietLogger()); !errors.Is(err, ErrInvalidRootConfig) {
		t.Fatalf("unreadable root configuration = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if manifest, err := Scan(ctx, contenttest.MemSource(validTree), quietLogger()); !errors.Is(err, context.Canceled) || manifest != nil {
		t.Fatalf("canceled scan = %v, %v", manifest, err)
	}
}

type traceKey struct{}

// traceHandler derives a trace attribute from the context, as an
// established context-aware handler would.
type traceHandler struct{ slog.Handler }

func (h traceHandler) Handle(ctx context.Context, record slog.Record) error {
	if id, ok := ctx.Value(traceKey{}).(string); ok {
		record.AddAttrs(slog.String("trace_id", id))
	}
	return h.Handler.Handle(ctx, record)
}

func TestScanLogsCorrelatedSummary(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(traceHandler{slog.NewJSONHandler(&buf, nil)})
	ctx := context.WithValue(context.Background(), traceKey{}, "trace-34")
	tree := contenttest.MemSource{".configuration": "", "Valid.md": "ok", "Bad_Name.md": "", "Secret.md": "+++\ntoken = \"do-not-log\"\n+++\n"}
	if _, err := Scan(ctx, tree, logger); err != nil {
		t.Fatal(err)
	}
	var record map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &record); err != nil {
		t.Fatalf("summary log is not one JSON record: %v\n%s", err, buf.String())
	}
	if record["trace_id"] != "trace-34" || record["articles"] != 2.0 || record["validation_errors"] != 1.0 {
		t.Fatalf("summary = %v", record)
	}
	if strings.Contains(buf.String(), "do-not-log") {
		t.Fatal("configuration values were logged")
	}
}

// TestScanDeterministic checks that listing order does not change the manifest.
func TestScanDeterministic(t *testing.T) {
	render := func(source content.Source) string {
		manifest, err := Scan(context.Background(), source, quietLogger())
		if err != nil {
			t.Fatal(err)
		}
		var b strings.Builder
		for _, article := range manifest.Articles {
			fmt.Fprintf(&b, "%s %s %v\n", article.SourcePath, article.URLPath, article.Config)
		}
		fmt.Fprintf(&b, "%+v %v", project(manifest.Root), manifest.Errors)
		return b.String()
	}
	want := render(contenttest.NewFS(t, validTree))
	for range 20 {
		if got := render(contenttest.MemSource(validTree)); got != want {
			t.Fatalf("manifest differs between sources or listing orders\n got %s\nwant %s", got, want)
		}
	}
}
