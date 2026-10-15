package engineering

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strings"
	"testing"
	"time"
)

type linearLookup []Record

func (records linearLookup) Get(ctx context.Context, key Key) (Record, error) {
	if err := ctx.Err(); err != nil {
		return Record{}, err
	}
	for _, record := range records {
		if err := ctx.Err(); err != nil {
			return Record{}, err
		}
		if record.Key == key {
			return record, nil
		}
	}
	return Record{}, ErrNotFound
}

type indexedLookup map[Key]Record

func (records indexedLookup) Get(ctx context.Context, key Key) (Record, error) {
	if err := ctx.Err(); err != nil {
		return Record{}, err
	}
	record, ok := records[key]
	if !ok {
		return Record{}, ErrNotFound
	}
	return record, nil
}

func implementations(records []Record) map[string]Lookup {
	indexed := make(indexedLookup, len(records))
	for _, record := range records {
		indexed[record.Key] = record
	}
	return map[string]Lookup{"linear": linearLookup(records), "indexed": indexed}
}

func mustKey(t testing.TB, raw string) Key {
	t.Helper()
	key, err := ParseKey(raw)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

func TestParseKey(t *testing.T) {
	for _, raw := range []string{"a", "record-1", strings.Repeat("a", 64)} {
		t.Run(raw, func(t *testing.T) {
			key, err := ParseKey(raw)
			if err != nil || key.String() != raw {
				t.Fatalf("parse = %q, %v", key.String(), err)
			}
		})
	}
	for _, raw := range []string{"", "1record", "UPPER", "two words", "../record", "é", "a\x00", strings.Repeat("a", 65)} {
		t.Run(fmt.Sprintf("invalid-%q", raw), func(t *testing.T) {
			key, err := ParseKey(raw)
			if !errors.Is(err, ErrInvalidKey) || key != (Key{}) {
				t.Fatalf("invalid input escaped: %q, %v", key.String(), err)
			}
		})
	}
}

func TestParseKeyInput(t *testing.T) {
	key, err := ParseKeyInput([]byte(`{"key":"record-1"}`))
	if err != nil || key.String() != "record-1" {
		t.Fatalf("valid input = %q, %v", key.String(), err)
	}
	for _, raw := range []string{
		`null`, `{}`, `{"key":null}`, `{"key":""}`, `{"key":"../record"}`,
		`{"key":0}`, `{"key":false}`, `{"key":{}}`, `[]`, `{"key":`,
	} {
		t.Run(raw, func(t *testing.T) {
			key, err := ParseKeyInput([]byte(raw))
			if !errors.Is(err, ErrInvalidKey) || key != (Key{}) {
				t.Fatalf("invalid wire input escaped: %q, %v", key.String(), err)
			}
		})
	}
}

func TestReadContract(t *testing.T) {
	key := mustKey(t, "record-1")
	want := Record{Key: key, Payload: "private-payload"}
	for name, source := range implementations([]Record{want}) {
		t.Run(name, func(t *testing.T) {
			got, err := Read(context.Background(), source, key, quietLogger())
			if err != nil || got != want {
				t.Fatalf("read = %v, %v; want %v", got, err, want)
			}
			got, err = Read(context.Background(), source, mustKey(t, "missing"), quietLogger())
			if !errors.Is(err, ErrNotFound) || got != (Record{}) {
				t.Fatalf("missing read = %v, %v", got, err)
			}
			got, err = Read(context.Background(), source, Key{}, quietLogger())
			if !errors.Is(err, ErrInvalidKey) || got != (Record{}) {
				t.Fatalf("zero key read = %v, %v", got, err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			got, err = Read(ctx, source, key, quietLogger())
			if !errors.Is(err, context.Canceled) || got != (Record{}) {
				t.Fatalf("cancelled read = %v, %v", got, err)
			}
		})
	}
}

func TestLookupCancellation(t *testing.T) {
	key := mustKey(t, "record-1")
	for _, records := range [][]Record{nil, {{Key: key, Payload: "fixture"}}} {
		for name, source := range implementations(records) {
			t.Run(fmt.Sprintf("%s/records=%d", name, len(records)), func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				got, err := source.Get(ctx, key)
				if !errors.Is(err, context.Canceled) || got != (Record{}) {
					t.Fatalf("cancelled lookup = %v, %v", got, err)
				}
			})
		}
	}
}

type observingLookup struct {
	t   *testing.T
	err error
}

func (source observingLookup) Get(ctx context.Context, key Key) (Record, error) {
	if id, _ := ctx.Value(requestIDKey{}).(string); id != "request-123" {
		source.t.Errorf("request correlation lost: %q", id)
	}
	if _, ok := ctx.Deadline(); !ok {
		source.t.Error("deadline lost at boundary")
	}
	return Record{}, source.err
}

func TestRequestCorrelationAndFailure(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	ctx, cancel := context.WithTimeout(WithRequestID(context.Background(), "request-123"), time.Minute)
	defer cancel()
	wantErr := errors.New("provider error with private-payload")
	_, err := Read(ctx, observingLookup{t: t, err: wantErr}, mustKey(t, "record-1"), logger)
	if !errors.Is(err, wantErr) {
		t.Fatalf("provider failure not preserved: %v", err)
	}
	assertRequestEvents(t, &output, "started", "failed")
}

func TestRequestCorrelationAndCompletion(t *testing.T) {
	key := mustKey(t, "record-1")
	want := Record{Key: key, Payload: "private-payload"}
	for name, source := range implementations([]Record{want}) {
		t.Run(name, func(t *testing.T) {
			var output bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&output, nil))
			ctx := WithRequestID(context.Background(), "request-123")
			if got, err := Read(ctx, source, key, logger); err != nil || got != want {
				t.Fatalf("read = %v, %v", got, err)
			}
			assertRequestEvents(t, &output, "started", "completed")
		})
	}
}

