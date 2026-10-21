package rebuild_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/reingma/Unframed/backend/internal/content/contenttest"
	"github.com/reingma/Unframed/backend/internal/document"
	"github.com/reingma/Unframed/backend/internal/event"
	"github.com/reingma/Unframed/backend/internal/publish"
	"github.com/reingma/Unframed/backend/internal/rebuild"
	"github.com/reingma/Unframed/backend/internal/snapshot"
)

const root = "title = \"Unframed\"\n[theme]\naccent = \"#336699\"\nbackground = \"#FFFFFF\"\ntext = \"#111111\"\n"

func quiet() *slog.Logger { return slog.New(slog.DiscardHandler) }

// source is a content source whose files can change between builds.
type source struct {
	mu    sync.Mutex
	files contenttest.MemSource
}

func (s *source) set(files contenttest.MemSource) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.files = files
}

func (s *source) build(ctx context.Context) (*snapshot.Snapshot, error) {
	s.mu.Lock()
	files := s.files
	s.mu.Unlock()
	return rebuild.FromSource(files, publish.NewLocal(), quiet())(ctx)
}

// counting wraps a build, recording overlap and the number of builds, and
// blocks each build until release is closed when release is set.
type counting struct {
	build   rebuild.SnapshotFunc
	running atomic.Int32
	overlap atomic.Bool
	builds  atomic.Int32
	release chan struct{}
}

func (c *counting) run(ctx context.Context) (*snapshot.Snapshot, error) {
	if c.running.Add(1) > 1 {
		c.overlap.Store(true)
	}
	defer c.running.Add(-1)
	c.builds.Add(1)
	if c.release != nil {
		<-c.release
	}
	return c.build(ctx)
}

func TestStartFailsOnInvalidRoot(t *testing.T) {
	src := &source{files: contenttest.MemSource{".configuration": "title = 1\n"}}
	var store snapshot.Store
	err := rebuild.New(&store, src.build, quiet()).Start(context.Background())
	if err == nil {
		t.Fatal("startup build of an invalid root succeeded")
	}
	if store.Current() != nil {
		t.Fatal("failed startup build installed a snapshot")
	}
}

func TestBurstOfSignalsIsCoalesced(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		src := &source{files: contenttest.MemSource{".configuration": root}}
		c := &counting{build: src.build, release: make(chan struct{})}
		var store snapshot.Store
		b := rebuild.New(&store, c.run, quiet())
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		go b.Run(ctx)

		b.Signal(rebuild.TriggerWatch)
		synctest.Wait() // the first build is now running and blocked
		for range 1000 {
			b.Signal(rebuild.TriggerWatch)
		}
		close(c.release)
		synctest.Wait()

		if got := c.builds.Load(); got != 2 {
			t.Fatalf("builds = %d, want 2: one running, one coalescing the rest", got)
		}
		if c.overlap.Load() {
			t.Fatal("builds overlapped")
		}
		if last := b.Last(); last.Signals != 1000 || last.Trigger != rebuild.TriggerWatch || last.Err != nil {
			t.Fatalf("last outcome = %+v, want 1000 coalesced watch signals", last)
		}
	})
}

func TestFailedRebuildKeepsPreviousSnapshot(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		src := &source{files: contenttest.MemSource{".configuration": root, "A.md": "A.\n"}}
		var store snapshot.Store
		b := rebuild.New(&store, src.build, quiet())
		if err := b.Start(t.Context()); err != nil {
			t.Fatal(err)
		}
		good := store.Current()
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		go b.Run(ctx)

		src.set(contenttest.MemSource{".configuration": "broken = ["})
		_, err := b.Reload(t.Context())
		if !errors.Is(err, rebuild.ErrBuildFailed) {
			t.Fatalf("reload error = %v, want ErrBuildFailed", err)
		}
		if store.Current() != good {
			t.Fatal("failed rebuild replaced the serving snapshot")
		}
		if last := b.Last(); last.Err == nil || last.Trigger != rebuild.TriggerReload {
			t.Fatalf("last outcome = %+v, want the failed reload", last)
		}
		if _, ok := store.Current().Envelope("/a", false); !ok {
			t.Fatal("previous snapshot lost its content")
		}
	})
}

func TestReloadWaitsForABuildThatStartedAfterIt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		src := &source{files: contenttest.MemSource{".configuration": root}}
		c := &counting{build: src.build, release: make(chan struct{})}
		var store snapshot.Store
		b := rebuild.New(&store, c.run, quiet())
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		go b.Run(ctx)

		b.Signal(rebuild.TriggerWatch)
		synctest.Wait() // a build of the old content is in flight
		src.set(contenttest.MemSource{".configuration": root, "New.md": "New.\n"})
		requested := time.Now()
		result := make(chan rebuild.Outcome, 1)
		go func() {
			outcome, err := b.Reload(t.Context())
			if err != nil {
				t.Error(err)
			}
			result <- outcome
		}()
		synctest.Wait()
		close(c.release)
		outcome := <-result

		if outcome.Started.Before(requested) {
			t.Fatalf("reload returned a build started at %v, before its request at %v", outcome.Started, requested)
		}
		if outcome.BuildID == "" || outcome.BuildID != store.Current().BuildID() || outcome.Envelopes != 1 {
			t.Fatalf("outcome = %+v, want the serving build with the new article", outcome)
		}
	})
}

func TestReloadIsBoundedByItsContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		src := &source{files: contenttest.MemSource{".configuration": root}}
		c := &counting{build: src.build, release: make(chan struct{})}
		var store snapshot.Store
		b := rebuild.New(&store, c.run, quiet())
		runCtx, cancel := context.WithCancel(t.Context())
		defer cancel()
		go b.Run(runCtx)

		ctx, stop := context.WithTimeout(t.Context(), time.Second)
		defer stop()
		if _, err := b.Reload(ctx); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("reload error = %v, want its deadline", err)
		}
		close(c.release)
	})
}

func TestFromSourceKeepsSnapshotWhenPublishingFails(t *testing.T) {
	failing := publish.NewLocal(failingSubscriber{})
	snapshotFunc := rebuild.FromSource(contenttest.MemSource{".configuration": root, "A.md": "A.\n"}, failing, quiet())
	s, err := snapshotFunc(context.Background())
	if err != nil || s == nil {
		t.Fatalf("build = %v, %v; want a snapshot despite the publishing failure", s, err)
	}
	doc, ok := s.Envelope("/a", false)
	if !ok {
		t.Fatal("snapshot lost its envelope")
	}
	if _, err := document.DecodeEnvelope(doc.JSON); err != nil {
		t.Fatal(err)
	}
}

type failingSubscriber struct{}

func (failingSubscriber) Receive(context.Context, []event.Event) error { return errors.New("bus down") }
