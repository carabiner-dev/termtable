// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import "errors"

var (
	// ErrSpanConflict is returned when a cell's span would overlap a grid
	// slot already occupied by another cell, and the table is not
	// configured with WithSpanOverwrite(true). Returned errors wrap this
	// sentinel with positional context.
	ErrSpanConflict = errors.New("cell span conflicts with an occupied grid slot")

	// ErrReaderAlreadyConsumed is a defensive guard returned if a cell's
	// reader has already been consumed when a render pass attempts to
	// resolve it again. In normal operation the cell buffers the reader's
	// content on first use so this error should not surface.
	ErrReaderAlreadyConsumed = errors.New("cell reader already consumed")

	// ErrTargetTooNarrow is returned during layout when the sum of
	// per-column minimum widths exceeds the configured target width and
	// no rendering is possible without collapsing content below
	// readability.
	ErrTargetTooNarrow = errors.New("target width too narrow for content minimums")

	// ErrCrossSectionSpan is the sentinel wrapped by CrossSectionSpanEvent
	// for callers that want to use errors.Is on a warning's backing
	// error. No API returns this error directly — rowspan clamping
	// is reported as a warning, not a returned error.
	ErrCrossSectionSpan = errors.New("row span crosses section boundary")
)

// Warning is implemented by non-fatal events surfaced during table
// construction or rendering. Retrieve them via Table.Warnings.
type Warning interface {
	warningTag()
	String() string
}

// OverwriteEvent is recorded when WithSpanOverwrite(true) causes a later
// cell's span to drop or truncate an earlier cell.
type OverwriteEvent struct {
	// DroppedID is set when an existing cell was entirely covered by the
	// new cell and removed from its row.
	DroppedID string

	// TruncatedID is set when an existing cell's span was reduced to
	// avoid the new cell. NewColSpan / NewRowSpan describe the resulting
	// span.
	TruncatedID string
	NewColSpan  int
	NewRowSpan  int

	// At is the grid anchor of the overwriting cell.
	At [2]int
}

func (OverwriteEvent) warningTag() {}

func (e OverwriteEvent) String() string {
	if e.DroppedID != "" {
		return "overwrite: dropped cell id=" + quote(e.DroppedID)
	}
	return "overwrite: truncated cell id=" + quote(e.TruncatedID)
}

// SpanOverflowEvent is recorded when a column-span cell cannot fit within
// the column budget its span covers, even after layout borrow/repay.
// Rendering continues but the cell overflows its allotted width.
type SpanOverflowEvent struct {
	CellID   string
	Required int
	Got      int
}

func (SpanOverflowEvent) warningTag() {}

func (e SpanOverflowEvent) String() string {
	return "span overflow: cell id=" + quote(e.CellID)
}

// ReaderErrorEvent is recorded when Measure's lazy reader consumption
// fails. The affected cell renders as empty; the error is preserved
// for inspection via Table.Warnings.
type ReaderErrorEvent struct {
	CellID string
	Err    error
}

func (ReaderErrorEvent) warningTag() {}

func (e ReaderErrorEvent) String() string {
	return "reader error: cell id=" + quote(e.CellID) + ": " + e.Err.Error()
}

// CrossSectionSpanEvent is recorded when a rowSpan declared on a cell
// reaches beyond the last row of its section (headers, body, or
// footers). Rendering clamps the effective rowspan to the section
// boundary; authored rowSpan on the Cell is preserved as-is.
type CrossSectionSpanEvent struct {
	CellID        string
	DeclaredSpan  int
	EffectiveSpan int
	Section       string
}

func (CrossSectionSpanEvent) warningTag() {}

func (e CrossSectionSpanEvent) String() string {
	return "rowspan crosses section boundary: cell id=" + quote(e.CellID)
}

// ContentSourceReplacedEvent is recorded when a cell is configured
// with both WithContent and WithReader. The later option wins; the
// earlier source is discarded. FinalSource is "content" or "reader"
// depending on which option was applied last.
type ContentSourceReplacedEvent struct {
	CellID      string
	FinalSource string
}

func (ContentSourceReplacedEvent) warningTag() {}

func (e ContentSourceReplacedEvent) String() string {
	return "content source replaced on cell id=" + quote(e.CellID) +
		" (final source: " + e.FinalSource + ")"
}

// DuplicateIDEvent is recorded when an element tries to register an
// ID already claimed by a different element. The first element
// keeps its registration; the second element's ID is cleared so
// Table.GetElementByID cannot return ambiguous results.
type DuplicateIDEvent struct {
	ID   string
	Kind string // "cell", "row", "header", "footer", "column", "table"
}

func (DuplicateIDEvent) warningTag() {}

func (e DuplicateIDEvent) String() string {
	return "duplicate id: " + quote(e.ID) + " (kind: " + e.Kind + ")"
}

func quote(s string) string {
	if s == "" {
		return `""`
	}
	return `"` + s + `"`
}
