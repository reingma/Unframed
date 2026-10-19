// Package publish is the build's outbound event boundary. A build hands its
// events to one Publisher, which puts them on a bus; the bus, not the build,
// reaches every subscriber. Local is the in-process bus; a message-bus
// publisher such as SNS or EventBridge replaces it in production, with
// subscribers running outside this process.
package publish

import (
	"context"
	"errors"

	"github.com/reingma/Unframed/backend/internal/event"
)

// Publisher delivers the events of one build. Publish honors cancellation
// and deadlines; an error means some subscribers may not have received the
// events.
type Publisher interface {
	Publish(ctx context.Context, events []event.Event) error
}

// Subscriber receives the events of one build from a Local bus.
type Subscriber interface {
	Receive(ctx context.Context, events []event.Event) error
}

// Local is an in-process bus: it delivers every build's events to each of
// its subscribers, even after one fails, and joins their errors.
type Local struct {
	subscribers []Subscriber
}

// NewLocal returns a bus delivering to subscribers in order.
func NewLocal(subscribers ...Subscriber) *Local {
	return &Local{subscribers: subscribers}
}

func (l *Local) Publish(ctx context.Context, events []event.Event) error {
	var errs []error
	for _, subscriber := range l.subscribers {
		errs = append(errs, subscriber.Receive(ctx, events))
	}
	return errors.Join(errs...)
}
