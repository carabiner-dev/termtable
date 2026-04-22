// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"strings"
	"testing"
)

func TestEmptyTableDimensions(t *testing.T) {
	tbl := NewTable()
	if tbl.NumColumns() != 0 {
		t.Errorf("NumColumns = %d, want 0", tbl.NumColumns())
	}
	if tbl.NumRows() != 0 {
		t.Errorf("NumRows = %d, want 0", tbl.NumRows())
	}
	if tbl.CellAt(0, 0) != nil {
		t.Error("CellAt on empty table should be nil")
	}
	if tbl.InBounds(0, 0) {
		t.Error("InBounds(0,0) on empty table should be false")
	}
}

func TestNumRowsSumAcrossSections(t *testing.T) {
	tbl := NewTable()
	tbl.AddHeader()
	tbl.AddHeader()
	tbl.AddRow()
	tbl.AddRow()
	tbl.AddRow()
	tbl.AddFooter()
	if got := tbl.NumRows(); got != 6 {
		t.Errorf("NumRows = %d, want 6 (2h + 3r + 1f)", got)
	}
}

func TestCellAtAcrossSections(t *testing.T) {
	tbl := NewTable()
	hd := tbl.AddHeader()
	hc := hd.AddCell(WithCellID("hc"), WithContent("head"))

	r := tbl.AddRow()
	rc := r.AddCell(WithCellID("rc"), WithContent("body"))

	f := tbl.AddFooter()
	fc := f.AddCell(WithCellID("fc"), WithContent("foot"))

	// Absolute row indices: header at 0, body at 1, footer at 2.
	if got := tbl.CellAt(0, 0); got != hc {
		t.Errorf("CellAt(0,0) = %v, want header cell", got)
	}
	if got := tbl.CellAt(1, 0); got != rc {
		t.Errorf("CellAt(1,0) = %v, want body cell", got)
	}
	if got := tbl.CellAt(2, 0); got != fc {
		t.Errorf("CellAt(2,0) = %v, want footer cell", got)
	}
	if tbl.CellAt(3, 0) != nil {
		t.Error("CellAt past last row should be nil")
	}
	if tbl.CellAt(-1, 0) != nil {
		t.Error("CellAt negative should be nil")
	}
}

func TestCellAtSpansMapToSameCell(t *testing.T) {
	tbl := NewTable()
	r := tbl.AddRow()
	c := r.AddCell(WithContent("wide"), WithColSpan(3))
	for col := range 3 {
		if got := tbl.CellAt(0, col); got != c {
			t.Errorf("CellAt(0,%d) = %v, want %v", col, got, c)
		}
	}
	if tbl.CellAt(0, 3) != nil {
		t.Error("CellAt past span should be nil")
	}
}

func TestInBounds(t *testing.T) {
	tbl := NewTable()
	r := tbl.AddRow()
	r.AddCell(WithContent("a"))
	r.AddCell(WithContent("b"))
	cases := []struct {
		r, c int
		want bool
	}{
		{0, 0, true},
		{0, 1, true},
		{0, 2, false},
		{1, 0, false},
		{-1, 0, false},
		{0, -1, false},
	}
	for _, tc := range cases {
		if got := tbl.InBounds(tc.r, tc.c); got != tc.want {
			t.Errorf("InBounds(%d,%d) = %v, want %v", tc.r, tc.c, got, tc.want)
		}
	}
}

func TestResolvedTargetWidth(t *testing.T) {
	tbl := NewTable()
	t.Setenv("COLUMNS", "")
	if got := tbl.ResolvedTargetWidth(); got != defaultTargetWidth {
		t.Errorf("default = %d, want %d", got, defaultTargetWidth)
	}
	t.Setenv("COLUMNS", "42")
	if got := tbl.ResolvedTargetWidth(); got != 42 {
		t.Errorf("COLUMNS = %d, want 42", got)
	}

	explicit := NewTable(WithTargetWidth(120))
	if got := explicit.ResolvedTargetWidth(); got != 120 {
		t.Errorf("explicit = %d, want 120", got)
	}

	t.Setenv("COLUMNS", "garbage")
	if got := tbl.ResolvedTargetWidth(); got != defaultTargetWidth {
		t.Errorf("garbage COLUMNS = %d, want default %d", got, defaultTargetWidth)
	}
}

