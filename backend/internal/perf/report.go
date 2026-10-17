package perf

import (
	"fmt"
	"html"
	"io"
	"math"
	"strings"
)

// WriteHTML renders a comparison as a self-contained HTML report with inline
// SVG histograms. Reports are generated on demand and never committed.
func (c Comparison) WriteHTML(w io.Writer) error {
	var sb strings.Builder
	c.writeHTML(&sb)
	_, err := w.Write([]byte(sb.String()))
	return err
}

// writeHTML renders the report markup into w.
func (c Comparison) writeHTML(w io.Writer) {
	esc := html.EscapeString
	fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8">
<title>Performance comparison</title>
<style>
body{font-family:system-ui,sans-serif;margin:2rem;color:#1a1a1a}
table{border-collapse:collapse;margin:1rem 0}
th,td{border:1px solid #ccc;padding:.3rem .6rem;text-align:right}
th:first-child,td:first-child,th:nth-child(2),td:nth-child(2){text-align:left}
.warn{background:#fff3cd;border:1px solid #e0a800;padding:.6rem 1rem;margin:1rem 0}
h2{margin-top:2rem}
code{background:#f4f4f4;padding:.1rem .3rem}
</style></head><body>
`)
	fmt.Fprint(w, "<h1>Performance comparison</h1>\n")
	fmt.Fprintf(w, "<p><strong>A:</strong> <code>%s</code> (seq %d, %s, <code>%s</code>)<br>\n",
		esc(c.NameA), c.RecordA.Sequence, esc(c.RecordA.Tag), c.RecordA.Revision[:8])
	fmt.Fprintf(w, "<strong>B:</strong> <code>%s</code> (seq %d, %s, <code>%s</code>)</p>\n",
		esc(c.NameB), c.RecordB.Sequence, esc(c.RecordB.Tag), c.RecordB.Revision[:8])
	if !c.EnvMatch {
		fmt.Fprint(w, "<p class=\"warn\">Records come from different environments; deltas are not directly comparable.</p>\n")
	}
	fmt.Fprint(w, "<table><tr><th>environment</th><th>A</th><th>B</th></tr>\n")
	envRows := [][3]string{
		{"machine", c.RecordA.Environment.Machine, c.RecordB.Environment.Machine},
		{"os", c.RecordA.Environment.OS, c.RecordB.Environment.OS},
		{"arch", c.RecordA.Environment.Arch, c.RecordB.Environment.Arch},
		{"cpu", c.RecordA.Environment.CPU, c.RecordB.Environment.CPU},
	}
	for _, r := range envRows {
		fmt.Fprintf(w, "<tr><td>%s</td><td>%s</td><td>%s</td></tr>\n", r[0], esc(r[1]), esc(r[2]))
	}
	fmt.Fprint(w, "</table>\n")
	for _, tc := range c.ToolchainChanges {
		fmt.Fprintf(w, "<p class=\"warn\">Workload <code>%s</code> changed toolchain (A: <code>%s</code>, B: <code>%s</code>); its deltas include the toolchain effect.</p>\n",
			esc(tc.WorkloadID), esc(tc.A.String()), esc(tc.B.String()))
	}

	fmt.Fprint(w, "<table><tr><th>workload</th><th>metric</th><th>n(A)</th><th>n(B)</th>"+
		"<th>mean(A)</th><th>mean(B)</th><th>delta</th><th>95% CI</th><th>p95 delta</th></tr>\n")
	for _, d := range c.Deltas {
		p95 := "—"
		if v, ok := d.QuantileDeltas["p95"]; ok {
			p95 = fmt.Sprintf("%+.1f", v)
		}
		fmt.Fprintf(w, "<tr><td>%s</td><td>%s</td><td>%d</td><td>%d</td><td>%.1f</td><td>%.1f</td><td>%+.1f</td><td>[%+.1f, %+.1f]</td><td>%s</td></tr>\n",
			esc(d.WorkloadID), esc(d.Metric), d.AValues, d.BValues, d.AMean, d.BMean, d.Delta, d.CILow, d.CIHigh, p95)
	}
	fmt.Fprint(w, "</table>\n")

	for _, d := range c.Deltas {
		fmt.Fprintf(w, "<h2>%s — %s</h2>\n", esc(d.WorkloadID), esc(d.Metric))
		c.writeHistogramSVG(w, d)
	}
	if len(c.OnlyInA) > 0 {
		fmt.Fprintf(w, "<p>Only in A: %s</p>\n", esc(strings.Join(c.OnlyInA, ", ")))
	}
	if len(c.OnlyInB) > 0 {
		fmt.Fprintf(w, "<p>Only in B: %s</p>\n", esc(strings.Join(c.OnlyInB, ", ")))
	}
	fmt.Fprint(w, "</body></html>\n")
}

// writeHistogramSVG draws A and B histograms overlaid on a shared x-axis,
// with vertical markers at each record's mean.
func (c Comparison) writeHistogramSVG(w io.Writer, d MetricDelta) {
	const width, height, pad = 640.0, 160.0, 4.0
	ha, hb := d.AStats.Histogram, d.BStats.Histogram
	x0 := math.Min(ha.Min, hb.Min)
	x1 := math.Max(ha.Min+ha.BucketWidth*float64(len(ha.Counts)),
		hb.Min+hb.BucketWidth*float64(len(hb.Counts)))
	if x1 <= x0 {
		x1 = x0 + 1
	}
	maxCount := 1
	for _, cts := range [][]int{ha.Counts, hb.Counts} {
		for _, n := range cts {
			maxCount = max(maxCount, n)
		}
	}
	scaleX := func(v float64) float64 { return (v - x0) / (x1 - x0) * width }
	scaleY := func(n int) float64 { return float64(n) / float64(maxCount) * (height - pad) }

	fmt.Fprintf(w, "<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"%.0f\" height=\"%.0f\" viewBox=\"0 0 %.0f %.0f\">\n",
		width, height+20, width, height+20)
	drawHist := func(h Histogram, color string) {
		for i, n := range h.Counts {
			if n == 0 {
				continue
			}
			x := scaleX(h.Min + float64(i)*h.BucketWidth)
			bw := math.Max(scaleX(h.Min+float64(i+1)*h.BucketWidth)-x-1, 1)
			y := height - scaleY(n)
			fmt.Fprintf(w, "<rect x=\"%.1f\" y=\"%.1f\" width=\"%.1f\" height=\"%.1f\" fill=\"%s\" fill-opacity=\"0.5\"/>\n",
				x, y, bw, scaleY(n), color)
		}
	}
	drawHist(ha, "#1f77b4")
	drawHist(hb, "#ff7f0e")
	for _, m := range []struct {
		mean  float64
		color string
	}{{d.AStats.Mean, "#1f77b4"}, {d.BStats.Mean, "#ff7f0e"}} {
		x := scaleX(m.mean)
		fmt.Fprintf(w, "<line x1=\"%.1f\" y1=\"0\" x2=\"%.1f\" y2=\"%.0f\" stroke=\"%s\" stroke-width=\"1.5\"/>\n",
			x, x, height, m.color)
	}
	fmt.Fprintf(w, "<text x=\"0\" y=\"%.0f\" font-size=\"10\">%.0f</text>\n", height+12, x0)
	fmt.Fprintf(w, "<text x=\"%.0f\" y=\"%.0f\" font-size=\"10\" text-anchor=\"end\">%.0f</text>\n", width, height+12, x1)
	fmt.Fprint(w, "</svg>\n")
	fmt.Fprint(w, "<p><span style=\"color:#1f77b4\">&#9632;</span> A &nbsp; "+
		"<span style=\"color:#ff7f0e\">&#9632;</span> B; vertical lines mark each record's mean.</p>\n")
}
