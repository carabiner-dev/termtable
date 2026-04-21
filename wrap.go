// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import "strings"

// ellipsis is appended to the final line when a wrapped or clipped
// result is truncated due to a width or height cap.
const ellipsis = "…"

// outputLine is a single wrapped line prior to rendering: the cluster
// sequence that forms its content plus the ANSI state that needs to be
// re-emitted at its start.
type outputLine struct {
	runs     []GraphemeRun
	startEsc string
}

// span is a half-open [start, end) range into a slice of runs.
type span struct{ start, end int }

// Wrap transforms a slice of natural lines (e.g. as returned by
// NaturalLines) into a slice of rendered lines each at most width
// terminal columns wide. When wrap is true, content breaks at
// whitespace where possible and hard-breaks on runs longer than width.
// When wrap is false, each natural line yields one output line,
// optionally truncated with "…" if trim is true and the line exceeds
// width.
//
// ANSI escape state is preserved across line boundaries: the
// cumulative escape bytes that were active at the start of each output
// line are re-emitted at its head, and every line that carries any
// escape bytes is terminated with a reset ("\x1b[0m"). This is
// deliberately redundant — a reduced-fidelity policy that keeps colored
// content readable without full SGR state tracking.
//
// maxHeight > 0 caps the total number of output lines. Exceeding
// content is dropped; when trim is also true the last kept line is
// truncated to end in "…". maxHeight == 0 disables the cap.
func Wrap(lines [][]GraphemeRun, width int, wrap, trim bool, maxHeight int) []string {
	if width <= 0 {
		return nil
	}
	var olines []outputLine
	for _, runs := range lines {
		olines = append(olines, wrapNaturalLine(runs, width, wrap, trim)...)
	}
	if maxHeight > 0 && len(olines) > maxHeight {
		kept := olines[:maxHeight]
		if trim {
			kept[maxHeight-1] = ellipsizeLine(kept[maxHeight-1], width)
		}
		olines = kept
	}
	out := make([]string, len(olines))
	for i, ol := range olines {
		out[i] = renderOutputLine(ol)
	}
	return out
}

// wrapNaturalLine wraps a single natural-line's runs into output lines.
// A natural line with zero runs produces one empty output line so the
// hard break is preserved in the output.
func wrapNaturalLine(runs []GraphemeRun, width int, wrap, trim bool) []outputLine {
	if len(runs) == 0 {
		return []outputLine{{}}
	}
	cum := cumulativeEsc(runs)
	if !wrap {
		sp := span{0, len(runs)}
		if trim && lineDisplayWidth(runs) > width {
			sp = clipSpanToWidth(runs, sp, width-1)
			clipped := append([]GraphemeRun{}, runs[sp.start:sp.end]...)
			clipped = append(clipped, GraphemeRun{Text: ellipsis, Width: 1})
			return []outputLine{{runs: clipped, startEsc: cum[sp.start]}}
		}
		return []outputLine{{runs: runs[sp.start:sp.end], startEsc: cum[sp.start]}}
	}
	widths := make([]int, len(runs))
	isWS := make([]bool, len(runs))
	for i, r := range runs {
		widths[i] = r.Width
		isWS[i] = isWhitespaceCluster(r.Text)
	}
	spans := wrapBreaks(widths, isWS, width)
	if len(spans) == 0 {
		// Input was non-empty but entirely whitespace. Preserve the
		// hard break by emitting one empty line.
		return []outputLine{{startEsc: cum[0]}}
	}
	out := make([]outputLine, 0, len(spans))
	for _, sp := range spans {
		out = append(out, outputLine{
			runs:     runs[sp.start:sp.end],
			startEsc: cum[sp.start],
		})
	}
	return out
}

