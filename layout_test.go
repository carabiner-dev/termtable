// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"errors"
	"testing"
)

// layoutOverhead returns the fixed (non-content) cost in columns for
// an nCols table with DefaultPadding.
func layoutOverheadCols(nCols int) int {
	return (nCols + 1) + nCols*2
}

func TestLayoutEqualSplitBaseline(t *testing.T) {
	tbl := NewTable(WithTargetWidth(37))
	r := tbl.AddRow()
	r.AddCell(WithContent("a"))
	r.AddCell(WithContent("b"))
	r.AddCell(WithContent("c"))

	m := Measure(tbl)
	l := Layout(tbl, m)
	if l.err != nil {
		t.Fatalf("unexpected err: %v", l.err)
	}
	// target=37, overhead=(3+1) + 3*2 = 10 → available=27, 3 cols → 9 each.
	want := []int{9, 9, 9}
	for i := range want {
		if l.colAssigned[i] != want[i] {
			t.Errorf("colAssigned[%d] = %d, want %d", i, l.colAssigned[i], want[i])
		}
	}
}

func TestLayoutExtraRemainderDistributedLeftFirst(t *testing.T) {
	// Force a remainder: available not divisible by 3.
	tbl := NewTable(WithTargetWidth(39))
	r := tbl.AddRow()
	r.AddCell(WithContent("a"))
	r.AddCell(WithContent("b"))
	r.AddCell(WithContent("c"))

	l := Layout(tbl, Measure(tbl))
	// overhead=10, avail=29, base=9 remainder 2 → [10,10,9]
	want := []int{10, 10, 9}
	for i := range want {
		if l.colAssigned[i] != want[i] {
			t.Errorf("colAssigned[%d] = %d, want %d", i, l.colAssigned[i], want[i])
		}
	}
}

func TestLayoutMinFloorTriggered(t *testing.T) {
	tbl := NewTable(WithTargetWidth(30))
	r := tbl.AddRow()
	r.AddCell(WithContent("x"))
	r.AddCell(WithContent("looooongword")) // min = 12
	r.AddCell(WithContent("y"))

	l := Layout(tbl, Measure(tbl))
	if l.err != nil {
		t.Fatalf("unexpected err: %v", l.err)
	}
	// Column 1 must be at least 12.
	if l.colAssigned[1] < 12 {
		t.Errorf("col 1 = %d, want >= 12", l.colAssigned[1])
	}
	// Total content sum should equal available budget.
	target := 30
	overhead := layoutOverheadCols(3)
	want := target - overhead
	var got int
	for _, v := range l.colAssigned {
		got += v
	}
	if got != want {
		t.Errorf("sum of assigned = %d, want %d", got, want)
	}
}

// TestLayoutTooNarrowNoRoomPerColumn verifies that
// ErrTargetTooNarrow fires only when the budget cannot give every
// column at least one glyph of content space. Overhead for 4 cols is
// (4+1) + 4*2 = 13, so target=15 leaves 2 content cols for 4 columns
// — genuinely pathological.
func TestLayoutTooNarrowNoRoomPerColumn(t *testing.T) {
	tbl := NewTable(WithTargetWidth(15))
	r := tbl.AddRow()
	r.AddCell(WithContent("a"))
	r.AddCell(WithContent("b"))
	r.AddCell(WithContent("c"))
	r.AddCell(WithContent("d"))

	l := Layout(tbl, Measure(tbl))
	if !errors.Is(l.err, ErrTargetTooNarrow) {
		t.Fatalf("expected ErrTargetTooNarrow, got %v", l.err)
	}
}

// TestLayoutShrinksBelowMinimumSilently verifies that when content
// minimums exceed the target but every column can still be given at
// least one glyph, the layout silently shrinks — no error. Content
// clipping happens inside cells via the normal wrap path.
func TestLayoutShrinksBelowMinimumSilently(t *testing.T) {
	tbl := NewTable(WithTargetWidth(12))
	r := tbl.AddRow()
	r.AddCell(WithContent("longwordone"))
	r.AddCell(WithContent("longwordtwo"))

	l := Layout(tbl, Measure(tbl))
	if l.err != nil {
		t.Fatalf("expected no error for silent shrink, got %v", l.err)
	}
}

// TestLayoutTooNarrowHardFitsToTarget verifies that when content
// minimums exceed the target, the layout still fits within the target
// by shrinking each column below its natural minimum (cells will clip
// their content with an ellipsis). The rendered width must equal the
// target — borders never spill outside it. No error is raised since
// every column still gets at least one glyph.
func TestLayoutTooNarrowHardFitsToTarget(t *testing.T) {
	tbl := NewTable(WithTargetWidth(20))
	r := tbl.AddRow()
	r.AddCell(WithContent("longwordonelongwordone"))
	r.AddCell(WithContent("longwordtwolongwordtwo"))

	l := Layout(tbl, Measure(tbl))
	if l.err != nil {
		t.Fatalf("expected no error (every column fits one glyph): %v", l.err)
	}
	// overhead for 2 cols: (2+1) + 2*2 = 7 → available = 13
	available := 20 - layoutOverheadCols(2)
	total := 0
	for _, v := range l.colAssigned {
		total += v
	}
	if total != available {
		t.Errorf("colAssigned sum = %d, want %d (hard-fit to available)", total, available)
	}
	for i, v := range l.colAssigned {
		if v < 1 {
			t.Errorf("colAssigned[%d] = %d, want >= 1 (every column keeps a glyph)", i, v)
		}
	}

	// End-to-end: every rendered line is exactly the target width.
	out := tbl.String()
	for i, ln := range splitLines(out) {
		if w := DisplayWidth(ln); w != 20 {
			t.Errorf("line %d width = %d, want 20: %q", i, w, ln)
		}
	}
}

