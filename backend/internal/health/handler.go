package health

import (
	"encoding/json/v2"
	"log/slog"
	"net/http"
)

func Handler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.MarshalWrite(w, struct {
		Status string `json:"status"`
	}{Status: "ok"}); err != nil {
		slog.Error("failed to write JSON healthcheck response", "error", err)
	}
}
