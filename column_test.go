// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"strings"
	"testing"
)

func TestColumnDefaults(t *testing.T) {
	tbl := NewTable()
	col := tbl.Column(0)
	if col.Weight() != 1.0 {
		t.Errorf("default weight = %v, want 1.0", col.Weight())
	}
	if col.Width() != 0 || col.Min() != 0 || col.Max() != 0 {
		t.Errorf("defaults width/min/max = %d/%d/%d, want 0/0/0",
			col.Width(), col.Min(), col.Max())
	}
	if col.HasAlign() {
		t.Error("HasAlign should default to false")
	}
}

func TestColumnSettersChain(t *testing.T) {
	tbl := NewTable()
	col := tbl.Column(0).
		SetMin(5).
		SetMax(20).
		SetWeight(2.5).
		SetAlign(AlignRight)
	if col.Min() != 5 || col.Max() != 20 {
		t.Errorf("min/max = %d/%d, want 5/20", col.Min(), col.Max())
	}
	if col.Weight() != 2.5 {
		t.Errorf("weight = %v, want 2.5", col.Weight())
	}
	if !col.HasAlign() || col.Align() != AlignRight {
		t.Errorf("align = %v (set=%v), want AlignRight (set=true)", col.Align(), col.HasAlign())
	}
}

func TestColumnSettersClearOnNonPositive(t *testing.T) {
	tbl := NewTable()
	col := tbl.Column(0).SetWidth(10).SetMin(3).SetMax(20)
	col.SetWidth(0).SetMin(-1).SetMax(0)
	if col.Width() != 0 || col.Min() != 0 || col.Max() != 0 {
		t.Errorf("after clear: width=%d min=%d max=%d, want 0/0/0",
			col.Width(), col.Min(), col.Max())
	}
}

func TestLayoutExplicitWidthPins(t *testing.T) {
	tbl := NewTable(WithTargetWidth(40))
	tbl.Column(1).SetWidth(8)

	r := tbl.AddRow()
	r.AddCell(WithContent("alpha"))
	r.AddCell(WithContent("x"))
	r.AddCell(WithContent("gamma"))

	l := Layout(tbl, Measure(tbl))
	if l.err != nil {
		t.Fatalf("unexpected err: %v", l.err)
	}
	if l.colAssigned[1] != 8 {
		t.Errorf("col 1 = %d, want 8 (pinned)", l.colAssigned[1])
	}
}

func TestLayoutMaxCapsRedistributesLeftover(t *testing.T) {
	tbl := NewTable(WithTargetWidth(40))
	tbl.Column(1).SetMax(6)

	r := tbl.AddRow()
	r.AddCell(WithContent("a"))
	r.AddCell(WithContent("b"))
	r.AddCell(WithContent("c"))

	l := Layout(tbl, Measure(tbl))
	if l.colAssigned[1] > 6 {
		t.Errorf("col 1 = %d, want <= 6 (capped)", l.colAssigned[1])
	}
	// Others absorb the leftover.
	var sum int
	for _, v := range l.colAssigned {
		sum += v
	}
	overhead := (3 + 1) + 3*2
	if sum != 40-overhead {
		t.Errorf("sum = %d, want %d (full budget)", sum, 40-overhead)
	}
}

func TestLayoutUserMinHonored(t *testing.T) {
	tbl := NewTable(WithTargetWidth(40))
	tbl.Column(0).SetMin(15)

	r := tbl.AddRow()
	r.AddCell(WithContent("a")) // content min = 1
	r.AddCell(WithContent("b"))
	r.AddCell(WithContent("c"))

	l := Layout(tbl, Measure(tbl))
	if l.colAssigned[0] < 15 {
		t.Errorf("col 0 = %d, want >= 15 (user min)", l.colAssigned[0])
	}
}

func TestLayoutWeightsProportional(t *testing.T) {
	tbl := NewTable(WithTargetWidth(40))
	tbl.Column(1).SetWeight(3)

	r := tbl.AddRow()
	r.AddCell(WithContent("a"))
	r.AddCell(WithContent("b"))
	r.AddCell(WithContent("c"))

	l := Layout(tbl, Measure(tbl))
	// Col 1 should be noticeably wider than cols 0 and 2.
	if l.colAssigned[1] <= l.colAssigned[0] ||
		l.colAssigned[1] <= l.colAssigned[2] {
		t.Errorf("weight=3 col not widest: %v", l.colAssigned)
	}
	// Roughly 3x the share of the others (after each gets its min).
	// With content mins all 1 and overhead=10, available=30, min sum=3,
	// flex remaining=27, weights [1, 3, 1] → cols get ~5, ~16, ~5 plus
	// their mins → [6, 17, 6]. Approximate check:
	if l.colAssigned[1] < 14 {
		t.Errorf("col 1 = %d, expected ~16 for weight 3", l.colAssigned[1])
	}
}

