// Package report is a local event subscriber that writes one JSON report file
// per build, in a directory outside the content tree.
package report

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/reingma/Unframed/backend/internal/event"
)

var (
	// ErrInsideContent reports a report directory inside the content root,
	// where report files would become content.
	ErrInsideContent = errors.New("report directory is inside the content root")
	// ErrInvalidBuildID reports a build ID that is not safe as a file name
	// component, or events from more than one build.
	ErrInvalidBuildID = errors.New("invalid build ID")
)

// Subscriber writes the events of each build as a JSON array in a report
// file.
type Subscriber struct {
	dir string
}

// New returns a subscriber writing to dir, creating it if
// needed. Both paths are resolved through symbolic links, and dir must not be
// contentRoot or inside it; a rejected dir is not created.
func New(dir, contentRoot string) (*Subscriber, error) {
	resolvedDir, err := resolve(dir)
	if err != nil {
		return nil, fmt.Errorf("resolve report directory: %w", err)
	}
	resolvedRoot, err := resolve(contentRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve content root: %w", err)
	}
	rel, err := filepath.Rel(resolvedRoot, resolvedDir)
	if err != nil {
		return nil, fmt.Errorf("compare report directory with content root: %w", err)
	}
	if rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("%w: %s is inside %s", ErrInsideContent, resolvedDir, resolvedRoot)
	}
	if err := os.MkdirAll(resolvedDir, 0o755); err != nil {
		return nil, fmt.Errorf("create report directory: %w", err)
	}
	return &Subscriber{dir: resolvedDir}, nil
}

// resolve returns the absolute, symlink-free form of path. Missing trailing
// components are kept as written, since they cannot be links yet.
func resolve(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	missing := ""
	for {
		resolved, err := filepath.EvalSymlinks(abs)
		if err == nil {
			return filepath.Join(resolved, missing), nil
		}
		parent := filepath.Dir(abs)
		if !errors.Is(err, fs.ErrNotExist) || parent == abs {
			return "", err
		}
		missing = filepath.Join(filepath.Base(abs), missing)
		abs = parent
	}
}

// validBuildID reports whether id is nonempty and contains only ASCII letters,
// digits, '-' and '_', so it cannot change the report's directory.
func validBuildID(id string) bool {
	if id == "" {
		return false
	}
	for _, r := range id {
		if !('a' <= r && r <= 'z' || 'A' <= r && r <= 'Z' || '0' <= r && r <= '9' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

// Receive writes build-<time>-<build ID>.json atomically: readers see either
// no file or the complete report. A build without events writes no file.
func (s *Subscriber) Receive(ctx context.Context, events []event.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}
	first := events[0]
	if !validBuildID(first.BuildID) {
		return fmt.Errorf("%w: %q", ErrInvalidBuildID, first.BuildID)
	}
	for _, e := range events[1:] {
		if e.BuildID != first.BuildID {
			return fmt.Errorf("%w: events from builds %q and %q", ErrInvalidBuildID, first.BuildID, e.BuildID)
		}
	}
	data, err := json.Marshal(events, jsontext.WithIndent("  "))
	if err != nil {
		return fmt.Errorf("encode build report: %w", err)
	}
	name := fmt.Sprintf("build-%s-%s.json", first.Time.UTC().Format("20060102T150405.000000000Z"), first.BuildID)
	tmp, err := os.CreateTemp(s.dir, ".build-*.tmp")
	if err != nil {
		return fmt.Errorf("write build report: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write build report: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write build report: %w", err)
	}
	if err := os.Rename(tmp.Name(), filepath.Join(s.dir, name)); err != nil {
		return fmt.Errorf("write build report: %w", err)
	}
	return nil
}
