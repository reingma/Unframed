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
)

// ErrPublish wraps a publisher failure. Build still returns its result.
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

// Build scans source, renders every valid article and publishes one batch
// of events: article.published for each published article and
// validation.failed for each excluded item. Every log record carries the
// build ID. Build fails when the scan fails; an invalid root configuration is
// still published as a validation.failed event. When only publishing fails,
// Build returns the result with an error wrapping ErrPublish.
func Build(ctx context.Context, source content.Source, publisher event.Publisher, logger *slog.Logger) (*Result, error) {
	started := time.Now()
	buildID := rand.Text()
	logger = logger.With("build_id", buildID)
	batch := event.Batch{BuildID: buildID, Time: started.UTC()}

	manifest, err := scan.Scan(ctx, source, logger)
	if err != nil {
		var rootErr content.ValidationError
		if errors.Is(err, scan.ErrInvalidRootConfig) && errors.As(err, &rootErr) {
			batch.Events = []event.Event{validationFailed(rootErr)}
			if publishErr := publish(ctx, publisher, batch); publishErr != nil {
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
			batch.Events = append(batch.Events, event.Event{Kind: event.KindArticlePublished, Path: article.SourcePath, URLPath: article.URLPath})
		}
	}
	for _, validationErr := range manifest.Errors {
		batch.Events = append(batch.Events, validationFailed(validationErr))
	}

	result := &Result{BuildID: buildID, Content: manifest, Manifest: document.NewManifest(manifest.Root), Envelopes: envelopes}
	err = publish(ctx, publisher, batch)
	logger.InfoContext(ctx, "content built",
		"envelopes", len(envelopes), "events", len(batch.Events), "validation_errors", len(manifest.Errors),
		"publish_error", err != nil, "duration", time.Since(started))
	return result, err
}

func validationFailed(err content.ValidationError) event.Event {
	return event.Event{Kind: event.KindValidationFailed, Path: err.Path, Message: err.Err.Error()}
}

func publish(ctx context.Context, publisher event.Publisher, batch event.Batch) error {
	if err := publisher.Publish(ctx, batch); err != nil {
		return fmt.Errorf("%w: %w", ErrPublish, err)
	}
	return nil
}
