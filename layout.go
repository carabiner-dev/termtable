// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import "fmt"

// layoutGeometry snapshots the display-width contributions of the
// table's horizontal padding and borders for use by the solver and
// renderer. seam is the inter-column cost: one border glyph plus the
// right padding of the left column and the left padding of the right
// column. perColumnPad is the combined horizontal padding added
// around each column's content area.
type layoutGeometry struct {
	perColumnPad int
	seam         int
}

func tableGeometry(t *Table) layoutGeometry {
	p := t.opts.padding
	return layoutGeometry{
		perColumnPad: p.Left + p.Right,
		seam:         1 + p.Left + p.Right,
	}
}

// effectivelyUnbounded stands in for "no user-specified cap" in the
// column-width solver. Large enough that the math never pretends to
// hit it for real tables.
const effectivelyUnbounded = 1 << 30

// layoutResult is the output of Pass 2. colAssigned gives the content
// width per column (not including padding or borders). rowHeights gives
// the number of rendered lines per absolute row (headers first, then
// body, then footers). wrapped is the per-cell wrapped content used by
// the renderer; each slice entry is one rendered line including ANSI
// state. warnings aggregates non-fatal events from solving. err is
// ErrTargetTooNarrow only in the pathological case where the target
// has no room for even one glyph per column.
type layoutResult struct {
	colAssigned []int
	rowHeights  []int
	wrapped     map[*Cell][]string
	warnings    []Warning
	err         error
}

// Layout performs Pass 2 of rendering: it takes per-column widths from
// Measure, combines them with user-supplied column configuration
// (width / min / max / weight), solves a width assignment that fits
// within the table's target width, then wraps every cell's content to
// those widths and computes per-row heights.
//
// The solver algorithm:
//
//  1. Effective bounds: effMin = max(contentMin, userMin); effMax =
//     userMax (or unbounded). Columns with SetWidth are pinned so
//     effMin = effMax = userWidth (clamped up to contentMin on conflict,
//     producing a best-effort overflow rather than silently cropping
//     content).
//  2. Feasibility: when sum(effMin) exceeds the budget, shrink every
//     column proportionally so each still gets at least one glyph of
//     width; cells with unbreakable content wider than their slot clip
//     it with an ellipsis via the wrap pass. ErrTargetTooNarrow only
//     fires when the budget cannot even give every column a single
//     glyph (available < nCols).
//  3. Initialize colAssigned to effMin. Water-fill the remaining
//     budget by column weights (default 1.0), capped at effMax per
//     column. Rounding leftovers distribute one-per-column left-first
//     until consumed.
//  4. Multi-span constraint pass: for each multi-span cell, borrow
//     width from outside-span slack until the cell's min fits across
//     its columns. Donors must remain >= effMin; receivers must
//     remain <= effMax.
//  5. Wrap every cell to its content width and compute row heights,
//     bumping the tail row of rowspan cells when their wrapped output
//     exceeds the natural sum of covered rows.
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
	geom := tableGeometry(t)
	fixedOverhead := (nCols + 1) + nCols*geom.perColumnPad
	available := target - fixedOverhead
	if available < 0 {
		available = 0
	}

	effMin, effMax, weights := effectiveBounds(t, m, nCols)

	var minSum int
	for _, v := range effMin {
		minSum += v
	}
	if available < nCols {
		// Genuinely pathological: the target width doesn't have room
		// for one glyph per column after paying for borders and
		// padding. Distribute what we have left-first and surface a
		// hard error — the caller needs to widen the target or drop
		// a column.
		out.err = fmt.Errorf(
			"target width %d leaves %d for content but table has %d columns: %w",
			target, available, nCols, ErrTargetTooNarrow,
		)
		distributeShrunkBudget(out.colAssigned, effMin, available)
		out.rowHeights = computeRowHeights(t, out.colAssigned, out.wrapped)
		return out
	}
	if minSum > available {
		// Content wants more room than the target gives. Shrink each
		// column below its natural minimum so the frame still fits;
		// cells that end up narrower than their unbreakable content
		// will clip it with an ellipsis via the normal wrap path.
		// This is not an error — output is well-formed at the target
		// width. Callers that care can still observe the compression
		// through the resulting per-column widths.
		distributeShrunkBudget(out.colAssigned, effMin, available)
		out.rowHeights = computeRowHeights(t, out.colAssigned, out.wrapped)
		return out
	}

	copy(out.colAssigned, effMin)
	distributeByWeights(out.colAssigned, effMax, weights, available)

	for _, cons := range m.multiSpans {
		if !applyMultiSpanConstraint(out.colAssigned, effMin, effMax, cons, geom.seam) {
			out.warnings = append(out.warnings, SpanOverflowEvent{
				CellID:   cons.cellID,
				Required: cons.minWidth,
				Got:      contentSum(out.colAssigned, cons.colStart, cons.colSpan),
			})
		}
	}

	for _, re := range m.readerErrs {
		out.warnings = append(out.warnings, ReaderErrorEvent{
			CellID: re.cellID,
			Err:    re.err,
		})
	}

	detectCrossSectionSpans(t, &out.warnings)

	out.rowHeights = computeRowHeights(t, out.colAssigned, out.wrapped)
	return out
}

