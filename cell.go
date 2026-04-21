// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import "io"

// Padding controls the empty space reserved inside a cell around its
// content. Values are measured in terminal columns (left/right) and rows
// (top/bottom).
type Padding struct {
	Left, Right, Top, Bottom int
}

// DefaultPadding is the padding applied to a cell when WithPadding is
// not supplied: one column of left/right breathing room, no vertical
// padding.
func DefaultPadding() Padding {
	return Padding{Left: 1, Right: 1, Top: 0, Bottom: 0}
}

// Cell is the fundamental content-bearing element. Cells belong to at
// most one row and occupy a rectangle in the table's grid defined by
// their anchor (GridRow, GridCol) and span (ColSpan, RowSpan).
type Cell struct {
	id string

	// Content source: at most one of content / reader is set. If
	// hasContent is true, content holds the authored string. Otherwise,
	// reader (if non-nil) is consumed lazily on the first render pass.
	content    string
	hasContent bool
	reader     io.Reader
	resolved   bool
	resolveErr error

	colSpan int
	rowSpan int

	section    sectionKind
	sectionRow int
	gridCol    int

	opts cellOptions

	// adopted is set once the cell belongs to a row.
	adopted bool
}

type cellOptions struct {
	align    Alignment
	wrap     bool
	trim     bool
	padding  Padding
	maxLines int // 0 = unbounded; reserved hook for a later phase
}

func defaultCellOptions() cellOptions {
	return cellOptions{
		align:   AlignLeft,
		wrap:    true,
		trim:    true,
		padding: DefaultPadding(),
	}
}

// NewCell constructs a detached cell with the given options. A detached
// cell has no grid position until it is passed to a row via WithCell or
// Row.AttachCell.
func NewCell(opts ...CellOption) *Cell {
	c := &Cell{
		colSpan: 1,
		rowSpan: 1,
		opts:    defaultCellOptions(),
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// ID returns the cell's user-assigned ID, or the empty string.
func (c *Cell) ID() string { return c.id }

// ColSpan returns the number of columns the cell occupies.
func (c *Cell) ColSpan() int { return c.colSpan }

// RowSpan returns the number of rows the cell occupies within its
// section.
func (c *Cell) RowSpan() int { return c.rowSpan }

// GridRow returns the absolute grid row of the cell's anchor across the
// whole table (headers, body, then footers concatenated).
func (c *Cell) GridRow() int {
	// Phase 1: callers that need section-local coordinates use SectionRow.
	// The absolute row is computed lazily against the owning table; until
	// that wiring lands in Phase 2, return the section-local row so tests
	// can still assert geometry.
	return c.sectionRow
}

// GridCol returns the zero-based column of the cell's anchor.
func (c *Cell) GridCol() int { return c.gridCol }

// Align returns the cell's horizontal alignment.
func (c *Cell) Align() Alignment { return c.opts.align }

// Content returns the cell's authored string content. If the cell was
// configured with WithReader and the reader has not yet been consumed,
// the empty string is returned.
func (c *Cell) Content() string { return c.content }

func (c *Cell) elementID() string { return c.id }
