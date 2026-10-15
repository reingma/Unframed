package perf

import (
	"encoding/json/v2"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	workloadIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	revisionPattern   = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

// Workload is one declarative registry entry. Kind selects which fields
// apply: http workloads carry rate (requests per second) and duration;
// browser workloads carry repetitions (reserved for the frontend runner).
type Workload struct {
	ID          string   `json:"id"`
	Kind        string   `json:"kind"`
	Target      string   `json:"target"`
	Rate        *float64 `json:"rate,omitempty"`
	Duration    string   `json:"duration,omitempty"`
	Repetitions *int     `json:"repetitions,omitempty"`
}

// Registry is the declarative workload list. Adding a measurable path is one
// entry, not new plumbing.
type Registry struct {
	Version   int        `json:"version"`
	Workloads []Workload `json:"workloads"`
}

// isLocalTarget enforces the local-only load-target contract: targets must
// resolve to a loopback address so the registry can never aim load outward.
func isLocalTarget(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid target URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("target scheme %q is not http(s)", u.Scheme)
	}
	host := u.Hostname()
	if host == "localhost" {
		return nil
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return nil
	}
	return fmt.Errorf("target host %q is not a loopback address", host)
}

func (w *Workload) validate() error {
	if !workloadIDPattern.MatchString(w.ID) {
		return fmt.Errorf("workload id %q must match %s", w.ID, workloadIDPattern)
	}
	switch w.Kind {
	case KindHTTP:
		switch {
		case w.Rate == nil || *w.Rate <= 0:
			return fmt.Errorf("http workload %q requires a positive rate", w.ID)
		case w.Duration == "":
			return fmt.Errorf("http workload %q requires a duration", w.ID)
		case w.Repetitions != nil:
			return fmt.Errorf("http workload %q must not set repetitions", w.ID)
		}
		d, err := time.ParseDuration(w.Duration)
		if err != nil || d <= 0 {
			return fmt.Errorf("http workload %q has invalid duration %q", w.ID, w.Duration)
		}
		if err := isLocalTarget(w.Target); err != nil {
			return fmt.Errorf("http workload %q: %w", w.ID, err)
		}
	case KindBrowser:
		switch {
		case w.Repetitions == nil || *w.Repetitions <= 0:
			return fmt.Errorf("browser workload %q requires positive repetitions", w.ID)
		case w.Rate != nil || w.Duration != "":
			return fmt.Errorf("browser workload %q must not set rate or duration", w.ID)
		case w.Target == "":
			return fmt.Errorf("browser workload %q requires a target", w.ID)
		}
		if u, err := url.Parse(w.Target); err == nil && u.IsAbs() {
			if err := isLocalTarget(w.Target); err != nil {
				return fmt.Errorf("browser workload %q: %w", w.ID, err)
			}
		} else if !strings.HasPrefix(w.Target, "/") || strings.HasPrefix(w.Target, "//") ||
			strings.ContainsAny(w.Target, "\\") {
			return fmt.Errorf("browser workload %q target %q is not a route path or local URL", w.ID, w.Target)
		}
	default:
		return fmt.Errorf("workload %q has invalid kind %q", w.ID, w.Kind)
	}
	return nil
}

// ParseRegistry decodes and validates a workload registry: version, unique
// well-formed ids and valid local targets.
func ParseRegistry(data []byte) (Registry, error) {
	var r Registry
	if err := json.Unmarshal(data, &r, json.RejectUnknownMembers(true)); err != nil {
		return Registry{}, err
	}
	if r.Version != SchemaVersion {
		return Registry{}, fmt.Errorf("unsupported registry version %d", r.Version)
	}
	if len(r.Workloads) == 0 {
		return Registry{}, fmt.Errorf("registry has no workloads")
	}
	seen := map[string]bool{}
	for i := range r.Workloads {
		w := &r.Workloads[i]
		if err := w.validate(); err != nil {
			return Registry{}, err
		}
		if seen[w.ID] {
			return Registry{}, fmt.Errorf("duplicate workload id %q", w.ID)
		}
		seen[w.ID] = true
	}
	return r, nil
}
