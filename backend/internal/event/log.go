package event

import (
	"context"
	"log/slog"
)

// LogPublisher writes each event as a structured log record with the build
// ID. Published articles log at debug level so routine rebuilds stay quiet;
// validation failures log as warnings.
type LogPublisher struct {
	Logger *slog.Logger
}

func (p LogPublisher) Publish(ctx context.Context, batch Batch) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, event := range batch.Events {
		attrs := []slog.Attr{slog.String("build_id", batch.BuildID), slog.String("kind", string(event.Kind)), slog.String("path", event.Path)}
		switch event.Kind {
		case KindValidationFailed:
			attrs = append(attrs, slog.String("error", event.Message))
			p.Logger.LogAttrs(ctx, slog.LevelWarn, "content item excluded", attrs...)
		default:
			attrs = append(attrs, slog.String("url_path", event.URLPath))
			p.Logger.LogAttrs(ctx, slog.LevelDebug, "article published", attrs...)
		}
	}
	return nil
}
