package perf

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const testRev = "e461931122f0d9f2b1a3c4e5f60718293a4b5c6d"

func TestCompressArtifactGolden(t *testing.T) {
	a := testArtifact() // latencies 1000, 2000, 3000; one error
	ws, err := CompressArtifact(a)
	if err != nil {
		t.Fatal(err)
	}
	if ws.ID != "backend-healthcheck" || ws.Kind != KindHTTP || ws.Samples != 3 || ws.Errors != 1 {
		t.Fatalf("got %+v", ws)
	}
	s, ok := ws.Metrics["latency_ns"]
	if !ok {
		t.Fatal("latency_ns metric missing")
	}
	if s.Values != 3 || !almostEqual(s.Mean, 2000) || !almostEqual(s.Variance, 1e6) {
		t.Fatalf("got %+v", s)
	}
}

func TestBuildRecordRejectsMixedEnvironments(t *testing.T) {
	a := testArtifact()
	b := testArtifact()
	b.WorkloadID = "other"
	b.Environment.Machine = "other-machine"
	if _, err := BuildRecord([]RunArtifact{a, b}, testRev, "ENG-30", time.Now()); err == nil {
		t.Error("expected mixed-environment rejection")
	}
}

func TestBuildRecordRequiresTagAndRevision(t *testing.T) {
	a := testArtifact()
	if _, err := BuildRecord([]RunArtifact{a}, testRev, "", time.Now()); err == nil {
		t.Error("expected error for empty tag")
	}
	if _, err := BuildRecord([]RunArtifact{a}, "deadbeef", "ENG-30", time.Now()); err == nil {
		t.Error("expected error for short revision")
	}
	if _, err := BuildRecord(nil, testRev, "ENG-30", time.Now()); err == nil {
		t.Error("expected error for no artifacts")
	}
}

func TestWriteAndLoadHistory(t *testing.T) {
	dir := t.TempDir()
	records := []HistoryRecord{
		testRecord(0, "e461931122f0d9f2b1a3c4e5f60718293a4b5c6d", "ENG-30"),
		testRecord(0, "537af8d3aabbccddee11223344556677889900aa", "ENG-31"),
	}
	var paths []string
	for i, r := range records {
		written, p, err := WriteRecord(dir, r)
		if err != nil {
			t.Fatal(err)
		}
		if written.Sequence != i+1 {
			t.Fatalf("assigned sequence %d, want %d", written.Sequence, i+1)
		}
		paths = append(paths, p)
	}
	if filepath.Base(paths[0]) != "0001-e4619311.json" ||
		filepath.Base(paths[1]) != "0002-537af8d3.json" {
		t.Fatalf("unexpected file names %v", paths)
	}
	got, err := LoadHistory(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Sequence != 1 || got[1].Sequence != 2 ||
		got[0].Tag != "ENG-30" || got[1].Tag != "ENG-31" {
		t.Fatalf("got %+v", got)
	}
}

func TestLoadHistoryRejectsBadFileName(t *testing.T) {
	dir := t.TempDir()
	data, err := MarshalHistoryRecord(testRecord(1, testRev, "ENG-30"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "badname.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadHistory(dir); err == nil {
		t.Error("expected error for non-conforming file name")
	}
	if err := os.WriteFile(filepath.Join(dir, "0009-e4619311.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadHistory(dir); err == nil {
		t.Error("expected error for name/sequence mismatch")
	}
}

func TestLatestRunDir(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"20260101T000000", "20260202T000000"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	got, err := LatestRunDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "20260202T000000" {
		t.Fatalf("got %s", got)
	}
}
