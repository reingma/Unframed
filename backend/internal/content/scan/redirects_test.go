package scan

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/reingma/Unframed/backend/internal/content"
	"github.com/reingma/Unframed/backend/internal/content/contenttest"
)

func aliases(paths ...string) string {
	return "aliases = [\"" + strings.Join(paths, "\", \"") + "\"]\n"
}

func frontmatter(config string) string { return "+++\n" + config + "+++\nBody\n" }

func scanTree(t *testing.T, tree map[string]string) *content.Manifest {
	t.Helper()
	manifest, err := Scan(context.Background(), contenttest.MemSource(tree), quietLogger())
	if err != nil {
		t.Fatal(err)
	}
	return manifest
}

func TestScanRedirects(t *testing.T) {
	manifest := scanTree(t, map[string]string{
		".configuration":              validRoot,
		"Essays/.configuration":       aliases("/writing", "/archive/essays"),
		"Essays/New-Title.md":         frontmatter(aliases("/essays/old-title", "/essays/older-title")),
		"Essays/Moved.md":             frontmatter(aliases("/elsewhere/moved")),
		"Essays/Unpublished.draft.md": frontmatter(aliases("/essays/draft-old")),
		"Ideas/Only.draft.md":         "",
		"Ideas/.configuration":        aliases("/thoughts"),
	})
	if len(manifest.Errors) != 0 {
		t.Fatalf("errors = %v", manifest.Errors)
	}
	want := []content.Redirect{
		{From: "/archive/essays", To: "/essays", Prefix: true},
		{From: "/archive/essays/old-title", To: "/essays/new-title"},
		{From: "/archive/essays/older-title", To: "/essays/new-title"},
		{From: "/elsewhere/moved", To: "/essays/moved"},
		{From: "/essays/old-title", To: "/essays/new-title"},
		{From: "/essays/older-title", To: "/essays/new-title"},
		{From: "/writing", To: "/essays", Prefix: true},
		{From: "/writing/old-title", To: "/essays/new-title"},
		{From: "/writing/older-title", To: "/essays/new-title"},
	}
	if !reflect.DeepEqual(manifest.Redirects, want) {
		t.Fatalf("redirects =\n%v\nwant\n%v", manifest.Redirects, want)
	}
}

func TestScanRedirectChainsThroughNestedDirectories(t *testing.T) {
	manifest := scanTree(t, map[string]string{
		".configuration":            validRoot,
		"Notes/.configuration":      aliases("/old-notes"),
		"Notes/Math/.configuration": aliases("/notes/maths"),
		"Notes/Math/Proof.md":       frontmatter(aliases("/notes/math/lemma")),
	})
	got := map[string]content.Redirect{}
	for _, r := range manifest.Redirects {
		got[r.From] = r
	}
	for from, want := range map[string]content.Redirect{
		"/old-notes":             {From: "/old-notes", To: "/notes", Prefix: true},
		"/old-notes/maths":       {From: "/old-notes/maths", To: "/notes/math", Prefix: true},
		"/old-notes/maths/lemma": {From: "/old-notes/maths/lemma", To: "/notes/math/proof"},
		"/old-notes/math/lemma":  {From: "/old-notes/math/lemma", To: "/notes/math/proof"},
		"/notes/maths/lemma":     {From: "/notes/maths/lemma", To: "/notes/math/proof"},
	} {
		if got[from] != want {
			t.Errorf("redirect %s = %+v, want %+v", from, got[from], want)
		}
	}
	checkRedirectInvariants(t, manifest)
}

func TestScanRejectsInvalidAliases(t *testing.T) {
	manifest := scanTree(t, map[string]string{
		".configuration":      validRoot,
		"Self.md":             frontmatter(aliases("/self", "/kept")),
		"First.md":            frontmatter(aliases("/shared")),
		"Second.md":           frontmatter(aliases("/shared")),
		"Shadow.md":           frontmatter(aliases("/live")),
		"Live.md":             "",
		"Ping.md":             frontmatter(aliases("/pong")),
		"Pong.md":             frontmatter(aliases("/ping")),
		"Dir/.configuration":  aliases("/dir"),
		"Dir/Article.md":      "",
		"Hidden.draft.md":     frontmatter(aliases("/hidden", "/live", "/shared")),
		"Bad.md":              frontmatter(aliases("https://evil.example")),
		"Nest/.configuration": aliases("/nest/old"),
		"Nest/Article.md":     "",
	})
	want := map[string]error{
		"Self.md":             ErrSelfAlias,
		"First.md":            ErrDuplicateAlias,
		"Second.md":           ErrDuplicateAlias,
		"Shadow.md":           ErrAliasShadowsPath,
		"Ping.md":             ErrRedirectLoop,
		"Pong.md":             ErrRedirectLoop,
		"Dir/.configuration":  ErrSelfAlias,
		"Hidden.draft.md":     ErrSelfAlias,
		"Bad.md":              content.ErrInvalidConfig,
		"Nest/.configuration": ErrRedirectLoop,
	}
	got := map[string][]error{}
	for _, validation := range manifest.Errors {
		got[validation.Path] = append(got[validation.Path], validation.Err)
	}
	for itemPath, sentinel := range want {
		if len(got[itemPath]) == 0 || !errors.Is(got[itemPath][0], sentinel) {
			t.Errorf("%s: errors %v, want %v", itemPath, got[itemPath], sentinel)
		}
	}
	if len(got) != len(want) {
		t.Errorf("errors = %v", manifest.Errors)
	}
	if hidden := got["Hidden.draft.md"]; len(hidden) != 2 || !errors.Is(hidden[1], ErrAliasShadowsPath) {
		t.Errorf("draft alias errors = %v, want self alias then shadowed path; drafts claim no alias", hidden)
	}
	if !slices.Equal(manifest.Redirects, []content.Redirect{{From: "/kept", To: "/self"}}) {
		t.Errorf("redirects = %v, want only the valid alias of a kept item", manifest.Redirects)
	}
	published := map[string]bool{}
	for _, article := range manifest.Articles {
		published[article.SourcePath] = true
	}
	for _, itemPath := range []string{"Self.md", "First.md", "Second.md", "Shadow.md", "Ping.md", "Pong.md"} {
		if !published[itemPath] {
			t.Errorf("%s was excluded; only its aliases should be", itemPath)
		}
	}
	if published["Bad.md"] {
		t.Error("Bad.md with malformed aliases was published")
	}
}

