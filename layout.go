// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import "fmt"

// seamWidth is the display cost of the boundary between two adjacent
// columns — one border glyph plus two padding columns for the default
// padding of 1 column on each side. Phase 4+ will compute this from
// the actual cell padding; Phase 2 / 3 assume the default.
const seamWidth = 1 + 2 // border + right-pad of left col + left-pad of right col

// layoutResult is the output of Pass 2. colAssigned gives the content
// width per column (not including padding or borders). rowHeights gives
// the number of rendered lines per absolute row (headers first, then
// body, then footers). wrapped is the per-cell wrapped content used by
// the renderer; each slice entry is one rendered line including ANSI
// state. warnings aggregates non-fatal events from solving. err is
// ErrTargetTooNarrow when the minimum widths cannot fit.
type layoutResult struct {
	colAssigned []int
	rowHeights  []int
	wrapped     map[*Cell][]string
	warnings    []Warning
	err         error
}

// Layout performs Pass 2 of rendering: it takes per-column widths from
// Measure and solves a width assignment that fits within the table's
// target width, then wraps every cell's content to those widths and
// computes per-row heights.
//
// Empty tables (zero columns) return an empty result with no error.
func Layout(t *Table, m *measureResult) *layoutResult {
	nCols := t.NumColumns()
	out := &layoutResult{
		colAssigned: make([]int, nCols),
		wrapped:     make(map[*Cell][]string),
	}
	if nCols == 0 {
		return out
	}

	target := t.ResolvedTargetWidth()
	fixedOverhead := (nCols + 1) + nCols*2 // borders + default L+R padding per column
	available := target - fixedOverhead
	if available < 0 {
		available = 0
	}

	// Feasibility: the sum of column minimums must fit.
	var minSum int
	for _, v := range m.colMin {
		minSum += v
	}
	if minSum > available {
		out.err = fmt.Errorf(
			"target width %d leaves %d for content but content minimum sums to %d: %w",
			target, available, minSum, ErrTargetTooNarrow,
		)
		// Fall through and still produce a best-effort layout at the
		// minimum widths so callers can inspect / warn on partial
		// output if they wish.
		copy(out.colAssigned, m.colMin)
		out.rowHeights = computeRowHeights(t, out.colAssigned, out.wrapped)
		return out
	}

	// Equal-split baseline with leftmost-first distribution of the
	// remainder so the result is deterministic.
	base := available / nCols
	extra := available - base*nCols
	for i := range nCols {
		out.colAssigned[i] = base
		if i < extra {
			out.colAssigned[i]++
		}
	}

	// Min-floor pass: raise any column below its minimum and remember
	// the debt owed to the global budget.
	var debt int
	for i := range nCols {
		if out.colAssigned[i] < m.colMin[i] {
			debt += m.colMin[i] - out.colAssigned[i]
			out.colAssigned[i] = m.colMin[i]
		}
	}
	// Pay the debt by shrinking the widest slack column (furthest
	// above its minimum). Ties break leftward.
	for debt > 0 {
		idx := widestSlackColumn(out.colAssigned, m.colMin)
		if idx < 0 {
			break
		}
		slack := out.colAssigned[idx] - m.colMin[idx]
		take := slack
		if take > debt {
			take = debt
		}
		out.colAssigned[idx] -= take
		debt -= take
	}

	// Multi-span constraint pass: for each multi-span cell, ensure
	// the sum of its columns' assigned widths plus seams covers its
	// minimum. Borrow from outside-span slack columns if needed.
	for _, cons := range m.multiSpans {
		satisfied := applyMultiSpanConstraint(out.colAssigned, m.colMin, cons)
		if !satisfied {
			out.warnings = append(out.warnings, SpanOverflowEvent{
				CellID:   cons.cellID,
				Required: cons.minWidth,
				Got:      contentSum(out.colAssigned, cons.colStart, cons.colSpan),
			})
		}
	}

	// Desired-width upgrade pass: distribute any unused budget to the
	// columns farthest below their desired width.
	leftover := available - sumInts(out.colAssigned)
	for leftover > 0 {
		idx, deficit := largestDeficit(out.colAssigned, m.colDesired)
		if idx < 0 {
			break
		}
		add := deficit
		if add > leftover {
			add = leftover
		}
		out.colAssigned[idx] += add
		leftover -= add
	}

	out.rowHeights = computeRowHeights(t, out.colAssigned, out.wrapped)
	return out
}

// widestSlackColumn returns the index of the column with the greatest
// (assigned - min) difference. Returns -1 when no slack exists. Ties
// break leftward.
func widestSlackColumn(assigned, minima []int) int {
	best := -1
	var bestSlack int
	for i := range assigned {
		slack := assigned[i] - minima[i]
		if slack > bestSlack {
			bestSlack = slack
			best = i
		}
	}
	return best
}

