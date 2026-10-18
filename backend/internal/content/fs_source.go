package content

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// FSSource is a Source over a local directory. Every access is confined to
// that directory with os.Root, so paths and symbolic links cannot escape it.
type FSSource struct {
	root *os.Root
}

var _ Source = (*FSSource)(nil)

// OpenFS opens dir as a content root. Close releases it.
func OpenFS(dir string) (*FSSource, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("open content root: %w", err)
	}
	return &FSSource{root: root}, nil
}

func (s *FSSource) Close() error { return s.root.Close() }

func (s *FSSource) Files(ctx context.Context) ([]File, error) {
	var files []File
	err := fs.WalkDir(s.root.FS(), ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if name == "." {
			return nil
		}
		if entry.IsDir() {
			if strings.HasPrefix(entry.Name(), ".") {
				return fs.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() || !isContentPath(name) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		files = append(files, File{Path: name, Size: info.Size(), ModTime: info.ModTime()})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list content files: %w", err)
	}
	return files, nil
}

func (s *FSSource) ReadFile(ctx context.Context, name string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !isContentPath(name) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidPath, name)
	}
	info, err := s.root.Lstat(name)
	if err != nil {
		return nil, fmt.Errorf("read content file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: %q is not a regular file", ErrInvalidPath, name)
	}
	data, err := s.root.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("read content file: %w", err)
	}
	return data, nil
}
