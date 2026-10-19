package logs

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/reingma/Unframed/backend/internal/event"
)

var events = []event.Event{
	{SchemaVersion: event.SchemaVersion, ID: "BUILD1-1", BuildID: "BUILD1", Kind: event.KindArticlePublished, Path: "Essays/Hello-World.md", URLPath: "/essays/hello-world"},
	{SchemaVersion: event.SchemaVersion, ID: "BUILD1-2", BuildID: "BUILD1", Kind: event.KindValidationFailed, Path: "Bad Name.md", Message: `invalid name "Bad Name"`},
}

func TestSubscriberCorrelatesEvents(t *testing.T) {
	var out bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&out, &slog.HandlerOptions{Level: slog.LevelDebug}))
	if err := (Subscriber{Logger: logger}).Receive(context.Background(), events); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != len(events) {
		t.Fatalf("got %d records, want %d:\n%s", len(lines), len(events), out.String())
	}
	for i, line := range lines {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatal(err)
		}
		e := events[i]
		if record["build_id"] != e.BuildID || record["event_id"] != e.ID || record["kind"] != string(e.Kind) || record["path"] != e.Path {
			t.Errorf("record %d = %v", i, record)
		}
	}
}

func TestSubscriberHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := (Subscriber{Logger: slog.New(slog.DiscardHandler)}).Receive(ctx, events); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}
