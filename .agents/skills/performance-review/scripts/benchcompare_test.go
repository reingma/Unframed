package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const machine = "goos: linux\ngoarch: amd64\ncpu: Test CPU\npkg: example.com/b\n"

func writeRun(t *testing.T, name, header string, benches map[string][]float64) string {
	t.Helper()
	var b strings.Builder
	b.WriteString(header)
	for bench, values := range benches {
		for _, v := range values {
			fmt.Fprintf(&b, "%s-8\t1000000\t%.1f ns/op\t0 B/op\t0 allocs/op\n", bench, v)
		}
	}
	b.WriteString("PASS\nok\texample.com/b\t1.0s\n")
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func spread(start float64) []float64 {
	values := make([]float64, 8)
	for i := range values {
		values[i] = start + float64(i)
	}
	return values
}

// sectionText returns the summary lines listed under the heading starting with title.
func sectionText(t *testing.T, summary, title string) string {
	t.Helper()
	_, rest, ok := strings.Cut(summary, "\n"+title)
	if !ok {
		t.Fatalf("summary has no %q section:\n%s", title, summary)
	}
	body, _, _ := strings.Cut(rest, "\n\n")
	return body
}

func mustCompare(t *testing.T, base, current string) string {
	t.Helper()
	summary, table, err := compare(base, current, 6)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(table, "vs base") {
		t.Fatalf("benchstat table missing:\n%s", table)
	}
	return summary
}

func TestCompareReportsMeaningfulRegression(t *testing.T) {
	base := writeRun(t, "base.txt", machine, map[string][]float64{"BenchmarkLookup": spread(100)})
	current := writeRun(t, "current.txt", machine, map[string][]float64{"BenchmarkLookup": spread(150)})
	summary := mustCompare(t, base, current)
	if got := sectionText(t, summary, "Significant changes"); !strings.Contains(got, "sec/op Lookup-8: +") {
		t.Fatalf("50%% slowdown not reported as significant:\n%s", summary)
	}
}

func TestCompareLeavesFeatureCostForContextualAssessment(t *testing.T) {
	base := writeRun(t, "base.txt", machine, map[string][]float64{"BenchmarkLookup": spread(100)})
	current := writeRun(t, "current.txt", machine, map[string][]float64{
		"BenchmarkLookup": spread(106),
		"BenchmarkRender": spread(500),
	})
	summary := mustCompare(t, base, current)
	if got := sectionText(t, summary, "Significant changes"); !strings.Contains(got, "Lookup-8: +") {
		t.Fatalf("small significant cost hidden:\n%s", summary)
	}
	if got := sectionText(t, summary, "New workloads with no baseline"); !strings.Contains(got, "BenchmarkRender-8") {
		t.Fatalf("new workload not listed without a baseline:\n%s", summary)
	}
}

func TestCompareTreatsNoiseAsInconclusive(t *testing.T) {
	base := writeRun(t, "base.txt", machine, map[string][]float64{"BenchmarkLookup": {100, 180, 90, 150, 110, 170, 95, 160}})
	current := writeRun(t, "current.txt", machine, map[string][]float64{"BenchmarkLookup": {105, 175, 92, 155, 115, 165, 98, 158}})
	summary := mustCompare(t, base, current)
	if got := sectionText(t, summary, "Significant changes"); strings.Contains(got, "Lookup") {
		t.Fatalf("noise reported as a change:\n%s", summary)
	}
	if got := sectionText(t, summary, "No significant difference"); !strings.Contains(got, "sec/op Lookup-8: ~") {
		t.Fatalf("noise not reported as inconclusive:\n%s", summary)
	}
}

func TestCompareRejectsInvalidInputs(t *testing.T) {
	good := writeRun(t, "good.txt", machine, map[string][]float64{"BenchmarkLookup": spread(100)})
	tests := map[string]struct {
		base string
		want string
	}{
		"missing baseline":         {filepath.Join(t.TempDir(), "absent.txt"), "baseline run unavailable"},
		"baseline without results": {writeRun(t, "empty.txt", machine, nil), "no benchmark results"},
		"failed benchmark":         {writeRun(t, "fail.txt", machine+"--- FAIL: BenchmarkLookup-8\nFAIL\n", map[string][]float64{"BenchmarkLookup": spread(100)}), "failure"},
		"nonzero exit":             {writeRun(t, "exit.txt", "exit-code: 1\n"+machine, map[string][]float64{"BenchmarkLookup": spread(100)}), "exit code 1"},
		"different machine":        {writeRun(t, "cpu.txt", strings.Replace(machine, "Test CPU", "Other CPU", 1), map[string][]float64{"BenchmarkLookup": spread(100)}), "incomparable"},
		"too few samples":          {writeRun(t, "few.txt", machine, map[string][]float64{"BenchmarkLookup": {100, 101, 102}}), "too few samples"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			summary, _, err := compare(tt.base, good, 6)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got err %v, want %q; summary:\n%s", err, tt.want, summary)
			}
			if summary != "" {
				t.Fatalf("comparison produced despite invalid input:\n%s", summary)
			}
		})
	}
}

func TestRunRecordsRevisionAndKeepsFailures(t *testing.T) {
	repo := t.TempDir()
	files := map[string]string{
		"go.mod": "module example.com/b\n\ngo 1.26\n",
		"b_test.go": `package b

import "testing"

func BenchmarkOK(b *testing.B) {
	for b.Loop() {
	}
}

func BenchmarkBad(b *testing.B) { b.Fatal("broken workload") }
`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(repo, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	gitT(t, repo, "init", "-q")
	gitT(t, repo, "add", ".")
	gitT(t, repo, "-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "-m", "init")
	head := gitT(t, repo, "rev-parse", "HEAD")

	out := filepath.Join(t.TempDir(), "ok.txt")
	if _, err := run(runConfig{repo: repo, rev: "HEAD", dir: ".", bench: "BenchmarkOK", count: 2, out: out, targets: []string{"."}}); err != nil {
		t.Fatal(err)
	}
	r, err := parseRun(out)
	if err != nil {
		t.Fatal(err)
	}
	if r.config["revision"] != head || r.config["exit-code"] != "0" || !hasSamples(r, "BenchmarkOK", 2) {
		t.Fatalf("unexpected run record: config %v, samples %v", r.config, r.samples)
	}
	if list := gitT(t, repo, "worktree", "list"); strings.Count(list, "\n") != 0 {
		t.Fatalf("temporary worktree left behind:\n%s", list)
	}

	bad := filepath.Join(t.TempDir(), "bad.txt")
	if _, err := run(runConfig{repo: repo, dir: ".", bench: "BenchmarkBad", count: 2, out: bad, targets: []string{"."}}); err == nil {
		t.Fatal("failed benchmark reported as success")
	}
	if _, _, err := compare(bad, out, 2); err == nil {
		t.Fatal("failed run accepted as a baseline")
	}
}

func gitT(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func hasSamples(r *benchRun, prefix string, n int) bool {
	for name, got := range r.samples {
		if strings.HasPrefix(name, prefix) && got == n {
			return true
		}
	}
	return false
}
