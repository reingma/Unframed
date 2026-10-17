package perf

import (
	"fmt"
	"io"
	"slices"
	"strings"
)

// MetricDelta is the per-metric comparison of two records: the delta of the
// means with its 95% confidence interval, plus point deltas of the stored
// quantiles (their distributions are not stored, so they carry no interval).
type MetricDelta struct {
	WorkloadID     string
	Kind           string
	Metric         string
	AValues        int
	BValues        int
	AMean          float64
	BMean          float64
	Delta          float64
	CILow          float64
	CIHigh         float64
	QuantileDeltas map[string]float64
	AStats         MetricStats
	BStats         MetricStats
}

// ToolchainChange records a workload measured with different toolchains in
// the two records; its deltas mix code and toolchain effects.
type ToolchainChange struct {
	WorkloadID string
	A          Toolchain
	B          Toolchain
}

// Comparison is the analysis of two history records.
type Comparison struct {
	NameA            string
	NameB            string
	RecordA          HistoryRecord
	RecordB          HistoryRecord
	EnvMatch         bool
	ToolchainChanges []ToolchainChange
	Deltas           []MetricDelta
	OnlyInA          []string
	OnlyInB          []string
}

// CompareRecords computes per-workload, per-metric deltas B - A. Records from
// different host environments are flagged via EnvMatch and workloads whose
// toolchain changed via ToolchainChanges; they are never pooled.
func CompareRecords(nameA string, a HistoryRecord, nameB string, b HistoryRecord) Comparison {
	c := Comparison{
		NameA:    nameA,
		NameB:    nameB,
		RecordA:  a,
		RecordB:  b,
		EnvMatch: a.Environment.Equal(b.Environment),
	}
	bByID := map[string]WorkloadStats{}
	for _, w := range b.Workloads {
		bByID[w.ID] = w
	}
	aByID := map[string]WorkloadStats{}
	for _, wa := range a.Workloads {
		aByID[wa.ID] = wa
		wb, ok := bByID[wa.ID]
		if !ok {
			c.OnlyInA = append(c.OnlyInA, wa.ID)
			continue
		}
		if !wa.Toolchain.Equal(wb.Toolchain) {
			c.ToolchainChanges = append(c.ToolchainChanges, ToolchainChange{WorkloadID: wa.ID, A: wa.Toolchain, B: wb.Toolchain})
		}
		for metric, sa := range wa.Metrics {
			sb, ok := wb.Metrics[metric]
			if !ok {
				c.OnlyInA = append(c.OnlyInA, wa.ID+"/"+metric)
				continue
			}
			delta, lo, hi := MeanDeltaCI(sa, sb)
			d := MetricDelta{
				WorkloadID:     wa.ID,
				Kind:           wa.Kind,
				Metric:         metric,
				AValues:        sa.Values,
				BValues:        sb.Values,
				AMean:          sa.Mean,
				BMean:          sb.Mean,
				Delta:          delta,
				CILow:          lo,
				CIHigh:         hi,
				QuantileDeltas: map[string]float64{},
				AStats:         sa,
				BStats:         sb,
			}
			for key, qa := range sa.Quantiles {
				if qb, ok := sb.Quantiles[key]; ok {
					d.QuantileDeltas[key] = qb - qa
				}
			}
			c.Deltas = append(c.Deltas, d)
		}
	}
	for _, wb := range b.Workloads {
		wa, ok := aByID[wb.ID]
		if !ok {
			c.OnlyInB = append(c.OnlyInB, wb.ID)
			continue
		}
		for metric := range wb.Metrics {
			if _, ok := wa.Metrics[metric]; !ok {
				c.OnlyInB = append(c.OnlyInB, wb.ID+"/"+metric)
			}
		}
	}
	slices.SortFunc(c.Deltas, func(x, y MetricDelta) int {
		return strings.Compare(x.WorkloadID+x.Metric, y.WorkloadID+y.Metric)
	})
	slices.SortFunc(c.ToolchainChanges, func(x, y ToolchainChange) int {
		return strings.Compare(x.WorkloadID, y.WorkloadID)
	})
	slices.Sort(c.OnlyInA)
	slices.Sort(c.OnlyInB)
	return c
}

// WriteText renders a comparison as a plain-text table.
func (c Comparison) WriteText(w io.Writer) error {
	var sb strings.Builder
	fmt.Fprintf(&sb, "A: %s (seq %d, %s, %s)\n", c.NameA, c.RecordA.Sequence, c.RecordA.Tag, c.RecordA.Revision[:8])
	fmt.Fprintf(&sb, "B: %s (seq %d, %s, %s)\n", c.NameB, c.RecordB.Sequence, c.RecordB.Tag, c.RecordB.Revision[:8])
	if !c.EnvMatch {
		fmt.Fprintln(&sb, "WARNING: records come from different environments; deltas are not directly comparable.")
		fmt.Fprintf(&sb, "  A environment: %+v\n", c.RecordA.Environment)
		fmt.Fprintf(&sb, "  B environment: %+v\n", c.RecordB.Environment)
	}
	for _, tc := range c.ToolchainChanges {
		fmt.Fprintf(&sb, "WARNING: workload %s changed toolchain; its deltas include the toolchain effect.\n", tc.WorkloadID)
		fmt.Fprintf(&sb, "  A toolchain: %s\n  B toolchain: %s\n", tc.A, tc.B)
	}
	fmt.Fprintln(&sb)
	fmt.Fprintf(&sb, "%-24s %-14s %8s %8s %14s %14s %14s %26s\n",
		"workload", "metric", "n(A)", "n(B)", "mean(A)", "mean(B)", "delta", "95% CI")
	for _, d := range c.Deltas {
		fmt.Fprintf(&sb, "%-24s %-14s %8d %8d %14.1f %14.1f %+14.1f [%+.1f, %+.1f]\n",
			d.WorkloadID, d.Metric, d.AValues, d.BValues, d.AMean, d.BMean, d.Delta, d.CILow, d.CIHigh)
		for _, key := range []string{"p50", "p90", "p95", "p99"} {
			if qd, ok := d.QuantileDeltas[key]; ok {
				fmt.Fprintf(&sb, "%-24s %-14s %8s %8s %14s %14s %+14.1f\n",
					"", key, "", "", "", "", qd)
			}
		}
	}
	if len(c.OnlyInA) > 0 {
		fmt.Fprintf(&sb, "\nOnly in A: %s\n", strings.Join(c.OnlyInA, ", "))
	}
	if len(c.OnlyInB) > 0 {
		fmt.Fprintf(&sb, "\nOnly in B: %s\n", strings.Join(c.OnlyInB, ", "))
	}
	_, err := w.Write([]byte(sb.String()))
	return err
}