// wrapBreaks decides where to split a sequence of grapheme widths into
// lines of at most width columns. It returns half-open spans into the
// input; whitespace runs at line boundaries are dropped.
func wrapBreaks(widths []int, isWS []bool, width int) []span {
	var spans []span
	n := len(widths)
	i := 0
	for i < n {
		for i < n && isWS[i] {
			i++
		}
		if i >= n {
			break
		}
		start := i
		curW := 0
		lastWS := -1
		for i < n {
			w := widths[i]
			if curW+w > width {
				break
			}
			curW += w
			if isWS[i] {
				lastWS = i
			}
			i++
		}
		switch {
		case i < n && lastWS > start:
			// Word break: end the line just before the whitespace and
			// resume after it.
			spans = append(spans, span{start, lastWS})
			i = lastWS + 1
		case i == start:
			// A single grapheme wider than the column. Emit it alone
			// (overflowing) so progress is guaranteed.
			spans = append(spans, span{start, start + 1})
			i = start + 1
		default:
			spans = append(spans, span{start, i})
		}
	}
	return spans
}

// cumulativeEsc returns, for each run index i (and one final entry at
// len(runs)), the escape bytes accumulated strictly before run i. The
// rendering loop re-emits cum[first-run-of-line] to restore state at
// each line boundary.
func cumulativeEsc(runs []GraphemeRun) []string {
	out := make([]string, len(runs)+1)
	var b strings.Builder
	for i, r := range runs {
		out[i] = b.String()
		b.WriteString(r.EscPrefix)
	}
	out[len(runs)] = b.String()
	return out
}

// renderOutputLine materializes a wrapped line as a printable string.
// The accumulated startEsc is emitted first so any active color or
// attribute from prior lines is restored; per-cluster EscPrefix values
// are then emitted inline; finally a "\x1b[0m" reset is appended when
// any escape bytes were emitted.
func renderOutputLine(ol outputLine) string {
	var b strings.Builder
	hadEsc := false
	if ol.startEsc != "" {
		b.WriteString(ol.startEsc)
		hadEsc = true
	}
	for _, r := range ol.runs {
		if r.EscPrefix != "" {
			b.WriteString(r.EscPrefix)
			hadEsc = true
		}
		b.WriteString(r.Text)
	}
	if hadEsc {
		b.WriteString("\x1b[0m")
	}
	return b.String()
}

func lineDisplayWidth(runs []GraphemeRun) int {
	var w int
	for _, r := range runs {
		w += r.Width
	}
	return w
}

// clipSpanToWidth truncates sp to at most targetWidth display columns.
// Clusters are kept whole; if adding the next cluster would exceed the
// target, it is dropped.
func clipSpanToWidth(runs []GraphemeRun, sp span, targetWidth int) span {
	if targetWidth <= 0 {
		return span{sp.start, sp.start}
	}
	w := 0
	for i := sp.start; i < sp.end; i++ {
		if w+runs[i].Width > targetWidth {
			return span{sp.start, i}
		}
		w += runs[i].Width
	}
	return sp
}

// ellipsizeLine produces a copy of ol truncated (from the right) so
// its rendered width is at most width columns, with "…" as the final
// cluster. When the line already fits with room to spare, "…" is
// simply appended.
func ellipsizeLine(ol outputLine, width int) outputLine {
	if width <= 0 {
		return outputLine{startEsc: ol.startEsc}
	}
	total := lineDisplayWidth(ol.runs)
	if total+1 <= width {
		newRuns := make([]GraphemeRun, len(ol.runs), len(ol.runs)+1)
		copy(newRuns, ol.runs)
		newRuns = append(newRuns, GraphemeRun{Text: ellipsis, Width: 1})
		return outputLine{runs: newRuns, startEsc: ol.startEsc}
	}
	sp := clipSpanToWidth(ol.runs, span{0, len(ol.runs)}, width-1)
	newRuns := make([]GraphemeRun, 0, sp.end-sp.start+1)
	newRuns = append(newRuns, ol.runs[sp.start:sp.end]...)
	newRuns = append(newRuns, GraphemeRun{Text: ellipsis, Width: 1})
	return outputLine{runs: newRuns, startEsc: ol.startEsc}
}
