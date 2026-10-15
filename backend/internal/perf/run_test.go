package perf

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestRunHTTPWorkloadEndToEnd drives a fixture run against a live local
// server and checks the artifact is well-formed and compressible.
func TestRunHTTPWorkloadEndToEnd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	rate := 20.0
	w := Workload{
		ID:       "fixture-endpoint",
		Kind:     KindHTTP,
		Target:   server.URL,
		Rate:     &rate,
		Duration: "200ms",
	}
	artifact, err := RunHTTPWorkload(context.Background(), w, testEnvironment())
	if err != nil {
		t.Fatal(err)
	}

	data, err := MarshalRunArtifact(artifact)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseRunArtifact(data)
	if err != nil {
		t.Fatalf("artifact is not well-formed: %v", err)
	}
	if len(parsed.Samples) == 0 {
		t.Fatal("no samples recorded")
	}
	for i, s := range parsed.Samples {
		if s.Error != "" || s.Status != http.StatusOK {
			t.Fatalf("sample %d failed: %+v", i, s)
		}
		if _, ok := s.Metrics["latency_ns"]; !ok {
			t.Fatalf("sample %d lacks latency_ns", i)
		}
	}

	ws, err := CompressArtifact(parsed)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Samples != len(parsed.Samples) || ws.Errors != 0 ||
		ws.Metrics["latency_ns"].Values != len(parsed.Samples) {
		t.Fatalf("compressed stats %+v", ws)
	}
}
