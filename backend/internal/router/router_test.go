package router_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/reingma/Unframed/backend/internal/router"
)

func TestRoutes(t *testing.T) {
	for _, test := range []struct {
		method string
		path   string
		status int
	}{
		{http.MethodGet, "/healthcheck", http.StatusOK},
		{http.MethodPost, "/healthcheck", http.StatusMethodNotAllowed},
		{http.MethodGet, "/healthz", http.StatusNotFound},
		{http.MethodGet, "/missing", http.StatusNotFound},
	} {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.New().ServeHTTP(response, httptest.NewRequest(test.method, test.path, nil))
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d", response.Code, test.status)
			}
			if test.status == http.StatusMethodNotAllowed && !strings.Contains(response.Header().Get("Allow"), http.MethodGet) {
				t.Fatalf("Allow = %q, want GET", response.Header().Get("Allow"))
			}
		})
	}
}
