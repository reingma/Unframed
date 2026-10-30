// Package perf implements the shared performance measurement machinery:
// the workload registry, raw run artifacts, compressed history records and
// their analysis. Raw artifacts are ephemeral and gitignored; history records
// are committed data. The JSON formats are versioned and consumable by
// non-Go producers.
package perf

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"
	"time"
)

// SchemaVersion is the current version of the run-artifact and
// history-record JSON formats. Bump it only for incompatible shape changes.
// Version 3 added content_hash. Version 1 and 2 history records remain
// readable; see ParseHistoryRecord.
const SchemaVersion = 3

// legacyRecordVersion is the last record version without content_hash.
const legacyRecordVersion = 2

// RegistryVersion is the current version of the workload registry format.
const RegistryVersion = 2

// Workload kinds in the registry.
const (
	KindHTTP    = "http"
	KindBrowser = "browser"
	KindBuild   = "build"
)

// Environment is the host factor a run or record belongs to. Machine is an
// opaque label, never a hostname, so records carry no identifying system
// data. OS and Arch use Go's GOOS/GOARCH names for every producer.
type Environment struct {
	Machine string `json:"machine"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
	CPU     string `json:"cpu"`
}

func (e Environment) validate() error {
	for name, value := range map[string]string{
		"machine": e.Machine, "os": e.OS, "arch": e.Arch, "cpu": e.CPU,
	} {
		if value == "" {
			return fmt.Errorf("environment field %q is empty", name)
		}
	}
	return nil
}

// Equal reports whether two environments are the same factor. Comparisons
// across different environments must be flagged, not silently pooled.
func (e Environment) Equal(o Environment) bool {
	return e == o
}

// Toolchain names the software versions that produced one workload's
// measurements, keyed by component (for example "go", "node", "chromium").
// It is per workload because producers differ: one record can hold Go and
// browser workloads measured on the same host.
type Toolchain map[string]string

func (t Toolchain) validate() error {
	if len(t) == 0 {
		return fmt.Errorf("toolchain is empty")
	}
	for name, version := range t {
		if !workloadIDPattern.MatchString(name) {
			return fmt.Errorf("toolchain component %q must match %s", name, workloadIDPattern)
		}
		if version == "" {
			return fmt.Errorf("toolchain component %q has an empty version", name)
		}
	}
	return nil
}

// Equal reports whether two toolchains name the same components and versions.
func (t Toolchain) Equal(o Toolchain) bool {
	return maps.Equal(t, o)
}

// String renders the toolchain as sorted name=version pairs.
func (t Toolchain) String() string {
	pairs := make([]string, 0, len(t))
	for _, name := range slices.Sorted(maps.Keys(t)) {
		pairs = append(pairs, name+"="+t[name])
	}
	return strings.Join(pairs, " ")
}

// Sample is one measured observation in a raw run artifact. Metrics carries
// the measured values (for example "latency_ns"); Error non-empty marks a
// failed observation; Status is an optional kind-specific diagnostic such as
// an HTTP response code.
type Sample struct {
	Metrics map[string]float64 `json:"metrics"`
	Status  int                `json:"status,omitempty"`
	Error   string             `json:"error,omitempty"`
}

// RunArtifact is the ephemeral per-workload output of one measurement run.
// It is regenerable by re-running the workload at the same git state.
type RunArtifact struct {
	Version     int         `json:"version"`
	WorkloadID  string      `json:"workload_id"`
	Kind        string      `json:"kind"`
	StartedAt   time.Time   `json:"started_at"`
	FinishedAt  time.Time   `json:"finished_at"`
	Environment Environment `json:"environment"`
	Toolchain   Toolchain   `json:"toolchain"`
	ContentHash string      `json:"content_hash"`
	Samples     []Sample    `json:"samples"`
}

// Histogram is an equal-width bucket layout over [Min, Max] of a metric.
type Histogram struct {
	Min         float64 `json:"min"`
	BucketWidth float64 `json:"bucket_width"`
	Counts      []int   `json:"counts"`
}

// MetricStats holds the sufficient statistics for one metric: count, mean,
// sample variance, quantiles and a histogram. Together they allow deltas and
// confidence intervals without resampling raw data.
type MetricStats struct {
	Values    int                `json:"values"`
	Mean      float64            `json:"mean"`
	Variance  float64            `json:"variance"`
	Min       float64            `json:"min"`
	Max       float64            `json:"max"`
	Quantiles map[string]float64 `json:"quantiles"`
	Histogram Histogram          `json:"histogram"`
}

// WorkloadStats is the compressed form of one workload inside a record.
type WorkloadStats struct {
	ID        string                 `json:"id"`
	Kind      string                 `json:"kind"`
	Toolchain Toolchain              `json:"toolchain"`
	Samples   int                    `json:"samples"`
	Errors    int                    `json:"errors"`
	Metrics   map[string]MetricStats `json:"metrics"`
}

// HistoryRecord is the committed, compressed summary of one measurement run.
// One record is committed per merge to main under perf/history/.
// ContentHash (see ContentHash) binds the record to the measured content;
// Revision is the measured commit, informational only because a squash or
// rebase merge can drop it from main. Version 1 and 2 records have no
// ContentHash.
type HistoryRecord struct {
	Version     int             `json:"version"`
	Sequence    int             `json:"sequence"`
	Revision    string          `json:"revision"`
	ContentHash string          `json:"content_hash,omitzero"`
	Tag         string          `json:"tag"`
	RecordedAt  time.Time       `json:"recorded_at"`
	Environment Environment     `json:"environment"`
	Workloads   []WorkloadStats `json:"workloads"`
}

func validKind(kind string) bool {
	return kind == KindHTTP || kind == KindBrowser || kind == KindBuild
}

func validSample(s Sample) error {
	if len(s.Metrics) == 0 {
		return fmt.Errorf("sample has no metrics")
	}
	for name, v := range s.Metrics {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("metric %q has non-finite value", name)
		}
	}
	return nil
}

func (a *RunArtifact) validate() error {
	switch {
	case a.Version != SchemaVersion:
		return fmt.Errorf("unsupported artifact version %d", a.Version)
	case !contentHashPattern.MatchString(a.ContentHash):
		return fmt.Errorf("content_hash %q is not a git object ID", a.ContentHash)
	case !workloadIDPattern.MatchString(a.WorkloadID):
		return fmt.Errorf("workload_id %q must match %s", a.WorkloadID, workloadIDPattern)
	case !validKind(a.Kind):
		return fmt.Errorf("invalid kind %q", a.Kind)
	case a.StartedAt.IsZero():
		return fmt.Errorf("started_at is missing")
	case a.FinishedAt.Before(a.StartedAt):
		return fmt.Errorf("finished_at is before started_at")
	}
	if err := a.Environment.validate(); err != nil {
		return err
	}
	if err := a.Toolchain.validate(); err != nil {
		return err
	}
	for i, s := range a.Samples {
		if err := validSample(s); err != nil {
			return fmt.Errorf("sample %d: %w", i, err)
		}
	}
	return nil
}

// ParseRunArtifact decodes and validates a raw run artifact. Unknown members
// are rejected: version changes gate shape changes.
func ParseRunArtifact(data []byte) (RunArtifact, error) {
	var a RunArtifact
	if err := json.Unmarshal(data, &a, json.RejectUnknownMembers(true)); err != nil {
		return RunArtifact{}, err
	}
	if err := a.validate(); err != nil {
		return RunArtifact{}, err
	}
	return a, nil
}

// MarshalRunArtifact encodes an artifact in its canonical indented form.
func MarshalRunArtifact(a RunArtifact) ([]byte, error) {
	return json.Marshal(a, jsontext.WithIndent("  "))
}

func (s MetricStats) validate() error {
	switch {
	case s.Values <= 0:
		return fmt.Errorf("values must be positive, got %d", s.Values)
	case math.IsNaN(s.Mean) || math.IsInf(s.Mean, 0):
		return fmt.Errorf("mean is not finite")
	case s.Variance < 0 || math.IsNaN(s.Variance) || math.IsInf(s.Variance, 0):
		return fmt.Errorf("variance must be finite and non-negative")
	case s.Min > s.Max:
		return fmt.Errorf("min %v exceeds max %v", s.Min, s.Max)
	}
	for key := range s.Quantiles {
		if !quantileKeys[key] {
			return fmt.Errorf("unknown quantile %q", key)
		}
	}
	for _, key := range []string{"p50", "p90", "p95", "p99"} {
		v, ok := s.Quantiles[key]
		if !ok {
			return fmt.Errorf("quantile %q is missing", key)
		}
		if v < s.Min || v > s.Max {
			return fmt.Errorf("quantile %q value %v outside [%v, %v]", key, v, s.Min, s.Max)
		}
	}
	if !(s.Quantiles["p50"] <= s.Quantiles["p90"] &&
		s.Quantiles["p90"] <= s.Quantiles["p95"] &&
		s.Quantiles["p95"] <= s.Quantiles["p99"]) {
		return fmt.Errorf("quantiles are not ordered")
	}
	h := s.Histogram
	if len(h.Counts) == 0 || h.BucketWidth <= 0 {
		return fmt.Errorf("histogram is empty")
	}
	total := 0
	for _, c := range h.Counts {
		if c < 0 {
			return fmt.Errorf("histogram has a negative count")
		}
		total += c
	}
	if total != s.Values {
		return fmt.Errorf("histogram counts sum to %d, want %d", total, s.Values)
	}
	return nil
}

func (r *HistoryRecord) validate() error {
	switch {
	case r.Version != SchemaVersion && r.Version != legacyRecordVersion:
		return fmt.Errorf("unsupported record version %d", r.Version)
	case r.Version == SchemaVersion && !contentHashPattern.MatchString(r.ContentHash):
		return fmt.Errorf("content_hash %q is not a git object ID", r.ContentHash)
	case r.Version == legacyRecordVersion && r.ContentHash != "":
		return fmt.Errorf("version %d records have no content_hash", r.Version)
	case r.Sequence <= 0:
		return fmt.Errorf("sequence must be positive, got %d", r.Sequence)
	case !revisionPattern.MatchString(r.Revision):
		return fmt.Errorf("revision %q is not a 40-digit hex SHA", r.Revision)
	case r.Tag == "":
		return fmt.Errorf("tag is empty")
	case r.RecordedAt.IsZero():
		return fmt.Errorf("recorded_at is missing")
	}
	if err := r.Environment.validate(); err != nil {
		return err
	}
	if len(r.Workloads) == 0 {
		return fmt.Errorf("record has no workloads")
	}
	seen := map[string]bool{}
	for _, w := range r.Workloads {
		switch {
		case !workloadIDPattern.MatchString(w.ID):
			return fmt.Errorf("workload id %q must match %s", w.ID, workloadIDPattern)
		case seen[w.ID]:
			return fmt.Errorf("duplicate workload id %q", w.ID)
		case !validKind(w.Kind):
			return fmt.Errorf("workload %q has invalid kind %q", w.ID, w.Kind)
		case w.Samples <= 0:
			return fmt.Errorf("workload %q has no samples", w.ID)
		case w.Errors < 0 || w.Errors > w.Samples:
			return fmt.Errorf("workload %q has invalid error count %d", w.ID, w.Errors)
		case len(w.Metrics) == 0:
			return fmt.Errorf("workload %q has no metrics", w.ID)
		}
		if err := w.Toolchain.validate(); err != nil {
			return fmt.Errorf("workload %q: %w", w.ID, err)
		}
		seen[w.ID] = true
		for name, m := range w.Metrics {
			if err := m.validate(); err != nil {
				return fmt.Errorf("workload %q metric %q: %w", w.ID, name, err)
			}
		}
	}
	return nil
}

// historyRecordV1 is the version 1 record shape: one record-level toolchain
// string, which only the Go runner produced.
type historyRecordV1 struct {
	Version     int       `json:"version"`
	Sequence    int       `json:"sequence"`
	Revision    string    `json:"revision"`
	Tag         string    `json:"tag"`
	RecordedAt  time.Time `json:"recorded_at"`
	Environment struct {
		Environment `json:",inline"`
		Toolchain   string `json:"toolchain"`
	} `json:"environment"`
	Workloads []struct {
		ID      string                 `json:"id"`
		Kind    string                 `json:"kind"`
		Samples int                    `json:"samples"`
		Errors  int                    `json:"errors"`
		Metrics map[string]MetricStats `json:"metrics"`
	} `json:"workloads"`
}

// upgrade converts a version 1 record to the version 2 shape. Its toolchain
// string was the Go toolchain, so it becomes each workload's "go" component.
func (v historyRecordV1) upgrade() HistoryRecord {
	r := HistoryRecord{
		Version:     legacyRecordVersion,
		Sequence:    v.Sequence,
		Revision:    v.Revision,
		Tag:         v.Tag,
		RecordedAt:  v.RecordedAt,
		Environment: v.Environment.Environment,
	}
	for _, w := range v.Workloads {
		var toolchain Toolchain
		if v.Environment.Toolchain != "" {
			toolchain = Toolchain{"go": v.Environment.Toolchain}
		}
		r.Workloads = append(r.Workloads, WorkloadStats{
			ID: w.ID, Kind: w.Kind, Toolchain: toolchain,
			Samples: w.Samples, Errors: w.Errors, Metrics: w.Metrics,
		})
	}
	return r
}

// ParseHistoryRecord decodes and validates a committed history record.
// Version 1 records are upgraded in memory to the version 2 shape, and
// version 2 records keep their version and empty ContentHash, so committed
// history is never rewritten.
func ParseHistoryRecord(data []byte) (HistoryRecord, error) {
	var header struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return HistoryRecord{}, err
	}
	var r HistoryRecord
	if header.Version == 1 {
		var v1 historyRecordV1
		if err := json.Unmarshal(data, &v1, json.RejectUnknownMembers(true)); err != nil {
			return HistoryRecord{}, err
		}
		r = v1.upgrade()
	} else if err := json.Unmarshal(data, &r, json.RejectUnknownMembers(true)); err != nil {
		return HistoryRecord{}, err
	}
	if err := r.validate(); err != nil {
		return HistoryRecord{}, err
	}
	return r, nil
}

// MarshalHistoryRecord encodes a record in its canonical indented form.
func MarshalHistoryRecord(r HistoryRecord) ([]byte, error) {
	return json.Marshal(r, jsontext.WithIndent("  "))
}