// checkRedirectInvariants checks the redirect map contract: sorted unique
// sources that are never published, published targets, and one hop, since
// no target is at or below any source.
func checkRedirectInvariants(t *testing.T, manifest *content.Manifest) {
	t.Helper()
	live := map[string]bool{}
	var walk func(*content.Node)
	walk = func(node *content.Node) {
		live[node.URLPath] = true
		for _, child := range node.Children {
			walk(child)
		}
	}
	walk(manifest.Root)
	for i, r := range manifest.Redirects {
		if i > 0 && manifest.Redirects[i-1].From >= r.From {
			t.Fatalf("redirects not sorted and unique at %d: %v", i, manifest.Redirects)
		}
		if _, err := content.ParseAlias(r.From); err != nil || live[r.From] || !live[r.To] {
			t.Fatalf("redirect %+v: source must be an unpublished alias and target published", r)
		}
		for _, other := range manifest.Redirects {
			if r.To == other.From || (other.Prefix && strings.HasPrefix(r.To, other.From+"/")) {
				t.Fatalf("redirect %+v reaches redirect %+v: chain not resolved", r, other)
			}
		}
	}
}

// TestScanRedirectProperties scans random trees whose items alias random
// paths, including live, draft and each other's, and checks the redirect map
// invariants and that every alias is either published or reported.
func TestScanRedirectProperties(t *testing.T) {
	rng := rand.New(rand.NewPCG(50, 50))
	words := []string{"a", "b", "c", "d"}
	randomPath := func() string {
		var segments []string
		for range 1 + rng.IntN(3) {
			segments = append(segments, words[rng.IntN(len(words))])
		}
		return "/" + strings.Join(segments, "/")
	}
	for trial := range 500 {
		tree := map[string]string{".configuration": validRoot}
		for range 1 + rng.IntN(8) {
			source := strings.TrimPrefix(randomPath(), "/")
			var claimed []string
			for range rng.IntN(3) {
				if alias := randomPath(); !slices.Contains(claimed, alias) {
					claimed = append(claimed, alias)
				}
			}
			config := ""
			if len(claimed) > 0 {
				config = aliases(claimed...)
			}
			switch rng.IntN(3) {
			case 0:
				tree[source+"/.configuration"] = config
			case 1:
				tree[source+".draft.md"] = frontmatter(config)
			default:
				tree[source+".md"] = frontmatter(config)
			}
		}
		t.Run(fmt.Sprint(trial), func(t *testing.T) {
			manifest := scanTree(t, tree)
			checkRedirectInvariants(t, manifest)
			declared := map[string]bool{}
			for _, r := range manifest.Redirects {
				declared[r.From] = true
			}
			reported := map[string]int{}
			for _, validation := range manifest.Errors {
				reported[validation.Path]++
			}
			for _, node := range append(slices.Clone(manifest.Articles), directories(manifest.Root)...) {
				missing := 0
				for _, alias := range node.Aliases {
					if node.Draft || !declared[alias.String()] {
						missing++
					}
				}
				if node.Draft && reported[aliasSource(node)] > missing || !node.Draft && reported[aliasSource(node)] != missing {
					t.Fatalf("%s: %d unpublished aliases of %v, %d reported; tree %v errors %v",
						node.SourcePath, missing, node.Aliases, reported[aliasSource(node)], tree, manifest.Errors)
				}
			}
		})
	}
}

func directories(node *content.Node) []*content.Node {
	var found []*content.Node
	for _, child := range node.Children {
		if child.Kind == content.KindDirectory {
			found = append(found, child)
			found = append(found, directories(child)...)
		}
	}
	return found
}
