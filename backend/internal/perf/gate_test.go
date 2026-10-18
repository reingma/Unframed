package perf

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The gate tests exercise the real contract in real git repositories: a
// scratch repo per test, with commits standing in for merged history and the
// change under review.

func gitRun(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// commitFiles writes name→content files (creating directories), commits
// everything and returns the new HEAD SHA.
func commitFiles(t *testing.T, dir string, files map[string]string, msg string) string {
	t.Helper()
	for name, content := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	gitRun(t, dir, "add", "-A")
	gitRun(t, dir, "commit", "-qm", msg)
	return gitRun(t, dir, "rev-parse", "HEAD")
}

// commitRecord marshals r, writes it under its canonical file name and
// commits it alone — the record commit convention the gate assumes.
func commitRecord(t *testing.T, dir string, r HistoryRecord) string {
	t.Helper()
	data, err := MarshalHistoryRecord(r)
	if err != nil {
		t.Fatal(err)
	}
	return commitFiles(t, dir, map[string]string{
		"perf/history/" + RecordFileName(r): string(data),
	}, "record "+RecordFileName(r))
}

func gateRecord(seq int, rev, tag, machine string) HistoryRecord {
	r := testRecord(seq, rev, tag)
	r.Environment.Machine = machine
	return r
}

// gateFixture returns a repo whose base commit holds one canonical record
// (sequence 1) measuring the initial code commit, mirroring merged history.
func gateFixture(t *testing.T) (dir, base string) {
	t.Helper()
	dir = t.TempDir()
	gitRun(t, dir, "init", "-q")
	measured := commitFiles(t, dir, map[string]string{"app.txt": "v1"}, "seed")
	base = commitRecord(t, dir, gateRecord(1, measured, "BASE", CanonicalMachine))
	return dir, base
}

func gateFailures(res GateResult) string {
	return strings.Join(res.Failures, "\n")
}

func TestGateCanonicalRecordPass(t *testing.T) {
	dir, base := gateFixture(t)
	measured := commitFiles(t, dir, map[string]string{"app.txt": "v2"}, "feature")
	head := commitRecord(t, dir, gateRecord(2, measured, "PR", CanonicalMachine))

	res, err := CheckGate(context.Background(), dir, "perf/history", base, head)
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK() {
		t.Fatalf("expected pass, failures: %s", gateFailures(res))
	}
	if len(res.Added) != 1 || !strings.HasPrefix(res.Added[0], "0002-") {
		t.Fatalf("added %v", res.Added)
	}
	if res.Newest != res.Added[0] || len(res.Warnings) != 0 {
		t.Fatalf("newest %q warnings %v", res.Newest, res.Warnings)
	}
}

func TestGateFailsWithoutAddedRecord(t *testing.T) {
	dir, base := gateFixture(t)
	head := commitFiles(t, dir, map[string]string{"app.txt": "v2"}, "feature")

	res, err := CheckGate(context.Background(), dir, "perf/history", base, head)
	if err != nil {
		t.Fatal(err)
	}
	if res.OK() {
		t.Fatal("expected failure for a recordless change")
	}
	if !strings.Contains(gateFailures(res), "record added") {
		t.Fatalf("expected a missing-record failure, got: %s", gateFailures(res))
	}
}

func TestGateFailsOnStaleRevision(t *testing.T) {
	dir, base := gateFixture(t)
	measured := commitFiles(t, dir, map[string]string{"app.txt": "v2"}, "feature")
	commitRecord(t, dir, gateRecord(2, measured, "PR", CanonicalMachine))
	// A push after measuring — even a tiny one — invalidates the record.
	head := commitFiles(t, dir, map[string]string{"app.txt": "v3"}, "review fix")

	res, err := CheckGate(context.Background(), dir, "perf/history", base, head)
	if err != nil {
		t.Fatal(err)
	}
	if res.OK() {
		t.Fatal("expected failure for a record whose revision is not the head content")
	}
	if !strings.Contains(gateFailures(res), "0002-") || !strings.Contains(gateFailures(res), "app.txt") {
		t.Fatalf("expected a stale-revision failure naming the record and drift, got: %s", gateFailures(res))
	}
}

func TestGateFailsOnModifiedOrDeletedBaseRecord(t *testing.T) {
	for _, tc := range []struct {
		name   string
		delete bool
	}{
		{"modified", false},
		{"deleted", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir, base := gateFixture(t)
			entries, err := os.ReadDir(filepath.Join(dir, "perf/history"))
			if err != nil {
				t.Fatal(err)
			}
			baseName := entries[0].Name()
			measured := commitFiles(t, dir, map[string]string{"app.txt": "v2"}, "feature")
			if tc.delete {
				gitRun(t, dir, "rm", "-q", filepath.Join("perf/history", baseName))
				gitRun(t, dir, "commit", "-qm", "remove record")
			} else {
				p := filepath.Join(dir, "perf/history", baseName)
				if err := os.WriteFile(p, []byte("tampered"), 0o644); err != nil {
					t.Fatal(err)
				}
				gitRun(t, dir, "add", "-A")
				gitRun(t, dir, "commit", "-qm", "tamper record")
			}
			head := commitRecord(t, dir, gateRecord(2, measured, "PR", CanonicalMachine))

			res, err := CheckGate(context.Background(), dir, "perf/history", base, head)
			if err != nil {
				t.Fatal(err)
			}
			if res.OK() || !strings.Contains(gateFailures(res), "immutable") {
				t.Fatalf("expected an immutability failure, got: %s", gateFailures(res))
			}
		})
	}
}

