package api

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/reingma/Unframed/backend/internal/rebuild"
	"github.com/reingma/Unframed/backend/internal/snapshot"
)

type fakeReloader struct{ err error }

func (f fakeReloader) Reload(context.Context) (rebuild.Outcome, error) {
	return rebuild.Outcome{}, f.err
}
func (f fakeReloader) Last() rebuild.Outcome { return rebuild.Outcome{} }

func TestReloadStatusIsLogged(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
	}{
		{fmt.Errorf("%w: %w", rebuild.ErrStopped, context.Canceled), http.StatusServiceUnavailable},
		{context.Canceled, StatusClientClosedRequest},
		{context.DeadlineExceeded, http.StatusGatewayTimeout},
		{fmt.Errorf("%w: boom", rebuild.ErrBuildFailed), http.StatusInternalServerError},
	} {
		var logs bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&logs, nil))
		h := New(&snapshot.Store{}, fakeReloader{tc.err}, NewToken(""), NewToken("admin"), logger)
		req := httptest.NewRequest(http.MethodPost, "/admin/reload", nil)
		req.Header.Set("Authorization", "Bearer admin")
		rec := httptest.NewRecorder()
		Observe(http.HandlerFunc(h.Reload), logger).ServeHTTP(rec, req)
		if rec.Code != tc.status || !strings.Contains(logs.String(), fmt.Sprintf(`"status":%d`, tc.status)) {
			t.Errorf("reload error %v: responded %d, logged\n%s\nwant %d", tc.err, rec.Code, logs.String(), tc.status)
		}
	}
}
