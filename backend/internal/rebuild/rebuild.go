// Package rebuild runs every content build on one goroutine and swaps each
// successful build's snapshot into the serving store. Triggers only signal
// the builder: at most one build runs and at most one is pending, so any
// volume of signals coalesces into bounded work.
package rebuild

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/reingma/Unframed/backend/internal/build"
	"github.com/reingma/Unframed/backend/internal/content"
	"github.com/reingma/Unframed/backend/internal/publish"
	"github.com/reingma/Unframed/backend/internal/snapshot"
)

// Trigger names what requested a build.
type Trigger string

const (
	TriggerStartup Trigger = "startup"
	TriggerWatch   Trigger = "watch"
	TriggerReload  Trigger = "reload"
)

// ErrBuildFailed reports that a build failed; the previous snapshot keeps serving.
var ErrBuildFailed = errors.New("content build failed")

// BuildFunc builds one snapshot. It must honor cancellation.
type BuildFunc func(ctx context.Context) (*snapshot.Snapshot, error)

// FromSource returns a BuildFunc that builds source and publishes its events.
// A publishing failure is logged and does not discard the snapshot.
func FromSource(source content.Source, publisher publish.Publisher, logger *slog.Logger) BuildFunc {
	return func(ctx context.Context) (*snapshot.Snapshot, error) {
		result, err := build.Build(ctx, source, publisher, logger)
		if result == nil {
			return nil, err
		}
		if err != nil {
			logger.WarnContext(ctx, "failed to publish build events", "build_id", result.BuildID, "error", err)
		}
		return snapshot.New(result, time.Now().UTC())
	}
}

// Outcome describes one finished build. BuildID and the counts are set only
// when it succeeded.
type Outcome struct {
	Trigger Trigger
	// Signals is the number of coalesced requests the build served.
	Signals          int
	Started          time.Time
	Duration         time.Duration
	BuildID          string
	Envelopes        int
	ValidationErrors int
	Err              error
}

// Notifier reports content changes by calling changed until ctx ends. The
// local filesystem watcher and an object-store event feed are its credible
// implementations. A Notifier may coalesce changes.
type Notifier interface {
	Watch(ctx context.Context, changed func()) error
}

// Builder owns the serving store and is the only writer to it.
type Builder struct {
	build  BuildFunc
	store  *snapshot.Store
	logger *slog.Logger
	wake   chan struct{}

	mu sync.Mutex
	// pending is the first trigger of the pending build and signals the
	// number of requests coalesced into it.
	pending Trigger
	signals int
	waiters []chan Outcome
	last    Outcome
}

// New returns a builder writing to store. Call Start, then Run.
func New(store *snapshot.Store, buildFunc BuildFunc, logger *slog.Logger) *Builder {
	return &Builder{build: buildFunc, store: store, logger: logger, wake: make(chan struct{}, 1)}
}

// Start runs the startup build synchronously. Its failure is fatal to the
// caller: there is no snapshot to serve.
func (b *Builder) Start(ctx context.Context) error {
	outcome := b.run(ctx, TriggerStartup, 1)
	return outcome.Err
}

// Run builds on signals until ctx ends. Only one Run may be active.
func (b *Builder) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			b.mu.Lock()
			waiters := b.waiters
			b.waiters = nil
			b.mu.Unlock()
			for _, w := range waiters {
				w <- Outcome{Trigger: TriggerReload, Err: ctx.Err()}
			}
			return
		case <-b.wake:
		}
		b.mu.Lock()
		trigger, signals, waiters := b.pending, b.signals, b.waiters
		b.pending, b.signals, b.waiters = "", 0, nil
		b.mu.Unlock()
		if signals == 0 {
			continue
		}
		outcome := b.run(ctx, trigger, signals)
		for _, w := range waiters {
			w <- outcome
		}
	}
}

// Signal requests a build without waiting. Signals that arrive while a build
// is pending join it.
func (b *Builder) Signal(trigger Trigger) {
	b.mu.Lock()
	b.add(trigger)
	b.mu.Unlock()
	b.notify()
}

// Reload requests a build and waits for one that started after the call,
// until ctx ends. It returns ErrBuildFailed when that build failed.
func (b *Builder) Reload(ctx context.Context) (Outcome, error) {
	waiter := make(chan Outcome, 1)
	b.mu.Lock()
	b.add(TriggerReload)
	b.waiters = append(b.waiters, waiter)
	b.mu.Unlock()
	b.notify()
	select {
	case outcome := <-waiter:
		if outcome.Err != nil {
			return outcome, fmt.Errorf("%w: %w", ErrBuildFailed, outcome.Err)
		}
		return outcome, nil
	case <-ctx.Done():
		return Outcome{}, ctx.Err()
	}
}

// Last returns the most recent finished build, successful or not.
func (b *Builder) Last() Outcome {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.last
}

// Follow passes notifier's changes to the builder as watch signals.
func (b *Builder) Follow(ctx context.Context, notifier Notifier) error {
	return notifier.Watch(ctx, func() { b.Signal(TriggerWatch) })
}

// add records a signal; b.mu must be held.
func (b *Builder) add(trigger Trigger) {
	if b.signals == 0 {
		b.pending = trigger
	}
	b.signals++
}

func (b *Builder) notify() {
	select {
	case b.wake <- struct{}{}:
	default:
	}
}

func (b *Builder) run(ctx context.Context, trigger Trigger, signals int) Outcome {
	outcome := Outcome{Trigger: trigger, Signals: signals, Started: time.Now()}
	next, err := b.build(ctx)
	outcome.Duration = time.Since(outcome.Started)
	if err == nil && next == nil {
		err = errors.New("build returned no snapshot")
	}
	attrs := []any{"trigger", outcome.Trigger, "signals", outcome.Signals, "duration", outcome.Duration}
	if err != nil {
		outcome.Err = err
		b.logger.ErrorContext(ctx, "content rebuild failed; previous snapshot still serving",
			append(attrs, "outcome", "failed", "error", err)...)
	} else {
		outcome.BuildID = next.BuildID()
		outcome.Envelopes = next.Envelopes()
		outcome.ValidationErrors = len(next.Diagnostics())
		b.store.Swap(next)
		b.logger.InfoContext(ctx, "content rebuilt", append(attrs, "outcome", "succeeded", "build_id", outcome.BuildID,
			"envelopes", outcome.Envelopes, "validation_errors", outcome.ValidationErrors)...)
	}
	b.mu.Lock()
	b.last = outcome
	b.mu.Unlock()
	return outcome
}
