// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import "strings"

// renderContext carries the per-render caches and table geometry. It
// is created fresh for each call to Table.String / Table.WriteTo so
// multiple renders of the same table are independent.
type renderContext struct {
	t          *Table
	layout     *layoutResult
	border     BorderSet
	nCols      int
	nRows      int
	padL       int
	padR       int
	seam       int
	emojiWidth EmojiWidthMode
}

func newRenderContext(t *Table, l *layoutResult, b BorderSet) *renderContext {
	geom := tableGeometry(t)
	return &renderContext{
		t:          t,
		layout:     l,
		border:     b,
		nCols:      t.NumColumns(),
		nRows:      t.NumRows(),
		padL:       t.opts.padding.Left,
		padR:       t.opts.padding.Right,
		seam:       geom.seam,
		emojiWidth: t.resolveEmojiWidth(),
	}
}

// renderTable produces the full printable representation of t using
// the column widths and wrapped content from l. Each line, including
// the top and bottom border, is terminated with '\n'. An empty table
// (zero rows or zero columns) renders to the empty string. A border
// line is omitted entirely when every adjacent cell's per-edge
// directive resolves to BorderEdgeNone (see Style.borderTop /
// borderBottom).
func renderTable(t *Table, l *layoutResult, b BorderSet) string {
	rc := newRenderContext(t, l, b)
	if rc.nCols == 0 || rc.nRows == 0 {
		return ""
	}
	var out strings.Builder
	rc.writeBorderLine(&out, 0)
	for r := range rc.nRows {
		for s := range l.rowHeights[r] {
			out.WriteString(rc.contentLine(r, s))
			out.WriteByte('\n')
		}
		if r < rc.nRows-1 {
			rc.writeBorderLine(&out, r+1)
		}
	}
	rc.writeBorderLine(&out, rc.nRows)
	return out.String()
}

// writeBorderLine emits a border line at boundary r if any adjacent
// cell wants one; otherwise it writes nothing (not even a newline),
// so the preceding and following content rows sit flush together.
func (rc *renderContext) writeBorderLine(out *strings.Builder, r int) {
	if !rc.shouldEmitBorderLine(r) {
		return
	}
	out.WriteString(rc.borderLine(r))
	out.WriteByte('\n')
}

// columnCellWidth returns the number of terminal columns occupied by a
// column's cell area — left padding + assigned content width + right
// padding. It does not include adjacent border glyphs.
func (rc *renderContext) columnCellWidth(c int) int {
	return rc.padL + rc.layout.colAssigned[c] + rc.padR
}

// cellContentWidth returns the inner content width for a cell
// spanning cell.colSpan columns starting at cell.gridCol. Each
// internal seam contributes seamWidth columns (border + adjacent
// paddings) to the content area because the suppressed border and
// paddings become part of the cell's space.
func (rc *renderContext) cellContentWidth(cell *Cell) int {
	return contentSum(rc.layout.colAssigned, cell.gridCol, cell.colSpan) +
		(cell.colSpan-1)*rc.seam
}

// isBorderSuppressedH reports whether the horizontal border segment
// at column c in the gap between absolute rows r-1 and r is
// suppressed by a rowspan cell that covers both sides. At the top
// (r == 0) and bottom (r == nRows) borders the segment is always
// drawn.
func (rc *renderContext) isBorderSuppressedH(r, c int) bool {
	if r == 0 || r == rc.nRows {
		return false
	}
	above := rc.t.CellAt(r-1, c)
	below := rc.t.CellAt(r, c)
	return above != nil && above == below
}

// isBorderSuppressedV reports whether the vertical border segment in
// row r at the boundary between columns c-1 and c is suppressed by a
// colspan cell that covers both sides. The outer left (c == 0) and
// outer right (c == nCols) boundaries are always drawn.
func (rc *renderContext) isBorderSuppressedV(r, c int) bool {
	if c == 0 || c == rc.nCols {
		return false
	}
	left := rc.t.CellAt(r, c-1)
	right := rc.t.CellAt(r, c)
	return left != nil && left == right
}

