// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import "strings"

// renderContext carries the per-render caches and table geometry. It
// is created fresh for each call to Table.String / Table.WriteTo so
// multiple renders of the same table are independent.
type renderContext struct {
	t      *Table
	layout *layoutResult
	border BorderSet
	nCols  int
	nRows  int
	// Default padding: Phase 3 uses a uniform 1-column left/right pad.
	// Cell-level padding overrides are plumbed through options but not
	// honored here; Phase 4 will wire them up.
	padL int
	padR int
}

func newRenderContext(t *Table, l *layoutResult, b BorderSet) *renderContext {
	return &renderContext{
		t:      t,
		layout: l,
		border: b,
		nCols:  t.NumColumns(),
		nRows:  t.NumRows(),
		padL:   1,
		padR:   1,
	}
}

// renderTable produces the full printable representation of t using
// the column widths and wrapped content from l. Each line, including
// the top and bottom border, is terminated with '\n'. An empty table
// (zero rows or zero columns) renders to the empty string.
func renderTable(t *Table, l *layoutResult, b BorderSet) string {
	rc := newRenderContext(t, l, b)
	if rc.nCols == 0 || rc.nRows == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString(rc.borderLine(0))
	out.WriteByte('\n')
	for r := range rc.nRows {
		for s := range l.rowHeights[r] {
			out.WriteString(rc.contentLine(r, s))
			out.WriteByte('\n')
		}
		if r < rc.nRows-1 {
			out.WriteString(rc.borderLine(r + 1))
			out.WriteByte('\n')
		}
	}
	out.WriteString(rc.borderLine(rc.nRows))
	out.WriteByte('\n')
	return out.String()
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
		(cell.colSpan-1)*seamWidth
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
// rows r-1 and r and the column boundary c. Each arm is set only when
// the corresponding border segment is actually drawn (not suppressed
// by a span crossing through the junction).
func (rc *renderContext) junctionArms(r, c int) int {
	var mask int
	if r > 0 && !rc.isBorderSuppressedV(r-1, c) {
		mask |= armN
	}
	if r < rc.nRows && !rc.isBorderSuppressedV(r, c) {
		mask |= armS
	}
	if c < rc.nCols && !rc.isBorderSuppressedH(r, c) {
		mask |= armE
	}
	if c > 0 && !rc.isBorderSuppressedH(r, c-1) {
		mask |= armW
	}
	return mask
}

// borderLine renders a horizontal border line at boundary r. r == 0
// is the top border, r == nRows is the bottom, everything in between
// is an inter-row separator. Border segments that would cross a
// rowspan cell are replaced with that cell's content at the matching
// sub-line.
func (rc *renderContext) borderLine(r int) string {
	var b strings.Builder
	for c := 0; c <= rc.nCols; c++ {
		glyph := rc.border.Joins[rc.junctionArms(r, c)]
		if glyph == 0 {
			glyph = ' '
		}
		b.WriteRune(glyph)
		if c == rc.nCols {
			break
		}
		if rc.isBorderSuppressedH(r, c) {
			// The border passes through a rowspan cell. Emit the
			// cell's content at this sub-line instead of a fill run.
			cell := rc.t.CellAt(r, c)
			rc.writeCellSlice(&b, cell, rc.cellSubLineAtBorder(cell, r))
			// Advance past any further columns the same cell covers;
			// the inner-border junctions at those boundaries are also
			// suppressed (the junctionArms loop still visits them but
			// emits either the cross-cell glyph or a space — handled
			// via armE/armW suppression above).
			//
			// We still emit one junction per boundary, so the inner
			// loop continues as normal.
		} else {
			w := rc.columnCellWidth(c)
			b.WriteString(strings.Repeat(string(rc.border.Horizontal), w))
		}
	}
	return b.String()
}

// contentLine renders a single content line: absolute row r,
// sub-line index s within that row. Cells spanning multiple columns
// emit once at their anchor column; empty slots (where no cell
// exists at the (r, c) position) emit blank space.
func (rc *renderContext) contentLine(r, s int) string {
	var b strings.Builder
	b.WriteRune(rc.border.Vertical)
	c := 0
	for c < rc.nCols {
		cell := rc.t.CellAt(r, c)
		if cell == nil {
			// Empty slot: pad a blank cell area.
			b.WriteString(strings.Repeat(" ", rc.columnCellWidth(c)))
			c++
			if c < rc.nCols {
				b.WriteRune(rc.border.Vertical)
			}
			continue
		}
		// cell.gridCol must equal c at the anchor. If we hit a
		// continuation column it means our c advancement was off; be
		// defensive and skip past the span.
		if cell.gridCol != c {
			c = cell.gridCol + cell.colSpan
			if c < rc.nCols {
				b.WriteRune(rc.border.Vertical)
			}
			continue
		}
		rc.writeCellSlice(&b, cell, rc.cellSubLineAtContent(cell, r, s))
		c += cell.colSpan
		if c < rc.nCols {
			b.WriteRune(rc.border.Vertical)
		}
	}
	b.WriteRune(rc.border.Vertical)
	return b.String()
}

// writeCellSlice writes padding + aligned content + padding for cell
// at the given local sub-line index within the cell's wrapped output.
// Sub-lines past the wrapped content yield blank space, preserving the
// cell's width.
func (rc *renderContext) writeCellSlice(b *strings.Builder, cell *Cell, subLine int) {
	b.WriteString(strings.Repeat(" ", rc.padL))
	var line string
	if lines := rc.layout.wrapped[cell]; subLine >= 0 && subLine < len(lines) {
		line = lines[subLine]
	}
	b.WriteString(alignText(line, rc.cellContentWidth(cell), cell.opts.align))
	b.WriteString(strings.Repeat(" ", rc.padR))
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

// alignText pads s (whose visible width is DisplayWidth(s)) to the
// given width using ASCII spaces according to the alignment. When the
// visible content already meets or exceeds width, s is returned
// unchanged — rendered output may overflow its slot in that case.
func alignText(s string, width int, align Alignment) string {
	vw := DisplayWidth(s)
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
