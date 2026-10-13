package health

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func Handler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(struct {
		Status string `json:"status"`
	}{Status: "ok"}); err != nil {
		slog.Error("write health response", "error", err)
	}
}
