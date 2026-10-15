package engineering

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
)

var (
	ErrInvalidKey = errors.New("invalid record key")
	ErrNotFound   = errors.New("record not found")
)

type Key struct {
	value string
}

func ParseKey(raw string) (Key, error) {
	if len(raw) == 0 || len(raw) > 64 || raw[0] < 'a' || raw[0] > 'z' {
		return Key{}, ErrInvalidKey
	}
	for i := 1; i < len(raw); i++ {
		c := raw[i]
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
			return Key{}, ErrInvalidKey
		}
	}
	return Key{value: raw}, nil
}

func (key Key) String() string {
	return key.value
}

func ParseKeyInput(raw []byte) (Key, error) {
	var input struct {
		Key *string `json:"key"`
	}
	if err := json.Unmarshal(raw, &input); err != nil {
		return Key{}, ErrInvalidKey
	}
	if input.Key == nil {
		return Key{}, ErrInvalidKey
	}
	return ParseKey(*input.Key)
}

type Record struct {
	Key     Key
	Payload string
}

type Lookup interface {
	Get(context.Context, Key) (Record, error)
}

type requestIDKey struct{}

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

func Read(ctx context.Context, source Lookup, key Key, logger *slog.Logger) (Record, error) {
	if key.value == "" {
		return Record{}, ErrInvalidKey
	}
	if err := ctx.Err(); err != nil {
		return Record{}, err
	}
	id, _ := ctx.Value(requestIDKey{}).(string)
	logger = logger.With("request_id", id, "stage", "lookup")
	logger.InfoContext(ctx, "lookup", "event", "started")
	record, err := source.Get(ctx, key)
	if err != nil {
		logger.ErrorContext(ctx, "lookup", "event", "failed")
		return Record{}, err
	}
	logger.InfoContext(ctx, "lookup", "event", "completed")
	return record, nil
}
