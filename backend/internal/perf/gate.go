package perf

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// CanonicalMachine is the designated measurement host label. Records
// measured elsewhere are valid data but a different environment factor, so
// the gate accepts them with a warning rather than failing.
const CanonicalMachine = "workstation-1"

// GateResult is the outcome of CheckGate: every failed invariant is named in
// Failures (a non-empty list fails the gate) and every advisory is in
// Warnings. Added lists the record file names the change appends and Newest
// is the file name of the newest record at head, when the head history parses.
type GateResult struct {
	Added    []string
	Newest   string
	Failures []string
	Warnings []string
}

// OK reports whether the change satisfies every gate invariant.
func (g GateResult) OK() bool {
	return len(g.Failures) == 0
}

// fileChange is one entry of `git diff --name-status`: a single-letter
// status and the path it applies to.
type fileChange struct {
	Status string
	Path   string
}

// CheckGate enforces the per-merge history record contract for the change
// between base and head inside the git repository at repoDir:
//
//   - Committed records are immutable: the merge-base..head diff under
//     historyDir may only add files, never modify or delete existing ones.
//   - The change must append at least one record file named
//     NNNN-<sha8>.json.
//   - The history at head must satisfy the LoadHistory rules: parseable
//     records, file name matching the embedded sequence and revision, no
//     duplicate sequences.
//   - The newest record at head must be bound to what is being merged: its
//     revision must resolve to a commit whose entire content outside
//     historyDir is identical to head's — only record appends may come after
//     the measured commit. A record cannot name the commit that contains it,
//     so this content equality is the strict form of "the record measures
//     the merge"; any change pushed after measuring fails the gate.
//
// Records whose machine label is not CanonicalMachine produce warnings, not
// failures. A non-nil error means the gate itself could not run (unresolvable
// revisions, no merge base, git unavailable); invariant violations are
// reported in the result instead.
func CheckGate(ctx context.Context, repoDir, historyDir, base, head string) (GateResult, error) {
	historyDir = strings.TrimSuffix(historyDir, "/")
	baseSHA, err := resolveCommit(ctx, repoDir, base)
	if err != nil {
		return GateResult{}, fmt.Errorf("base %q: %w", base, err)
	}
	headSHA, err := resolveCommit(ctx, repoDir, head)
	if err != nil {
		return GateResult{}, fmt.Errorf("head %q: %w", head, err)
	}
	mergeBase, err := gitOutput(ctx, repoDir, "merge-base", baseSHA, headSHA)
	if err != nil {
		return GateResult{}, fmt.Errorf("no merge base between %s and %s", base, head)
	}
	var res GateResult

	changes, err := diffNameStatus(ctx, repoDir, mergeBase, headSHA, historyDir)
	if err != nil {
		return GateResult{}, err
	}
	for _, c := range changes {
		if c.Status != "A" {
			res.Failures = append(res.Failures, fmt.Sprintf(
				"%s: %s under %s; committed records are immutable (no M/D entries allowed)", c.Path, changeVerb(c.Status), historyDir))
			continue
		}
		if recordFilePattern.MatchString(filepath.Base(c.Path)) {
			res.Added = append(res.Added, filepath.Base(c.Path))
		}
	}
	if len(res.Added) == 0 {
		res.Failures = append(res.Failures, fmt.Sprintf(
			"no %s/NNNN-<sha8>.json record added; every merge must commit a record measured at the merge head", historyDir))
	}

	records, err := loadHistoryAt(ctx, repoDir, headSHA, historyDir)
	if err != nil {
		res.Failures = append(res.Failures, fmt.Sprintf("history at %s is invalid: %v", headSHA[:12], err))
		return res, nil
	}
	if len(records) == 0 {
		res.Failures = append(res.Failures, fmt.Sprintf("no records under %s at %s", historyDir, headSHA[:12]))
		return res, nil
	}
	newest := records[len(records)-1]
	res.Newest = RecordFileName(newest)

	measuredSHA, err := resolveCommit(ctx, repoDir, newest.Revision)
	if err != nil {
		res.Failures = append(res.Failures, fmt.Sprintf(
			"%s has revision %s which does not resolve to a commit; the record cannot be verified against the merged content",
			res.Newest, newest.Revision))
	} else {
		since, err := diffNameStatus(ctx, repoDir, measuredSHA, headSHA)
		if err != nil {
			return GateResult{}, err
		}
		var drift []string
		for _, c := range since {
			if c.Status != "A" || !strings.HasPrefix(c.Path, historyDir+"/") {
				drift = append(drift, c.Status+" "+c.Path)
			}
		}
		if len(drift) > 0 {
			const max = 5
			detail := strings.Join(drift[:min(len(drift), max)], ", ")
			if len(drift) > max {
				detail += fmt.Sprintf(", and %d more", len(drift)-max)
			}
			res.Failures = append(res.Failures, fmt.Sprintf(
				"%s measures %s but the merge head has drifted: %s — re-run the measurement on the designated machine and append a fresh record",
				res.Newest, newest.Revision[:8], detail))
		}
	}

	byName := map[string]HistoryRecord{}
	for _, r := range records {
		byName[RecordFileName(r)] = r
	}
	for _, name := range res.Added {
		if r, ok := byName[name]; ok && r.Environment.Machine != CanonicalMachine {
			res.Warnings = append(res.Warnings, fmt.Sprintf(
				"%s was measured on machine %q, not the canonical %q; the record is accepted but is a different environment factor",
				name, r.Environment.Machine, CanonicalMachine))
		}
	}
	return res, nil
}

