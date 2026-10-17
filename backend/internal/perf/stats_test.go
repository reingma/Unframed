package perf

import (
	"math"
	"testing"
)

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func TestComputeStatsGolden(t *testing.T) {
	// Values 1..10: mean 5.5, sample variance 9.166..., nearest-rank
	// quantiles p50=5, p90=9, p95=10, p99=10.
	s, err := ComputeStats([]float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	if err != nil {
		t.Fatal(err)
	}
	if s.Values != 10 || !almostEqual(s.Mean, 5.5) || !almostEqual(s.Variance, 55.0/6.0) {
		t.Fatalf("got %+v", s)
	}
	if s.Min != 1 || s.Max != 10 {
		t.Fatalf("got min %v max %v", s.Min, s.Max)
	}
	wantQ := map[string]float64{"p50": 5, "p90": 9, "p95": 10, "p99": 10}
	for k, v := range wantQ {
		if s.Quantiles[k] != v {
			t.Errorf("quantile %s = %v, want %v", k, s.Quantiles[k], v)
		}
	}
	total := 0
	for _, c := range s.Histogram.Counts {
		total += c
	}
	if total != 10 || s.Histogram.Min != 1 || !almostEqual(s.Histogram.BucketWidth, 9.0/40.0) {
		t.Fatalf("histogram %+v", s.Histogram)
	}
}

func TestComputeStatsSingleValue(t *testing.T) {
	s, err := ComputeStats([]float64{42})
	if err != nil {
		t.Fatal(err)
	}
	if s.Values != 1 || s.Mean != 42 || s.Variance != 0 || s.Min != 42 || s.Max != 42 {
		t.Fatalf("got %+v", s)
	}
	if s.Histogram.Counts[0] != 1 {
		t.Fatalf("histogram %+v", s.Histogram)
	}
}

func TestComputeStatsEmpty(t *testing.T) {
	if _, err := ComputeStats(nil); err == nil {
		t.Error("expected error for zero values")
	}
}

func TestMeanDeltaCIGolden(t *testing.T) {
	// n=100 each, means 100 and 110, variances 25: delta 10,
	// SE = sqrt(0.25+0.25) ~= 0.7071, CI = 10 +/- 1.96*0.7071.
	a := MetricStats{Values: 100, Mean: 100, Variance: 25}
	b := MetricStats{Values: 100, Mean: 110, Variance: 25}
	delta, lo, hi := MeanDeltaCI(a, b)
	se := math.Sqrt(0.5)
	if !almostEqual(delta, 10) || !almostEqual(lo, 10-z95*se) || !almostEqual(hi, 10+z95*se) {
		t.Fatalf("got delta=%v lo=%v hi=%v", delta, lo, hi)
	}
}