func assertRequestEvents(t *testing.T, output *bytes.Buffer, required ...string) {
	t.Helper()
	decoder := json.NewDecoder(output)
	events := make(map[string]bool)
	for {
		var event map[string]any
		if err := decoder.Decode(&event); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		if event["request_id"] != "request-123" || event["stage"] != "lookup" {
			t.Fatalf("uncorrelated event: %v", event)
		}
		events[fmt.Sprint(event["event"])] = true
		encoded, err := json.Marshal(event)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(encoded, []byte("private-payload")) {
			t.Fatal("sensitive provider data leaked to logs")
		}
	}
	for _, event := range required {
		if !events[event] {
			t.Fatalf("missing boundary lifecycle event %q: %v", event, events)
		}
	}
}

func FuzzParseKey(f *testing.F) {
	for _, seed := range []string{"", "a", "record-1", "../record", "a\x00", "é", strings.Repeat("a", 64), strings.Repeat("a", 65)} {
		f.Add(seed)
	}
	grammar := regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)
	f.Fuzz(func(t *testing.T, raw string) {
		key, err := ParseKey(raw)
		if grammar.MatchString(raw) {
			if err != nil || key.String() != raw {
				t.Fatalf("canonical input rejected or altered: %q, %v", key.String(), err)
			}
			again, err := ParseKey(key.String())
			if err != nil || again != key {
				t.Fatal("canonical key did not round-trip")
			}
		} else if !errors.Is(err, ErrInvalidKey) || key != (Key{}) {
			t.Fatalf("invalid input escaped: %q, %v", key.String(), err)
		}
	})
}

func BenchmarkLookup(b *testing.B) {
	for _, size := range []int{8, 1024} {
		records := make([]Record, size)
		for i := range records {
			records[i] = Record{Key: mustKey(b, fmt.Sprintf("record-%d", i)), Payload: "fixture"}
		}
		key := records[len(records)-1].Key
		for name, source := range implementations(records) {
			b.Run(fmt.Sprintf("%s/records=%d", name, size), func(b *testing.B) {
				ctx := context.Background()
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					got, err := source.Get(ctx, key)
					if err != nil || got.Key != key {
						b.Fatal("lookup contract failed", err)
					}
				}
			})
		}
	}
}