// junctionArms computes the four-bit arm mask for the border
// junction at the intersection of the horizontal border line between
// rows r-1 and r and the column boundary c. Each arm is set only
// when the corresponding border segment is actually drawn — either
// because a rowspan/colspan suppresses it, or because the aggregated
// per-edge directive at that segment evaluates to something other
// than Solid. Hidden and None arms both leave the bit cleared; the
// renderer substitutes a space at that position.
func (rc *renderContext) junctionArms(r, c int) int {
	var mask int
	if r > 0 && !rc.isBorderSuppressedV(r-1, c) && rc.vSeamEdge(r-1, c) == BorderEdgeSolid {
		mask |= armN
	}
	if r < rc.nRows && !rc.isBorderSuppressedV(r, c) && rc.vSeamEdge(r, c) == BorderEdgeSolid {
		mask |= armS
	}
	if c < rc.nCols && !rc.isBorderSuppressedH(r, c) && rc.hBoundaryColumnEdge(r, c) == BorderEdgeSolid {
		mask |= armE
	}
	if c > 0 && !rc.isBorderSuppressedH(r, c-1) && rc.hBoundaryColumnEdge(r, c-1) == BorderEdgeSolid {
		mask |= armW
	}
	return mask
}

// edgeSide names the four per-cell border edges.
type edgeSide int

const (
	edgeTop edgeSide = iota
	edgeRight
	edgeBottom
	edgeLeft
)

// cellEdge returns the resolved per-edge directive for cell on side.
// A nil cell falls back to the table-level default; the table-level
// default itself falls back to Solid when nothing in the cascade has
// set it, preserving today's "draw all borders" behaviour.
func (rc *renderContext) cellEdge(cell *Cell, side edgeSide) BorderEdge {
	if cell == nil {
		return rc.tableDefaultEdge(side)
	}
	s := rc.t.effectiveCellStyle(cell)
	return resolveEdgeWithDefault(edgeOf(s, side), rc.tableDefaultEdge(side))
}

// tableDefaultEdge returns the table-level default for side, or
// Solid when unset.
func (rc *renderContext) tableDefaultEdge(side edgeSide) BorderEdge {
	if rc.t.style == nil {
		return BorderEdgeSolid
	}
	e := edgeOf(rc.t.style, side)
	if e == BorderEdgeAuto {
		return BorderEdgeSolid
	}
	return e
}

func edgeOf(s *Style, side edgeSide) BorderEdge {
	if s == nil {
		return BorderEdgeAuto
	}
	switch side {
	case edgeTop:
		return s.borderTop
	case edgeRight:
		return s.borderRight
	case edgeBottom:
		return s.borderBottom
	case edgeLeft:
		return s.borderLeft
	}
	return BorderEdgeAuto
}

func resolveEdgeWithDefault(e, fallback BorderEdge) BorderEdge {
	if e == BorderEdgeAuto {
		return fallback
	}
	return e
}

// hBoundaryColumnEdge aggregates the per-column effective edge
// directive at the horizontal boundary between rows r-1 and r, in
// column c. Precedence: Solid > Hidden > None.
func (rc *renderContext) hBoundaryColumnEdge(r, c int) BorderEdge {
	if c < 0 || c >= rc.nCols {
		return BorderEdgeNone
	}
	var above, below BorderEdge
	if r > 0 {
		above = rc.cellEdge(rc.t.CellAt(r-1, c), edgeBottom)
	}
	if r < rc.nRows {
		below = rc.cellEdge(rc.t.CellAt(r, c), edgeTop)
	}
	return strongerEdge(above, below)
}

// vSeamEdge aggregates the per-row effective edge directive at the
// vertical seam between columns c-1 and c, in row r.
func (rc *renderContext) vSeamEdge(r, c int) BorderEdge {
	if r < 0 || r >= rc.nRows {
		return BorderEdgeNone
	}
	var left, right BorderEdge
	if c > 0 {
		left = rc.cellEdge(rc.t.CellAt(r, c-1), edgeRight)
	}
	if c < rc.nCols {
		right = rc.cellEdge(rc.t.CellAt(r, c), edgeLeft)
	}
	return strongerEdge(left, right)
}

// strongerEdge returns the higher-precedence of a and b. Solid wins
// over Hidden wins over None. Auto is treated as None for aggregation
// (callers should resolve it first).
func strongerEdge(a, b BorderEdge) BorderEdge {
	if a == BorderEdgeSolid || b == BorderEdgeSolid {
		return BorderEdgeSolid
	}
	if a == BorderEdgeHidden || b == BorderEdgeHidden {
		return BorderEdgeHidden
	}
	return BorderEdgeNone
}

