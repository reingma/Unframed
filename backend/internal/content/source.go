package content

import (
	"context"
	"errors"
	"io/fs"
	"path"
	"strings"
	"time"
)

// ErrInvalidPath reports a path that is not a content path of a Source.
var ErrInvalidPath = errors.New("invalid content path")

// configurationFile is the per-directory configuration file name.
const configurationFile = ".configuration"

// File describes one regular file in a content tree.
type File struct {
	// Path is slash-separated and relative to the content root, as accepted
	// by io/fs.ValidPath, such as "Mathematics/On-Martingales.md".
	Path    string
	Size    int64
	ModTime time.Time
}

// Source is a content tree: the local filesystem now, an object store later.
//
// Sources expose content paths only: regular files whose path segments are
// not hidden (dot-prefixed), except files named ".configuration". Hidden
// directories, symbolic links and other non-regular entries are omitted.
// Directories are implied by file paths, so a directory without files does
// not exist in the tree.
//
// Both methods honor cancellation and deadlines, returning the context's error.
type Source interface {
	// Files lists every content file, in no particular order.
	Files(ctx context.Context) ([]File, error)
	// ReadFile returns a file's contents. Paths that are not content paths
	// fail with ErrInvalidPath; missing files fail with fs.ErrNotExist.
	ReadFile(ctx context.Context, name string) ([]byte, error)
}

// isContentPath reports whether name is a valid path whose segments are not
// hidden, except for a final ".configuration" segment.
func isContentPath(name string) bool {
	if name == "." || !fs.ValidPath(name) {
		return false
	}
	dir, base := path.Split(name)
	if strings.HasPrefix(base, ".") && base != configurationFile {
		return false
	}
	for segment := range strings.SplitSeq(strings.TrimSuffix(dir, "/"), "/") {
		if strings.HasPrefix(segment, ".") {
			return false
		}
	}
	return true
}