func changeVerb(status string) string {
	switch status {
	case "M":
		return "modified"
	case "D":
		return "deleted"
	case "T":
		return "type-changed"
	default:
		return "changed (" + status + ")"
	}
}

// gitOutput runs a git command inside repoDir and returns trimmed stdout.
func gitOutput(ctx context.Context, repoDir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repoDir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		var detail string
		if ee, ok := err.(*exec.ExitError); ok {
			detail = strings.TrimSpace(string(ee.Stderr))
		}
		return "", fmt.Errorf("git %s: %w %s", strings.Join(args, " "), err, detail)
	}
	return strings.TrimSpace(string(out)), nil
}

// resolveCommit verifies rev names a commit and returns its full SHA.
func resolveCommit(ctx context.Context, repoDir, rev string) (string, error) {
	out, err := gitOutput(ctx, repoDir, "rev-parse", "--verify", "--quiet", rev+"^{commit}")
	if err != nil || out == "" {
		return "", fmt.Errorf("%q does not resolve to a commit", rev)
	}
	return out, nil
}

// diffNameStatus returns the name-status entries of `git diff from to`,
// optionally limited to paths. Renames are split into delete+add so a renamed
// record always surfaces as a deletion of an immutable file.
func diffNameStatus(ctx context.Context, repoDir, from, to string, paths ...string) ([]fileChange, error) {
	args := []string{"diff", "--name-status", "--no-renames", "-z", from, to, "--"}
	args = append(args, paths...)
	out, err := gitOutput(ctx, repoDir, args...)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	fields := strings.Split(strings.TrimSuffix(out, "\x00"), "\x00")
	if len(fields)%2 != 0 {
		return nil, fmt.Errorf("unexpected git diff --name-status output %q", out)
	}
	changes := make([]fileChange, 0, len(fields)/2)
	for i := 0; i < len(fields); i += 2 {
		changes = append(changes, fileChange{Status: fields[i], Path: fields[i+1]})
	}
	return changes, nil
}

// loadHistoryAt reads the committed record files of historyDir at rev through
// git, so the gate inspects committed content rather than the worktree.
func loadHistoryAt(ctx context.Context, repoDir, rev, historyDir string) ([]HistoryRecord, error) {
	listing, err := gitOutput(ctx, repoDir, "ls-tree", "-r", "-z", "--name-only", rev, historyDir)
	if err != nil {
		return nil, err
	}
	if listing == "" {
		return nil, nil
	}
	paths := strings.Split(strings.TrimSuffix(listing, "\x00"), "\x00")
	entries := make([]historyEntry, 0, len(paths))
	for _, p := range paths {
		data, err := exec.CommandContext(ctx, "git", "-C", repoDir, "show", rev+":"+p).Output()
		if err != nil {
			return nil, fmt.Errorf("read %s at %s: %w", p, rev, err)
		}
		entries = append(entries, historyEntry{display: p, name: filepath.Base(p), data: data})
	}
	return parseHistoryEntries(entries)
}
