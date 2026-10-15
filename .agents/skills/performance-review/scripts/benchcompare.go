// Command benchcompare records Go benchmark runs with identifying metadata and
// compares a baseline run with a current run through a pinned benchstat.
//
//	benchcompare run [-rev REV] [-dir DIR] [-bench RE] [-count N] -o FILE [packages or files]
//	benchcompare compare [-min-count N] [-o DIR] BASE CURRENT
//
// A zero exit from compare means a comparison was produced, not that the
// performance is acceptable. Invalid, failed or incomparable inputs exit 1.
package main

import (
	"bytes"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

const benchstatModule = "golang.org/x/perf/cmd/benchstat@v0.0.0-20260825160852-19be9d8e6c70"

// metadataKeys are written by run and ignored by benchstat so that runs of
// different revisions stay in one comparison table.
var metadataKeys = []string{"revision", "revision-ref", "command", "go-version", "goexperiment", "date", "exit-code"}

// machineKeys must match for two runs to be comparable.
var machineKeys = []string{"goos", "goarch", "cpu", "pkg"}

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	var err error
	switch os.Args[1] {
	case "run":
		err = runCmd(os.Args[2:], os.Stdout)
	case "compare":
		err = compareCmd(os.Args[2:], os.Stdout)
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "benchcompare:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: benchcompare run|compare [flags] ...")
	os.Exit(2)
}

type runConfig struct {
	repo, rev, dir, bench, out string
	count                      int
	targets                    []string
}

func runCmd(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	var cfg runConfig
	fs.StringVar(&cfg.rev, "rev", "", "git revision to measure in a temporary worktree; empty measures the working tree")
	fs.StringVar(&cfg.dir, "dir", ".", "directory relative to the repository root in which to run go test")
	fs.StringVar(&cfg.bench, "bench", ".", "benchmark regular expression")
	fs.IntVar(&cfg.count, "count", 10, "repetitions of each benchmark")
	fs.StringVar(&cfg.out, "o", "", "output file (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if cfg.out == "" {
		return errors.New("run: -o is required")
	}
	cfg.targets = fs.Args()
	if len(cfg.targets) == 0 {
		cfg.targets = []string{"./..."}
	}
	msg, err := run(cfg)
	if msg != "" {
		fmt.Fprintln(stdout, msg)
	}
	return err
}

