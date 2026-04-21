// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import "io"

// TableOption configures a *Table.
type TableOption func(*Table)

// RowOption configures a row being added via AddRow, AddHeader, or
// AddFooter. Internally it operates on the shared rowBody; users never
// interact with rowBody directly.
type RowOption func(*rowBody)

// CellOption configures a *Cell during NewCell, AddCell, or AttachCell.
type CellOption func(*Cell)

// ColumnOption configures a *Column. Reserved for a later phase; no
// column configuration exists yet.
type ColumnOption func(*Column)

// ---------------------------------------------------------------------
// Table options
// ---------------------------------------------------------------------

// WithTableID assigns a unique ID to the table itself, retrievable via
// Table.GetElementByID.
func WithTableID(id string) TableOption {
	return func(t *Table) { t.id = id }
}

// WithTargetWidth pins the layout target width to w terminal columns.
// When unset, the table reads the COLUMNS environment variable, then
// falls back to 80.
func WithTargetWidth(w int) TableOption {
	return func(t *Table) {
		t.opts.targetWidth = w
		t.opts.targetWidthSet = true
	}
}

// WithBorder replaces the table's border glyph set. Defaults to
// DefaultSingleLine.
func WithBorder(b BorderSet) TableOption {
	return func(t *Table) { t.opts.border = b }
}

// WithSpanOverwrite controls span-conflict behavior. When false (the
// default), a colliding cell span returns ErrSpanConflict. When true,
// later cells overwrite earlier spans: fully-covered cells are dropped
// and partially overlapped cells are truncated, with events recorded
// in Table.Warnings.
func WithSpanOverwrite(enable bool) TableOption {
	return func(t *Table) { t.opts.spanOverwrite = enable }
}

// ---------------------------------------------------------------------
// Row / Header / Footer options
// ---------------------------------------------------------------------

// WithRowID assigns a unique ID to the row being added.
func WithRowID(id string) RowOption {
	return func(r *rowBody) { r.id = id }
}

// WithCell queues a previously constructed cell for adoption into the
// row. Multiple WithCell options may be supplied; they are attached in
// the order given after the row itself has been inserted.
func WithCell(c *Cell) RowOption {
	return func(r *rowBody) {
		r.pendingCells = append(r.pendingCells, c)
	}
}

// ---------------------------------------------------------------------
// Cell options
// ---------------------------------------------------------------------

// WithCellID assigns a unique ID to the cell.
func WithCellID(id string) CellOption {
	return func(c *Cell) { c.id = id }
}

// WithContent sets the cell's textual content. Honors "\n" as a hard
// line break; combines with automatic wrapping when the cell is wider
// than its assigned column width.
func WithContent(s string) CellOption {
	return func(c *Cell) {
		c.content = s
		c.hasContent = true
	}
}

// WithReader sets the cell's content source to an io.Reader consumed
// lazily on the first render pass. Cannot be combined with WithContent
// (ErrContentAndReader).
func WithReader(r io.Reader) CellOption {
	return func(c *Cell) { c.reader = r }
}

// WithColSpan sets the number of columns the cell occupies. Must be
// >= 1 (ErrInvalidSpan).
func WithColSpan(n int) CellOption {
	return func(c *Cell) { c.colSpan = n }
}

// WithRowSpan sets the number of rows the cell occupies within its
// section. Must be >= 1 (ErrInvalidSpan). Rowspans cannot cross
// section boundaries.
func WithRowSpan(n int) CellOption {
	return func(c *Cell) { c.rowSpan = n }
}

// WithAlign sets the cell's horizontal alignment. Default is AlignLeft.
func WithAlign(a Alignment) CellOption {
	return func(c *Cell) { c.opts.align = a }
}

// WithWrap toggles automatic word-wrapping on whitespace. Default is
// true.
func WithWrap(enable bool) CellOption {
	return func(c *Cell) { c.opts.wrap = enable }
}

// WithTrim toggles ellipsis-based trimming when a cell's wrapped
// content exceeds its allotted height. Default is true. In Phase 3 no
// height cap is active, so this option has no observable effect yet;
// Phase 4+ will plumb the cap.
func WithTrim(enable bool) CellOption {
	return func(c *Cell) { c.opts.trim = enable }
}

// WithPadding overrides the default cell padding. Default is
// DefaultPadding().
func WithPadding(p Padding) CellOption {
	return func(c *Cell) { c.opts.padding = p }
}

// WithMaxLines caps the cell's wrapped content to at most n lines. A
// value of 0 means unbounded (the default). Plumbed for Phase 4+; has
// no observable effect yet.
func WithMaxLines(n int) CellOption {
	return func(c *Cell) { c.opts.maxLines = n }
}

// ---------------------------------------------------------------------
// Column options (reserved)
// ---------------------------------------------------------------------

// WithColumnID assigns a unique ID to a column.
func WithColumnID(id string) ColumnOption {
	return func(c *Column) { c.id = id }
}
