// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// defaultTargetWidth is used as the last-resort target width when no
// explicit WithTargetWidth is supplied, COLUMNS is unset or invalid,
// and no connected terminal can be queried for its size.
const defaultTargetWidth = 80

// Table is the root element. It owns the column list, the header / body
// / footer row collections, the ID registry, and the occupancy grids
// used for span tracking.
type Table struct {
	id      string
	opts    tableOptions
	columns []*Column

	headers []*Header
	rows    []*Row
	footers []*Footer

	headerOcc *occupancyGrid
	bodyOcc   *occupancyGrid
	footerOcc *occupancyGrid

	registry *idRegistry

	// authoringWarnings accumulate from construction-time events —
	// span overwrites, ID reassignments — and persist across renders.
	authoringWarnings []Warning

	// renderWarnings are produced by the most recent render pass
	// (span-overflow, cross-section spans, reader errors). They are
	// overwritten on every WriteTo call so repeated renders do not
	// compound the same events.
	renderWarnings []Warning

	// style applies table-wide defaults. Rows and cells merge their
	// own style over this. Border color lives here exclusively.
	style *Style

	// lastRenderErr captures any layout error surfaced by the most
	// recent call to String or WriteTo. Exposed via LastRenderError
	// so callers of String (which cannot return an error) can still
	// inspect failure modes.
	lastRenderErr error
}

type tableOptions struct {
	targetWidth    int
	targetWidthSet bool
	border         BorderSet
	padding        Padding
	emojiWidth     EmojiWidthMode
	spanOverwrite  bool
}

func defaultTableOptions() tableOptions {
	return tableOptions{
		border:        DefaultSingleLine(),
		padding:       DefaultPadding(),
		spanOverwrite: true, // safe default — conflicts never halt rendering
	}
}

// NewTable constructs an empty table configured with the given options.
func NewTable(opts ...TableOption) *Table {
	t := &Table{
		opts:      defaultTableOptions(),
		headerOcc: newOccupancyGrid(),
		bodyOcc:   newOccupancyGrid(),
		footerOcc: newOccupancyGrid(),
		registry:  newIDRegistry(),
	}
	for _, o := range opts {
		o(t)
	}
	if t.id != "" {
		// Safe direct insert: the registry is empty at construction so
		// there is no possible conflict to report.
		t.registry.m[t.id] = t
	}
	return t
}

// ID returns the table's user-assigned ID, or the empty string.
func (t *Table) ID() string { return t.id }

// NumColumns returns the number of columns currently present in the
// table. Columns grow as cells populate new positions.
func (t *Table) NumColumns() int { return len(t.columns) }

// NumRows returns the total number of rows across all sections
// (headers + body + footers).
func (t *Table) NumRows() int {
	return len(t.headers) + len(t.rows) + len(t.footers)
}

// Column returns the virtual Column element at index i, creating it
// (and any earlier missing columns) on demand.
func (t *Table) Column(i int) *Column {
	if i < 0 {
		return nil
	}
	for len(t.columns) <= i {
		t.growColumnTo(len(t.columns))
	}
	return t.columns[i]
}

// Columns returns a snapshot of the table's columns in index order.
func (t *Table) Columns() []*Column {
	out := make([]*Column, len(t.columns))
	copy(out, t.columns)
	return out
}

// Headers returns a snapshot of the table's header rows.
func (t *Table) Headers() []*Header {
	out := make([]*Header, len(t.headers))
	copy(out, t.headers)
	return out
}

// Rows returns a snapshot of the table's body rows.
func (t *Table) Rows() []*Row {
	out := make([]*Row, len(t.rows))
	copy(out, t.rows)
	return out
}

// Footers returns a snapshot of the table's footer rows.
func (t *Table) Footers() []*Footer {
	out := make([]*Footer, len(t.footers))
	copy(out, t.footers)
	return out
}