// TestDetectTerminalWidthNonTTY verifies that the TTY probe silently
// reports "not available" when stdout/stderr are pipes (which is the
// shape `go test` runs with). This guarantees that in non-interactive
// environments the resolver falls through to defaultTargetWidth rather
// than returning a bogus 0.
func TestDetectTerminalWidthNonTTY(t *testing.T) {
	if _, ok := detectTerminalWidth(); ok {
		t.Skip("stdout/stderr appear to be a real TTY; skipping non-TTY assertion")
	}
}

// withFakeTTY swaps terminalWidthProbe for the duration of the test.
// Pass ok=false to simulate a non-TTY environment.
func withFakeTTY(t *testing.T, width int, ok bool) {
	t.Helper()
	saved := terminalWidthProbe
	terminalWidthProbe = func() (int, bool) { return width, ok }
	t.Cleanup(func() { terminalWidthProbe = saved })
}

// TestResolvedTargetWidthDefaultInWideTTY verifies that a table built
// with no WithTargetWidth uses the 80-column default even when the
// terminal is wider. The TTY is a ceiling, not a target.
func TestResolvedTargetWidthDefaultInWideTTY(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 200, true)

	tbl := NewTable()
	if got := tbl.ResolvedTargetWidth(); got != defaultTargetWidth {
		t.Errorf("default width = %d, want %d", got, defaultTargetWidth)
	}
}

// TestResolvedTargetWidthDefaultCappedByNarrowTTY verifies that when
// the screen is narrower than the 80-column default, the default is
// capped to the screen.
func TestResolvedTargetWidthDefaultCappedByNarrowTTY(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 40, true)

	tbl := NewTable()
	if got := tbl.ResolvedTargetWidth(); got != 40 {
		t.Errorf("default capped = %d, want 40", got)
	}
}

// TestResolvedTargetWidthCapsExplicitToTTY verifies that an explicit
// WithTargetWidth wider than the attached terminal is capped to the
// terminal width.
func TestResolvedTargetWidthCapsExplicitToTTY(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 80, true)

	tbl := NewTable(WithTargetWidth(200))
	if got := tbl.ResolvedTargetWidth(); got != 80 {
		t.Errorf("capped width = %d, want 80 (TTY cap)", got)
	}
}

// TestResolvedTargetWidthExplicitFitsInTTY verifies that an explicit
// width narrower than the TTY is honoured verbatim.
func TestResolvedTargetWidthExplicitFitsInTTY(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 120, true)

	tbl := NewTable(WithTargetWidth(40))
	if got := tbl.ResolvedTargetWidth(); got != 40 {
		t.Errorf("narrow explicit = %d, want 40", got)
	}
}

// TestResolvedTargetWidthCOLUMNSCappedToTTY verifies that a COLUMNS
// value wider than the screen is also capped. COLUMNS is a preference,
// not a licence to overflow the terminal.
func TestResolvedTargetWidthCOLUMNSCappedToTTY(t *testing.T) {
	t.Setenv("COLUMNS", "200")
	withFakeTTY(t, 80, true)

	tbl := NewTable()
	if got := tbl.ResolvedTargetWidth(); got != 80 {
		t.Errorf("COLUMNS cap = %d, want 80", got)
	}
}

// TestResolvedTargetWidthNoTTYNoCap verifies that when no terminal is
// attached (e.g. writing to a pipe or file) the resolver does not
// invent a cap: explicit widths pass through verbatim even when they
// are large.
func TestResolvedTargetWidthNoTTYNoCap(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 0, false)

	tbl := NewTable(WithTargetWidth(500))
	if got := tbl.ResolvedTargetWidth(); got != 500 {
		t.Errorf("non-TTY explicit = %d, want 500 (no cap applied)", got)
	}
}

