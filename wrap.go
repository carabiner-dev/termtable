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
func Wrap(lines [][]GraphemeRun, width int, wrap, trim bool, maxHeight int, trimPos TrimPosition) []string {
	if width <= 0 {
		return nil
	}
	var olines []outputLine
	for _, runs := range lines {
		olines = append(olines, wrapNaturalLine(runs, width, wrap, trim, trimPos)...)
	}
	if maxHeight > 0 && len(olines) > maxHeight {
		kept := olines[:maxHeight]
		if trim {
			kept[maxHeight-1] = ellipsizeLine(kept[maxHeight-1], width, trimPos)
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
func wrapNaturalLine(runs []GraphemeRun, width int, wrap, trim bool, trimPos TrimPosition) []outputLine {
	if len(runs) == 0 {
		return []outputLine{{}}
	}
	cum := cumulativeEsc(runs)
	if !wrap {
		if lineDisplayWidth(runs) <= width {
			return []outputLine{{runs: runs, startEsc: cum[0]}}
		}
		// Content overflows its slot. Clip according to the trim
		// position, adding an ellipsis marker when trim is enabled.
		return []outputLine{clipOverflowingLine(runs, cum, width, trim, trimPos)}
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

// clipToTerminalWidth clips every overwide line in s to maxWidth,
// replacing the dropped tail with an ellipsis so the truncation is
// visible. Lines that already fit are returned verbatim; ANSI escape
// state is preserved for the kept prefix and closed with a reset.
// Widths are measured under mode so the result matches the table's
// emoji-width policy. Non-positive maxWidth is a no-op.
func clipToTerminalWidth(s string, maxWidth int, mode EmojiWidthMode) string {
	if maxWidth <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	changed := false
	for i, ln := range lines {
		if displayWidthFor(ln, mode) <= maxWidth {
			continue
		}
		runs := graphemeRunsOf(ln, mode)
		kept := clipEnd(runs, maxWidth, true)
		lines[i] = renderOutputLine(outputLine{runs: kept})
		changed = true
	}
	if !changed {
		return s
	}
	return strings.Join(lines, "\n")
}

// ellipsizeLine decorates the final kept line of a line-clamped
// cell. Two cases:
//
//  1. The line fits in the column with room to spare — append a
//     trailing "…" to signal that content below was dropped. This
//     marker always sits at the end because it indicates vertical
//     truncation, not horizontal.
//  2. The line itself overflows the column — clip horizontally per
//     trimPos so the user's chosen trim position still wins.
func ellipsizeLine(ol outputLine, width int, trimPos TrimPosition) outputLine {
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
	clipped := clipRuns(ol.runs, width, true, trimPos)
	return outputLine{runs: clipped, startEsc: ol.startEsc}
}

// clipOverflowingLine builds a single outputLine from runs clipped
// to width, honouring the trim policy and marker position. Called
// by wrapNaturalLine's single-line branch.
func clipOverflowingLine(runs []GraphemeRun, cum []string, width int, trim bool, trimPos TrimPosition) outputLine {
	clipped := clipRuns(runs, width, trim, trimPos)
	// The ANSI state that was active just before the first visible
	// run of the clipped output needs to be reconstructed. For
	// TrimEnd we keep the head, so cum[0] is correct. For TrimStart
	// we drop a prefix, so the state at the kept first index is the
	// right answer. TrimMiddle keeps head + tail; the head's state
	// (cum[0]) prevails.
	return outputLine{runs: clipped, startEsc: cum[0]}
}

// clipRuns returns runs truncated to fit width columns with an
// ellipsis (when trim=true) placed according to pos, or a hard clip
// (when trim=false) at the same position.
func clipRuns(runs []GraphemeRun, width int, trim bool, pos TrimPosition) []GraphemeRun {
	switch pos {
	case TrimStart:
		return clipStart(runs, width, trim)
	case TrimMiddle:
		return clipMiddle(runs, width, trim)
	case TrimEnd:
		return clipEnd(runs, width, trim)
	}
	return clipEnd(runs, width, trim)
}

// clipEnd keeps the prefix of runs that fits, optionally appending
// an ellipsis when trim is true.
func clipEnd(runs []GraphemeRun, width int, trim bool) []GraphemeRun {
	if !trim {
		return clipHeadToWidth(runs, width)
	}
	kept := clipHeadToWidth(runs, width-1)
	out := make([]GraphemeRun, len(kept), len(kept)+1)
	copy(out, kept)
	out = append(out, GraphemeRun{Text: ellipsis, Width: 1})
	return out
}

// clipStart keeps the suffix of runs that fits, optionally
// prepending an ellipsis when trim is true.
func clipStart(runs []GraphemeRun, width int, trim bool) []GraphemeRun {
	if !trim {
		return clipTailToWidth(runs, width)
	}
	kept := clipTailToWidth(runs, width-1)
	out := make([]GraphemeRun, 0, len(kept)+1)
	out = append(out, GraphemeRun{Text: ellipsis, Width: 1})
	out = append(out, kept...)
	return out
}

// clipMiddle keeps both ends of runs with an ellipsis (or hard cut)
// between them. With trim=true the total budget becomes
// leftWidth + 1 + rightWidth = width, leftWidth being floor((w-1)/2).
// With trim=false the budget splits evenly: floor(w/2) + ceil(w/2).
func clipMiddle(runs []GraphemeRun, width int, trim bool) []GraphemeRun {
	var leftWidth, rightWidth int
	if trim {
		leftWidth = (width - 1) / 2
		rightWidth = width - 1 - leftWidth
	} else {
		leftWidth = width / 2
		rightWidth = width - leftWidth
	}
	left := clipHeadToWidth(runs, leftWidth)
	right := clipTailToWidth(runs, rightWidth)
	// If the head and tail overlap in the source slice (possible
	// with very short content that still exceeds width due to wide
	// clusters), drop the overlap from the tail to preserve order
	// and avoid duplicate runs.
	if overlap := len(left) + len(right) - len(runs); overlap > 0 {
		if overlap >= len(right) {
			right = nil
		} else {
			right = right[overlap:]
		}
	}
	out := make([]GraphemeRun, 0, len(left)+len(right)+1)
	out = append(out, left...)
	if trim {
		out = append(out, GraphemeRun{Text: ellipsis, Width: 1})
	}
	out = append(out, right...)
	return out
}

// clipHeadToWidth returns the longest prefix of runs whose
// cumulative width does not exceed maxWidth.
func clipHeadToWidth(runs []GraphemeRun, maxWidth int) []GraphemeRun {
	if maxWidth <= 0 {
		return nil
	}
	var w int
	for i, r := range runs {
		if w+r.Width > maxWidth {
			return runs[:i]
		}
		w += r.Width
	}
	return runs
}

// clipTailToWidth returns the longest suffix of runs whose
// cumulative width does not exceed maxWidth.
func clipTailToWidth(runs []GraphemeRun, maxWidth int) []GraphemeRun {
	if maxWidth <= 0 {
		return nil
	}
	var w int
	for i := len(runs) - 1; i >= 0; i-- {
		if w+runs[i].Width > maxWidth {
			return runs[i+1:]
		}
		w += runs[i].Width
	}
	return runs
}
