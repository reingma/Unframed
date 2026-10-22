// Package api serves the current build snapshot as JSON: the manifest,
// published article envelopes, token-gated previews and diagnostics, and a
// token-gated synchronous reload. Requests read only the snapshot, never the
// content source.
package api

import (
	"context"
	"crypto/subtle"
	"encoding/json/v2"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/reingma/Unframed/backend/internal/content"
	"github.com/reingma/Unframed/backend/internal/rebuild"
	"github.com/reingma/Unframed/backend/internal/snapshot"
)

// ReloadTimeout bounds how long a reload request waits for its build. It is
// below the server's write timeout so the response can still be written.
const ReloadTimeout = 8 * time.Second

// Snapshots is the server-owned read side of the build output. A shared
// snapshot store can replace the in-process one behind it.
type Snapshots interface {
	Current() *snapshot.Snapshot
}

// StatusClientClosedRequest is logged when the client disconnects before a
// response, following the nginx convention.
const StatusClientClosedRequest = 499

// Reloader runs reloads and reports the latest build outcome.
type Reloader interface {
	Reload(ctx context.Context) (rebuild.Outcome, error)
	Last() rebuild.Outcome
}

// Token is a server-only shared secret. The zero Token disables what it
// gates. Its String and LogValue never reveal the value.
type Token struct{ value string }

// NewToken returns a Token for value; an empty value is the zero Token.
func NewToken(value string) Token { return Token{value: value} }

func (t Token) enabled() bool        { return t.value != "" }
func (t Token) String() string       { return "<redacted>" }
func (t Token) LogValue() slog.Value { return slog.StringValue("<redacted>") }
func (t Token) matches(got string) bool {
	return t.enabled() && subtle.ConstantTimeCompare([]byte(t.value), []byte(got)) == 1
}

// Handler serves the API routes.
type Handler struct {
	snapshots Snapshots
	rebuilds  Reloader
	preview   Token
	admin     Token
	logger    *slog.Logger
}

// New returns the API handlers. A zero preview token disables preview and
// diagnostics; a zero admin token disables reload.
func New(snapshots Snapshots, rebuilds Reloader, preview, admin Token, logger *slog.Logger) *Handler {
	return &Handler{snapshots: snapshots, rebuilds: rebuilds, preview: preview, admin: admin, logger: logger}
}

// Manifest serves the public manifest.
func (h *Handler) Manifest(w http.ResponseWriter, r *http.Request) {
	s := h.current(r)
	writeDocument(w, s.Manifest())
}

// Article serves the published envelope at the request's slug path.
func (h *Handler) Article(w http.ResponseWriter, r *http.Request) {
	urlPath, ok := articlePath(r)
	if !ok {
		h.notFound(w, r)
		return
	}
	doc, ok := h.current(r).Envelope(urlPath, false)
	if !ok {
		h.notFound(w, r)
		return
	}
	writeDocument(w, doc)
}

// PreviewArticle serves the draft at the slug path, or else the published
// envelope. A miss lists the validation errors that explain it.
func (h *Handler) PreviewArticle(w http.ResponseWriter, r *http.Request) {
	if !h.previewAllowed(r) {
		h.notFound(w, r)
		return
	}
	urlPath, ok := articlePath(r)
	if !ok {
		h.notFound(w, r)
		return
	}
	s := h.current(r)
	if doc, ok := s.Envelope(urlPath, true); ok {
		writeDocument(w, doc)
		return
	}
	if doc, ok := s.Envelope(urlPath, false); ok {
		writeDocument(w, doc)
		return
	}
	h.writeJSON(w, r, http.StatusNotFound, previewMiss{
		Error:       "not found",
		URLPath:     urlPath,
		BuildID:     s.BuildID(),
		Diagnostics: nonNil(s.DiagnosticsFor(urlPath)),
	})
}

type previewMiss struct {
	Error       string                `json:"error"`
	URLPath     string                `json:"urlPath"`
	BuildID     string                `json:"buildId"`
	Diagnostics []snapshot.Diagnostic `json:"diagnostics"`
}

// Diagnostics reports the serving snapshot, its validation errors and the
// latest build outcome, which may be a failure still served by an older
// snapshot.
func (h *Handler) Diagnostics(w http.ResponseWriter, r *http.Request) {
	if !h.previewAllowed(r) {
		h.notFound(w, r)
		return
	}
	s := h.current(r)
	diagnostics := s.Diagnostics()
	h.writeJSON(w, r, http.StatusOK, diagnosticsResponse{
		Serving: buildSummary{
			BuildID:          s.BuildID(),
			BuiltAt:          s.BuiltAt().UTC(),
			Envelopes:        s.Envelopes(),
			ValidationErrors: len(diagnostics),
		},
		Diagnostics: nonNil(diagnostics),
		LastBuild:   newLastBuild(h.rebuilds.Last()),
	})
}

type diagnosticsResponse struct {
	Serving     buildSummary          `json:"serving"`
	Diagnostics []snapshot.Diagnostic `json:"diagnostics"`
	LastBuild   lastBuild             `json:"lastBuild"`
}

