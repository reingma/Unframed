package perf

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// contentHashPattern accepts git object IDs of both object formats: SHA-1
// (40 hex digits) and SHA-256 (64). The gate compares a record's hash with
// one computed in the repository, so a hash of the wrong format never matches.
var contentHashPattern = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)

// ContentHash returns the content identity of rev: the git tree object ID of
// rev's tree with historyDir removed. Appending records leaves it unchanged
// and any other change alters it, so it identifies the measured code across
// squash, rebase and merge-commit merges. It is a genuine tree ID, computed
// without writing objects; `git mktree` over the same entries reproduces it.
func ContentHash(ctx context.Context, repoDir, rev, historyDir string) (string, error) {
	commit, err := resolveCommit(ctx, repoDir, rev)
	if err != nil {
		return "", err
	}
	root, err := gitOutput(ctx, repoDir, "rev-parse", commit+"^{tree}")
	if err != nil {
		return "", err
	}
	parts := strings.Split(strings.Trim(historyDir, "/"), "/")
	id, empty, err := treeWithout(ctx, repoDir, root, parts)
	if err != nil {
		return "", err
	}
	if empty {
		return hashTree(ctx, repoDir, nil)
	}
	return id, nil
}

// treeWithout returns the ID of tree with the path parts removed. empty
// reports that nothing remains, so the caller drops the entry instead.
func treeWithout(ctx context.Context, repoDir, tree string, parts []string) (string, bool, error) {
	listing, err := gitOutput(ctx, repoDir, "ls-tree", "-z", tree)
	if err != nil {
		return "", false, err
	}
	var entries []string
	if listing != "" {
		entries = strings.Split(strings.TrimSuffix(listing, "\x00"), "\x00")
	}
	kept := make([]string, 0, len(entries))
	changed := false
	for _, e := range entries {
		// ls-tree entries are "<mode> <type> <oid>\t<name>".
		meta, name, ok := strings.Cut(e, "\t")
		fields := strings.Fields(meta)
		if !ok || len(fields) != 3 {
			return "", false, fmt.Errorf("unexpected git ls-tree entry %q", e)
		}
		if name != parts[0] || fields[1] != "tree" {
			kept = append(kept, e)
			continue
		}
		changed = true
		if len(parts) == 1 {
			continue
		}
		sub, empty, err := treeWithout(ctx, repoDir, fields[2], parts[1:])
		if err != nil {
			return "", false, err
		}
		if !empty {
			kept = append(kept, fields[0]+" tree "+sub+"\t"+name)
		}
	}
	if !changed {
		return tree, false, nil
	}
	if len(kept) == 0 {
		return "", true, nil
	}
	id, err := hashTree(ctx, repoDir, kept)
	return id, false, err
}

// hashTree computes the object ID of a tree holding ls-tree formatted
// entries, in their existing (already canonical) order, without writing it.
func hashTree(ctx context.Context, repoDir string, entries []string) (string, error) {
	var buf bytes.Buffer
	for _, e := range entries {
		meta, name, _ := strings.Cut(e, "\t")
		fields := strings.Fields(meta)
		oid, err := hex.DecodeString(fields[2])
		if err != nil {
			return "", fmt.Errorf("git ls-tree entry %q: %w", e, err)
		}
		// Tree objects store modes without leading zeros ("40000", not "040000").
		buf.WriteString(strings.TrimLeft(fields[0], "0"))
		buf.WriteByte(' ')
		buf.WriteString(name)
		buf.WriteByte(0)
		buf.Write(oid)
	}
	cmd := exec.CommandContext(ctx, "git", "-C", repoDir, "hash-object", "-t", "tree", "--stdin")
	cmd.Stdin = &buf
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git hash-object -t tree: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// CheckClean refuses a worktree with uncommitted or untracked changes outside
// historyDir; ignored files are allowed. A measurement taken there would not
// be the content its hash names.
func CheckClean(ctx context.Context, repoDir, historyDir string) error {
	// Raw output: porcelain entries start with a space-padded status.
	raw, err := exec.CommandContext(ctx, "git", "-C", repoDir, "status", "--porcelain", "-z",
		"--untracked-files=all", "--", ".", ":(exclude)"+strings.Trim(historyDir, "/")).Output()
	if err != nil {
		return fmt.Errorf("git status: %w", err)
	}
	out := string(raw)
	if out == "" {
		return nil
	}
	var paths []string
	for _, e := range strings.Split(strings.TrimSuffix(out, "\x00"), "\x00") {
		if len(e) > 3 {
			paths = append(paths, e[3:])
		}
	}
	const max = 5
	detail := strings.Join(paths[:min(len(paths), max)], ", ")
	if len(paths) > max {
		detail += fmt.Sprintf(", and %d more", len(paths)-max)
	}
	return fmt.Errorf("worktree has uncommitted changes outside %s (%s); commit or stash them before measuring", historyDir, detail)
}

// Locate maps each wanted content hash to the oldest commit on ref's
// first-parent history whose content matches: the commit where the measured
// content landed on that branch. Hashes with no match are absent.
func Locate(ctx context.Context, repoDir, ref, historyDir string, hashes ...string) (map[string]string, error) {
	want := map[string]bool{}
	for _, h := range hashes {
		if h != "" {
			want[h] = true
		}
	}
	found := map[string]string{}
	if len(want) == 0 {
		return found, nil
	}
	tip, err := resolveCommit(ctx, repoDir, ref)
	if err != nil {
		return nil, err
	}
	list, err := gitOutput(ctx, repoDir, "rev-list", "--first-parent", "--reverse", tip)
	if err != nil {
		return nil, err
	}
	for _, commit := range strings.Fields(list) {
		h, err := ContentHash(ctx, repoDir, commit, historyDir)
		if err != nil {
			return nil, err
		}
		if want[h] {
			if _, ok := found[h]; !ok {
				found[h] = commit
				if len(found) == len(want) {
					break
				}
			}
		}
	}
	return found, nil
}
