// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"errors"
	"testing"
)

// TestColSpanClaimsMultipleColumns verifies that a cell with colSpan=3
// causes the table to grow to 3 columns and occupies positions 0..2 in
// its row.
func TestColSpanClaimsMultipleColumns(t *testing.T) {
	h := th{t}
	tbl := NewTable()
	r := h.row(tbl.AddRow())
	c := h.cell(r.AddCell(WithContent("wide"), WithColSpan(3)))
	if tbl.NumColumns() != 3 {
		t.Errorf("NumColumns = %d, want 3", tbl.NumColumns())
	}
	for col := range 3 {
		if got := tbl.bodyOcc.at(0, col); got != c {
			t.Errorf("occ[0][%d] = %v, want %v", col, got, c)
		}
	}
}

// TestAddCellAdvancesPastReservedRowspan verifies that when row 0 has a
// rowspan=2 cell in column 0, the next row's first AddCell lands at
// column 1 (column 0 is reserved by the rowspan).
func TestAddCellAdvancesPastReservedRowspan(t *testing.T) {
	h := th{t}
	tbl := NewTable()
	r0 := h.row(tbl.AddRow())
	h.cell(r0.AddCell(WithContent("tall"), WithRowSpan(2)))
	side := h.cell(r0.AddCell(WithContent("side")))
	if side.GridCol() != 1 {
		t.Errorf("side grid col = %d, want 1", side.GridCol())
	}

	r1 := h.row(tbl.AddRow())
	first := h.cell(r1.AddCell(WithContent("below")))
	if first.GridCol() != 1 {
		t.Errorf("below grid col = %d, want 1 (col 0 reserved)", first.GridCol())
	}
}

// TestSpanConflictErrors verifies that a rowspan reaching into a row
// that already has content in the overlapping column triggers
// ErrSpanConflict.
func TestSpanConflictErrors(t *testing.T) {
	h := th{t}
	tbl := NewTable()
	r0 := h.row(tbl.AddRow())
	r1 := h.row(tbl.AddRow())
	h.cell(r1.AddCell(WithContent("below"), WithCellID("below")))
	// r0 tries to place a rowspan=2 cell at col 0 — but r1[0] is taken.
	_, err := r0.AddCell(WithContent("reach"), WithRowSpan(2))
	if !errors.Is(err, ErrSpanConflict) {
		t.Fatalf("expected ErrSpanConflict, got %v", err)
	}
	if err.Error() == "" {
		t.Error("error message empty")
	}
}

// TestSpanConflictAutoAdvanceWithinRow confirms that within a single
// row, new cells auto-advance past occupied slots rather than erroring.
func TestSpanConflictAutoAdvanceWithinRow(t *testing.T) {
	h := th{t}
	tbl := NewTable()
	r0 := h.row(tbl.AddRow())
	h.cell(r0.AddCell(WithContent("a"), WithColSpan(3)))
	c := h.cell(r0.AddCell(WithContent("b")))
	if c.GridCol() != 3 {
		t.Errorf("advanced col = %d, want 3", c.GridCol())
	}
}

// TestSpanOverwriteDropsAnchorCovered verifies that with
// WithSpanOverwrite(true), a new cell whose rectangle fully covers an
// existing cell's anchor removes the victim entirely.
func TestSpanOverwriteDropsAnchorCovered(t *testing.T) {
	h := th{t}
	tbl := NewTable(WithSpanOverwrite(true))
	r0 := h.row(tbl.AddRow())
	r1 := h.row(tbl.AddRow())
	h.cell(r1.AddCell(WithCellID("victim"), WithContent("v")))
	// r0 cell with rowspan=2 anchored at col 0 — covers (r1, c0),
	// which is the victim's anchor.
	h.cell(r0.AddCell(WithContent("over"), WithRowSpan(2)))
	if got := len(r1.Cells()); got != 0 {
		t.Errorf("victim remained in row, cells=%d", got)
	}
	if tbl.GetElementByID("victim") != nil {
		t.Error("victim id should be unregistered")
	}
	var sawDrop bool
	for _, w := range tbl.Warnings() {
		ev, ok := w.(OverwriteEvent)
		if ok && ev.DroppedID == "victim" {
			sawDrop = true
		}
	}
	if !sawDrop {
		t.Errorf("expected OverwriteEvent{DroppedID: victim}, got %v", tbl.Warnings())
	}
}

// TestSpanOverwriteTruncatesPartial verifies that with overwrite on, a
// new cell whose rectangle overlaps an existing cell's span without
// covering the victim's anchor reduces (truncates) the victim's span
// rather than dropping it.
//
// Scenario:
//
//	r0: [A] . .
//	r1: [B] . .
//	r2: [V V] .       (vic: (2, 0..1), rowSpan=2 — reserves (2..3, 0..1))
//
// Then we attach D to r0 with colSpan=2, rowSpan=3. Auto-advance pushes
// D to col 1 (col 0 is taken by A). D's rectangle becomes (0..2, 1..2).
// That intersects vic at (2, 1). Vic's anchor (2, 0) lies outside D's
// rectangle, so truncation applies: vic's colSpan drops from 2 to 1.
func TestSpanOverwriteTruncatesPartial(t *testing.T) {
	h := th{t}
	tbl := NewTable(WithSpanOverwrite(true))
	r0 := h.row(tbl.AddRow())
	h.cell(r0.AddCell(WithContent("A")))
	r1 := h.row(tbl.AddRow())
	h.cell(r1.AddCell(WithContent("B")))
	r2 := h.row(tbl.AddRow())
	vic := h.cell(r2.AddCell(
		WithCellID("vic"), WithContent("V"),
		WithColSpan(2), WithRowSpan(2),
	))

	h.cell(r0.AddCell(WithContent("D"), WithColSpan(2), WithRowSpan(3)))

	if vic.ColSpan() != 1 {
		t.Errorf("vic colSpan = %d, want 1", vic.ColSpan())
	}
	if vic.RowSpan() != 2 {
		t.Errorf("vic rowSpan = %d, want 2", vic.RowSpan())
	}
	if tbl.bodyOcc.at(2, 0) != vic {
		t.Error("vic should still be at (2,0)")
	}
	if tbl.bodyOcc.at(2, 1) == vic {
		t.Error("vic should no longer occupy (2,1)")
	}

	var sawTrunc bool
	for _, w := range tbl.Warnings() {
		ev, ok := w.(OverwriteEvent)
		if ok && ev.TruncatedID == "vic" {
			sawTrunc = true
		}
	}
	if !sawTrunc {
		t.Errorf("expected OverwriteEvent{TruncatedID: vic}, got %v", tbl.Warnings())
	}
}

// TestRowSpanReservesAcrossRows verifies the occupancy grid grows to
// cover a rowspan even when subsequent rows haven't been added yet.
func TestRowSpanReservesAcrossRows(t *testing.T) {
	h := th{t}
	tbl := NewTable()
	r0 := h.row(tbl.AddRow())
	c := h.cell(r0.AddCell(WithContent("tall"), WithRowSpan(3)))
	if tbl.bodyOcc.at(2, 0) != c {
		t.Error("rowspan should reserve row 2 col 0")
	}
	if tbl.bodyOcc.numRows() < 3 {
		t.Errorf("occ numRows = %d, want >= 3", tbl.bodyOcc.numRows())
	}
}
