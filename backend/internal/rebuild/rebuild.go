// Package rebuild runs every content build on one goroutine and swaps each
// successful build's snapshot into the serving store. Triggers only signal
// the build manager: at most one build runs and at most one is pending, so any
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

// ErrStopped reports a build or reload cut short because the manager is
// shutting down. It is not a build failure.
var ErrStopped = errors.New("build manager stopped")

// SnapshotFunc builds one snapshot. It must honor cancellation.
type SnapshotFunc func(ctx context.Context) (*snapshot.Snapshot, error)

// FromSource returns a SnapshotFunc that builds source and publishes its events.
// A publishing failure is logged and does not discard the snapshot.
func FromSource(source content.Source, publisher publish.Publisher, logger *slog.Logger) SnapshotFunc {
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

// Manager owns the serving store and is the only writer to it.
type Manager struct {
	build  SnapshotFunc
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

// New returns a build manager writing to store. Call Start, then Run.
func New(store *snapshot.Store, snapshot SnapshotFunc, logger *slog.Logger) *Manager {
	return &Manager{build: snapshot, store: store, logger: logger, wake: make(chan struct{}, 1)}
}

// Start runs the startup build synchronously. Its failure is fatal to the
// caller: there is no snapshot to serve.
func (m *Manager) Start(ctx context.Context) error {
	outcome := m.run(ctx, TriggerStartup, 1)
	return outcome.Err
}

// Run builds on signals until ctx ends. Only one Run may be active.
func (m *Manager) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			m.mu.Lock()
			waiters := m.waiters
			m.waiters = nil
			m.mu.Unlock()
			for _, w := range waiters {
				w <- Outcome{Trigger: TriggerReload, Err: fmt.Errorf("%w: %w", ErrStopped, ctx.Err())}
			}
			return
		case <-m.wake:
		}
		m.mu.Lock()
		trigger, signals, waiters := m.pending, m.signals, m.waiters
		m.pending, m.signals, m.waiters = "", 0, nil
		m.mu.Unlock()
		if signals == 0 {
			continue
		}
		outcome := m.run(ctx, trigger, signals)
		for _, w := range waiters {
			w <- outcome
		}
	}
}

// Signal requests a build without waiting. Signals that arrive while a build
// is pending join it.
func (m *Manager) Signal(trigger Trigger) {
	m.mu.Lock()
	m.add(trigger)
	m.mu.Unlock()
	m.notify()
}

// Reload requests a build and waits for one that started after the call,
// until ctx ends. It returns ErrBuildFailed when that build failed.
func (m *Manager) Reload(ctx context.Context) (Outcome, error) {
	waiter := make(chan Outcome, 1)
	m.mu.Lock()
	m.add(TriggerReload)
	m.waiters = append(m.waiters, waiter)
	m.mu.Unlock()
	m.notify()
	select {
	case outcome := <-waiter:
		switch {
		case errors.Is(outcome.Err, ErrStopped):
			return outcome, outcome.Err
		case outcome.Err != nil:
			return outcome, fmt.Errorf("%w: %w", ErrBuildFailed, outcome.Err)
		}
		return outcome, nil
	case <-ctx.Done():
		return Outcome{}, ctx.Err()
	}
}

// Last returns the most recent finished build, successful or not.
func (m *Manager) Last() Outcome {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.last
}

// Follow passes notifier's changes to the manager as watch signals.
func (m *Manager) Follow(ctx context.Context, notifier Notifier) error {
	return notifier.Watch(ctx, func() { m.Signal(TriggerWatch) })
}

// add records a signal; m.mu must be held.
func (m *Manager) add(trigger Trigger) {
	if m.signals == 0 {
		m.pending = trigger
	}
	m.signals++
}

func (m *Manager) notify() {
	select {
	case m.wake <- struct{}{}:
	default:
	}
}

func (m *Manager) run(ctx context.Context, trigger Trigger, signals int) Outcome {
	outcome := Outcome{Trigger: trigger, Signals: signals, Started: time.Now()}
	next, err := m.build(ctx)
	outcome.Duration = time.Since(outcome.Started)
	if err == nil && next == nil {
		err = errors.New("build returned no snapshot")
	}
	attrs := []any{"trigger", outcome.Trigger, "signals", outcome.Signals, "duration", outcome.Duration}
	if err != nil && ctx.Err() != nil {
		// Shutdown, not a failure: the last outcome stays the last real build.
		outcome.Err = fmt.Errorf("%w: %w", ErrStopped, err)
		m.logger.InfoContext(ctx, "content rebuild canceled; shutting down", append(attrs, "outcome", "canceled")...)
		return outcome
	}
	if err != nil {
		outcome.Err = err
		m.logger.ErrorContext(ctx, "content rebuild failed; previous snapshot still serving",
			append(attrs, "outcome", "failed", "error", err)...)
	} else {
		outcome.BuildID = next.BuildID()
		outcome.Envelopes = next.Envelopes()
		outcome.ValidationErrors = len(next.Diagnostics())
		m.store.Swap(next)
		m.logger.InfoContext(ctx, "content rebuilt", append(attrs, "outcome", "succeeded", "build_id", outcome.BuildID,
			"envelopes", outcome.Envelopes, "validation_errors", outcome.ValidationErrors)...)
	}
	m.mu.Lock()
	m.last = outcome
	m.mu.Unlock()
	return outcome
}
