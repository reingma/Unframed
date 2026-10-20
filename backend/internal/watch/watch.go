// Package watch reports changes to a local content directory with fsnotify.
// It implements rebuild.Notifier. Network filesystems (NFS, SMB, FUSE) may
// not deliver notifications; there is no polling fallback.
package watch

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Debounce is the quiet window after the last change before a burst of
// changes is reported once.
const Debounce = 200 * time.Millisecond

// FS watches every non-hidden directory under a root, including directories
// created later. It watches directories rather than files, so editors that
// save by renaming a temporary file are seen.
type FS struct {
	dir      string
	debounce time.Duration
	logger   *slog.Logger
}

// New returns a watcher of dir reporting bursts after debounce.
func New(dir string, debounce time.Duration, logger *slog.Logger) *FS {
	return &FS{dir: dir, debounce: debounce, logger: logger}
}

// Watch calls changed once after each burst of changes until ctx ends.
// Watcher errors, including event overflow, are logged and reported as a
// change, so a missed event still leads to a full rebuild. Watch fails only
// when the watcher cannot start.
func (w *FS) Watch(ctx context.Context, changed func()) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("start content watcher: %w", err)
	}
	defer watcher.Close()
	if err := w.addTree(watcher, w.dir); err != nil {
		return err
	}
	w.logger.InfoContext(ctx, "watching content tree", "debounce", w.debounce)
	return w.loop(ctx, watcher.Events, watcher.Errors, func(dir string) error { return w.addTree(watcher, dir) }, changed)
}

// loop debounces events and errors into calls to changed. add watches a newly
// created directory and its subdirectories.
func (w *FS) loop(ctx context.Context, events <-chan fsnotify.Event, errs <-chan error, add func(string) error, changed func()) error {
	timer := time.NewTimer(w.debounce)
	timer.Stop()
	for {
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			if ev.Has(fsnotify.Create) {
				if info, err := os.Lstat(ev.Name); err == nil && info.IsDir() && !hidden(ev.Name) {
					if err := add(ev.Name); err != nil {
						w.logger.WarnContext(ctx, "failed to watch new content directory; forcing a rebuild", "error", err)
					}
				}
			}
			timer.Reset(w.debounce)
		case err, ok := <-errs:
			if !ok {
				return nil
			}
			if errors.Is(err, fsnotify.ErrEventOverflow) {
				w.logger.WarnContext(ctx, "content watcher overflowed; forcing a rebuild", "error", err)
			} else {
				w.logger.WarnContext(ctx, "content watcher error; forcing a rebuild", "error", err)
			}
			timer.Reset(w.debounce)
		case <-timer.C:
			changed()
		}
	}
}

func (w *FS) addTree(watcher *fsnotify.Watcher, dir string) error {
	return filepath.WalkDir(dir, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) && name != w.dir {
				return nil
			}
			return fmt.Errorf("watch content tree: %w", err)
		}
		if !entry.IsDir() {
			return nil
		}
		if name != w.dir && hidden(name) {
			return fs.SkipDir
		}
		if err := watcher.Add(name); err != nil {
			return fmt.Errorf("watch content directory: %w", err)
		}
		return nil
	})
}

func hidden(name string) bool { return strings.HasPrefix(filepath.Base(name), ".") }
