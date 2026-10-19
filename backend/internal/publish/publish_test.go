package publish

import (
	"context"
	"errors"
	"testing"

	"github.com/reingma/Unframed/backend/internal/event"
)

type recorder struct {
	received [][]event.Event
	err      error
}

func (r *recorder) Receive(_ context.Context, events []event.Event) error {
	r.received = append(r.received, events)
	return r.err
}

func TestLocalDeliversToEverySubscriber(t *testing.T) {
	failed := errors.New("unreachable")
	first, second := &recorder{err: failed}, &recorder{}
	events := []event.Event{{Kind: event.KindArticlePublished, Path: "A.md"}}
	err := NewLocal(first, second).Publish(context.Background(), events)
	if !errors.Is(err, failed) {
		t.Fatalf("error = %v, want the failing subscriber's error", err)
	}
	for i, r := range []*recorder{first, second} {
		if len(r.received) != 1 || len(r.received[0]) != 1 || r.received[0][0] != events[0] {
			t.Fatalf("subscriber %d received %v, want one delivery of %v", i, r.received, events)
		}
	}
}

func TestLocalWithoutSubscribers(t *testing.T) {
	if err := NewLocal().Publish(context.Background(), []event.Event{{}}); err != nil {
		t.Fatal(err)
	}
}