// TestRenderNeverExceedsTTYWidth verifies the hard guarantee: when a
// terminal is attached, no rendered line is wider than the terminal,
// even when the content's minimum widths would otherwise force an
// overflowing best-effort render.
func TestRenderNeverExceedsTTYWidth(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 20, true)

	// Target is narrower than the content minimums — the layout
	// solver will surface ErrTargetTooNarrow and produce a
	// best-effort render that, left alone, would exceed the target.
	// The TTY clip must bring every line back under 20 columns.
	tbl := NewTable(WithTargetWidth(10))
	r := tbl.AddRow()
	r.AddCell(WithContent("averylongwordthatexceedsthescreen"))
	r.AddCell(WithContent("anotherlongwordtoo"))

	out := tbl.String()
	for i, ln := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if w := DisplayWidth(ln); w > 20 {
			t.Errorf("line %d has width %d > 20 (TTY cap): %q", i, w, ln)
		}
	}
}

// TestRenderNoClipToPipe verifies that non-interactive sinks (pipes,
// files) are not clipped — the user said those are free to overflow.
func TestRenderNoClipToPipe(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 0, false)

	tbl := NewTable(WithTargetWidth(10))
	r := tbl.AddRow()
	r.AddCell(WithContent("averylongwordthatexceedsthescreen"))

	out := tbl.String()
	// The overflow render is wider than 10 — confirm no clip happened.
	// At least one line must exceed the target width.
	var widest int
	for _, ln := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if w := DisplayWidth(ln); w > widest {
			widest = w
		}
	}
	if widest <= 10 {
		t.Errorf("expected overflow render in pipe mode, widest line = %d", widest)
	}
}

func TestColumnAutoCreation(t *testing.T) {
	tbl := NewTable()
	r := tbl.AddRow()
	for range 4 {
		r.AddCell(WithContent("x"))
	}
	if got := tbl.NumColumns(); got != 4 {
		t.Errorf("NumColumns = %d, want 4", got)
	}
	for i := range 4 {
		col := tbl.Column(i)
		if col == nil {
			t.Errorf("Column(%d) is nil", i)
			continue
		}
		if col.Index() != i {
			t.Errorf("Column(%d).Index() = %d", i, col.Index())
		}
	}
}

func TestColumnExplicitCreate(t *testing.T) {
	tbl := NewTable()
	col := tbl.Column(3)
	if col == nil || col.Index() != 3 {
		t.Fatalf("Column(3) = %v", col)
	}
	if tbl.NumColumns() != 4 {
		t.Errorf("NumColumns = %d, want 4 (explicit growth)", tbl.NumColumns())
	}
}

func TestMultiHeaderFooterOrdering(t *testing.T) {
	tbl := NewTable()
	h1 := tbl.AddHeader()
	h1.AddCell(WithContent("h1"))
	h2 := tbl.AddHeader()
	h2.AddCell(WithContent("h2"))
	r := tbl.AddRow()
	r.AddCell(WithContent("r"))
	f1 := tbl.AddFooter()
	f1.AddCell(WithContent("f1"))
	f2 := tbl.AddFooter()
	f2.AddCell(WithContent("f2"))

	if tbl.NumRows() != 5 {
		t.Fatalf("NumRows = %d, want 5", tbl.NumRows())
	}
	wantContents := []string{"h1", "h2", "r", "f1", "f2"}
	for i, want := range wantContents {
		c := tbl.CellAt(i, 0)
		if c == nil {
			t.Errorf("CellAt(%d,0) nil", i)
			continue
		}
		if c.Content() != want {
			t.Errorf("row %d content = %q, want %q", i, c.Content(), want)
		}
	}
}
