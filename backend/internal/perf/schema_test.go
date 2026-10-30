package perf

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func testEnvironment() Environment {
	return Environment{
		Machine: "workstation-1",
		OS:      "linux",
		Arch:    "amd64",
		CPU:     "Test CPU",
	}
}

func testArtifact() RunArtifact {
	return RunArtifact{
		Version:     SchemaVersion,
		WorkloadID:  "backend-healthcheck",
		Kind:        KindHTTP,
		StartedAt:   time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC),
		FinishedAt:  time.Date(2026, 10, 15, 12, 0, 10, 0, time.UTC),
		Environment: testEnvironment(),
		Toolchain:   Toolchain{"go": "go1.27.2"},
		ContentHash: testContent,
		Samples: []Sample{
			{Metrics: map[string]float64{"latency_ns": 1000}, Status: 200},
			{Metrics: map[string]float64{"latency_ns": 2000}, Status: 200},
			{Metrics: map[string]float64{"latency_ns": 3000}, Status: 500, Error: "500"},
		},
	}
}

func TestRunArtifactRoundTrip(t *testing.T) {
	want := testArtifact()
	data, err := MarshalRunArtifact(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseRunArtifact(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.WorkloadID != want.WorkloadID || got.Kind != want.Kind ||
		!got.StartedAt.Equal(want.StartedAt) || !got.FinishedAt.Equal(want.FinishedAt) ||
		got.Environment != want.Environment || !got.Toolchain.Equal(want.Toolchain) ||
		len(got.Samples) != len(want.Samples) {
		t.Fatalf("round trip mismatch: %+v", got)
	}
	for i, s := range got.Samples {
		if s.Status != want.Samples[i].Status || s.Error != want.Samples[i].Error ||
			s.Metrics["latency_ns"] != want.Samples[i].Metrics["latency_ns"] {
			t.Fatalf("sample %d mismatch: %+v", i, s)
		}
	}
}

func testStats() MetricStats {
	stats, err := ComputeStats([]float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	if err != nil {
		panic(err)
	}
	return stats
}

func testRecord(seq int, rev, tag string) HistoryRecord {
	return HistoryRecord{
		Version:     SchemaVersion,
		Sequence:    seq,
		Revision:    rev,
		ContentHash: testContent,
		Tag:         tag,
		RecordedAt:  time.Date(2026, 10, 15, 12, 30, 0, 0, time.UTC),
		Environment: testEnvironment(),
		Workloads: []WorkloadStats{{
			ID:        "backend-healthcheck",
			Kind:      KindHTTP,
			Toolchain: Toolchain{"go": "go1.27.2"},
			Samples:   10,
			Errors:    0,
			Metrics:   map[string]MetricStats{"latency_ns": testStats()},
		}},
	}
}

// testContent is a content hash; the scratch-repo tests compute real ones.
const testContent = "2a66ce5990e1d520265d13ed0b712dacc43cb7b8"

func TestParseHistoryRecordKeepsV2(t *testing.T) {
	v2 := testRecord(1, "e461931122f0d9f2b1a3c4e5f60718293a4b5c6d", "ENG-30")
	v2.Version, v2.ContentHash = legacyRecordVersion, ""
	data, err := MarshalHistoryRecord(v2)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "content_hash") {
		t.Fatalf("a version 2 record must marshal without content_hash:\n%s", data)
	}
	got, err := ParseHistoryRecord(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != legacyRecordVersion || got.ContentHash != "" {
		t.Fatalf("got version %d content %q", got.Version, got.ContentHash)
	}
}

func TestHistoryRecordRoundTrip(t *testing.T) {
	want := testRecord(1, "e461931122f0d9f2b1a3c4e5f60718293a4b5c6d", "ENG-30")
	data, err := MarshalHistoryRecord(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseHistoryRecord(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.Sequence != want.Sequence || got.Revision != want.Revision || got.Tag != want.Tag ||
		!got.RecordedAt.Equal(want.RecordedAt) || got.Environment != want.Environment ||
		len(got.Workloads) != 1 {
		t.Fatalf("round trip mismatch: %+v", got)
	}
	gotStats := got.Workloads[0].Metrics["latency_ns"]
	wantStats := want.Workloads[0].Metrics["latency_ns"]
	if !reflect.DeepEqual(gotStats, wantStats) {
		t.Fatalf("stats mismatch: got %+v want %+v", gotStats, wantStats)
	}
}

func TestParseRunArtifactRejects(t *testing.T) {
	valid, err := MarshalRunArtifact(testArtifact())
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"unknown member": strings.Replace(string(valid), `"version": 3`, `"version": 3, "extra": true`, 1),
		"bad version":    strings.Replace(string(valid), `"version": 3`, `"version": 2`, 1),
		"no hash":        strings.Replace(string(valid), testContent, ``, 1),
		"short hash":     strings.Replace(string(valid), testContent, testContent[:39], 1),
		"upper hash":     strings.Replace(string(valid), testContent, strings.ToUpper(testContent), 1),
		"empty id":       strings.Replace(string(valid), `"workload_id": "backend-healthcheck"`, `"workload_id": ""`, 1),
		"bad id shape":   strings.Replace(string(valid), `"workload_id": "backend-healthcheck"`, `"workload_id": "Has Space"`, 1),
		"bad kind":       strings.Replace(string(valid), `"kind": "http"`, `"kind": "rpc"`, 1),
		"empty machine":  strings.Replace(string(valid), `"machine": "workstation-1"`, `"machine": ""`, 1),
		"no toolchain":   strings.Replace(string(valid), `"go": "go1.27.2"`, ``, 1),
		"bad component":  strings.Replace(string(valid), `"go": "go1.27.2"`, `"Go": "go1.27.2"`, 1),
		"empty version":  strings.Replace(string(valid), `"go": "go1.27.2"`, `"go": ""`, 1),
	}
	for name, doc := range cases {
		if _, err := ParseRunArtifact([]byte(doc)); err == nil {
			t.Errorf("%s: expected parse error", name)
		}
	}

	empty := testArtifact()
	empty.Samples[0].Metrics = nil
	emptyDoc, err := MarshalRunArtifact(empty)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseRunArtifact(emptyDoc); err == nil {
		t.Error("sample without metrics: expected parse error")
	}
}

func TestParseHistoryRecordRejects(t *testing.T) {
	valid, err := MarshalHistoryRecord(testRecord(1, "e461931122f0d9f2b1a3c4e5f60718293a4b5c6d", "ENG-30"))
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"unknown member":   strings.Replace(string(valid), `"version": 3`, `"version": 3, "extra": 1`, 1),
		"bad version":      strings.Replace(string(valid), `"version": 3`, `"version": 9`, 1),
		"v3 without hash":  strings.Replace(string(valid), `"content_hash": "`+testContent+`",`, ``, 1),
		"v2 with hash":     strings.Replace(string(valid), `"version": 3`, `"version": 2`, 1),
		"non-hex hash":     strings.Replace(string(valid), testContent, "g"+testContent[1:], 1),
		"no toolchain":     strings.Replace(string(valid), `"go": "go1.27.2"`, ``, 1),
		"bad revision":     strings.Replace(string(valid), `"e461931122f0d9f2b1a3c4e5f60718293a4b5c6d"`, `"notasha"`, 1),
		"empty tag":        strings.Replace(string(valid), `"tag": "ENG-30"`, `"tag": ""`, 1),
		"zero sequence":    strings.Replace(string(valid), `"sequence": 1`, `"sequence": 0`, 1),
		"bad workload id":  strings.Replace(string(valid), `"id": "backend-healthcheck"`, `"id": "Has Space"`, 1),
		"unknown quantile": strings.Replace(string(valid), `"p99": 10`, `"p99": 10, "p999": 10`, 1),
	}
	for name, doc := range cases {
		if _, err := ParseHistoryRecord([]byte(doc)); err == nil {
			t.Errorf("%s: expected parse error", name)
		}
	}

	dup := testRecord(1, "e461931122f0d9f2b1a3c4e5f60718293a4b5c6d", "ENG-30")
	dup.Workloads = append(dup.Workloads, dup.Workloads[0])
	dupDoc, err := MarshalHistoryRecord(dup)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseHistoryRecord(dupDoc); err == nil {
		t.Error("duplicate workload ids: expected parse error")
	}
}

// v1RecordDoc is a version 1 record as ENG-30 committed it: one
// record-level Go toolchain string.
const v1RecordDoc = `{
  "version": 1,
  "sequence": 1,
  "revision": "ed2c70da4c5e420fbbb1a9e2fa45155c38ca1d3f",
  "tag": "ENG-30",
  "recorded_at": "2026-10-15T20:11:22Z",
  "environment": {"machine": "workstation-1", "os": "linux", "arch": "amd64", "cpu": "Test CPU", "toolchain": "go1.27.2"},
  "workloads": [{
    "id": "backend-healthcheck", "kind": "http", "samples": 2, "errors": 0,
    "metrics": {"latency_ns": {"values": 2, "mean": 1.5, "variance": 0.5, "min": 1, "max": 2,
      "quantiles": {"p50": 1, "p90": 2, "p95": 2, "p99": 2},
      "histogram": {"min": 1, "bucket_width": 0.5, "counts": [1, 1]}}}
  }]
}`

func TestParseHistoryRecordUpgradesV1(t *testing.T) {
	r, err := ParseHistoryRecord([]byte(v1RecordDoc))
	if err != nil {
		t.Fatal(err)
	}
	if r.Version != legacyRecordVersion || r.ContentHash != "" || r.Environment != testEnvironment() {
		t.Fatalf("got version %d environment %+v", r.Version, r.Environment)
	}
	if !r.Workloads[0].Toolchain.Equal(Toolchain{"go": "go1.27.2"}) {
		t.Fatalf("got toolchain %v", r.Workloads[0].Toolchain)
	}
	// The upgraded record compares against current records without a
	// spurious toolchain warning.
	current := testRecord(2, "537af8d3aabbccddee11223344556677889900aa", "ENG-31")
	if c := CompareRecords("v1", r, "v2", current); !c.EnvMatch || len(c.ToolchainChanges) != 0 {
		t.Fatalf("got env match %v toolchain changes %+v", c.EnvMatch, c.ToolchainChanges)
	}

	for name, doc := range map[string]string{
		"v1 without toolchain":    strings.Replace(v1RecordDoc, `, "toolchain": "go1.27.2"`, ``, 1),
		"v1 with v2 toolchain":    strings.Replace(v1RecordDoc, `"kind": "http",`, `"kind": "http", "toolchain": {"go": "go1.27.2"},`, 1),
		"v2 with v1 toolchain":    strings.Replace(v1RecordDoc, `"version": 1`, `"version": 2`, 1),
		"v1 with unknown member":  strings.Replace(v1RecordDoc, `"version": 1`, `"version": 1, "extra": 1`, 1),
		"v1 with build kind type": strings.Replace(v1RecordDoc, `"kind": "http"`, `"kind": "rpc"`, 1),
	} {
		if _, err := ParseHistoryRecord([]byte(doc)); err == nil {
			t.Errorf("%s: expected parse error", name)
		}
	}
}

func FuzzParseRunArtifact(f *testing.F) {
	valid, err := MarshalRunArtifact(testArtifact())
	if err != nil {
		f.Fatal(err)
	}
	f.Add(valid)
	f.Add([]byte(`{}`))
	f.Add([]byte(`not json`))
	f.Fuzz(func(t *testing.T, data []byte) {
		a, err := ParseRunArtifact(data)
		if err != nil {
			return
		}
		out, err := MarshalRunArtifact(a)
		if err != nil {
			t.Fatalf("accepted artifact does not marshal: %v", err)
		}
		if _, err := ParseRunArtifact(out); err != nil {
			t.Fatalf("canonical artifact does not reparse: %v", err)
		}
	})
}

func FuzzParseHistoryRecord(f *testing.F) {
	valid, err := MarshalHistoryRecord(testRecord(1, "e461931122f0d9f2b1a3c4e5f60718293a4b5c6d", "ENG-30"))
	if err != nil {
		f.Fatal(err)
	}
	f.Add(valid)
	f.Add([]byte(v1RecordDoc))
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Fuzz(func(t *testing.T, data []byte) {
		r, err := ParseHistoryRecord(data)
		if err != nil {
			return
		}
		out, err := MarshalHistoryRecord(r)
		if err != nil {
			t.Fatalf("accepted record does not marshal: %v", err)
		}
		if _, err := ParseHistoryRecord(out); err != nil {
			t.Fatalf("canonical record does not reparse: %v", err)
		}
	})
}
