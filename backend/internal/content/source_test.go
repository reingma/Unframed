package content

import (
	"context"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

// memSource is an in-memory Source for scanner tests. It applies the same
// content-path contract as FSSource, verified by TestSourceContract.
type memSource map[string]string

func (m memSource) Files(ctx context.Context) ([]File, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var files []File
	for name, data := range m {
		if isContentPath(name) {
			files = append(files, File{Path: name, Size: int64(len(data)), ModTime: time.Unix(0, 0)})
		}
	}
	return files, nil
}

func (m memSource) ReadFile(ctx context.Context, name string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !isContentPath(name) {
		return nil, ErrInvalidPath
	}
	data, ok := m[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return []byte(data), nil
}

func newFSSource(t testing.TB, files map[string]string) *FSSource {
	t.Helper()
	dir := t.TempDir()
	for name, data := range files {
		target := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	source, err := OpenFS(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = source.Close() })
	return source
}

func sourceImplementations(t *testing.T, files map[string]string) map[string]Source {
	return map[string]Source{"fs": newFSSource(t, files), "memory": memSource(files)}
}

// TestSourceContract runs the Source contract against every implementation.
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
	for name, source := range sourceImplementations(t, files) {
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
				if _, err := source.ReadFile(ctx, name); !errors.Is(err, ErrInvalidPath) {
					t.Errorf("ReadFile(%q) = %v; want ErrInvalidPath", name, err)
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
	source := newFSSource(t, map[string]string{".configuration": "", "Real.md": "real"})
	dir := source.root.Name()
	for link, target := range map[string]string{
		"Escape.md":  filepath.Join(outside, "secret.md"),
		"Inside.md":  filepath.Join(dir, "Real.md"),
		"Linked-Dir": outside,
	} {
		if err := os.Symlink(target, filepath.Join(dir, link)); err != nil {
			t.Skipf("symlinks unsupported: %v", err)
		}
	}
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
	if _, err := OpenFS(filepath.Join(t.TempDir(), "missing")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("OpenFS(missing) = %v", err)
	}
}
