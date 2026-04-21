// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

// sectionKind tags which logical section of the table a row belongs to.
// Rowspans cannot cross section boundaries.
type sectionKind uint8

const (
	sectionHeader sectionKind = iota
	sectionBody
	sectionFooter
)

func (s sectionKind) String() string {
	switch s {
	case sectionHeader:
		return "header"
	case sectionFooter:
		return "footer"
	case sectionBody:
		return "body"
	}
	return "body"
}

// rowBody holds state shared by Row, Header, and Footer. It is never
// exposed directly; users interact through the wrapper types.
type rowBody struct {
	id         string
	table      *Table
	section    sectionKind
	sectionRow int
	cells      []*Cell

	// style applies to every cell in the row unless the cell sets its
	// own style fields. nil means "inherit from the table".
	style *Style

	// pendingCells are populated by WithCell options during AddRow /
	// AddHeader / AddFooter; they are attached after the row is inserted
	// into its section.
	pendingCells []*Cell
}

// Row is a row in the table body.
type Row struct{ rowBody }

// Header is a header row. Headers are rendered above body rows and may
// carry styling or layout differences once styling support lands.
type Header struct{ rowBody }

// Footer is a footer row. Footers are rendered below body rows.
type Footer struct{ rowBody }

// ---------------------------------------------------------------------
// Public API on the wrapper types
// ---------------------------------------------------------------------

// AddCell constructs a cell from the given options and attaches it to
// the row. Returns ErrSpanConflict (wrapped with coordinates) if the
// cell's span overlaps an occupied grid slot and the table is not
// configured with WithSpanOverwrite(true).
func (r *Row) AddCell(opts ...CellOption) (*Cell, error) { return r.addCell(opts) }

// AttachCell attaches a previously constructed cell to the row. The
// cell must not already belong to another row (ErrCellAlreadyAdopted).
func (r *Row) AttachCell(c *Cell) (*Cell, error) { return r.attachCell(c) }

// Cell returns the i-th cell declared in the row (logical
// coordinate). Returns nil if i is out of range.
func (r *Row) Cell(i int) *Cell { return r.cellAt(i) }

// Cells returns a snapshot of the cells declared in the row, in
// declaration order. Spans are not expanded.
func (r *Row) Cells() []*Cell { return r.cellsCopy() }

// ID returns the row's user-assigned ID, or the empty string.
func (r *Row) ID() string { return r.id }

func (r *Row) elementID() string { return r.id }

// AddCell constructs a cell from the given options and attaches it to
// the header row.
func (h *Header) AddCell(opts ...CellOption) (*Cell, error) { return h.addCell(opts) }

// AttachCell attaches a previously constructed cell to the header row.
func (h *Header) AttachCell(c *Cell) (*Cell, error) { return h.attachCell(c) }

// Cell returns the i-th cell declared in the header row.
func (h *Header) Cell(i int) *Cell { return h.cellAt(i) }

// Cells returns a snapshot of the header row's cells.
func (h *Header) Cells() []*Cell { return h.cellsCopy() }

// ID returns the header's user-assigned ID.
func (h *Header) ID() string { return h.id }

func (h *Header) elementID() string { return h.id }

// AddCell constructs a cell from the given options and attaches it to
// the footer row.
func (f *Footer) AddCell(opts ...CellOption) (*Cell, error) { return f.addCell(opts) }

// AttachCell attaches a previously constructed cell to the footer row.
func (f *Footer) AttachCell(c *Cell) (*Cell, error) { return f.attachCell(c) }

// Cell returns the i-th cell declared in the footer row.
func (f *Footer) Cell(i int) *Cell { return f.cellAt(i) }

// Cells returns a snapshot of the footer row's cells.
func (f *Footer) Cells() []*Cell { return f.cellsCopy() }

// ID returns the footer's user-assigned ID.
func (f *Footer) ID() string { return f.id }

func (f *Footer) elementID() string { return f.id }

// ---------------------------------------------------------------------
// Shared implementation on rowBody
// ---------------------------------------------------------------------

func (r *rowBody) cellAt(i int) *Cell {
	if i < 0 || i >= len(r.cells) {
		return nil
	}
	return r.cells[i]
}

func (r *rowBody) cellsCopy() []*Cell {
	out := make([]*Cell, len(r.cells))
	copy(out, r.cells)
	return out
}

// addCell builds a new cell from the options, attaches it to the row,
// and returns it. A failure to attach returns the attach error without
// adding the cell to r.cells.
func (r *rowBody) addCell(opts []CellOption) (*Cell, error) {
	c := NewCell(opts...)
	return r.attachCell(c)
}

// attachCell anchors a pre-built cell into the row, stamping the
// section's occupancy grid. On success, the cell is appended to
// r.cells, its ID is registered with the table, and it is marked
// adopted.
func (r *rowBody) attachCell(c *Cell) (*Cell, error) {
	if c == nil {
		return nil, nil
	}
	if c.adopted {
		return nil, ErrCellAlreadyAdopted
	}
	if c.colSpan < 1 || c.rowSpan < 1 {
		return nil, ErrInvalidSpan
	}
	if c.hasContent && c.reader != nil {
		return nil, ErrContentAndReader
	}

	c.section = r.section
	c.sectionRow = r.sectionRow
	c.table = r.table

	if err := r.table.stampCell(c); err != nil {
		return nil, err
	}

	if err := r.table.registry.register(c.id, c); err != nil {
		// Unstamp on ID conflict.
		r.table.unstampCell(c)
		return nil, err
	}

	c.adopted = true
	r.cells = append(r.cells, c)
	return c, nil
}