func TestLayoutWeightZeroDoesNotGrow(t *testing.T) {
	tbl := NewTable(WithTargetWidth(40))
	tbl.Column(0).SetWeight(0) // pinned near content min

	r := tbl.AddRow()
	r.AddCell(WithContent("abc")) // content min = 3
	r.AddCell(WithContent("b"))
	r.AddCell(WithContent("c"))

	l := Layout(tbl, Measure(tbl))
	if l.colAssigned[0] != 3 {
		t.Errorf("col 0 = %d, want 3 (weight=0 pins to content min)", l.colAssigned[0])
	}
}

func TestLayoutExplicitWidthBelowContentMinOverflows(t *testing.T) {
	tbl := NewTable(WithTargetWidth(40))
	tbl.Column(0).SetWidth(3) // below content min of "widecontent" = 11

	r := tbl.AddRow()
	r.AddCell(WithContent("widecontent"))
	r.AddCell(WithContent("b"))

	l := Layout(tbl, Measure(tbl))
	// Content min wins; col 0 overflows the requested pin.
	if l.colAssigned[0] < 11 {
		t.Errorf("col 0 = %d, want >= 11 (content min wins over pinned 3)",
			l.colAssigned[0])
	}
}

func TestColumnAlignmentCascadesToCells(t *testing.T) {
	tbl := NewTable(WithTargetWidth(30))
	tbl.Column(1).SetAlign(AlignCenter)

	r := tbl.AddRow()
	r.AddCell(WithContent("L"))                        // no align: defaults left
	r.AddCell(WithContent("X"))                        // no align: inherits column center
	r.AddCell(WithContent("R"), WithAlign(AlignRight)) // explicit: right

	out := tbl.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	content := lines[1]
	// 3 cols at target=30: overhead=10, available=20, split [7,7,6] → cell
	// areas 9, 9, 8.
	//   col 0 (left default):  " L       "
	//   col 1 (inherits center): "    X    "
	//   col 2 (explicit right):  "      R "
	want := "│ L       │    X    │      R │"
	if content != want {
		t.Errorf("got:\n%q\nwant:\n%q", content, want)
	}
}

func TestCellAlignmentOverridesColumn(t *testing.T) {
	tbl := NewTable(WithTargetWidth(30))
	tbl.Column(0).SetAlign(AlignRight)

	r := tbl.AddRow()
	// Cell sets its own alignment — column is overridden.
	r.AddCell(WithContent("X"), WithAlign(AlignLeft))
	r.AddCell(WithContent("y"))
	r.AddCell(WithContent("z"))

	out := tbl.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	content := lines[1]
	// Col 0 cell is explicitly left-aligned: " X       " (not right-aligned).
	if !strings.HasPrefix(content, "│ X       │") {
		t.Errorf("cell should override column align-right to left; got %q", content)
	}
}

func TestLayoutAllowsWidthLeftoverBelowTarget(t *testing.T) {
	// Cap all columns; solver can't use the full budget. Output should
	// be narrower than target.
	tbl := NewTable(WithTargetWidth(100))
	tbl.Column(0).SetMax(5)
	tbl.Column(1).SetMax(5)
	tbl.Column(2).SetMax(5)

	r := tbl.AddRow()
	r.AddCell(WithContent("a"))
	r.AddCell(WithContent("b"))
	r.AddCell(WithContent("c"))

	l := Layout(tbl, Measure(tbl))
	for i, v := range l.colAssigned {
		if v > 5 {
			t.Errorf("col %d = %d, want <= 5 (capped)", i, v)
		}
	}
	// Total assigned should equal sum of caps since weights consumed
	// them all.
	var sum int
	for _, v := range l.colAssigned {
		sum += v
	}
	if sum > 15 {
		t.Errorf("sum = %d, want <= 15", sum)
	}
}
