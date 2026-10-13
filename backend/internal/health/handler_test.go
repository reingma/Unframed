package health_test

import (
	"bytes"
	"encoding/json/v2"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/reingma/Unframed/backend/internal/health"
)

func TestHandler(t *testing.T) {
	response := httptest.NewRecorder()
	health.Handler(response, httptest.NewRequest(http.MethodGet, "/healthcheck", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content type = %q, want application/json", got)
	}
	var body map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["status"] != "ok" || len(body) != 1 {
		t.Fatalf("body = %v, want only status: ok", body)
	}
}

func TestHandlerLogsWriteFailure(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	response := &failingResponseWriter{header: make(http.Header)}
	health.Handler(response, httptest.NewRequest(http.MethodGet, "/healthcheck", nil))

	var record map[string]string
	if err := json.Unmarshal(logs.Bytes(), &record); err != nil {
		t.Fatalf("invalid log JSON: %v", err)
	}
	if record["level"] != "ERROR" || record["msg"] != "failed to write JSON healthcheck response" {
		t.Fatalf("unexpected log record: %v", record)
	}
	if !strings.Contains(record["error"], io.ErrClosedPipe.Error()) {
		t.Fatalf("error = %q, want write failure details", record["error"])
	}
}

type failingResponseWriter struct {
	header http.Header
}

func (w *failingResponseWriter) Header() http.Header { return w.header }
func (*failingResponseWriter) WriteHeader(int)       {}
func (*failingResponseWriter) Write([]byte) (int, error) {
	return 0, io.ErrClosedPipe
}