func run(cfg runConfig) (string, error) {
	out, err := filepath.Abs(cfg.out)
	if err != nil {
		return "", err
	}
	root, err := git(cfg.repo, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	workRoot, revision, ref := root, "", cfg.rev
	if cfg.rev != "" {
		if revision, err = git(root, "rev-parse", "--verify", cfg.rev+"^{commit}"); err != nil {
			return "", err
		}
		tmp, err := os.MkdirTemp("", "benchcompare-")
		if err != nil {
			return "", err
		}
		workRoot = filepath.Join(tmp, "tree")
		if _, err := git(root, "worktree", "add", "--detach", workRoot, revision); err != nil {
			os.RemoveAll(tmp)
			return "", err
		}
		defer func() {
			git(root, "worktree", "remove", "--force", workRoot)
			os.RemoveAll(tmp)
		}()
	} else {
		ref = "working-tree"
		if revision, err = git(root, "rev-parse", "HEAD"); err != nil {
			return "", err
		}
		if status, err := git(root, "status", "--porcelain"); err != nil {
			return "", err
		} else if status != "" {
			revision += "-dirty"
		}
	}

	dir := filepath.Join(workRoot, cfg.dir)
	goArgs := append([]string{"test", "-run", "^$", "-bench", cfg.bench, "-benchmem", "-count", strconv.Itoa(cfg.count)}, cfg.targets...)
	version, err := command(dir, "go", "env", "GOVERSION")
	if err != nil {
		return "", err
	}
	var output bytes.Buffer
	cmd := exec.Command("go", goArgs...)
	cmd.Dir = dir
	cmd.Stdout, cmd.Stderr = &output, &output
	runErr := cmd.Run()
	exitCode := 0
	if exitErr, ok := errors.AsType[*exec.ExitError](runErr); ok {
		exitCode = exitErr.ExitCode()
	} else if runErr != nil {
		return "", runErr
	}

	experiment := os.Getenv("GOEXPERIMENT")
	if experiment == "" {
		experiment = "none"
	}
	var file bytes.Buffer
	fmt.Fprintf(&file, "revision: %s\nrevision-ref: %s\ncommand: go %s (in %s)\ngo-version: %s\ngoexperiment: %s\ndate: %s\nexit-code: %d\n",
		revision, ref, strings.Join(goArgs, " "), cfg.dir, version, experiment, time.Now().UTC().Format(time.RFC3339), exitCode)
	file.Write(output.Bytes())
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(out, file.Bytes(), 0o644); err != nil {
		return "", err
	}
	if exitCode != 0 {
		return "", fmt.Errorf("go test exited %d; output kept in %s", exitCode, out)
	}
	parsed, err := parseRun(out)
	if err != nil {
		return "", err
	}
	if len(parsed.samples) == 0 {
		return "", fmt.Errorf("no benchmark matched %q; output kept in %s", cfg.bench, out)
	}
	return fmt.Sprintf("wrote %s: %d benchmarks at %s", out, len(parsed.samples), revision), nil
}

type benchRun struct {
	path    string
	config  map[string]string
	samples map[string]int
	failure string
}

var configLine = regexp.MustCompile(`^([a-z][^\s:]*): (.*)$`)

func parseRun(path string) (*benchRun, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	r := &benchRun{path: path, config: map[string]string{}, samples: map[string]int{}}
	for line := range strings.Lines(string(data)) {
		line = strings.TrimRight(line, "\r\n")
		switch {
		case line == "FAIL", strings.HasPrefix(line, "FAIL\t"), strings.HasPrefix(line, "--- FAIL"), strings.HasPrefix(line, "panic: "):
			if r.failure == "" {
				r.failure = line
			}
		case strings.HasPrefix(line, "Benchmark"):
			if f := strings.Fields(line); len(f) >= 4 {
				if _, err := strconv.Atoi(f[1]); err == nil {
					r.samples[f[0]]++
				}
			}
		default:
			if m := configLine.FindStringSubmatch(line); m != nil {
				r.config[m[1]] = m[2]
			}
		}
	}
	return r, nil
}

func compareCmd(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("compare", flag.ContinueOnError)
	minCount := fs.Int("min-count", 6, "minimum samples per benchmark in each run")
	outDir := fs.String("o", "", "directory for summary.txt and benchstat.txt")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		return errors.New("compare: need BASE and CURRENT files")
	}
	summary, table, err := compare(fs.Arg(0), fs.Arg(1), *minCount)
	if err != nil {
		return err
	}
	if *outDir != "" {
		if err := os.MkdirAll(*outDir, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(*outDir, "benchstat.txt"), []byte(table), 0o644); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(*outDir, "summary.txt"), []byte(summary), 0o644); err != nil {
			return err
		}
	}
	_, err = io.WriteString(stdout, table+"\n"+summary)
	return err
}

// compare validates both runs and returns a summary and the benchstat table.
func compare(basePath, currentPath string, minCount int) (string, string, error) {
	base, err := loadValid(basePath, "baseline")
	if err != nil {
		return "", "", err
	}
	current, err := loadValid(currentPath, "current")
	if err != nil {
		return "", "", err
	}
	for _, k := range machineKeys {
		b, okB := base.config[k]
		c, okC := current.config[k]
		if okB && okC && b != c {
			return "", "", fmt.Errorf("incomparable runs: %s is %q in the baseline and %q in the current run", k, b, c)
		}
	}
	var few []string
	for _, r := range []*benchRun{base, current} {
		for name, n := range r.samples {
			if n < minCount {
				few = append(few, fmt.Sprintf("%s has %d samples in %s", name, n, r.path))
			}
		}
	}
	if len(few) > 0 {
		slices.Sort(few)
		return "", "", fmt.Errorf("too few samples (minimum %d): %s", minCount, strings.Join(few, "; "))
	}

	ignore := []string{"-ignore", strings.Join(metadataKeys, ",")}
	table, err := benchstat(append(ignore, basePath, currentPath)...)
	if err != nil {
		return "", "", err
	}
	csvOut, err := benchstat(append(ignore, "-format", "csv", basePath, currentPath)...)
	if err != nil {
		return "", "", err
	}
	changed, unchanged, err := classify(csvOut)
	if err != nil {
		return "", "", err
	}

	var s strings.Builder
	fmt.Fprintf(&s, "Baseline: %s (revision %s, %s)\n", basePath, orUnknown(base.config["revision"]), orUnknown(base.config["go-version"]))
	fmt.Fprintf(&s, "Current:  %s (revision %s, %s)\n", currentPath, orUnknown(current.config["revision"]), orUnknown(current.config["go-version"]))
	if b, c := base.config["go-version"], current.config["go-version"]; b != c {
		fmt.Fprintf(&s, "Toolchains differ (%s vs %s): the toolchain is part of the change.\n", orUnknown(b), orUnknown(c))
	}
	section(&s, "Significant changes; assess each against the changelog and diff", changed)
	section(&s, "No significant difference; inconclusive, not proof of equality", unchanged)
	section(&s, "New workloads with no baseline", onlyIn(current, base))
	section(&s, "Removed workloads", onlyIn(base, current))
	return s.String(), table, nil
}

