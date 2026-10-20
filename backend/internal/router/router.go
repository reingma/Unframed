package router

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/reingma/Unframed/backend/internal/api"
	"github.com/reingma/Unframed/backend/internal/health"
)

// New registers every route. Path cleaning and trailing-slash redirects are
// off so malformed article paths fail instead of being rewritten.
func New(handlers *api.Handler) *httprouter.Router {
	router := httprouter.New()
	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false
	router.NotFound = handlers.NotFound()
	router.HandlerFunc(http.MethodGet, "/healthcheck", health.Handler)
	router.HandlerFunc(http.MethodGet, "/manifest", handlers.Manifest)
	router.HandlerFunc(http.MethodGet, "/articles/*path", handlers.Article)
	router.HandlerFunc(http.MethodGet, "/preview/:token/articles/*path", handlers.PreviewArticle)
	router.HandlerFunc(http.MethodGet, "/preview/:token/diagnostics", handlers.Diagnostics)
	router.HandlerFunc(http.MethodPost, "/admin/reload", handlers.Reload)
	return router
}