// detectCrossSectionSpans walks every multi-row cell and records a
// CrossSectionSpanEvent for any that extend past the last row of
// their section. The effective rowspan used by the row-height solver
// and the renderer is clamped by effectiveRowSpan; the authored
// rowSpan on the cell is preserved as-is.
func detectCrossSectionSpans(t *Table, ws *[]Warning) {
	visit := func(c *Cell) {
		if c.rowSpan <= 1 {
			return
		}
		eff := effectiveRowSpan(t, c)
		if eff == c.rowSpan {
			return
		}
		*ws = append(*ws, CrossSectionSpanEvent{
			CellID:        c.id,
			DeclaredSpan:  c.rowSpan,
			EffectiveSpan: eff,
			Section:       c.section.String(),
		})
	}
	for _, h := range t.headers {
		for _, c := range h.cells {
			visit(c)
		}
	}
	for _, r := range t.rows {
		for _, c := range r.cells {
			visit(c)
		}
	}
	for _, f := range t.footers {
		for _, c := range f.cells {
			visit(c)
		}
	}
}

// effectiveRowSpan returns the cell's rowSpan clamped so it does not
// extend past the last row of its section. Returns at least 1.
func effectiveRowSpan(t *Table, c *Cell) int {
	var sectionRows int
	switch c.section {
	case sectionHeader:
		sectionRows = len(t.headers)
	case sectionFooter:
		sectionRows = len(t.footers)
	case sectionBody:
		sectionRows = len(t.rows)
	}
	remaining := sectionRows - c.sectionRow
	if remaining < 1 {
		remaining = 1
	}
	if c.rowSpan > remaining {
		return remaining
	}
	return c.rowSpan
}

// effectiveBounds combines content measurements with the per-column
// user configuration into the three parallel slices the solver needs:
// effMin[i] is the lower bound, effMax[i] the upper bound, and
// weights[i] the distribution weight (defaulting to 1.0).
func effectiveBounds(t *Table, m *measureResult, nCols int) (effMin, effMax []int, weights []float64) {
	effMin = make([]int, nCols)
	effMax = make([]int, nCols)
	weights = make([]float64, nCols)
	for i := range nCols {
		col := t.Column(i)
		contentMin := m.colMin[i]

		minV := contentMin
		if col.set&cMin != 0 && col.minW > minV {
			minV = col.minW
		}

		var maxV int
		switch {
		case col.set&cWidth != 0:
			pin := col.width
			if pin < minV {
				// Content minimum wins; the pinned width would force
				// truncation, which we prefer to avoid. The column
				// will overflow its requested pin.
				maxV = minV
			} else {
				minV = pin
				maxV = pin
			}
		case col.set&cMax != 0:
			maxV = col.maxW
			if maxV < minV {
				maxV = minV
			}
		default:
			maxV = effectivelyUnbounded
		}

		effMin[i] = minV
		effMax[i] = maxV

		if col.set&cWeight != 0 {
			weights[i] = col.weight
		} else {
			weights[i] = 1.0
		}
	}
	return effMin, effMax, weights
}

