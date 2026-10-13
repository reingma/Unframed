package router

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/reingma/Unframed/backend/internal/health"
)

func New() *httprouter.Router {
	router := httprouter.New()
	router.HandlerFunc(http.MethodGet, "/healthcheck", health.Handler)
	return router
}