// Warnings returns the concatenation of authoring-time events (span
// overwrites, ID reassignments) and the events produced by the most
// recent render pass (span overflow, cross-section spans, reader
// errors). Calling String or WriteTo multiple times does not
// duplicate render-time events — each render overwrites them.
func (t *Table) Warnings() []Warning {
	out := make([]Warning, 0, len(t.authoringWarnings)+len(t.renderWarnings))
	out = append(out, t.authoringWarnings...)
	out = append(out, t.renderWarnings...)
	return out
}

// AddHeader appends a new header row and returns it. Under the
// default table configuration this call cannot fail. If strict
// mode is enabled via WithSpanOverwrite(false) and a pre-built
// cell supplied via WithCell produces a span conflict, AddHeader
// panics — use AttachCellWithError on the returned row for
// explicit error handling instead.
func (t *Table) AddHeader(opts ...RowOption) *Header {
	h := &Header{}
	commit := func() { t.headers = append(t.headers, h) }
	rollback := func() { t.headers = t.headers[:len(t.headers)-1] }
	if err := t.addSectionRow(&h.rowBody, sectionHeader, len(t.headers), h, opts, commit, rollback); err != nil {
		panic(err)
	}
	return h
}

// AddRow appends a new body row and returns it. See AddHeader for
// the panic contract.
func (t *Table) AddRow(opts ...RowOption) *Row {
	r := &Row{}
	commit := func() { t.rows = append(t.rows, r) }
	rollback := func() { t.rows = t.rows[:len(t.rows)-1] }
	if err := t.addSectionRow(&r.rowBody, sectionBody, len(t.rows), r, opts, commit, rollback); err != nil {
		panic(err)
	}
	return r
}

// AddFooter appends a new footer row and returns it. See AddHeader
// for the panic contract.
func (t *Table) AddFooter(opts ...RowOption) *Footer {
	f := &Footer{}
	commit := func() { t.footers = append(t.footers, f) }
	rollback := func() { t.footers = t.footers[:len(t.footers)-1] }
	if err := t.addSectionRow(&f.rowBody, sectionFooter, len(t.footers), f, opts, commit, rollback); err != nil {
		panic(err)
	}
	return f
}

// addSectionRow wires a rowBody into its section, applies RowOptions,
// reserves occupancy, registers the wrapper element's ID, and flushes
// any pending cells. commit/rollback manage the wrapper's membership in
// its typed section slice so error paths leave the table consistent.
func (t *Table) addSectionRow(
	body *rowBody,
	section sectionKind,
	sectionRow int,
	wrapper Element,
	opts []RowOption,
	commit func(),
	rollback func(),
) error {
	body.table = t
	body.section = section
	body.sectionRow = sectionRow
	for _, o := range opts {
		o(body)
	}
	t.occForSection(section).ensure(sectionRow+1, 0)
	if !t.registry.register(body.id, wrapper) {
		t.authoringWarnings = append(t.authoringWarnings,
			DuplicateIDEvent{ID: body.id, Kind: section.String()})
		body.id = ""
	}
	commit()
	if err := t.flushPendingCells(body); err != nil {
		rollback()
		t.registry.unregister(body.id)
		return err
	}
	return nil
}

// flushPendingCells attaches any cells accumulated by WithCell options
// during row construction. On the first error the newly-attached cells
// are unstamped and the error is returned.
func (t *Table) flushPendingCells(r *rowBody) error {
	attached := make([]*Cell, 0, len(r.pendingCells))
	for _, c := range r.pendingCells {
		if _, err := r.attachCell(c); err != nil {
			// Roll back any cells attached so far in this flush.
			for _, a := range attached {
				t.unstampCell(a)
				t.registry.unregister(a.id)
			}
			r.cells = r.cells[:len(r.cells)-len(attached)]
			return err
		}
		attached = append(attached, c)
	}
	r.pendingCells = nil
	return nil
}

// GetElementByID looks up any named element in the table: the table
// itself, a column, a header/body/footer row, or a cell. Returns nil
// if no element with that ID is registered.
func (t *Table) GetElementByID(id string) Element {
	return t.registry.lookup(id)
}