// distributeByWeights grows assigned towards effMax until the
// available budget is fully consumed (or all columns are capped).
// Each round allocates floor(weight/totalWeight * remaining) to every
// flex column; leftover from rounding is distributed one-per-column
// left-first so the output is deterministic.
// distributeShrunkBudget allocates a tight `available` budget into
// `dst` when per-column minimums cannot be honoured. `proportions`
// (typically effMin) guides the split: columns that naturally want
// more space get a proportionally larger slice.
//
// Invariants:
//   - When `available >= len(dst)`, every column gets at least 1 and the
//     allocations sum to exactly `available`.
//   - When `available < len(dst)`, the first `available` columns each get
//     1 and the rest get 0 — a pathological case where even one glyph
//     per column does not fit.
//   - When `available <= 0`, every column is set to 0.
func distributeShrunkBudget(dst, proportions []int, available int) {
	n := len(dst)
	if n == 0 {
		return
	}
	if available <= 0 {
		for i := range dst {
			dst[i] = 0
		}
		return
	}
	if available < n {
		for i := range dst {
			if i < available {
				dst[i] = 1
			} else {
				dst[i] = 0
			}
		}
		return
	}
	// Phase 1: seed every column with 1.
	for i := range dst {
		dst[i] = 1
	}
	remaining := available - n

	// Phase 2: proportional share of what's left.
	var propSum int
	for _, p := range proportions {
		if p > 0 {
			propSum += p
		}
	}
	if propSum > 0 {
		for i, p := range proportions {
			if p <= 0 {
				continue
			}
			dst[i] += p * remaining / propSum
		}
	}

	// Phase 3: reconcile rounding. Push any leftover out to the left,
	// or reclaim from the left when proportional math overshot.
	assigned := sumInts(dst)
	diff := available - assigned
	for diff > 0 {
		for i := 0; i < n && diff > 0; i++ {
			dst[i]++
			diff--
		}
	}
	for diff < 0 {
		progress := false
		for i := 0; i < n && diff < 0; i++ {
			if dst[i] > 1 {
				dst[i]--
				diff++
				progress = true
			}
		}
		if !progress {
			return
		}
	}
}

func distributeByWeights(assigned, effMax []int, weights []float64, available int) {
	remaining := available - sumInts(assigned)
	for remaining > 0 {
		flex, totalW := flexColumns(assigned, effMax, weights)
		if len(flex) == 0 || totalW == 0 {
			return
		}
		gave := 0
		for _, f := range flex {
			share := int(float64(remaining) * f.weight / totalW)
			room := effMax[f.idx] - assigned[f.idx]
			if share > room {
				share = room
			}
			if share > 0 {
				assigned[f.idx] += share
				gave += share
			}
		}
		if gave == 0 {
			// Rounding gave every column zero. Distribute the
			// remainder one-per-column, left to right, to keep the
			// growth balanced.
			toGive := remaining
			for _, f := range flex {
				if toGive == 0 {
					break
				}
				if assigned[f.idx] < effMax[f.idx] {
					assigned[f.idx]++
					gave++
					toGive--
				}
			}
			if gave == 0 {
				return
			}
		}
		remaining -= gave
	}
}

type flexCol struct {
	idx    int
	weight float64
}

// flexColumns returns the columns that are below their effMax and
// carry a positive weight, along with the total of those weights.
func flexColumns(assigned, effMax []int, weights []float64) (flex []flexCol, totalWeight float64) {
	for i := range assigned {
		if weights[i] <= 0 {
			continue
		}
		if assigned[i] >= effMax[i] {
			continue
		}
		flex = append(flex, flexCol{idx: i, weight: weights[i]})
		totalWeight += weights[i]
	}
	return flex, totalWeight
}