type buildSummary struct {
	BuildID          string    `json:"buildId"`
	BuiltAt          time.Time `json:"builtAt,omitzero"`
	Envelopes        int       `json:"envelopes"`
	ValidationErrors int       `json:"validationErrors"`
}

type lastBuild struct {
	Trigger          rebuild.Trigger `json:"trigger"`
	Signals          int             `json:"signals"`
	Started          time.Time       `json:"started"`
	DurationMS       float64         `json:"durationMs"`
	Outcome          string          `json:"outcome"`
	BuildID          string          `json:"buildId,omitempty"`
	Envelopes        int             `json:"envelopes"`
	ValidationErrors int             `json:"validationErrors"`
	Error            string          `json:"error,omitempty"`
}

func newLastBuild(o rebuild.Outcome) lastBuild {
	last := lastBuild{
		Trigger:          o.Trigger,
		Signals:          o.Signals,
		Started:          o.Started.UTC(),
		DurationMS:       float64(o.Duration.Microseconds()) / 1000,
		Outcome:          "succeeded",
		BuildID:          o.BuildID,
		Envelopes:        o.Envelopes,
		ValidationErrors: o.ValidationErrors,
	}
	if o.Err != nil {
		last.Outcome, last.Error = "failed", o.Err.Error()
	}
	return last
}

// Reload runs a build that starts after the request and reports it. Only
// "Authorization: Bearer <admin token>" authorizes it.
func (h *Handler) Reload(w http.ResponseWriter, r *http.Request) {
	if !h.admin.enabled() {
		h.notFound(w, r)
		return
	}
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || !h.admin.matches(token) {
		h.writeJSON(w, r, http.StatusUnauthorized, errorBody{Error: "unauthorized"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), ReloadTimeout)
	defer cancel()
	outcome, err := h.rebuilds.Reload(ctx)
	switch {
	case errors.Is(err, rebuild.ErrStopped):
		h.writeJSON(w, r, http.StatusServiceUnavailable, errorBody{Error: "server shutting down"})
	case errors.Is(err, rebuild.ErrBuildFailed):
		h.writeJSON(w, r, http.StatusInternalServerError, errorBody{Error: "build failed; see diagnostics"})
	case errors.Is(err, context.DeadlineExceeded):
		h.writeJSON(w, r, http.StatusGatewayTimeout, errorBody{Error: "reload timed out"})
	case err != nil:
		// The client went away; nobody reads the response, but the request
		// log records it as closed by the client.
		w.WriteHeader(StatusClientClosedRequest)
	default:
		setBuildID(r.Context(), outcome.BuildID)
		h.writeJSON(w, r, http.StatusOK, reloadResponse{
			BuildID:          outcome.BuildID,
			Envelopes:        outcome.Envelopes,
			ValidationErrors: outcome.ValidationErrors,
		})
	}
}

type reloadResponse struct {
	BuildID          string `json:"buildId"`
	Envelopes        int    `json:"envelopes"`
	ValidationErrors int    `json:"validationErrors"`
}

type errorBody struct {
	Error string `json:"error"`
}

func (h *Handler) previewAllowed(r *http.Request) bool {
	return h.preview.matches(httprouter.ParamsFromContext(r.Context()).ByName("token"))
}

// current reads the serving snapshot once per request and records its build
// ID for the request log.
func (h *Handler) current(r *http.Request) *snapshot.Snapshot {
	s := h.snapshots.Current()
	setBuildID(r.Context(), s.BuildID())
	return s
}

// notFound is the one public miss response, so it reveals nothing about why.
func (h *Handler) notFound(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, r, http.StatusNotFound, errorBody{Error: "not found"})
}

// NotFound serves the generic miss for unregistered routes.
func (h *Handler) NotFound() http.Handler { return http.HandlerFunc(h.notFound) }

func (h *Handler) writeJSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.MarshalWrite(w, v); err != nil {
		h.logger.WarnContext(r.Context(), "failed to write JSON response", "error", err, requestIDAttr(r.Context()))
	}
}

func writeDocument(w http.ResponseWriter, doc snapshot.Document) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(doc.JSON)
}

// articlePath parses the catch-all slug path of an article route into a URL
// path: "" is the root ("/"), otherwise each segment must be a lowercase slug.
// Encoded separators, dot segments, empty segments and trailing slashes fail.
func articlePath(r *http.Request) (string, bool) {
	escaped := strings.ToLower(r.URL.EscapedPath())
	if strings.Contains(escaped, "%2f") || strings.Contains(escaped, "%5c") {
		return "", false
	}
	raw := strings.TrimPrefix(httprouter.ParamsFromContext(r.Context()).ByName("path"), "/")
	if raw == "" {
		return "/", true
	}
	for segment := range strings.SplitSeq(raw, "/") {
		if _, err := content.ParseSlug(segment); err != nil {
			return "", false
		}
	}
	return "/" + raw, true
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