// CellAt returns the cell covering absolute grid coordinate (r, c). r
// is interpreted as: rows [0, len(Headers)) index into headers; rows
// [len(Headers), len(Headers)+len(Rows)) index into the body; the
// remainder index into footers. Returns nil if (r, c) is out of
// bounds or the slot is unoccupied.
func (t *Table) CellAt(r, c int) *Cell {
	hEnd := len(t.headers)
	bEnd := hEnd + len(t.rows)
	switch {
	case r < 0:
		return nil
	case r < hEnd:
		return t.headerOcc.at(r, c)
	case r < bEnd:
		return t.bodyOcc.at(r-hEnd, c)
	default:
		return t.footerOcc.at(r-bEnd, c)
	}
}

// InBounds reports whether (r, c) is a valid grid coordinate within the
// table's current dimensions.
func (t *Table) InBounds(r, c int) bool {
	if r < 0 || c < 0 {
		return false
	}
	if r >= t.NumRows() {
		return false
	}
	return c < t.NumColumns()
}

// ResolvedTargetWidth returns the target width the table will use for
// layout. The resolution cascade, in order of preference:
//
//  1. explicit WithTargetWidth(n)
//  2. the COLUMNS environment variable, when it parses to a positive int
//  3. defaultTargetWidth (80)
//
// Whatever value the cascade produces is then clamped to the attached
// terminal's width when one is detected (stdout or stderr, via
// golang.org/x/term), so output never exceeds the physical screen.
// Pipes and other non-interactive sinks leave the value uncapped.
func (t *Table) ResolvedTargetWidth() int {
	tty, ttyOK := terminalWidthProbe()

	want := defaultTargetWidth
	switch {
	case t.opts.targetWidthSet && t.opts.targetWidth > 0:
		want = t.opts.targetWidth
	default:
		if v := os.Getenv("COLUMNS"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				want = n
			}
		}
	}

	if ttyOK && want > tty {
		return tty
	}
	return want
}

// terminalWidthProbe is the injection point for TTY-size detection.
// Tests replace it to simulate a connected terminal of a given size.
var terminalWidthProbe = detectTerminalWidth

// detectTerminalWidth reports the attached terminal's column count when
// stdout (or stderr as a fallback) is a TTY. The probe is silent: any
// non-terminal fd or query failure returns (_, false) and the caller
// falls through to the next resolution step.
func detectTerminalWidth() (int, bool) {
	for _, f := range []*os.File{os.Stdout, os.Stderr} {
		if f == nil {
			continue
		}
		fd := int(f.Fd()) //nolint:gosec // fd values always fit in int
		if !term.IsTerminal(fd) {
			continue
		}
		w, _, err := term.GetSize(fd)
		if err != nil || w <= 0 {
			continue
		}
		return w, true
	}
	return 0, false
}

// String renders the table to a string. Layout errors (e.g., a target
// width too narrow to fit content minimums) do not prevent best-effort
// output — the renderer falls back to the minimum column widths and
// produces a possibly-overflowing table. Inspect Table.Warnings() to
// see non-fatal events collected during rendering; use WriteTo for
// access to the underlying error.
func (t *Table) String() string {
	var b strings.Builder
	_, t.lastRenderErr = t.WriteTo(&b)
	return b.String()
}

// LastRenderError returns the error from the most recent call to
// String, or nil if the last call succeeded. The value is overwritten
// on every String call (so calling String a second time after a
// successful render will clear a previous error). WriteTo callers do
// not need this accessor — they receive the error directly.
func (t *Table) LastRenderError() error { return t.lastRenderErr }

// WriteTo renders the table to w. Returns the number of bytes written
// and either a write error from w or a layout error (e.g.,
// ErrTargetTooNarrow) — write errors take precedence when both occur.
func (t *Table) WriteTo(w io.Writer) (int64, error) {
	m := Measure(t)
	l := Layout(t, m)
	t.renderWarnings = append(t.renderWarnings[:0], l.warnings...)
	out := renderTable(t, l, t.opts.border)
	n, err := w.Write([]byte(out))
	if err != nil {
		return int64(n), err
	}
	return int64(n), l.err
}

func (t *Table) elementID() string { return t.id }

