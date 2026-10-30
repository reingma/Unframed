package perf

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTreeWithoutHistory is an independent oracle for ContentHash: the
// tree git itself writes for HEAD's index with perf/history removed.
func writeTreeWithoutHistory(t *testing.T, dir string) string {
	t.Helper()
	gitRun(t, dir, "rm", "-rq", "--cached", "--ignore-unmatch", "perf/history")
	defer gitRun(t, dir, "reset", "-q")
	return gitRun(t, dir, "write-tree")
}

// TestContentHashInvariants checks, over random trees, that ContentHash
// matches git's own tree for the content outside perf/history, ignores any
// record append, and changes with any other file.
func TestContentHashInvariants(t *testing.T) {
	ctx := context.Background()
	rng := rand.New(rand.NewPCG(47, 1))
	// Names that share prefixes with the excluded path exercise the tree walk.
	dirs := []string{"", "perf", "perf/history", "perf/history/sub", "perf/historyx", "perfx", "a/perf/history", "backend"}
	randomPath := func(prefix string) string {
		return filepath.Join(prefix, fmt.Sprintf("f%d.txt", rng.IntN(1000)))
	}
	for i := range 8 {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			dir := t.TempDir()
			gitRun(t, dir, "init", "-q")
			files := map[string]string{"seed.txt": "seed"}
			for range 1 + rng.IntN(6) {
				files[randomPath(dirs[rng.IntN(len(dirs))])] = fmt.Sprint(rng.Uint64())
			}
			commitFiles(t, dir, files, "seed")
			h0, err := ContentHash(ctx, dir, "HEAD", "perf/history")
			if err != nil {
				t.Fatal(err)
			}
			if want := writeTreeWithoutHistory(t, dir); h0 != want {
				t.Fatalf("content hash %s, git write-tree %s", h0, want)
			}

			commitFiles(t, dir, map[string]string{
				randomPath("perf/history"): "record",
			}, "record")
			if h, _ := ContentHash(ctx, dir, "HEAD", "perf/history"); h != h0 {
				t.Fatalf("a record append changed the hash: %s -> %s", h0, h)
			}

			outside := []string{"", "perf", "perf/historyx", "perfx", "a/perf/history", "backend"}
			changed := randomPath(outside[rng.IntN(len(outside))])
			commitFiles(t, dir, map[string]string{changed: fmt.Sprint("changed", rng.Uint64())}, "change")
			h1, err := ContentHash(ctx, dir, "HEAD", "perf/history")
			if err != nil {
				t.Fatal(err)
			}
			if h1 == h0 {
				t.Fatalf("changing %s left the hash unchanged", changed)
			}
			if want := writeTreeWithoutHistory(t, dir); h1 != want {
				t.Fatalf("content hash %s, git write-tree %s", h1, want)
			}
		})
	}
}

func TestContentHashOnlyHistory(t *testing.T) {
	dir := t.TempDir()
	gitRun(t, dir, "init", "-q")
	commitFiles(t, dir, map[string]string{"perf/history/0001-aaaaaaaa.json": "{}"}, "only history")
	h, err := ContentHash(context.Background(), dir, "HEAD", "perf/history")
	if err != nil {
		t.Fatal(err)
	}
	if want := writeTreeWithoutHistory(t, dir); h != want {
		t.Fatalf("content hash %s, want the empty tree %s", h, want)
	}
}

func TestCheckClean(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	gitRun(t, dir, "init", "-q")
	commitFiles(t, dir, map[string]string{"app.txt": "v1", ".gitignore": "ignored/\n"}, "seed")
	write := func(name, content string) {
		t.Helper()
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("ignored/out.bin", "x")
	write("perf/history/0002-bbbbbbbb.json", "{}")
	if err := CheckClean(ctx, dir, "perf/history"); err != nil {
		t.Fatalf("ignored files and history changes must be allowed: %v", err)
	}
	write("notes.txt", "untracked")
	if err := CheckClean(ctx, dir, "perf/history"); err == nil || !strings.Contains(err.Error(), "(notes.txt)") {
		t.Fatalf("expected the untracked file to be refused by name, got %v", err)
	}
	os.Remove(filepath.Join(dir, "notes.txt"))
	write("app.txt", "v2")
	if err := CheckClean(ctx, dir, "perf/history"); err == nil || !strings.Contains(err.Error(), "(app.txt)") {
		t.Fatalf("expected the modified file to be refused by name, got %v", err)
	}
}

func TestLocateMissingContent(t *testing.T) {
	dir, _ := gateFixture(t)
	found, err := Locate(context.Background(), dir, "main", "perf/history", testContent)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 0 {
		t.Fatalf("located %v for content that never existed", found)
	}
}
