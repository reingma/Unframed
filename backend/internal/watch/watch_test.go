package watch

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/fsnotify/fsnotify"
)

func quiet() *slog.Logger { return slog.New(slog.DiscardHandler) }

// watching starts a real watcher of dir and returns its change notifications.
func watching(t *testing.T, dir string) <-chan struct{} {
	t.Helper()
	changes := make(chan struct{}, 100)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- New(dir, 20*time.Millisecond, quiet()).Watch(ctx, func() { changes <- struct{}{} })
	}()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Error(err)
		}
	})
	// fsnotify has no readiness signal: touch the root until a change arrives.
	deadline := time.After(5 * time.Second)
	for {
		write(t, filepath.Join(dir, ".ready"), "")
		select {
		case <-changes:
			drain(changes)
			return changes
		case <-time.After(50 * time.Millisecond):
		case <-deadline:
			t.Fatal("watcher never became ready")
		}
	}
}

func write(t *testing.T, name, data string) {
	t.Helper()
	if err := os.WriteFile(name, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func drain(changes <-chan struct{}) {
	for {
		select {
		case <-changes:
		case <-time.After(100 * time.Millisecond):
			return
		}
	}
}

func expectChange(t *testing.T, changes <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-changes:
		drain(changes)
	case <-time.After(5 * time.Second):
		t.Fatalf("no change reported after %s", what)
	}
}

func TestWatchReportsContentChanges(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "A.md"), "A.\n")
	changes := watching(t, dir)

	write(t, filepath.Join(dir, "B.md"), "B.\n")
	expectChange(t, changes, "create")

	write(t, filepath.Join(dir, "B.md"), "B, edited.\n")
	expectChange(t, changes, "edit")

	tmp := filepath.Join(dir, ".A.md.swp")
	write(t, tmp, "A, saved atomically.\n")
	drain(changes)
	if err := os.Rename(tmp, filepath.Join(dir, "A.md")); err != nil {
		t.Fatal(err)
	}
	expectChange(t, changes, "atomic rename")

	if err := os.Remove(filepath.Join(dir, "B.md")); err != nil {
		t.Fatal(err)
	}
	expectChange(t, changes, "delete")

	sub := filepath.Join(dir, "Essays", "Deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	expectChange(t, changes, "new subdirectory")
	write(t, filepath.Join(sub, "C.md"), "C.\n")
	expectChange(t, changes, "create inside a new subdirectory")
}

func TestWatchFailsOnMissingRoot(t *testing.T) {
	err := New(filepath.Join(t.TempDir(), "missing"), Debounce, quiet()).Watch(t.Context(), func() {})
	if err == nil {
		t.Fatal("watching a missing root succeeded")
	}
}

func TestBurstIsDebouncedAndErrorsForceAChange(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		events := make(chan fsnotify.Event)
		errs := make(chan error)
		var changed atomic.Int32
		w := New("", Debounce, quiet())
		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan error)
		go func() {
			done <- w.loop(ctx, events, errs, func(string) error { return nil }, func() { changed.Add(1) })
		}()

		for range 50 {
			events <- fsnotify.Event{Name: "A.md", Op: fsnotify.Write}
			time.Sleep(Debounce / 2)
		}
		synctest.Wait()
		if changed.Load() != 0 {
			t.Fatalf("changed %d times during a burst", changed.Load())
		}
		time.Sleep(Debounce)
		synctest.Wait()
		if changed.Load() != 1 {
			t.Fatalf("changed %d times after a burst, want 1", changed.Load())
		}

		errs <- fsnotify.ErrEventOverflow
		time.Sleep(Debounce)
		synctest.Wait()
		errs <- errors.New("watch descriptor lost")
		time.Sleep(Debounce)
		synctest.Wait()
		if changed.Load() != 3 {
			t.Fatalf("changed %d times, want watcher errors to force a change each", changed.Load())
		}

		cancel()
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	})
}
