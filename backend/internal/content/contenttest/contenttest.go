// Package contenttest provides content.Source implementations for tests.
package contenttest

import (
	"context"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/reingma/Unframed/backend/internal/content"
)

// MemSource is an in-memory content.Source keyed by path. It applies the same
// content-path contract as content.FSSource, verified by the Source contract test.
type MemSource map[string]string

func (m MemSource) Files(ctx context.Context) ([]content.File, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var files []content.File
	for name, data := range m {
		if content.IsContentPath(name) {
			files = append(files, content.File{Path: name, Size: int64(len(data)), ModTime: time.Unix(0, 0)})
		}
	}
	return files, nil
}

func (m MemSource) ReadFile(ctx context.Context, name string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !content.IsContentPath(name) {
		return nil, content.ErrInvalidPath
	}
	data, ok := m[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return []byte(data), nil
}

// WriteTree writes files into a new temporary directory and returns its path.
func WriteTree(t testing.TB, files map[string]string) string {
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
	return dir
}

// OpenFS opens dir as a content.FSSource closed at test cleanup.
func OpenFS(t testing.TB, dir string) *content.FSSource {
	t.Helper()
	source, err := content.OpenFS(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = source.Close() })
	return source
}

// NewFS writes files to a temporary directory and opens it as a content.FSSource.
func NewFS(t testing.TB, files map[string]string) *content.FSSource {
	t.Helper()
	return OpenFS(t, WriteTree(t, files))
}

// Sources returns every content.Source implementation loaded with files.
func Sources(t testing.TB, files map[string]string) map[string]content.Source {
	t.Helper()
	return map[string]content.Source{"fs": NewFS(t, files), "memory": MemSource(files)}
}

// WithMaxAliases returns a copy of files in which every directory
// .configuration and every article below the root declares
// content.MaxAliases former paths, for measuring alias-heavy builds.
func WithMaxAliases(files map[string]string) map[string]string {
	aliased := maps.Clone(files)
	for name, data := range files {
		dir, file := path.Split(name)
		if dir == "" {
			continue
		}
		slug := strings.ToLower(strings.TrimSuffix(name, ".md"))
		if file == content.ConfigurationFile {
			slug = strings.ToLower(strings.TrimSuffix(dir, "/"))
		}
		paths := make([]string, 0, content.MaxAliases)
		for k := range content.MaxAliases {
			paths = append(paths, fmt.Sprintf("%q", fmt.Sprintf("/old/%s-%d", slug, k)))
		}
		line := "aliases = [" + strings.Join(paths, ", ") + "]\n"
		if file == content.ConfigurationFile {
			aliased[name] = line + data
		} else {
			aliased[name] = "+++\n" + line + strings.TrimPrefix(data, "+++\n")
		}
	}
	return aliased
}