func TestGateFailsOnBrokenHistoryAtHead(t *testing.T) {
	dir, base := gateFixture(t)
	// File name and embedded sequence/revision must agree; here the record's
	// revision does not match the file's <sha8> part.
	r := gateRecord(2, strings.Repeat("1", 40), "PR", CanonicalMachine)
	data, err := MarshalHistoryRecord(r)
	if err != nil {
		t.Fatal(err)
	}
	head := commitFiles(t, dir, map[string]string{
		"perf/history/0002-22222222.json": string(data),
	}, "bad record")

	res, err := CheckGate(context.Background(), dir, "perf/history", base, head)
	if err != nil {
		t.Fatal(err)
	}
	if res.OK() || !strings.Contains(gateFailures(res), "invalid") {
		t.Fatalf("expected a name/sequence failure, got: %s", gateFailures(res))
	}
}

func TestGateMultiRecordAppend(t *testing.T) {
	dir, base := gateFixture(t)
	// Backfill of an old merge plus this change's own record, committed
	// together: the gate permits multiple appended records.
	measured := commitFiles(t, dir, map[string]string{"app.txt": "v2"}, "feature")
	backfill := gateRecord(2, strings.Repeat("a", 40), "OLD", CanonicalMachine)
	own := gateRecord(3, measured, "PR", CanonicalMachine)
	b0, err := MarshalHistoryRecord(backfill)
	if err != nil {
		t.Fatal(err)
	}
	b1, err := MarshalHistoryRecord(own)
	if err != nil {
		t.Fatal(err)
	}
	head := commitFiles(t, dir, map[string]string{
		"perf/history/" + RecordFileName(backfill): string(b0),
		"perf/history/" + RecordFileName(own):      string(b1),
	}, "records")

	res, err := CheckGate(context.Background(), dir, "perf/history", base, head)
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK() {
		t.Fatalf("expected pass, failures: %s", gateFailures(res))
	}
	if len(res.Added) != 2 || res.Newest != RecordFileName(own) {
		t.Fatalf("added %v newest %q", res.Added, res.Newest)
	}
}

func TestGateNonCanonicalMachineWarns(t *testing.T) {
	dir, base := gateFixture(t)
	measured := commitFiles(t, dir, map[string]string{"app.txt": "v2"}, "feature")
	head := commitRecord(t, dir, gateRecord(2, measured, "PR", "cloud-agent"))

	res, err := CheckGate(context.Background(), dir, "perf/history", base, head)
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK() {
		t.Fatalf("non-canonical record must not fail the gate, failures: %s", gateFailures(res))
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], CanonicalMachine) {
		t.Fatalf("expected a non-canonical warning, got %v", res.Warnings)
	}
}

func TestGateFailsOnUnresolvableMeasuredRevision(t *testing.T) {
	dir, base := gateFixture(t)
	commitFiles(t, dir, map[string]string{"app.txt": "v2"}, "feature")
	// A revision that is not a commit in this repository cannot be verified.
	head := commitRecord(t, dir, gateRecord(2, strings.Repeat("b", 40), "PR", CanonicalMachine))

	res, err := CheckGate(context.Background(), dir, "perf/history", base, head)
	if err != nil {
		t.Fatal(err)
	}
	if res.OK() || !strings.Contains(gateFailures(res), "resolve") {
		t.Fatalf("expected an unverifiable-revision failure, got: %s", gateFailures(res))
	}
}