// shouldEmitBorderLine reports whether the horizontal border line at
// boundary r should be written. Lines are omitted when every column
// position and every junction resolves to BorderEdgeNone.
func (rc *renderContext) shouldEmitBorderLine(r int) bool {
	for c := range rc.nCols {
		if rc.isBorderSuppressedH(r, c) {
			// A rowspan-suppressed segment forces the line to exist —
			// the cell's content shows through here.
			return true
		}
		if rc.hBoundaryColumnEdge(r, c) != BorderEdgeNone {
			return true
		}
	}
	// Check junction arms too: a vertical run meeting the boundary
	// may request a glyph even when every horizontal column is None.
	for c := 0; c <= rc.nCols; c++ {
		if rc.junctionArms(r, c) != 0 {
			return true
		}
	}
	return false
}

// borderLine renders a horizontal border line at boundary r. r == 0
// is the top border, r == nRows is the bottom, everything in between
// is an inter-row separator. Border segments that would cross a
// rowspan cell are replaced with that cell's content at the matching
// sub-line. Per-column border directives determine whether each
// column position emits the BorderSet's horizontal glyph (Solid) or
// a run of spaces (Hidden or None).
func (rc *renderContext) borderLine(r int) string {
	var b strings.Builder
	rc.writeJunction(&b, r, 0)
	c := 0
	for c < rc.nCols {
		if rc.isBorderSuppressedH(r, c) {
			// Suppressed by a rowspan: emit the cell's content once
			// over its full colspan (the writeCellSlice output covers
			// any internal seams). Advance past the cell so we don't
			// re-emit at continuation columns.
			cell := rc.t.CellAt(r, c)
			rc.writeCellSlice(&b, cell, rc.cellSubLineAtBorder(cell, r))
			c += cell.colSpan
		} else {
			w := rc.columnCellWidth(c)
			glyph := ' '
			if rc.hBoundaryColumnEdge(r, c) == BorderEdgeSolid {
				glyph = rc.border.Horizontal
			}
			b.WriteString(rc.styleBorder(strings.Repeat(string(glyph), w)))
			c++
		}
		rc.writeJunction(&b, r, c)
	}
	return b.String()
}

func (rc *renderContext) writeJunction(b *strings.Builder, r, c int) {
	arms := rc.junctionArms(r, c)
	glyph := rune(0)
	if arms != 0 {
		glyph = rc.border.Joins[arms]
	}
	if glyph == 0 {
		glyph = ' '
	}
	b.WriteString(rc.styleBorder(string(glyph)))
}

// styleBorder wraps s with the table-level border color when one is
// set, leaving it unchanged otherwise.
func (rc *renderContext) styleBorder(s string) string {
	return rc.t.style.applyBorder(s)
}

// contentLine renders a single content line: absolute row r,
// sub-line index s within that row. Cells spanning multiple columns
// emit once at their anchor column; empty slots (where no cell
// exists at the (r, c) position) emit blank space. Vertical seams
// consult per-cell border-left/right directives — Solid uses the
// BorderSet's Vertical glyph, Hidden/None substitute a space.
func (rc *renderContext) contentLine(r, s int) string {
	var b strings.Builder
	b.WriteString(rc.seamGlyph(r, 0))
	c := 0
	for c < rc.nCols {
		cell := rc.t.CellAt(r, c)
		if cell == nil {
			b.WriteString(strings.Repeat(" ", rc.columnCellWidth(c)))
			c++
			b.WriteString(rc.seamGlyph(r, c))
			continue
		}
		if cell.gridCol != c {
			c = cell.gridCol + cell.colSpan
			b.WriteString(rc.seamGlyph(r, c))
			continue
		}
		rc.writeCellSlice(&b, cell, rc.cellSubLineAtContent(cell, r, s))
		c += cell.colSpan
		b.WriteString(rc.seamGlyph(r, c))
	}
	return b.String()
}

// seamGlyph returns the styled glyph drawn at the vertical seam
// between columns c-1 and c in row r. Outer seams (c == 0 or
// c == nCols) have only one adjacent cell; internal seams aggregate
// both sides through vSeamEdge.
func (rc *renderContext) seamGlyph(r, c int) string {
	if rc.vSeamEdge(r, c) == BorderEdgeSolid {
		return rc.styleBorder(string(rc.border.Vertical))
	}
	return " "
}

