// Package event is the contract of the facts a build emits. Each event is
// self-contained, because a message bus delivers events one at a time to
// consumers that may run in other processes.
package event

import "time"

// SchemaVersion is the event schema version. Consumers reject events with a
// missing or unknown version; within a version, members are additive.
const SchemaVersion = 1

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
	SchemaVersion int `json:"schemaVersion"`
	// ID is unique per event, so consumers can drop redelivered duplicates.
	ID string `json:"id"`
	// BuildID is shared by every event of a build and its log records.
	BuildID string    `json:"buildId"`
	Time    time.Time `json:"time"`
	Kind    Kind      `json:"kind"`
	// Path is the item's slash-separated path in the content source.
	Path string `json:"path"`
	// URLPath is set for KindArticlePublished.
	URLPath string `json:"urlPath,omitempty"`
	// Message is the validation error for KindValidationFailed.
	Message string `json:"message,omitempty"`
}
