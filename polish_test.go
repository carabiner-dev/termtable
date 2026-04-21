// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------
// Tier 1: Cell.GridRow returns absolute row
// ---------------------------------------------------------------------

func TestCellGridRowAbsoluteAcrossSections(t *testing.T) {
	h := th{t}
	tbl := NewTable()
	hd := h.header(tbl.AddHeader())
	hc := h.cell(hd.AddCell(WithContent("H")))

	r := h.row(tbl.AddRow())
	rc := h.cell(r.AddCell(WithContent("B")))

	f := h.footer(tbl.AddFooter())
	fc := h.cell(f.AddCell(WithContent("F")))

	cases := []struct {
		c    *Cell
		want int
	}{
		{hc, 0}, // header row 0 → abs 0
		{rc, 1}, // body row 0 → abs 1 (after 1 header)
		{fc, 2}, // footer row 0 → abs 2 (after 1 header + 1 body)
	}
	for _, tc := range cases {
		if got := tc.c.GridRow(); got != tc.want {
			t.Errorf("cell %q GridRow = %d, want %d", tc.c.content, got, tc.want)
		}
	}
}

func TestCellGridRowDetachedCellFallsBack(t *testing.T) {
	c := NewCell(WithContent("detached"))
	// Detached cell has no table; GridRow should return the
	// section-local value (which is zero for an unattached cell).
	if got := c.GridRow(); got != 0 {
		t.Errorf("detached cell GridRow = %d, want 0", got)
	}
}

// ---------------------------------------------------------------------
// Tier 1: Warnings don't duplicate across renders
// ---------------------------------------------------------------------

