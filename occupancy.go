// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

// occupancyGrid tracks which cell, if any, occupies each (row, col)
// position within a table section. Rows and columns grow on demand.
type occupancyGrid struct {
	rows  [][]*Cell // rows[r][c] is the cell covering that slot, or nil.
	nCols int
}

func newOccupancyGrid() *occupancyGrid {
	return &occupancyGrid{}
}

// ensure grows the grid so that at least nRows and nCols are
// addressable. New slots are nil.
func (g *occupancyGrid) ensure(nRows, nCols int) {
	if nCols > g.nCols {
		for i, row := range g.rows {
			g.rows[i] = extend(row, nCols)
		}
		g.nCols = nCols
	}
	for len(g.rows) < nRows {
		g.rows = append(g.rows, make([]*Cell, g.nCols))
	}
}

func extend(row []*Cell, nCols int) []*Cell {
	if cap(row) >= nCols {
		return row[:nCols]
	}
	out := make([]*Cell, nCols)
	copy(out, row)
	return out
}

// numRows returns the number of rows currently represented in the grid.
func (g *occupancyGrid) numRows() int { return len(g.rows) }

// at returns the cell at (r, c), or nil if that slot is empty or out of
// bounds.
func (g *occupancyGrid) at(r, c int) *Cell {
	if r < 0 || r >= len(g.rows) {
		return nil
	}
	if c < 0 || c >= g.nCols {
		return nil
	}
	return g.rows[r][c]
}

// nextFreeInRow returns the lowest column index >= fromCol whose slot
// in row r is unoccupied. Reserved slots (stamped by prior rowspans)
// count as occupied.
func (g *occupancyGrid) nextFreeInRow(r, fromCol int) int {
	if r < 0 {
		return fromCol
	}
	g.ensure(r+1, fromCol+1)
	c := fromCol
	for c < g.nCols && g.rows[r][c] != nil {
		c++
	}
	return c
}

// stamp places cell in every slot of the rectangle
// [r..r+rowSpan-1] x [c..c+colSpan-1], growing the grid as needed.
// Any non-nil existing occupant is NOT checked here; callers must use
// conflictsWith before calling stamp when conflict detection is needed.
func (g *occupancyGrid) stamp(cell *Cell, r, c, rowSpan, colSpan int) {
	g.ensure(r+rowSpan, c+colSpan)
	for rr := r; rr < r+rowSpan; rr++ {
		for cc := c; cc < c+colSpan; cc++ {
			g.rows[rr][cc] = cell
		}
	}
}

// unstamp clears every slot currently occupied by cell within the
// rectangle anchored at (r, c) with the given spans. Used to roll back
// a partial stamp or to drop an overwritten victim.
func (g *occupancyGrid) unstamp(cell *Cell, r, c, rowSpan, colSpan int) {
	if r < 0 || c < 0 {
		return
	}
	for rr := r; rr < r+rowSpan && rr < len(g.rows); rr++ {
		for cc := c; cc < c+colSpan && cc < g.nCols; cc++ {
			if g.rows[rr][cc] == cell {
				g.rows[rr][cc] = nil
			}
		}
	}
}

// occupantsIn returns the distinct cells occupying any slot in the
// rectangle [r..r+rowSpan-1] x [c..c+colSpan-1], preserving first-seen
// order. Out-of-bounds slots contribute nothing.
func (g *occupancyGrid) occupantsIn(r, c, rowSpan, colSpan int) []*Cell {
	var seen []*Cell
	for rr := r; rr < r+rowSpan && rr < len(g.rows); rr++ {
		for cc := c; cc < c+colSpan && cc < g.nCols; cc++ {
			cell := g.rows[rr][cc]
			if cell == nil {
				continue
			}
			if !contains(seen, cell) {
				seen = append(seen, cell)
			}
		}
	}
	return seen
}

func contains(xs []*Cell, x *Cell) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
