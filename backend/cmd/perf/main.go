// Command perf is the repository's performance measurement tool: it runs
// registry workloads into raw artifacts, compresses them into committed
// history records, and compares or reports on records. It runs from the
// repository root; default paths point at perf/.
package main

import (
	"context"
	"encoding/json/v2"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/reingma/Unframed/backend/internal/perf"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil {
		slog.Error("perf failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: perf <run|compress|compare|report|validate> [flags]")
	}
	switch args[0] {
	case "run":
		return cmdRun(ctx, args[1:])
	case "compress":
		return cmdCompress(ctx, args[1:])
	case "compare":
		return cmdCompare(ctx, args[1:])
	case "report":
		return cmdReport(ctx, args[1:])
	case "validate":
		return cmdValidate(ctx, args[1:])
	default:
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

func loadRegistry(path string) (perf.Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return perf.Registry{}, err
	}
	return perf.ParseRegistry(data)
}

func machineLabel(flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	if env := os.Getenv("PERF_MACHINE"); env != "" {
		return env, nil
	}
	return "", fmt.Errorf("machine label required: pass --machine or set PERF_MACHINE (an opaque label, not a hostname)")
}

func cmdRun(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	registryPath := fs.String("registry", "perf/workloads.json", "workload registry file")
	runsDir := fs.String("runs", "perf/runs", "raw artifact output root")
	runDirFlag := fs.String("run", "", "run directory to write (default: new UTC timestamp under --runs)")
	machine := fs.String("machine", "", "opaque machine label (or PERF_MACHINE)")
	only := fs.String("only", "", "run a single workload id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	label, err := machineLabel(*machine)
	if err != nil {
		return err
	}
	registry, err := loadRegistry(*registryPath)
	if err != nil {
		return err
	}
	env := perf.DetectEnvironment(label)
	runDir := *runDirFlag
	if runDir == "" {
		runDir = filepath.Join(*runsDir, time.Now().UTC().Format("20060102T150405"))
	}
	ran := false
	for _, w := range registry.Workloads {
		if *only != "" && w.ID != *only {
			continue
		}
		if w.Kind != perf.KindHTTP {
			if *only == w.ID {
				return fmt.Errorf("workload %q kind %q is reserved for the frontend runner", w.ID, w.Kind)
			}
			slog.Info("skipping non-http workload", "id", w.ID, "kind", w.Kind)
			continue
		}
		ran = true
		slog.Info("running workload", "id", w.ID, "target", w.Target)
		artifact, err := perf.RunHTTPWorkload(ctx, w, env)
		if err != nil {
			return err
		}
		data, err := perf.MarshalRunArtifact(artifact)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(runDir, 0o755); err != nil {
			return err
		}
		path := filepath.Join(runDir, w.ID+".json")
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		if _, err := f.Write(data); err != nil {
			f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
		errCount := 0
		for _, s := range artifact.Samples {
			if s.Error != "" {
				errCount++
			}
		}
		slog.Info("workload complete", "id", w.ID, "samples", len(artifact.Samples), "errors", errCount, "artifact", path)
	}
	if *only != "" && !ran {
		return fmt.Errorf("no workload matched --only %q", *only)
	}
	return nil
}

func cmdCompress(_ context.Context, args []string) error {
	fs := flag.NewFlagSet("compress", flag.ContinueOnError)
	runsDir := fs.String("runs", "perf/runs", "raw artifact root")
	runDir := fs.String("run", "", "run directory to compress (default: latest under --runs)")
	historyDir := fs.String("history", "perf/history", "committed record directory")
	tag := fs.String("tag", "", "changelog or issue tag for the record (required)")
	revision := fs.String("revision", "", "40-digit measured git SHA (default: git rev-parse HEAD)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *tag == "" {
		return fmt.Errorf("--tag is required")
	}
	dir := *runDir
	if dir == "" {
		latest, err := perf.LatestRunDir(*runsDir)
		if err != nil {
			return err
		}
		dir = latest
	}
	rev := *revision
	if rev == "" {
		out, err := exec.Command("git", "rev-parse", "HEAD").Output()
		if err != nil {
			return fmt.Errorf("resolve revision: %w (or pass --revision)", err)
		}
		rev = strings.TrimSpace(string(out))
	}
	artifacts, err := perf.LoadRunDir(dir)
	if err != nil {
		return err
	}
	slog.Info("compressing run", "dir", dir, "artifacts", len(artifacts))
	record, err := perf.BuildRecord(artifacts, rev, *tag, time.Now())
	if err != nil {
		return err
	}
	written, path, err := perf.WriteRecord(*historyDir, record)
	if err != nil {
		return err
	}
	slog.Info("record written", "path", path, "sequence", written.Sequence, "tag", written.Tag)
	return nil
}

// selectRecords resolves the pair to compare: explicit --a/--b files or the
// two most recent committed records.
func selectRecords(historyDir, aPath, bPath string) (string, perf.HistoryRecord, string, perf.HistoryRecord, error) {
	if aPath != "" && bPath != "" {
		load := func(path string) (perf.HistoryRecord, error) {
			data, err := os.ReadFile(path)
			if err != nil {
				return perf.HistoryRecord{}, err
			}
			return perf.ParseHistoryRecord(data)
		}
		a, err := load(aPath)
		if err != nil {
			return "", perf.HistoryRecord{}, "", perf.HistoryRecord{}, err
		}
		b, err := load(bPath)
		if err != nil {
			return "", perf.HistoryRecord{}, "", perf.HistoryRecord{}, err
		}
		return filepath.Base(aPath), a, filepath.Base(bPath), b, nil
	}
	if aPath != "" || bPath != "" {
		return "", perf.HistoryRecord{}, "", perf.HistoryRecord{}, fmt.Errorf("pass both --a and --b or neither")
	}
	records, err := perf.LoadHistory(historyDir)
	if err != nil {
		return "", perf.HistoryRecord{}, "", perf.HistoryRecord{}, err
	}
	if len(records) < 2 {
		return "", perf.HistoryRecord{}, "", perf.HistoryRecord{}, fmt.Errorf("need at least two records in %s", historyDir)
	}
	a, b := records[len(records)-2], records[len(records)-1]
	return perf.RecordFileName(a), a, perf.RecordFileName(b), b, nil
}

func cmdCompare(_ context.Context, args []string) error {
	fs := flag.NewFlagSet("compare", flag.ContinueOnError)
	historyDir := fs.String("history", "perf/history", "committed record directory")
	aPath := fs.String("a", "", "baseline record file (default: second-latest)")
	bPath := fs.String("b", "", "current record file (default: latest)")
	asJSON := fs.Bool("json", false, "emit the comparison as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	nameA, a, nameB, b, err := selectRecords(*historyDir, *aPath, *bPath)
	if err != nil {
		return err
	}
	cmp := perf.CompareRecords(nameA, a, nameB, b)
	if *asJSON {
		out, err := json.Marshal(cmp)
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		return nil
	}
	return cmp.WriteText(os.Stdout)
}

func cmdReport(_ context.Context, args []string) error {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	historyDir := fs.String("history", "perf/history", "committed record directory")
	aPath := fs.String("a", "", "baseline record file (default: second-latest)")
	bPath := fs.String("b", "", "current record file (default: latest)")
	out := fs.String("out", "", "output HTML file (required; keep it outside committed paths)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *out == "" {
		return fmt.Errorf("--out is required")
	}
	nameA, a, nameB, b, err := selectRecords(*historyDir, *aPath, *bPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		return err
	}
	f, err := os.Create(*out)
	if err != nil {
		return err
	}
	if err := perf.CompareRecords(nameA, a, nameB, b).WriteHTML(f); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	slog.Info("report written", "path", *out)
	return nil
}

func cmdValidate(_ context.Context, args []string) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	registryPath := fs.String("registry", "perf/workloads.json", "workload registry file")
	historyDir := fs.String("history", "perf/history", "committed record directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	registry, err := loadRegistry(*registryPath)
	if err != nil {
		return fmt.Errorf("%s: %w", *registryPath, err)
	}
	records, err := perf.LoadHistory(*historyDir)
	if err != nil {
		return err
	}
	slog.Info("validation passed", "workloads", len(registry.Workloads), "records", len(records))
	return nil
}
