// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

// BorderSet is the glyph set used to draw table borders. The Joins
// array is indexed by a four-bit arm mask with bit layout
// (least-significant first): N=1, E=2, S=4, W=8. Only the 11 valid
// entries (runs, corners, T-joins, crosses) are consulted during
// rendering; the remaining entries are left zero.
//
// Phase 1 ships the struct and DefaultSingleLine so options and tests
// can reference a concrete border set. Actual use lands with Phase 3
// rendering.
type BorderSet struct {
	// Horizontal is the glyph repeated along horizontal border runs.
	Horizontal rune
	// Vertical is the glyph repeated along vertical border runs.
	Vertical rune
	// Joins maps arm bitmasks to the glyph drawn at corners and
	// junctions.
	Joins [16]rune
}

// Arm bits for BorderSet.Joins.
const (
	armN = 1 << 0 // north (up)
	armE = 1 << 1 // east (right)
	armS = 1 << 2 // south (down)
	armW = 1 << 3 // west (left)
)

// DefaultSingleLine returns the Unicode single-line box-drawing
// BorderSet.
func DefaultSingleLine() BorderSet {
	var b BorderSet
	b.Horizontal = '─'
	b.Vertical = '│'
	b.Joins[armN|armS] = '│'
	b.Joins[armE|armW] = '─'
	b.Joins[armS|armE] = '┌'
	b.Joins[armS|armW] = '┐'
	b.Joins[armN|armE] = '└'
	b.Joins[armN|armW] = '┘'
	b.Joins[armN|armS|armE] = '├'
	b.Joins[armN|armS|armW] = '┤'
	b.Joins[armE|armS|armW] = '┬'
	b.Joins[armN|armE|armW] = '┴'
	b.Joins[armN|armE|armS|armW] = '┼'
	return b
}