// applyMultiSpanConstraint borrows width from outside-span columns
// and gives it to the narrowest in-span column until the cell's
// minimum is satisfied across its span (accounting for inter-column
// seams) or no further progress is possible. Donors must stay at or
// above effMin; receivers must stay at or below effMax. Returns true
// on full satisfaction.
func applyMultiSpanConstraint(assigned, effMin, effMax []int, cons multiSpanConstraint, seam int) bool {
	required := cons.minWidth - (cons.colSpan-1)*seam
	if required <= 0 {
		return true
	}
	for {
		if contentSum(assigned, cons.colStart, cons.colSpan) >= required {
			return true
		}
		donorIdx := outsideWidestSlack(assigned, effMin, cons.colStart, cons.colSpan)
		if donorIdx < 0 {
			return false
		}
		receiverIdx := insideNarrowestWithRoom(assigned, effMax, cons.colStart, cons.colSpan)
		if receiverIdx < 0 {
			return false
		}
		have := contentSum(assigned, cons.colStart, cons.colSpan)
		need := required - have
		donorSlack := assigned[donorIdx] - effMin[donorIdx]
		receiverRoom := effMax[receiverIdx] - assigned[receiverIdx]
		transfer := need
		if transfer > donorSlack {
			transfer = donorSlack
		}
		if transfer > receiverRoom {
			transfer = receiverRoom
		}
		if transfer <= 0 {
			return false
		}
		assigned[donorIdx] -= transfer
		assigned[receiverIdx] += transfer
	}
}

// outsideWidestSlack returns the index of the column outside
// [colStart, colStart+colSpan) with the greatest slack relative to
// its effective minimum. Returns -1 when no outside slack exists.
func outsideWidestSlack(assigned, effMin []int, colStart, colSpan int) int {
	best := -1
	var bestSlack int
	for i := range assigned {
		if i >= colStart && i < colStart+colSpan {
			continue
		}
		slack := assigned[i] - effMin[i]
		if slack > bestSlack {
			bestSlack = slack
			best = i
		}
	}
	return best
}

// insideNarrowestWithRoom returns the index of the narrowest in-span
// column that still has room to grow under its effMax. Returns -1
// when every in-span column is capped.
func insideNarrowestWithRoom(assigned, effMax []int, colStart, colSpan int) int {
	best := -1
	for i := colStart; i < colStart+colSpan; i++ {
		if assigned[i] >= effMax[i] {
			continue
		}
		if best < 0 || assigned[i] < assigned[best] {
			best = i
		}
	}
	return best
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
	geom := tableGeometry(t)
	mode := t.resolveEmojiWidth()

	wrapCell := func(c *Cell) []string {
		w := contentSum(assigned, c.gridCol, c.colSpan) + (c.colSpan-1)*geom.seam
		if w <= 0 {
			return nil
		}
		text, err := resolveCellContent(c)
		if err != nil {
			// The error was already recorded during Measure's walk;
			// here we just fall back to whatever bytes were buffered.
			text = c.content
		}
		wrap, trim, maxLines, trimPos := effectiveWrapParams(t, c)
		return Wrap(naturalLinesFor(text, mode), w, wrap, trim, maxLines, trimPos)
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

	// Rowspan bump pass. Use the effective (section-clamped) rowspan
	// so a rowspan declared past the last header/footer doesn't index
	// into the neighboring section's rows.
	for cell, lines := range wrapped {
		if cell.rowSpan == 1 {
			continue
		}
		absStart := absRowOf(t, cell)
		eff := effectiveRowSpan(t, cell)
		var have int
		for i := range eff {
			have += heights[absStart+i]
		}
		need := len(lines)
		if need > have {
			heights[absStart+eff-1] += need - have
		}
	}

	return heights
}

// effectiveWrapParams resolves the wrap/trim/maxLines/trimPosition
// knobs for a cell through the Style cascade. Fields not set at any
// level fall back to their package defaults (wrap=true, trim=true,
// maxLines=0 for unbounded, trimPosition=TrimEnd).
func effectiveWrapParams(t *Table, c *Cell) (wrap, trim bool, maxLines int, trimPos TrimPosition) {
	style := t.effectiveCellStyle(c)
	wrap = true
	trim = true
	trimPos = TrimEnd
	if style.set&sWrap != 0 {
		wrap = style.wrap
	}
	if style.set&sTrim != 0 {
		trim = style.trim
	}
	if style.set&sMaxLines != 0 {
		maxLines = style.maxLines
	}
	if style.set&sTrimPos != 0 {
		trimPos = style.trimPosition
	}
	return wrap, trim, maxLines, trimPos
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
