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

// AddCell constructs a cell from the given options and attaches it
// to the row. Under the default table configuration
// (WithSpanOverwrite(true)) this call cannot fail — span conflicts
// absorb existing cells and are recorded as OverwriteEvent
// warnings on Table.Warnings. Under WithSpanOverwrite(false),
// a conflict panics; use AddCellWithError for explicit error
// handling instead.
func (r *Row) AddCell(opts ...CellOption) *Cell { return mustCell(r.addCell(opts)) }

// AddCellWithError is the error-returning counterpart to AddCell.
// Useful when the table runs in strict mode (WithSpanOverwrite(false))
// and callers want to recover from span conflicts without a panic.
func (r *Row) AddCellWithError(opts ...CellOption) (*Cell, error) { return r.addCell(opts) }

// AttachCell attaches a previously constructed cell to the row.
// Cells already belonging to another row are migrated — their
// previous attachment is cleaned up before the new one is applied.
// The panic / WithError contract mirrors AddCell.
func (r *Row) AttachCell(c *Cell) *Cell { return mustCell(r.attachCell(c)) }

// AttachCellWithError is the error-returning counterpart to AttachCell.
func (r *Row) AttachCellWithError(c *Cell) (*Cell, error) { return r.attachCell(c) }

// Cell returns the i-th cell declared in the row (logical
// coordinate). Returns nil if i is out of range.
func (r *Row) Cell(i int) *Cell { return r.cellAt(i) }

// Cells returns a snapshot of the cells declared in the row, in
// declaration order. Spans are not expanded.
func (r *Row) Cells() []*Cell { return r.cellsCopy() }

// ID returns the row's user-assigned ID, or the empty string.
func (r *Row) ID() string { return r.id }

func (r *Row) elementID() string { return r.id }

// AddCell constructs a cell from the given options and attaches it
// to the header row. See Row.AddCell for the panic / error contract.
func (h *Header) AddCell(opts ...CellOption) *Cell { return mustCell(h.addCell(opts)) }

// AddCellWithError is the error-returning counterpart to AddCell.
func (h *Header) AddCellWithError(opts ...CellOption) (*Cell, error) { return h.addCell(opts) }

// AttachCell attaches a previously constructed cell to the header row.
func (h *Header) AttachCell(c *Cell) *Cell { return mustCell(h.attachCell(c)) }

// AttachCellWithError is the error-returning counterpart to AttachCell.
func (h *Header) AttachCellWithError(c *Cell) (*Cell, error) { return h.attachCell(c) }

// Cell returns the i-th cell declared in the header row.
func (h *Header) Cell(i int) *Cell { return h.cellAt(i) }

// Cells returns a snapshot of the header row's cells.
func (h *Header) Cells() []*Cell { return h.cellsCopy() }

// ID returns the header's user-assigned ID.
func (h *Header) ID() string { return h.id }

func (h *Header) elementID() string { return h.id }

// AddCell constructs a cell from the given options and attaches it
// to the footer row. See Row.AddCell for the panic / error contract.
func (f *Footer) AddCell(opts ...CellOption) *Cell { return mustCell(f.addCell(opts)) }

// AddCellWithError is the error-returning counterpart to AddCell.
func (f *Footer) AddCellWithError(opts ...CellOption) (*Cell, error) { return f.addCell(opts) }

// AttachCell attaches a previously constructed cell to the footer row.
func (f *Footer) AttachCell(c *Cell) *Cell { return mustCell(f.attachCell(c)) }

// AttachCellWithError is the error-returning counterpart to AttachCell.
func (f *Footer) AttachCellWithError(c *Cell) (*Cell, error) { return f.attachCell(c) }

// Cell returns the i-th cell declared in the footer row.
func (f *Footer) Cell(i int) *Cell { return f.cellAt(i) }

// Cells returns a snapshot of the footer row's cells.
func (f *Footer) Cells() []*Cell { return f.cellsCopy() }

// ID returns the footer's user-assigned ID.
func (f *Footer) ID() string { return f.id }

func (f *Footer) elementID() string { return f.id }

// mustCell is the internal helper that converts the
// (cell, error) return of the attach machinery into a single
// *Cell result by panicking on non-nil error. It keeps the panic
// vs. WithError split DRY across Row / Header / Footer.
func mustCell(c *Cell, err error) *Cell {
	if err != nil {
		panic(err)
	}
	return c
}

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

// attachCell anchors a cell into the row, stamping the section's
// occupancy grid. If the cell is already adopted by another row, it
// is first detached from its previous owner (a DOM-style move).
// Duplicate-ID and content-source-swap events are surfaced as
// warnings on Table.authoringWarnings rather than returned errors.
//
// The only case where this method returns a non-nil error is a
// span conflict under WithSpanOverwrite(false) — the single
// remaining "strict mode" failure mode.
func (r *rowBody) attachCell(c *Cell) (*Cell, error) {
	if c == nil {
		return nil, nil
	}
	if c.adopted {
		r.table.detachCell(c)
	}

	c.section = r.section
	c.sectionRow = r.sectionRow
	c.table = r.table

	if err := r.table.stampCell(c); err != nil {
		// Strict-mode span conflict. Keep c in the "unattached"
		// state (adopted=false, no row membership) so callers can
		// retry with a different cell definition.
		c.section = 0
		c.sectionRow = 0
		c.table = nil
		return nil, err
	}

	if !r.table.registry.register(c.id, c) {
		r.table.authoringWarnings = append(r.table.authoringWarnings,
			DuplicateIDEvent{ID: c.id, Kind: "cell"})
		// Clear the rejected ID so Cell.ID() matches what
		// Table.GetElementByID returns.
		c.id = ""
	}

	if c.contentSourceSwapped {
		r.table.authoringWarnings = append(r.table.authoringWarnings,
			ContentSourceReplacedEvent{
				CellID:      c.id,
				FinalSource: finalContentSourceLabel(c),
			})
		c.contentSourceSwapped = false
	}

	c.adopted = true
	r.cells = append(r.cells, c)
	return c, nil
}

func finalContentSourceLabel(c *Cell) string {
	if c.reader != nil {
		return "reader"
	}
	return "content"
}
