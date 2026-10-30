package perf

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"time"
)

// CompressArtifact reduces a raw run artifact to per-workload sufficient
// statistics. Samples carrying an error still contribute their measured
// metric values — the attempt is real load — and are counted in Errors.
func CompressArtifact(a RunArtifact) (WorkloadStats, error) {
	ws := WorkloadStats{
		ID:        a.WorkloadID,
		Kind:      a.Kind,
		Toolchain: a.Toolchain,
		Samples:   len(a.Samples),
		Metrics:   map[string]MetricStats{},
	}
	byMetric := map[string][]float64{}
	for _, s := range a.Samples {
		if s.Error != "" {
			ws.Errors++
		}
		for name, v := range s.Metrics {
			byMetric[name] = append(byMetric[name], v)
		}
	}
	if len(byMetric) == 0 {
		return WorkloadStats{}, fmt.Errorf("workload %q artifact has no metric values", a.WorkloadID)
	}
	for name, values := range byMetric {
		stats, err := ComputeStats(values)
		if err != nil {
			return WorkloadStats{}, fmt.Errorf("workload %q metric %q: %w", a.WorkloadID, name, err)
		}
		ws.Metrics[name] = stats
	}
	return ws, nil
}

// BuildRecord compresses one run's artifacts into a history record. All
// artifacts must share one host environment and one content hash; mixing
// machines or measured content inside a record is rejected rather than
// pooled. The record takes the artifacts' content hash. Each workload keeps its own toolchain, so
// Go and browser producers can contribute to the same record.
func BuildRecord(artifacts []RunArtifact, revision, tag string, now time.Time) (HistoryRecord, error) {
	if len(artifacts) == 0 {
		return HistoryRecord{}, fmt.Errorf("no artifacts to compress")
	}
	if !revisionPattern.MatchString(revision) {
		return HistoryRecord{}, fmt.Errorf("revision %q is not a 40-digit hex SHA", revision)
	}
	if tag == "" {
		return HistoryRecord{}, fmt.Errorf("tag is required (changelog or issue tag)")
	}
	env := artifacts[0].Environment
	content := artifacts[0].ContentHash
	workloads := make([]WorkloadStats, 0, len(artifacts))
	for _, a := range artifacts {
		if !a.Environment.Equal(env) {
			return HistoryRecord{}, fmt.Errorf("artifact %q has a different environment; refusing to pool", a.WorkloadID)
		}
		if a.ContentHash != content {
			return HistoryRecord{}, fmt.Errorf("artifact %q measured content %s, but %q measured %s; refusing to pool",
				a.WorkloadID, a.ContentHash, artifacts[0].WorkloadID, content)
		}
		ws, err := CompressArtifact(a)
		if err != nil {
			return HistoryRecord{}, err
		}
		workloads = append(workloads, ws)
	}
	slices.SortFunc(workloads, func(a, b WorkloadStats) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	return HistoryRecord{
		Version:     SchemaVersion,
		Revision:    revision,
		ContentHash: content,
		Tag:         tag,
		RecordedAt:  now.UTC(),
		Environment: env,
		Workloads:   workloads,
	}, nil
}

// LoadRunDir reads every run artifact in a run directory.
func LoadRunDir(dir string) ([]RunArtifact, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no run artifacts in %s", dir)
	}
	slices.Sort(paths)
	artifacts := make([]RunArtifact, 0, len(paths))
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		a, err := ParseRunArtifact(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		artifacts = append(artifacts, a)
	}
	return artifacts, nil
}

var runDirPattern = regexp.MustCompile(`^\d{8}T\d{6}$`)

// LatestRunDir returns the newest run directory under runsDir by name.
// Run directories are UTC timestamped, so lexical order is chronological;
// stray non-run directories are ignored.
func LatestRunDir(runsDir string) (string, error) {
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		return "", err
	}
	latest := ""
	for _, e := range entries {
		if e.IsDir() && runDirPattern.MatchString(e.Name()) && e.Name() > latest {
			latest = e.Name()
		}
	}
	if latest == "" {
		return "", fmt.Errorf("no run directories in %s", runsDir)
	}
	return filepath.Join(runsDir, latest), nil
}

// RecordFileName is the committed file name for a record: a zero-padded
// sequence plus the short revision, ordered lexically by sequence.
func RecordFileName(r HistoryRecord) string {
	return fmt.Sprintf("%04d-%s.json", r.Sequence, r.Revision[:8])
}

var recordFilePattern = regexp.MustCompile(`^(\d{4,})-([0-9a-f]{8})\.json$`)

// historyEntry is one committed record file as read from the worktree or a
// git revision: display names the source for error messages, name is the
// base file name and data is the raw JSON.
type historyEntry struct {
	display string
	name    string
	data    []byte
}

// parseHistoryEntries validates raw record files: each name must match
// NNNN-<sha8>.json and agree with the embedded sequence and revision, and
// sequences may not repeat. Records are returned sorted by sequence.
func parseHistoryEntries(entries []historyEntry) ([]HistoryRecord, error) {
	records := make([]HistoryRecord, 0, len(entries))
	seen := map[int]bool{}
	for _, e := range entries {
		m := recordFilePattern.FindStringSubmatch(e.name)
		if m == nil {
			return nil, fmt.Errorf("%s: file name does not match NNNN-<sha8>.json", e.display)
		}
		r, err := ParseHistoryRecord(e.data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.display, err)
		}
		fileSeq, _ := strconv.Atoi(m[1])
		if fileSeq != r.Sequence || m[2] != r.Revision[:8] {
			return nil, fmt.Errorf("%s: file name does not match embedded sequence/revision", e.display)
		}
		if seen[r.Sequence] {
			return nil, fmt.Errorf("%s: duplicate sequence %d", e.display, r.Sequence)
		}
		seen[r.Sequence] = true
		records = append(records, r)
	}
	slices.SortFunc(records, func(a, b HistoryRecord) int {
		return a.Sequence - b.Sequence
	})
	return records, nil
}

// LoadHistory reads and validates every committed record, sorted by
// sequence. File name and embedded sequence must agree.
func LoadHistory(dir string) ([]HistoryRecord, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	slices.Sort(paths)
	entries := make([]historyEntry, 0, len(paths))
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		entries = append(entries, historyEntry{display: p, name: filepath.Base(p), data: data})
	}
	return parseHistoryEntries(entries)
}

// WriteRecord assigns the next sequence and writes a record under dir in its
// canonical form. It returns the record as written (with its sequence) and
// the committed file path.
func WriteRecord(dir string, r HistoryRecord) (HistoryRecord, string, error) {
	records, err := LoadHistory(dir)
	if err != nil {
		return HistoryRecord{}, "", err
	}
	maxSeq := 0
	for _, existing := range records {
		maxSeq = max(maxSeq, existing.Sequence)
	}
	r.Sequence = maxSeq + 1
	if err := r.validate(); err != nil {
		return HistoryRecord{}, "", err
	}
	data, err := MarshalHistoryRecord(r)
	if err != nil {
		return HistoryRecord{}, "", err
	}
	path := filepath.Join(dir, RecordFileName(r))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return HistoryRecord{}, "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return HistoryRecord{}, "", err
	}
	return r, path, nil
}
