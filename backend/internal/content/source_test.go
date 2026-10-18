package content_test

import (
	"context"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/reingma/Unframed/backend/internal/content"
	"github.com/reingma/Unframed/backend/internal/content/contenttest"
)

// TestSourceContract runs the content.Source contract against every implementation.
func TestSourceContract(t *testing.T) {
	files := map[string]string{
		".configuration":            "title = \"Site\"\n",
		"Math/.configuration":       "",
		"Math/On-Martingales.md":    "body",
		"Math/figure.png":           "png",
		".git/config":               "secret",
		"Math/.DS_Store":            "junk",
		".hidden/Visible.md":        "hidden",
		"Math/.notes/Article.md":    "hidden",
		"Math/Drafts/Idea.draft.md": "draft",
	}
	want := []string{".configuration", "Math/.configuration", "Math/Drafts/Idea.draft.md", "Math/On-Martingales.md", "Math/figure.png"}
	for name, source := range contenttest.Sources(t, files) {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			listed, err := source.Files(ctx)
			if err != nil {
				t.Fatal(err)
			}
			got := map[string]int64{}
			for _, file := range listed {
				got[file.Path] = file.Size
			}
			if keys := slices.Sorted(maps.Keys(got)); !slices.Equal(keys, want) {
				t.Fatalf("Files = %v, want %v", keys, want)
			}
			for _, name := range want {
				data, err := source.ReadFile(ctx, name)
				if err != nil || string(data) != files[name] || got[name] != int64(len(data)) {
					t.Fatalf("ReadFile(%q) = %q, %v (listed size %d)", name, data, err, got[name])
				}
			}
			if _, err := source.ReadFile(ctx, "Math/Missing.md"); !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("missing file = %v; want fs.ErrNotExist", err)
			}
			for _, name := range []string{"../escape", "/etc/passwd", "Math//On-Martingales.md", "Math/../.git/config", ".git/config", "Math/.DS_Store", ".", ""} {
				if _, err := source.ReadFile(ctx, name); !errors.Is(err, content.ErrInvalidPath) {
					t.Errorf("ReadFile(%q) = %v; want content.ErrInvalidPath", name, err)
				}
			}
			canceled, cancel := context.WithCancel(ctx)
			cancel()
			if _, err := source.Files(canceled); !errors.Is(err, context.Canceled) {
				t.Errorf("Files after cancel = %v", err)
			}
			if _, err := source.ReadFile(canceled, "Math/On-Martingales.md"); !errors.Is(err, context.Canceled) {
				t.Errorf("ReadFile after cancel = %v", err)
			}
		})
	}
}

func TestFSSourceSymlinks(t *testing.T) {
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.md"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := contenttest.WriteTree(t, map[string]string{".configuration": "", "Real.md": "real"})
	for link, target := range map[string]string{
		"Escape.md":  filepath.Join(outside, "secret.md"),
		"Inside.md":  filepath.Join(dir, "Real.md"),
		"Linked-Dir": outside,
	} {
		if err := os.Symlink(target, filepath.Join(dir, link)); err != nil {
			t.Skipf("symlinks unsupported: %v", err)
		}
	}
	source := contenttest.OpenFS(t, dir)
	listed, err := source.Files(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range listed {
		if file.Path != ".configuration" && file.Path != "Real.md" {
			t.Errorf("Files listed non-regular entry %q", file.Path)
		}
	}
	for _, name := range []string{"Escape.md", "Inside.md", "Linked-Dir/secret.md"} {
		if data, err := source.ReadFile(context.Background(), name); err == nil {
			t.Errorf("ReadFile(%q) followed a symlink: %q", name, data)
		}
	}
}

func TestOpenFSMissingRoot(t *testing.T) {
	if _, err := content.OpenFS(filepath.Join(t.TempDir(), "missing")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("OpenFS(missing) = %v", err)
	}
}
