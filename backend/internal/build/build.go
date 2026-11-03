// Package build turns a content source into served documents: it scans the
// tree, renders every valid article and publishes the build's events.
package build

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/reingma/Unframed/backend/internal/content"
	"github.com/reingma/Unframed/backend/internal/content/render"
	"github.com/reingma/Unframed/backend/internal/content/scan"
	"github.com/reingma/Unframed/backend/internal/document"
	"github.com/reingma/Unframed/backend/internal/event"
	"github.com/reingma/Unframed/backend/internal/publish"
)

// ErrPublish wraps a publishing failure. Build still returns its result.
var ErrPublish = errors.New("publish build events")

// Key identifies an envelope: a published article and its draft share a URL.
type Key struct {
	URLPath string
	Draft   bool
}

// Result is the output of one build.
type Result struct {
	// BuildID correlates the build's log records and events.
	BuildID string
	// Content is the scanned manifest, with drafts and validation errors.
	Content *content.Manifest
	// Manifest is the public manifest document.
	Manifest document.Manifest
	// Envelopes holds one envelope per valid article, drafts included.
	Envelopes map[Key]document.Envelope
}

// Build scans source, renders every valid article and publishes the build's
// events in one call: article.published for each published article and
// validation.failed for each excluded item. A build without events publishes
// nothing. Every log record carries the
// build ID. Build fails when the scan fails; an invalid root configuration is
// still published as a validation.failed event. When only publishing fails,
// Build returns the result with an error wrapping ErrPublish.
func Build(ctx context.Context, source content.Source, publisher publish.Publisher, logger *slog.Logger) (*Result, error) {
	started := time.Now()
	buildID := rand.Text()
	logger = logger.With("build_id", buildID)
	events := &events{buildID: buildID, time: started.UTC()}

	manifest, err := scan.Scan(ctx, source, logger)
	if err != nil {
		var rootErr content.ValidationError
		if errors.Is(err, scan.ErrInvalidRootConfig) && errors.As(err, &rootErr) {
			events.validationFailed(rootErr)
			if publishErr := events.publish(ctx, publisher); publishErr != nil {
				err = errors.Join(err, publishErr)
			}
		}
		logger.ErrorContext(ctx, "content build failed", "error", err, "duration", time.Since(started))
		return nil, fmt.Errorf("build content: %w", err)
	}

	envelopes := make(map[Key]document.Envelope, len(manifest.Articles))
	for _, article := range manifest.Articles {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("build content: %w", err)
		}
		html, err := render.Render(article.Body)
		if err != nil {
			return nil, fmt.Errorf("build content: %s: %w", article.SourcePath, err)
		}
		envelopes[Key{URLPath: article.URLPath, Draft: article.Draft}] = document.NewEnvelope(article, html)
		if !article.Draft {
			events.add(event.Event{Kind: event.KindArticlePublished, Path: article.SourcePath, URLPath: article.URLPath})
		}
	}
	for _, validationErr := range manifest.Errors {
		events.validationFailed(validationErr)
	}

	result := &Result{BuildID: buildID, Content: manifest, Manifest: document.NewManifest(manifest.Root, manifest.Redirects), Envelopes: envelopes}
	err = events.publish(ctx, publisher)
	logger.InfoContext(ctx, "content built",
		"envelopes", len(envelopes), "events", len(events.list), "validation_errors", len(manifest.Errors),
		"publish_error", err != nil, "duration", time.Since(started))
	return result, err
}

// events collects a build's events, stamping each with the build's identity.
type events struct {
	buildID string
	time    time.Time
	list    []event.Event
}

func (e *events) add(ev event.Event) {
	ev.SchemaVersion = event.SchemaVersion
	ev.ID = fmt.Sprintf("%s-%d", e.buildID, len(e.list)+1)
	ev.BuildID = e.buildID
	ev.Time = e.time
	e.list = append(e.list, ev)
}

func (e *events) validationFailed(err content.ValidationError) {
	e.add(event.Event{Kind: event.KindValidationFailed, Path: err.Path, Message: err.Err.Error()})
}

func (e *events) publish(ctx context.Context, publisher publish.Publisher) error {
	if len(e.list) == 0 {
		return nil
	}
	if err := publisher.Publish(ctx, e.list); err != nil {
		return fmt.Errorf("%w: %w", ErrPublish, err)
	}
	return nil
}
