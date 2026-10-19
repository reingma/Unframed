package event

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
)

var (
	// ErrReportInsideContent reports a report directory inside the content
	// root, where report files would become content.
	ErrReportInsideContent = errors.New("report directory is inside the content root")
	// ErrInvalidBuildID reports a build ID that is not safe as a file name
	// component.
	ErrInvalidBuildID = errors.New("invalid build ID")
)

// ReportPublisher writes each batch as a JSON report file in a directory
// outside the content tree.
type ReportPublisher struct {
	dir string
}

// NewReportPublisher returns a publisher writing to dir, creating it if
// needed. Both paths are resolved through symbolic links, and dir must not be
// contentRoot or inside it; a rejected dir is not created.
func NewReportPublisher(dir, contentRoot string) (*ReportPublisher, error) {
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
		return nil, fmt.Errorf("%w: %s is inside %s", ErrReportInsideContent, resolvedDir, resolvedRoot)
	}
	if err := os.MkdirAll(resolvedDir, 0o755); err != nil {
		return nil, fmt.Errorf("create report directory: %w", err)
	}
	return &ReportPublisher{dir: resolvedDir}, nil
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

// Publish writes build-<time>-<build ID>.json atomically: readers see either
// no file or the complete report.
func (p *ReportPublisher) Publish(ctx context.Context, batch Batch) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !validBuildID(batch.BuildID) {
		return fmt.Errorf("%w: %q", ErrInvalidBuildID, batch.BuildID)
	}
	data, err := json.Marshal(batch, jsontext.WithIndent("  "))
	if err != nil {
		return fmt.Errorf("encode build report: %w", err)
	}
	name := fmt.Sprintf("build-%s-%s.json", batch.Time.UTC().Format("20060102T150405.000000000Z"), batch.BuildID)
	tmp, err := os.CreateTemp(p.dir, ".build-*.tmp")
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
	if err := os.Rename(tmp.Name(), filepath.Join(p.dir, name)); err != nil {
		return fmt.Errorf("write build report: %w", err)
	}
	return nil
}
