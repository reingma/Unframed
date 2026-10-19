// Package logs is a local event subscriber that writes structured log records.
package logs

import (
	"context"
	"log/slog"

	"github.com/reingma/Unframed/backend/internal/event"
)

// Subscriber writes each event as a structured log record with its build ID.
// Published articles log at debug level so routine rebuilds stay quiet;
// validation failures log as warnings.
type Subscriber struct {
	Logger *slog.Logger
}

func (s Subscriber) Receive(ctx context.Context, events []event.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, e := range events {
		attrs := []slog.Attr{
			slog.String("build_id", e.BuildID), slog.String("event_id", e.ID),
			slog.String("kind", string(e.Kind)), slog.String("path", e.Path),
		}
		switch e.Kind {
		case event.KindValidationFailed:
			attrs = append(attrs, slog.String("error", e.Message))
			s.Logger.LogAttrs(ctx, slog.LevelWarn, "content item excluded", attrs...)
		default:
			attrs = append(attrs, slog.String("url_path", e.URLPath))
			s.Logger.LogAttrs(ctx, slog.LevelDebug, "article published", attrs...)
		}
	}
	return nil
}