// effectiveCellStyle cascades table → column → row → cell styles
// into a freshly-allocated Style. Every field that is set at any
// level is copied through, with lower-level set fields overriding
// upper-level ones. Used by the layout solver and renderer to
// resolve per-cell display attributes.
func (t *Table) effectiveCellStyle(c *Cell) *Style {
	eff := &Style{}
	eff.merge(t.style)
	if col := t.Column(c.gridCol); col != nil {
		eff.merge(col.style)
	}
	if row := t.rowBodyFor(c); row != nil {
		eff.merge(row.style)
	}
	eff.merge(c.style)
	return eff
}

// ---------------------------------------------------------------------
// Helpers used by rowBody / cell attachment
// ---------------------------------------------------------------------

func (t *Table) occForSection(k sectionKind) *occupancyGrid {
	switch k {
	case sectionHeader:
		return t.headerOcc
	case sectionFooter:
		return t.footerOcc
	case sectionBody:
		return t.bodyOcc
	}
	return t.bodyOcc
}

// stampCell resolves the cell's anchor column within its row, performs
// conflict detection, and stamps the section's occupancy grid. On
// success it also grows the table's column list to cover the cell's
// span.
func (t *Table) stampCell(c *Cell) error {
	occ := t.occForSection(c.section)
	// Determine the anchor column: start at the column after the last
	// cell already in the row and advance past reserved slots.
	startCol := t.nextColInRow(c.section, c.sectionRow)
	// Verify no conflict in the full rectangle.
	if victims := occ.occupantsIn(c.sectionRow, startCol, c.rowSpan, c.colSpan); len(victims) > 0 {
		if !t.opts.spanOverwrite {
			return fmt.Errorf(
				"%s row %d col %d span %dx%d: %w",
				c.section, c.sectionRow, startCol, c.rowSpan, c.colSpan,
				ErrSpanConflict,
			)
		}
		t.overwriteVictims(victims, c.sectionRow, startCol, c.rowSpan, c.colSpan)
	}
	c.gridCol = startCol
	occ.stamp(c, c.sectionRow, startCol, c.rowSpan, c.colSpan)
	// Grow the table's columns to cover this cell's span.
	t.growColumnTo(startCol + c.colSpan - 1)
	return nil
}

// unstampCell removes a cell's span from its section's occupancy grid.
func (t *Table) unstampCell(c *Cell) {
	if c == nil {
		return
	}
	occ := t.occForSection(c.section)
	occ.unstamp(c, c.sectionRow, c.gridCol, c.rowSpan, c.colSpan)
}

// detachCell removes a cell from its current row and occupancy
// grid, returning it to an unattached state ready for re-adoption
// by a different row. Used by attachCell to migrate pre-adopted
// cells without duplicating them in multiple rows.
func (t *Table) detachCell(c *Cell) {
	if c == nil || !c.adopted {
		return
	}
	t.unstampCell(c)
	t.removeCellFromRow(c)
	c.adopted = false
	c.gridCol = 0
	c.sectionRow = 0
	c.section = 0
	c.table = nil
}

// nextColInRow finds the lowest column >= 0 in (section, row) that is
// not yet occupied by any prior cell or reservation. When every slot up
// to the grid's current right edge is occupied, the next free position
// is just past it.
func (t *Table) nextColInRow(section sectionKind, row int) int {
	occ := t.occForSection(section)
	free := occ.nextFreeInRow(row, 0)
	if free < occ.nCols {
		return free
	}
	return occ.nCols
}

// growColumnTo ensures the columns slice has an entry at index i,
// creating any missing columns along the way.
func (t *Table) growColumnTo(i int) {
	for len(t.columns) <= i {
		col := newColumn(len(t.columns))
		col.table = t
		t.columns = append(t.columns, col)
	}
}

