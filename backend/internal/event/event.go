// Package event is the build fanout boundary: builds publish what they
// produced, and publishers deliver it. Local publishers log events and write
// report files; a notification service can implement Publisher later.
package event

import (
	"context"
	"errors"
	"time"
)

// Kind names what an event reports.
type Kind string

const (
	// KindArticlePublished reports a published (non-draft) article.
	KindArticlePublished Kind = "article.published"
	// KindValidationFailed reports an item excluded by a validation error.
	KindValidationFailed Kind = "validation.failed"
)

// Event is one fact about a build.
type Event struct {
	Kind Kind `json:"kind"`
	// Path is the item's slash-separated path in the content source.
	Path string `json:"path"`
	// URLPath is set for KindArticlePublished.
	URLPath string `json:"urlPath,omitempty"`
	// Message is the validation error for KindValidationFailed.
	Message string `json:"message,omitempty"`
}

// Batch is every event of one build.
type Batch struct {
	BuildID string    `json:"buildId"`
	Time    time.Time `json:"time"`
	Events  []Event   `json:"events"`
}

// Publisher delivers a build's events. Publish honors cancellation and
// deadlines, and a failure means the batch may not have been delivered.
type Publisher interface {
	Publish(ctx context.Context, batch Batch) error
}

// Fanout returns a Publisher that delivers each batch to every publisher,
// even after one fails, and joins their errors.
func Fanout(publishers ...Publisher) Publisher {
	return fanout(publishers)
}

type fanout []Publisher

func (f fanout) Publish(ctx context.Context, batch Batch) error {
	var errs []error
	for _, publisher := range f {
		errs = append(errs, publisher.Publish(ctx, batch))
	}
	return errors.Join(errs...)
}