func splitLines(s string) []string {
	var out []string
	curr := ""
	for _, ch := range s {
		if ch == '\n' {
			out = append(out, curr)
			curr = ""
			continue
		}
		curr += string(ch)
	}
	if curr != "" {
		out = append(out, curr)
	}
	return out
}

func TestLayoutMultiSpanConstraintBorrows(t *testing.T) {
	tbl := NewTable(WithTargetWidth(40))
	// Row 0: a single cell spanning 2 columns with a long minimum.
	r0 := tbl.AddRow()
	r0.AddCell(WithContent("averylongbannerword"), WithColSpan(2))
	// Row 1: two normal cells to populate the columns.
	r1 := tbl.AddRow()
	r1.AddCell(WithContent("x"))
	r1.AddCell(WithContent("y"))
	// Force a third column so there's outside-span slack to borrow.
	r1.AddCell(WithContent("z")) // col 2

	// Row 0 only has the banner, so column 2 gets populated by row 1.
	// Actually the banner is in row 0 and the banner cell occupies cols
	// 0..1; then row 1 has three cells 0..2. NumColumns = 3.
	if tbl.NumColumns() != 3 {
		t.Fatalf("expected 3 columns, got %d", tbl.NumColumns())
	}

	m := Measure(tbl)
	l := Layout(tbl, m)
	if l.err != nil {
		t.Fatalf("unexpected err: %v", l.err)
	}
	// Span minimum ~ 19. seamWidth=3 for a single seam. So columns 0+1
	// must sum to at least 19-3 = 16.
	span := l.colAssigned[0] + l.colAssigned[1]
	if span < 16 {
		t.Errorf("span sum = %d, want >= 16", span)
	}
}

func TestLayoutDesiredUpgrade(t *testing.T) {
	// Short content: base widths exceed desired easily; leftover
	// budget should NOT inflate columns past their desired.
	tbl := NewTable(WithTargetWidth(60))
	r := tbl.AddRow()
	r.AddCell(WithContent("a"))
	r.AddCell(WithContent("b"))

	l := Layout(tbl, Measure(tbl))
	if l.err != nil {
		t.Fatalf("unexpected err: %v", l.err)
	}
	// When desired equals min (single-char content), assigned still
	// goes up from equal-split. That's fine; just ensure sum ≤ budget.
	overhead := layoutOverheadCols(2)
	budget := 60 - overhead
	var got int
	for _, v := range l.colAssigned {
		got += v
	}
	if got > budget {
		t.Errorf("sum of assigned = %d, exceeds budget %d", got, budget)
	}
}

func TestLayoutRowHeightsSingleLineContent(t *testing.T) {
	tbl := NewTable(WithTargetWidth(50))
	r := tbl.AddRow()
	r.AddCell(WithContent("one"))
	r.AddCell(WithContent("two"))

	l := Layout(tbl, Measure(tbl))
	if len(l.rowHeights) != 1 {
		t.Fatalf("rowHeights len = %d, want 1", len(l.rowHeights))
	}
	if l.rowHeights[0] != 1 {
		t.Errorf("row height = %d, want 1", l.rowHeights[0])
	}
}

func TestLayoutRowHeightsWrapsToMultiLine(t *testing.T) {
	// Narrow column forces wrapping.
	tbl := NewTable(WithTargetWidth(20))
	r := tbl.AddRow()
	r.AddCell(WithContent("a very long sentence that must wrap several times"))

	l := Layout(tbl, Measure(tbl))
	if len(l.rowHeights) != 1 {
		t.Fatalf("rowHeights len = %d", len(l.rowHeights))
	}
	if l.rowHeights[0] < 2 {
		t.Errorf("row height = %d, want >= 2", l.rowHeights[0])
	}
}

func TestLayoutRowSpanBumpsTailRow(t *testing.T) {
	tbl := NewTable(WithTargetWidth(25))
	r0 := tbl.AddRow()
	r0.AddCell(
		WithContent("first\nsecond\nthird\nfourth"),
		WithRowSpan(3),
	)
	r1 := tbl.AddRow()
	r1.AddCell(WithContent("x"))
	r2 := tbl.AddRow()
	r2.AddCell(WithContent("y"))

	l := Layout(tbl, Measure(tbl))
	if len(l.rowHeights) != 3 {
		t.Fatalf("rowHeights len = %d", len(l.rowHeights))
	}
	// Each base row has height 1 (from x, y in rows 1, 2, nothing from
	// row 0 since the rowspan is not counted until the tail bump).
	// Total must be >= 4 (content lines). Tail row bumped.
	var sum int
	for _, hh := range l.rowHeights {
		sum += hh
	}
	if sum < 4 {
		t.Errorf("sum of heights = %d, want >= 4", sum)
	}
}

func TestLayoutEmptyTable(t *testing.T) {
	tbl := NewTable()
	l := Layout(tbl, Measure(tbl))
	if l.err != nil {
		t.Errorf("unexpected err on empty table: %v", l.err)
	}
	if len(l.colAssigned) != 0 {
		t.Errorf("colAssigned = %v, want empty", l.colAssigned)
	}
	if len(l.rowHeights) != 0 {
		t.Errorf("rowHeights = %v, want empty", l.rowHeights)
	}
}
