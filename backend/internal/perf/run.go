package perf

import (
	"context"
	"fmt"
	"net/http"
	"time"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

// RunHTTPWorkload executes one http registry workload with vegeta's open
// load model at constant rate and collects the raw samples. Response bodies
// are never retained: artifacts carry metrics, status and error strings only.
func RunHTTPWorkload(ctx context.Context, w Workload, env Environment) (RunArtifact, error) {
	if w.Kind != KindHTTP {
		return RunArtifact{}, fmt.Errorf("workload %q kind %q is not http", w.ID, w.Kind)
	}
	if w.Rate == nil || *w.Rate <= 0 {
		return RunArtifact{}, fmt.Errorf("workload %q requires a positive rate", w.ID)
	}
	duration, err := time.ParseDuration(w.Duration)
	if err != nil || duration <= 0 {
		return RunArtifact{}, fmt.Errorf("workload %q has invalid duration %q", w.ID, w.Duration)
	}
	targeter := vegeta.NewStaticTargeter(vegeta.Target{
		Method: http.MethodGet,
		URL:    w.Target,
	})
	// Redirects(0): the loopback target contract applies to every request,
	// not just the first hop — a redirect must be a sample, not a bypass.
	attacker := vegeta.NewAttacker(vegeta.Redirects(0))
	rate := vegeta.Rate{Freq: 1, Per: time.Duration(float64(time.Second) / *w.Rate)}

	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			attacker.Stop()
		case <-done:
		}
	}()

	started := time.Now().UTC()
	var samples []Sample
	for res := range attacker.Attack(targeter, rate, duration, w.ID) {
		samples = append(samples, Sample{
			Metrics: map[string]float64{"latency_ns": float64(res.Latency)},
			Status:  int(res.Code),
			Error:   res.Error,
		})
	}
	if err := ctx.Err(); err != nil {
		return RunArtifact{}, fmt.Errorf("workload %q interrupted: %w", w.ID, err)
	}
	return RunArtifact{
		Version:     SchemaVersion,
		WorkloadID:  w.ID,
		Kind:        w.Kind,
		StartedAt:   started,
		FinishedAt:  time.Now().UTC(),
		Environment: env,
		Samples:     samples,
	}, nil
}