func TestWarningsDoNotDuplicateAcrossRenders(t *testing.T) {
	h := th{t}
	// Scenario: multi-span cell wider than the target budget can
	// accommodate. Each render produces a SpanOverflowEvent.
	tbl := NewTable(WithTargetWidth(20))
	r := h.row(tbl.AddRow())
	h.cell(r.AddCell(WithCellID("wide"),
		WithContent("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		WithColSpan(2),
	))
	h.cell(r.AddCell(WithContent("x"))) // forces 2 cols

	_ = tbl.String()
	first := len(tbl.Warnings())
	_ = tbl.String()
	second := len(tbl.Warnings())
	if first != second {
		t.Errorf("warnings grew across renders: %d → %d", first, second)
	}
}

func TestAuthoringWarningsPersistAcrossRenders(t *testing.T) {
	h := th{t}
	tbl := NewTable(WithSpanOverwrite(true), WithTargetWidth(30))
	r0 := h.row(tbl.AddRow())
	r1 := h.row(tbl.AddRow())
	h.cell(r1.AddCell(WithCellID("victim"), WithContent("v")))
	h.cell(r0.AddCell(WithContent("over"), WithRowSpan(2)))

	before := len(tbl.Warnings())
	_ = tbl.String()
	_ = tbl.String()
	after := len(tbl.Warnings())
	// Authoring warnings (the overwrite event) must stick; render
	// warnings should stay at zero. Total count should equal the
	// original plus whatever render warnings a passing render
	// produces (zero in this case).
	if after != before {
		t.Errorf("authoring warnings changed after render: before=%d after=%d",
			before, after)
	}
}

// ---------------------------------------------------------------------
// Tier 1: Reader errors surfaced as warnings
// ---------------------------------------------------------------------

type boomReader struct{}

func (boomReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestReaderErrorSurfacedAsWarning(t *testing.T) {
	h := th{t}
	tbl := NewTable(WithTargetWidth(30))
	r := h.row(tbl.AddRow())
	h.cell(r.AddCell(WithCellID("broken"), WithReader(boomReader{})))
	h.cell(r.AddCell(WithContent("ok")))

	_ = tbl.String()
	var saw bool
	for _, w := range tbl.Warnings() {
		ev, ok := w.(ReaderErrorEvent)
		if ok && ev.CellID == "broken" {
			saw = true
		}
	}
	if !saw {
		t.Errorf("expected ReaderErrorEvent, got %v", tbl.Warnings())
	}
}

// ---------------------------------------------------------------------
// Tier 1: Cross-section rowspan clamp + warning
// ---------------------------------------------------------------------

func TestCrossSectionRowSpanClampedAndWarned(t *testing.T) {
	h := th{t}
	tbl := NewTable(WithTargetWidth(40))
	// One header row, header cell with rowSpan=3 — would reach into
	// body territory. Must be clamped without panicking.
	hd := h.header(tbl.AddHeader())
	h.cell(hd.AddCell(WithCellID("overreach"),
		WithContent("banner"), WithRowSpan(3)))
	h.cell(hd.AddCell(WithContent("col2")))

	r := h.row(tbl.AddRow())
	h.cell(r.AddCell(WithContent("b1")))
	h.cell(r.AddCell(WithContent("b2")))

	// Just rendering would panic before the fix.
	_ = tbl.String()

	var saw bool
	for _, w := range tbl.Warnings() {
		ev, ok := w.(CrossSectionSpanEvent)
		if ok && ev.CellID == "overreach" {
			saw = true
			if ev.DeclaredSpan != 3 || ev.EffectiveSpan != 1 {
				t.Errorf("event spans: declared=%d effective=%d, want 3/1",
					ev.DeclaredSpan, ev.EffectiveSpan)
			}
			if ev.Section != "header" {
				t.Errorf("event section = %q, want %q", ev.Section, "header")
			}
		}
	}
	if !saw {
		t.Errorf("expected CrossSectionSpanEvent, got %v", tbl.Warnings())
	}
}

// ---------------------------------------------------------------------
// Tier 2: table-wide padding
// ---------------------------------------------------------------------

func TestWithTablePaddingChangesLayout(t *testing.T) {
	h := th{t}
	tbl := NewTable(
		WithTargetWidth(30),
		WithTablePadding(Padding{Left: 0, Right: 0}),
	)
	r := h.row(tbl.AddRow())
	h.cell(r.AddCell(WithContent("a")))
	h.cell(r.AddCell(WithContent("b")))

	out := tbl.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	// With zero padding, content areas abut the borders.
	// Overhead = nCols+1 = 3 borders. Content budget = 30 - 3 = 27.
	// Cells: 14, 13. Check first content line: │a(13 spaces)│b(12)│.
	if !strings.HasPrefix(lines[1], "│a") {
		t.Errorf("no-padding render should place content flush with border: %q", lines[1])
	}
}

// ---------------------------------------------------------------------
// Tier 2: Column.SetID + registry
// ---------------------------------------------------------------------

func TestColumnSetIDRegisters(t *testing.T) {
	tbl := NewTable()
	col := tbl.Column(0)
	if err := col.SetID("status"); err != nil {
		t.Fatalf("SetID: %v", err)
	}
	got := tbl.GetElementByID("status")
	if got != col {
		t.Errorf("GetElementByID = %v, want column", got)
	}
}

func TestColumnSetIDCollisionReturnsError(t *testing.T) {
	h := th{t}
	tbl := NewTable()
	r := h.row(tbl.AddRow(WithRowID("taken")))
	_ = r

	col := tbl.Column(0)
	err := col.SetID("taken")
	if !errors.Is(err, ErrDuplicateID) {
		t.Errorf("err = %v, want ErrDuplicateID", err)
	}
}

func TestColumnSetIDReassignUnregistersOld(t *testing.T) {
	tbl := NewTable()
	col := tbl.Column(0)
	if err := col.SetID("first"); err != nil {
		t.Fatal(err)
	}
	if err := col.SetID("second"); err != nil {
		t.Fatal(err)
	}
	if tbl.GetElementByID("first") != nil {
		t.Error("old id should be unregistered after reassign")
	}
	if tbl.GetElementByID("second") != col {
		t.Error("new id should resolve to the column")
	}
}

// ---------------------------------------------------------------------
// Tier 2: LastRenderError behavior
// ---------------------------------------------------------------------

func TestLastRenderErrorRoundTrips(t *testing.T) {
	h := th{t}
	tbl := NewTable(WithTargetWidth(5))
	r := h.row(tbl.AddRow())
	h.cell(r.AddCell(WithContent("longwordone")))
	h.cell(r.AddCell(WithContent("anothertoo")))

	// String() should capture the narrow-width error.
	_ = tbl.String()
	if !errors.Is(tbl.LastRenderError(), ErrTargetTooNarrow) {
		t.Fatalf("LastRenderError after narrow = %v", tbl.LastRenderError())
	}

	// Widen and re-render: error should clear.
	tbl.opts.targetWidth = 60
	tbl.opts.targetWidthSet = true
	_ = tbl.String()
	if tbl.LastRenderError() != nil {
		t.Errorf("LastRenderError after successful render = %v", tbl.LastRenderError())
	}
}

// ---------------------------------------------------------------------
// Tier 2: header-to-header rowspan
// ---------------------------------------------------------------------

func TestHeaderRowSpanAcrossMultipleHeaders(t *testing.T) {
	h := th{t}
	tbl := NewTable(WithTargetWidth(40))
	h1 := h.header(tbl.AddHeader())
	h.cell(h1.AddCell(WithContent("pivot"), WithRowSpan(2)))
	h.cell(h1.AddCell(WithContent("a")))

	h2 := h.header(tbl.AddHeader())
	// Column 0 is reserved by the rowspan; first cell auto-advances.
	h.cell(h2.AddCell(WithContent("b")))

	out := tbl.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	// Expect 3 border lines + 2 header content rows = 5 lines. The
	// horizontal border between h1 and h2 must be suppressed at col 0
	// (│ glyph, not ├).
	if len(lines) < 5 {
		t.Fatalf("expected at least 5 lines, got %d:\n%s", len(lines), out)
	}
	// The separator between h1 and h2 is lines[2]. Its first inner
	// junction (between cols 0 and 1) should be ┤ since the cell to
	// its left spans, the one to its right does not.
	if !strings.ContainsRune(lines[2], '┤') {
		t.Errorf("header-rowspan separator missing ┤ join: %q", lines[2])
	}
	// No CrossSectionSpanEvent should be emitted — the rowspan stays
	// within the header section.
	for _, w := range tbl.Warnings() {
		if _, ok := w.(CrossSectionSpanEvent); ok {
			t.Errorf("unexpected cross-section warning: %v", tbl.Warnings())
		}
	}
}

// ---------------------------------------------------------------------
// Tier 2: Custom BorderSet round-trips through WithBorder
// ---------------------------------------------------------------------

func TestCustomBorderSetUsedInRender(t *testing.T) {
	// Build an ASCII-only border set.
	var ascii BorderSet
	ascii.Horizontal = '-'
	ascii.Vertical = '|'
	ascii.Joins[armS|armE] = '+'
	ascii.Joins[armS|armW] = '+'
	ascii.Joins[armN|armE] = '+'
	ascii.Joins[armN|armW] = '+'
	ascii.Joins[armN|armS] = '|'
	ascii.Joins[armE|armW] = '-'
	ascii.Joins[armN|armS|armE] = '+'
	ascii.Joins[armN|armS|armW] = '+'
	ascii.Joins[armE|armS|armW] = '+'
	ascii.Joins[armN|armE|armW] = '+'
	ascii.Joins[armN|armE|armS|armW] = '+'

	h := th{t}
	tbl := NewTable(WithTargetWidth(15), WithBorder(ascii))
	r := h.row(tbl.AddRow())
	h.cell(r.AddCell(WithContent("a")))
	h.cell(r.AddCell(WithContent("b")))

	out := tbl.String()
	// Must not contain any unicode box glyphs.
	for _, bad := range []rune{'─', '│', '┌', '┐', '└', '┘', '┼'} {
		if strings.ContainsRune(out, bad) {
			t.Errorf("ascii border set leaked %q glyph: %q", bad, out)
		}
	}
	// Must contain the custom glyphs.
	for _, want := range []rune{'+', '-', '|'} {
		if !strings.ContainsRune(out, want) {
			t.Errorf("ascii border set missing %q: %q", want, out)
		}
	}
}

// ---------------------------------------------------------------------
// Tier 2: Reader content persists across multiple renders
// ---------------------------------------------------------------------

func TestReaderContentPersistsAcrossRenders(t *testing.T) {
	h := th{t}
	tbl := NewTable(WithTargetWidth(30))
	r := h.row(tbl.AddRow())
	c := h.cell(r.AddCell(WithCellID("lazy"),
		WithReader(strings.NewReader("content-from-reader")),
	))

	var buf bytes.Buffer
	if _, err := tbl.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	// Second render must not fail (reader is already drained on
	// first render) and must still show the content.
	buf.Reset()
	if _, err := tbl.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "content") {
		t.Errorf("second render dropped reader content:\n%s", buf.String())
	}
	if !c.resolved || !c.hasContent {
		t.Errorf("cell flags: resolved=%v hasContent=%v, want both true",
			c.resolved, c.hasContent)
	}
}

// ---------------------------------------------------------------------
// Tier 2: Empty-content cell renders cleanly
// ---------------------------------------------------------------------

func TestEmptyCellRendersBlank(t *testing.T) {
	h := th{t}
	tbl := NewTable(WithTargetWidth(20))
	r := h.row(tbl.AddRow())
	h.cell(r.AddCell(WithContent("")))
	h.cell(r.AddCell(WithContent("x")))

	out := tbl.String()
	// Must produce exactly 5 lines (top + 1 row + bottom = 3; no error
	// condition introduces extras). And grid alignment holds.
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Errorf("lines = %d, want 3:\n%s", len(lines), out)
	}
	target := DisplayWidth(lines[0])
	for i, ln := range lines {
		if DisplayWidth(ln) != target {
			t.Errorf("line %d width %d, want %d: %q",
				i, DisplayWidth(ln), target, ln)
		}
	}
}
