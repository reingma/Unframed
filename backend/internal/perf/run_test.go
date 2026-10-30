package perf

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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
	artifact, err := RunHTTPWorkload(context.Background(), w, testEnvironment(), testContent)
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
		if got := s.Metrics["response_bytes"]; got != float64(len(`{"status":"ok"}`)) {
			t.Fatalf("sample %d response_bytes = %v, want the body length", i, got)
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

// TestRunHTTPWorkloadNoRedirects checks the local-only contract survives a
// redirect: with redirects disabled the attempt surfaces as an error sample
// and no request leaves the loopback origin.
func TestRunHTTPWorkloadNoRedirects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://example.com/", http.StatusFound)
	}))
	defer server.Close()

	rate := 5.0
	w := Workload{ID: "fixture-redirect", Kind: KindHTTP, Target: server.URL, Rate: &rate, Duration: "200ms"}
	artifact, err := RunHTTPWorkload(context.Background(), w, testEnvironment(), testContent)
	if err != nil {
		t.Fatal(err)
	}
	if len(artifact.Samples) == 0 {
		t.Fatal("no samples recorded")
	}
	for i, s := range artifact.Samples {
		if s.Status == http.StatusOK {
			t.Fatalf("sample %d: redirect to external host was followed: %+v", i, s)
		}
		if s.Error == "" || !strings.Contains(s.Error, "redirect") {
			t.Fatalf("sample %d: expected a redirect refusal error: %+v", i, s)
		}
	}
}

// TestRunHTTPWorkloadCancelled checks an interrupted run errors instead of
// returning a silently truncated artifact.
func TestRunHTTPWorkloadCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rate := 50.0
	w := Workload{ID: "fixture-cancel", Kind: KindHTTP, Target: server.URL, Rate: &rate, Duration: "5s"}
	if _, err := RunHTTPWorkload(ctx, w, testEnvironment(), testContent); err == nil {
		t.Error("expected interruption error for cancelled context")
	}
}

func TestRunHTTPWorkloadRequiresRate(t *testing.T) {
	w := Workload{ID: "fixture-norate", Kind: KindHTTP, Target: "http://127.0.0.1:1/", Duration: "1s"}
	if _, err := RunHTTPWorkload(context.Background(), w, testEnvironment(), testContent); err == nil {
		t.Error("expected error for missing rate")
	}
}

// TestRunHTTPWorkloadExpectedStatus counts only the configured response code
// as success, so a not-found workload measures misses without errors.
func TestRunHTTPWorkloadExpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ok" {
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	rate := 20.0
	notFound := http.StatusNotFound
	for _, tc := range []struct {
		path   string
		status *int
		errors bool
	}{
		{"/missing", &notFound, false},
		{"/ok", &notFound, true},
		{"/missing", nil, true},
		{"/ok", nil, false},
	} {
		w := Workload{ID: "fixture-status", Kind: KindHTTP, Target: server.URL + tc.path, Rate: &rate, Duration: "100ms", Status: tc.status}
		artifact, err := RunHTTPWorkload(context.Background(), w, testEnvironment(), testContent)
		if err != nil {
			t.Fatal(err)
		}
		if len(artifact.Samples) == 0 {
			t.Fatal("no samples recorded")
		}
		for _, s := range artifact.Samples {
			if (s.Error != "") != tc.errors {
				t.Fatalf("%s with status %v: sample %+v", tc.path, tc.status, s)
			}
		}
	}
}