func loadValid(path, role string) (*benchRun, error) {
	r, err := parseRun(path)
	if err != nil {
		return nil, fmt.Errorf("%s run unavailable: %w", role, err)
	}
	if r.failure != "" {
		return nil, fmt.Errorf("%s run %s contains a failure: %q", role, path, r.failure)
	}
	if code, ok := r.config["exit-code"]; ok && code != "0" {
		return nil, fmt.Errorf("%s run %s recorded exit code %s", role, path, code)
	}
	if len(r.samples) == 0 {
		return nil, fmt.Errorf("%s run %s has no benchmark results", role, path)
	}
	return r, nil
}

func benchstat(args ...string) (string, error) {
	name, full := "go", append([]string{"run", benchstatModule}, args...)
	if bin := os.Getenv("BENCHSTAT"); bin != "" {
		name, full = bin, args
	}
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(name, full...)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("benchstat failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

// classify reads benchstat CSV rows: name, base, CI, current, CI, delta, P.
func classify(out string) (changed, unchanged []string, err error) {
	var table strings.Builder
	for line := range strings.Lines(out) {
		if !configLine.MatchString(strings.TrimRight(line, "\r\n")) {
			table.WriteString(line)
		}
	}
	r := csv.NewReader(strings.NewReader(table.String()))
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("unreadable benchstat output: %w", err)
	}
	unit := ""
	for _, row := range rows {
		if len(row) < 7 && row[0] != "" && (len(row) < 4 || row[1] == "" || row[3] == "") {
			continue // present in one run only; listed as a new or removed workload
		}
		if len(row) != 7 {
			return nil, nil, fmt.Errorf("unexpected benchstat row %q; compare only two runs of one configuration", row)
		}
		if row[0] == "" {
			if row[5] == "vs base" {
				unit = row[1]
			}
			continue
		}
		if row[0] == "geomean" || row[1] == "" || row[3] == "" {
			continue
		}
		line := fmt.Sprintf("%s %s: %s (%s; base ±%s, current ±%s)", unit, row[0], row[5], row[6], row[2], row[4])
		if row[5] == "~" {
			unchanged = append(unchanged, line)
		} else {
			changed = append(changed, line)
		}
	}
	if len(changed)+len(unchanged) == 0 {
		return nil, nil, errors.New("benchstat compared no benchmark present in both runs")
	}
	return changed, unchanged, nil
}

func onlyIn(a, b *benchRun) []string {
	var names []string
	for name := range a.samples {
		if _, ok := b.samples[name]; !ok {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

func section(s *strings.Builder, title string, lines []string) {
	fmt.Fprintf(s, "\n%s:\n", title)
	if len(lines) == 0 {
		s.WriteString("  none\n")
	}
	for _, l := range lines {
		fmt.Fprintf(s, "  %s\n", l)
	}
}

func orUnknown(v string) string {
	if v == "" {
		return "unknown"
	}
	return v
}

func git(dir string, args ...string) (string, error) {
	return command(dir, "git", args...)
}

func command(dir, name string, args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}