// overwriteVictims applies the WithSpanOverwrite(true) policy: cells
// whose anchors lie within the overwriter's rectangle are fully
// dropped; cells whose anchors lie outside but whose spans overlap are
// truncated back to the largest rectangle anchored at their original
// anchor that does not intersect the overwriter.
func (t *Table) overwriteVictims(victims []*Cell, r, c, rowSpan, colSpan int) {
	rEnd := r + rowSpan
	cEnd := c + colSpan
	for _, v := range victims {
		vREnd := v.sectionRow + v.rowSpan
		vCEnd := v.gridCol + v.colSpan
		anchorInside := v.sectionRow >= r && v.sectionRow < rEnd &&
			v.gridCol >= c && v.gridCol < cEnd
		if anchorInside {
			// Drop the victim entirely.
			t.unstampCell(v)
			t.removeCellFromRow(v)
			t.registry.unregister(v.id)
			t.authoringWarnings = append(t.authoringWarnings, OverwriteEvent{
				DroppedID: v.id,
				At:        [2]int{r, c},
			})
			continue
		}
		// Truncate: clear the victim completely, then re-stamp at the
		// largest rectangle from its anchor that does not intersect the
		// overwriter.
		newRowSpan, newColSpan := truncatedSpan(v.sectionRow, v.gridCol, vREnd, vCEnd, r, c, rEnd, cEnd)
		if newRowSpan < 1 || newColSpan < 1 {
			// Degenerate: treat as drop.
			t.unstampCell(v)
			t.removeCellFromRow(v)
			t.registry.unregister(v.id)
			t.authoringWarnings = append(t.authoringWarnings, OverwriteEvent{
				DroppedID: v.id,
				At:        [2]int{r, c},
			})
			continue
		}
		t.unstampCell(v)
		v.rowSpan = newRowSpan
		v.colSpan = newColSpan
		t.occForSection(v.section).stamp(v, v.sectionRow, v.gridCol, v.rowSpan, v.colSpan)
		t.authoringWarnings = append(t.authoringWarnings, OverwriteEvent{
			TruncatedID: v.id,
			NewColSpan:  newColSpan,
			NewRowSpan:  newRowSpan,
			At:          [2]int{r, c},
		})
	}
}

// truncatedSpan computes the largest rectangle anchored at
// (vr0, vc0) with original extent (vr1, vc1) that does not intersect
// the rectangle (r0, c0)..(r1, c1). The anchor is assumed outside the
// overwriter.
func truncatedSpan(vr0, vc0, vr1, vc1, r0, c0, r1, c1 int) (rowSpan, colSpan int) {
	rowSpan = vr1 - vr0
	colSpan = vc1 - vc0
	// Vertical clipping: if the victim's vertical range overlaps the
	// overwriter's vertical range AND the victim ends inside or past
	// it, clip the victim's height so it stops just before the
	// overwriter's top edge. Only meaningful when vr0 < r0.
	if vr0 < r0 && vr1 > r0 && colRangesOverlap(vc0, vc1, c0, c1) {
		rowSpan = r0 - vr0
	}
	// Horizontal clipping: analogous.
	if vc0 < c0 && vc1 > c0 && rowRangesOverlap(vr0, vr1, r0, r1) {
		colSpan = c0 - vc0
	}
	return rowSpan, colSpan
}

func colRangesOverlap(a0, a1, b0, b1 int) bool { return a0 < b1 && b0 < a1 }
func rowRangesOverlap(a0, a1, b0, b1 int) bool { return a0 < b1 && b0 < a1 }

// removeCellFromRow deletes cell c from its owning row's cells slice.
// No-op if the cell is not found.
func (t *Table) removeCellFromRow(c *Cell) {
	r := t.rowBodyFor(c)
	if r == nil {
		return
	}
	for i, cc := range r.cells {
		if cc == c {
			r.cells = append(r.cells[:i], r.cells[i+1:]...)
			return
		}
	}
}

func (t *Table) rowBodyFor(c *Cell) *rowBody {
	switch c.section {
	case sectionHeader:
		if c.sectionRow < len(t.headers) {
			return &t.headers[c.sectionRow].rowBody
		}
	case sectionFooter:
		if c.sectionRow < len(t.footers) {
			return &t.footers[c.sectionRow].rowBody
		}
	case sectionBody:
		if c.sectionRow < len(t.rows) {
			return &t.rows[c.sectionRow].rowBody
		}
	}
	return nil
}
