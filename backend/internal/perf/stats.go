package perf

import (
	"fmt"
	"math"
	"slices"
)

// HistogramBuckets is the fixed bucket count for stored histograms.
const HistogramBuckets = 40

var quantilePoints = []struct {
	key string
	p   float64
}{
	{"p50", 0.50},
	{"p90", 0.90},
	{"p95", 0.95},
	{"p99", 0.99},
}

// ComputeStats reduces raw metric values to sufficient statistics: count,
// mean, sample variance (n-1), nearest-rank quantiles and an equal-width
// histogram over [min, max].
func ComputeStats(values []float64) (MetricStats, error) {
	n := len(values)
	if n == 0 {
		return MetricStats{}, fmt.Errorf("cannot compute statistics of zero values")
	}
	sorted := slices.Clone(values)
	slices.Sort(sorted)

	var mean, m2 float64
	for i, v := range values {
		delta := v - mean
		mean += delta / float64(i+1)
		m2 += delta * (v - mean)
	}
	variance := 0.0
	if n > 1 {
		variance = m2 / float64(n-1)
	}

	quantiles := make(map[string]float64, len(quantilePoints))
	for _, q := range quantilePoints {
		idx := max(int(math.Ceil(q.p*float64(n)))-1, 0)
		quantiles[q.key] = sorted[idx]
	}

	minV, maxV := sorted[0], sorted[n-1]
	width := (maxV - minV) / HistogramBuckets
	counts := make([]int, HistogramBuckets)
	if width == 0 {
		// All values identical: a single non-empty bucket and a unit width.
		counts[0] = n
		width = 1
	} else {
		for _, v := range values {
			b := min(int((v-minV)/width), HistogramBuckets-1)
			counts[b]++
		}
	}

	return MetricStats{
		Values:    n,
		Mean:      mean,
		Variance:  variance,
		Min:       minV,
		Max:       maxV,
		Quantiles: quantiles,
		Histogram: Histogram{Min: minV, BucketWidth: width, Counts: counts},
	}, nil
}

// z95 is the 97.5th percentile of the standard normal distribution. With the
// sample sizes the registry produces (rate x duration), the normal
// approximation on stored n/mean/variance is adequate; richer modeling is a
// documented deferral.
const z95 = 1.959963986120195

// MeanDeltaCI returns the delta of two means and its 95% confidence interval
// from stored sufficient statistics (Welch form on n, mean, variance).
func MeanDeltaCI(a, b MetricStats) (delta, lo, hi float64) {
	se := math.Sqrt(a.Variance/float64(a.Values) + b.Variance/float64(b.Values))
	delta = b.Mean - a.Mean
	return delta, delta - z95*se, delta + z95*se
}
