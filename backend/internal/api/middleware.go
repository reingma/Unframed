package api

import (
	"context"
	"crypto/rand"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// RequestIDHeader carries the correlation ID from Next.js.
const RequestIDHeader = "X-Request-ID"

const maxRequestIDLength = 128

type requestKey struct{}

type requestInfo struct {
	id      string
	buildID string
}

func setBuildID(ctx context.Context, buildID string) {
	if info, ok := ctx.Value(requestKey{}).(*requestInfo); ok {
		info.buildID = buildID
	}
}

func requestIDAttr(ctx context.Context) slog.Attr {
	id := ""
	if info, ok := ctx.Value(requestKey{}).(*requestInfo); ok {
		id = info.id
	}
	return slog.String("request_id", id)
}

// Observe assigns each request a correlation ID, echoes it in the response
// and logs the request's completion with the build ID that served it. A
// valid incoming X-Request-ID is kept; otherwise one is generated. Logged
// paths never contain the preview token.
func Observe(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		info := &requestInfo{id: requestID(r.Header.Get(RequestIDHeader))}
		w.Header().Set(RequestIDHeader, info.id)
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r.WithContext(context.WithValue(r.Context(), requestKey{}, info)))

		level, msg := slog.LevelInfo, "request completed"
		if rec.status >= http.StatusInternalServerError {
			level, msg = slog.LevelError, "request failed"
		}
		logger.LogAttrs(r.Context(), level, msg,
			slog.String("request_id", info.id),
			slog.String("method", r.Method),
			slog.String("path", redactPath(r.URL.Path)),
			slog.Int("status", rec.status),
			slog.Duration("duration", time.Since(start)),
			slog.String("build_id", info.buildID),
		)
	})
}

func requestID(incoming string) string {
	if validRequestID(incoming) {
		return incoming
	}
	return rand.Text()
}

func validRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLength {
		return false
	}
	for _, c := range []byte(id) {
		ok := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == '-'
		if !ok {
			return false
		}
	}
	return true
}

const previewSegment = "/preview/"

// redactPath hides whatever may be a preview token: everything after a
// preview segment in any letter case, except that a well-formed
// /preview/<token>/articles/... or /preview/<token>/diagnostics path keeps a
// tail that has no preview segment of its own.
func redactPath(p string) string {
	i := indexPreviewSegment(p)
	if i < 0 {
		return p
	}
	head := p[:i+len(previewSegment)]
	if i == 0 {
		token, tail, _ := strings.Cut(p[len(head):], "/")
		wellFormed := tail == "diagnostics" || strings.HasPrefix(tail, "articles/")
		if token != "" && wellFormed && indexPreviewSegment("/"+tail) < 0 {
			return head + "<redacted>/" + tail
		}
	}
	return head + "<redacted>"
}

// indexPreviewSegment returns the index of the first "/preview/" in p,
// compared case-insensitively, or -1.
func indexPreviewSegment(p string) int {
	for i := 0; i+len(previewSegment) <= len(p); i++ {
		if strings.EqualFold(p[i:i+len(previewSegment)], previewSegment) {
			return i
		}
	}
	return -1
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }
