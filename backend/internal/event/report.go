package event

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrReportInsideContent reports a report directory inside the content root,
// where report files would become content.
var ErrReportInsideContent = errors.New("report directory is inside the content root")

// ReportPublisher writes each batch as a JSON report file in a directory
// outside the content tree.
type ReportPublisher struct {
	dir string
}

// NewReportPublisher creates dir if needed and returns a publisher writing to
// it. Both paths are resolved through symbolic links, and dir must not be
// contentRoot or inside it.
func NewReportPublisher(dir, contentRoot string) (*ReportPublisher, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create report directory: %w", err)
	}
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
	return &ReportPublisher{dir: resolvedDir}, nil
}

func resolve(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)
}

// Publish writes build-<time>-<build ID>.json atomically: readers see either
// no file or the complete report.
func (p *ReportPublisher) Publish(ctx context.Context, batch Batch) error {
	if err := ctx.Err(); err != nil {
		return err
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