// writeCellSlice writes padding + aligned content + padding for cell
// at the given local sub-line index within the cell's wrapped output.
// Sub-lines past the wrapped content yield blank space, preserving the
// cell's width. The effective style (table → column → row → cell) is
// applied to the whole slot so background colors extend into the
// padding. Vertical alignment shifts which wrapped line maps to the
// current sub-line — VAlignTop leaves idx = subLine (blanks at the
// bottom), VAlignBottom subtracts the excess (blanks at the top),
// VAlignMiddle splits the excess above and below.
func (rc *renderContext) writeCellSlice(b *strings.Builder, cell *Cell, subLine int) {
	lines := rc.layout.wrapped[cell]
	h := len(lines)
	vspan := rc.cellVerticalSpan(cell)

	offset := 0
	switch rc.effectiveCellVAlign(cell) {
	case VAlignMiddle:
		if vspan > h {
			offset = (vspan - h) / 2
		}
	case VAlignBottom:
		if vspan > h {
			offset = vspan - h
		}
	case VAlignTop:
		// no offset
	}
	idx := subLine - offset

	var slot strings.Builder
	slot.WriteString(strings.Repeat(" ", rc.padL))
	var line string
	if idx >= 0 && idx < h {
		line = lines[idx]
	}
	slot.WriteString(alignText(line, rc.cellContentWidth(cell), rc.effectiveCellAlign(cell), rc.emojiWidth))
	slot.WriteString(strings.Repeat(" ", rc.padR))

	style := rc.effectiveCellStyle(cell)
	b.WriteString(style.applyContent(slot.String()))
}

// cellVerticalSpan returns the number of output lines the cell
// occupies vertically — the sum of row heights across its effective
// rowspan plus one line per internal separator.
func (rc *renderContext) cellVerticalSpan(cell *Cell) int {
	a := absRowOf(rc.t, cell)
	rs := effectiveRowSpan(rc.t, cell)
	var sum int
	for i := a; i < a+rs; i++ {
		sum += rc.layout.rowHeights[i]
	}
	return sum + rs - 1
}

// effectiveCellVAlign resolves a cell's vertical alignment through
// the same cascade as its colour style. Defaults to VAlignTop.
func (rc *renderContext) effectiveCellVAlign(cell *Cell) VerticalAlignment {
	style := rc.effectiveCellStyle(cell)
	if style.set&sVAlign != 0 {
		return style.valign
	}
	return VAlignTop
}

// effectiveCellAlign resolves a cell's horizontal alignment. The
// value is pulled from the effective style (which already cascades
// table → column → row → cell); AlignLeft is the default when no
// level has set an alignment.
func (rc *renderContext) effectiveCellAlign(cell *Cell) Alignment {
	style := rc.effectiveCellStyle(cell)
	if style.set&sAlign != 0 {
		return style.align
	}
	return AlignLeft
}

// effectiveCellStyle delegates to Table.effectiveCellStyle so the
// same cascade logic is used by layout and render.
func (rc *renderContext) effectiveCellStyle(cell *Cell) *Style {
	return rc.t.effectiveCellStyle(cell)
}

// cellSubLineAtBorder computes the sub-line index within a rowspan
// cell that sits on the horizontal border line between rows r-1 and
// r. Valid only when the border is suppressed for this cell (i.e.
// the cell covers both sides of the border).
func (rc *renderContext) cellSubLineAtBorder(cell *Cell, r int) int {
	a := absRowOf(rc.t, cell)
	var sum int
	for i := a; i <= r-1; i++ {
		sum += rc.layout.rowHeights[i]
	}
	return sum + (r - 1 - a)
}

// cellSubLineAtContent computes the sub-line index within a cell
// that corresponds to absolute row r, sub-line index s within that
// row. For cells anchored at r (rowSpan == 1 or first row of a
// rowspan), this is simply s.
func (rc *renderContext) cellSubLineAtContent(cell *Cell, r, s int) int {
	a := absRowOf(rc.t, cell)
	var sum int
	for i := a; i <= r-1; i++ {
		sum += rc.layout.rowHeights[i]
	}
	return sum + (r - a) + s
}

// alignText pads s to the given width using ASCII spaces according
// to the alignment. The mode controls how composite emoji widths
// are counted — the renderer passes its resolved EmojiWidthMode so
// padding math matches layout math. When the visible content
// already meets or exceeds width, s is returned unchanged — rendered
// output may overflow its slot in that case.
func alignText(s string, width int, align Alignment, mode EmojiWidthMode) string {
	vw := displayWidthFor(s, mode)
	if vw >= width {
		return s
	}
	extra := width - vw
	switch align {
	case AlignLeft:
		return s + strings.Repeat(" ", extra)
	case AlignRight:
		return strings.Repeat(" ", extra) + s
	case AlignCenter:
		left := extra / 2
		right := extra - left
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
	}
	return s
}
