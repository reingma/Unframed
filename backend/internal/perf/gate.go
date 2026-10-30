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
//   - Every added record must carry a content hash (schema version 3).
//   - The newest record at head must be bound to what is being merged: its
//     content hash must equal ContentHash(head), so head's entire content
//     outside historyDir is what was measured and only record appends came
//     after it. A record cannot name the commit that contains it, so this
//     content equality is the strict form of "the record measures the
//     merge"; any change pushed after measuring fails the gate. The record's
//     revision is not required to resolve: squash and rebase merges drop it.
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

	byName := map[string]HistoryRecord{}
	for _, r := range records {
		byName[RecordFileName(r)] = r
	}
	for _, name := range res.Added {
		r, ok := byName[name]
		if !ok {
			continue
		}
		if r.ContentHash == "" {
			res.Failures = append(res.Failures, fmt.Sprintf(
				"%s is a version %d record without content_hash; new records must be version %d — re-run the measurement with the current perf tool",
				name, r.Version, SchemaVersion))
		}
		if r.Environment.Machine != CanonicalMachine {
			res.Warnings = append(res.Warnings, fmt.Sprintf(
				"%s was measured on machine %q, not the canonical %q; the record is accepted but is a different environment factor",
				name, r.Environment.Machine, CanonicalMachine))
		}
	}

	// A legacy newest record cannot be bound by content; it is either an
	// added record (failed above) or an old one (no record added, failed above).
	if newest.ContentHash == "" {
		return res, nil
	}
	headContent, err := ContentHash(ctx, repoDir, headSHA, historyDir)
	if err != nil {
		return GateResult{}, err
	}
	if newest.ContentHash != headContent {
		detail, err := driftSince(ctx, repoDir, newest.Revision, headSHA, historyDir)
		if err != nil {
			return GateResult{}, err
		}
		res.Failures = append(res.Failures, fmt.Sprintf(
			"%s measured content %s but the merge head's content outside %s is %s%s — re-run the measurement on the designated machine and append a fresh record",
			res.Newest, newest.ContentHash, historyDir, headContent, detail))
	}
	return res, nil
}

// driftSince describes what changed outside historyDir between the measured
// revision and head, when the revision still resolves; it is diagnostics
// only, since the content hash decides the gate.
func driftSince(ctx context.Context, repoDir, revision, headSHA, historyDir string) (string, error) {
	measuredSHA, err := resolveCommit(ctx, repoDir, revision)
	if err != nil {
		return "", nil
	}
	since, err := diffNameStatus(ctx, repoDir, measuredSHA, headSHA)
	if err != nil {
		return "", err
	}
	var drift []string
	for _, c := range since {
		if c.Status != "A" || !strings.HasPrefix(c.Path, historyDir+"/") {
			drift = append(drift, c.Status+" "+c.Path)
		}
	}
	if len(drift) == 0 {
		return "", nil
	}
	const max = 5
	detail := strings.Join(drift[:min(len(drift), max)], ", ")
	if len(drift) > max {
		detail += fmt.Sprintf(", and %d more", len(drift)-max)
	}
	return fmt.Sprintf("; changed since %s: %s", revision[:8], detail), nil
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