// applyMultiSpanConstraint borrows width from outside-span columns
// (tallest slack first) and gives it to the narrowest column inside
// the span until the constraint is satisfied or no outside slack
// remains. Returns true when the constraint was fully satisfied.
func applyMultiSpanConstraint(assigned, minima []int, cons multiSpanConstraint) bool {
	required := cons.minWidth - (cons.colSpan-1)*seamWidth
	if required <= 0 {
		return true
	}
	for {
		have := contentSum(assigned, cons.colStart, cons.colSpan)
		if have >= required {
			return true
		}
		need := required - have
		donorIdx := outsideWidestSlack(assigned, minima, cons.colStart, cons.colSpan)
		if donorIdx < 0 {
			return false
		}
		donorSlack := assigned[donorIdx] - minima[donorIdx]
		transfer := donorSlack
		if transfer > need {
			transfer = need
		}
		receiverIdx := insideNarrowest(assigned, cons.colStart, cons.colSpan)
		assigned[donorIdx] -= transfer
		assigned[receiverIdx] += transfer
	}
}

// outsideWidestSlack returns the index of the column outside
// [colStart, colStart+colSpan) with the greatest slack relative to its
// minimum. Returns -1 when no outside slack exists.
func outsideWidestSlack(assigned, minima []int, colStart, colSpan int) int {
	best := -1
	var bestSlack int
	for i := range assigned {
		if i >= colStart && i < colStart+colSpan {
			continue
		}
		slack := assigned[i] - minima[i]
		if slack > bestSlack {
			bestSlack = slack
			best = i
		}
	}
	return best
}

// insideNarrowest returns the index of the narrowest column inside
// [colStart, colStart+colSpan), leftmost on ties.
func insideNarrowest(assigned []int, colStart, colSpan int) int {
	best := colStart
	for i := colStart + 1; i < colStart+colSpan; i++ {
		if assigned[i] < assigned[best] {
			best = i
		}
	}
	return best
}

// largestDeficit returns the column index with the greatest
// (desired - assigned) gap and the gap's size. Returns (-1, 0) when
// all columns already meet or exceed desired.
func largestDeficit(assigned, desired []int) (colIdx, gap int) {
	best := -1
	var bestGap int
	for i := range assigned {
		gap := desired[i] - assigned[i]
		if gap > bestGap {
			bestGap = gap
			best = i
		}
	}
	return best, bestGap
}

// contentSum returns the sum of assigned[colStart:colStart+colSpan].
func contentSum(assigned []int, colStart, colSpan int) int {
	var sum int
	for i := colStart; i < colStart+colSpan; i++ {
		sum += assigned[i]
	}
	return sum
}

func sumInts(xs []int) int {
	var s int
	for _, v := range xs {
		s += v
	}
	return s
}

// computeRowHeights wraps every cell to its column's assigned width
// (adjusted by seams for colspans) and produces a heights slice in
// absolute row order. The second pass bumps the tail row of each
// rowspan cell if its wrapped height exceeds the natural sum of
// per-row heights across the span.
func computeRowHeights(t *Table, assigned []int, wrapped map[*Cell][]string) []int {
	totalRows := len(t.headers) + len(t.rows) + len(t.footers)
	heights := make([]int, totalRows)

	wrapCell := func(c *Cell) []string {
		w := contentSum(assigned, c.gridCol, c.colSpan) + (c.colSpan-1)*seamWidth
		if w <= 0 {
			return nil
		}
		text, err := resolveCellContent(c)
		if err != nil {
			// The error was already recorded during Measure's walk;
			// here we just fall back to whatever bytes were buffered.
			text = c.content
		}
		return Wrap(NaturalLines(text), w, c.opts.wrap, c.opts.trim, c.opts.maxLines)
	}

	headerOffset := 0
	bodyOffset := len(t.headers)
	footerOffset := bodyOffset + len(t.rows)

	processSection := func(rows []*rowBody, offset int) {
		for sectionRow, row := range rows {
			absRow := offset + sectionRow
			for _, c := range row.cells {
				lines := wrapCell(c)
				wrapped[c] = lines
				if c.rowSpan == 1 && len(lines) > heights[absRow] {
					heights[absRow] = len(lines)
				}
			}
		}
	}

	hBodies := make([]*rowBody, len(t.headers))
	for i, h := range t.headers {
		hBodies[i] = &h.rowBody
	}
	bBodies := make([]*rowBody, len(t.rows))
	for i, r := range t.rows {
		bBodies[i] = &r.rowBody
	}
	fBodies := make([]*rowBody, len(t.footers))
	for i, f := range t.footers {
		fBodies[i] = &f.rowBody
	}
	processSection(hBodies, headerOffset)
	processSection(bBodies, bodyOffset)
	processSection(fBodies, footerOffset)

	// Rowspan bump pass.
	for cell, lines := range wrapped {
		if cell.rowSpan == 1 {
			continue
		}
		absStart := absRowOf(t, cell)
		var have int
		for i := range cell.rowSpan {
			have += heights[absStart+i]
		}
		need := len(lines)
		if need > have {
			heights[absStart+cell.rowSpan-1] += need - have
		}
	}

	return heights
}

// absRowOf returns the absolute row index of cell c across the table
// (headers first, then body, then footers).
func absRowOf(t *Table, c *Cell) int {
	switch c.section {
	case sectionHeader:
		return c.sectionRow
	case sectionBody:
		return len(t.headers) + c.sectionRow
	case sectionFooter:
		return len(t.headers) + len(t.rows) + c.sectionRow
	}
	return c.sectionRow
}
