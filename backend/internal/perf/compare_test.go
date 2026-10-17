package perf

import (
	"math"
	"strings"
	"testing"
)

func recordWithMean(seq int, rev, tag string, mean, variance float64, n int) HistoryRecord {
	r := testRecord(seq, rev, tag)
	stats, err := ComputeStats([]float64{mean})
	if err != nil {
		panic(err)
	}
	stats.Values = n
	stats.Mean = mean
	stats.Variance = variance
	r.Workloads[0].Metrics["latency_ns"] = stats
	return r
}

func TestCompareRecordsGolden(t *testing.T) {
	a := recordWithMean(1, "e461931122f0d9f2b1a3c4e5f60718293a4b5c6d", "ENG-29", 100, 25, 100)
	b := recordWithMean(2, "537af8d3aabbccddee11223344556677889900aa", "ENG-30", 110, 25, 100)
	c := CompareRecords("a.json", a, "b.json", b)
	if !c.EnvMatch {
		t.Error("expected environments to match")
	}
	if len(c.Deltas) != 1 {
		t.Fatalf("got %d deltas", len(c.Deltas))
	}
	d := c.Deltas[0]
	se := math.Sqrt(0.5)
	if !almostEqual(d.Delta, 10) || !almostEqual(d.CILow, 10-z95*se) || !almostEqual(d.CIHigh, 10+z95*se) {
		t.Fatalf("got %+v", d)
	}
}

func TestCompareRecordsEnvironmentMismatch(t *testing.T) {
	a := testRecord(1, "e461931122f0d9f2b1a3c4e5f60718293a4b5c6d", "ENG-29")
	b := testRecord(2, "537af8d3aabbccddee11223344556677889900aa", "ENG-30")
	b.Environment.Machine = "other-machine"
	c := CompareRecords("a", a, "b", b)
	if c.EnvMatch {
		t.Error("expected environment mismatch to be flagged")
	}
	var out strings.Builder
	if err := c.WriteText(&out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "different environments") {
		t.Error("text output does not flag the mismatch")
	}
}

func TestCompareRecordsWorkloadSets(t *testing.T) {
	a := testRecord(1, "e461931122f0d9f2b1a3c4e5f60718293a4b5c6d", "ENG-29")
	b := testRecord(2, "537af8d3aabbccddee11223344556677889900aa", "ENG-30")
	b.Workloads[0].ID = "new-endpoint"
	c := CompareRecords("a", a, "b", b)
	if len(c.Deltas) != 0 {
		t.Fatalf("unexpected deltas %+v", c.Deltas)
	}
	if len(c.OnlyInA) != 1 || c.OnlyInA[0] != "backend-healthcheck" ||
		len(c.OnlyInB) != 1 || c.OnlyInB[0] != "new-endpoint" {
		t.Fatalf("got only-in-A %v only-in-B %v", c.OnlyInA, c.OnlyInB)
	}
}

func TestCompareRecordsToolchainChange(t *testing.T) {
	a := testRecord(1, "e461931122f0d9f2b1a3c4e5f60718293a4b5c6d", "ENG-29")
	b := testRecord(2, "537af8d3aabbccddee11223344556677889900aa", "ENG-30")
	b.Workloads[0].Toolchain = Toolchain{"go": "go1.28.0"}
	c := CompareRecords("a", a, "b", b)
	if !c.EnvMatch {
		t.Error("a toolchain change must not flag the host environment")
	}
	if len(c.ToolchainChanges) != 1 || c.ToolchainChanges[0].WorkloadID != "backend-healthcheck" {
		t.Fatalf("got %+v", c.ToolchainChanges)
	}
	if len(c.Deltas) != 1 {
		t.Fatalf("toolchain change must still report deltas, got %d", len(c.Deltas))
	}
	var text, report strings.Builder
	if err := c.WriteText(&text); err != nil {
		t.Fatal(err)
	}
	if err := c.WriteHTML(&report); err != nil {
		t.Fatal(err)
	}
	for name, out := range map[string]string{"text": text.String(), "html": report.String()} {
		if !strings.Contains(out, "changed toolchain") || !strings.Contains(out, "go=go1.28.0") {
			t.Errorf("%s output does not flag the toolchain change:\n%s", name, out)
		}
	}
}
