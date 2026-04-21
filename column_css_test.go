// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"strings"
	"testing"
)

func TestColumnStyleSizeProperties(t *testing.T) {
	tbl := NewTable()
	tbl.Column(0).Style("width: 12")
	if tbl.Column(0).Width() != 12 {
		t.Errorf("width = %d, want 12", tbl.Column(0).Width())
	}

	tbl.Column(1).Style("min-width: 5; max-width: 20")
	if tbl.Column(1).Min() != 5 {
		t.Errorf("min = %d, want 5", tbl.Column(1).Min())
	}
	if tbl.Column(1).Max() != 20 {
		t.Errorf("max = %d, want 20", tbl.Column(1).Max())
	}

	tbl.Column(2).Style("flex: 2.5")
	if tbl.Column(2).Weight() != 2.5 {
		t.Errorf("weight = %v, want 2.5", tbl.Column(2).Weight())
	}
}

func TestColumnStyleTextAlignEquivalentToSetAlign(t *testing.T) {
	tbl := NewTable()
	tbl.Column(0).Style("text-align: right")
	if !tbl.Column(0).HasAlign() {
		t.Fatal("HasAlign should be true")
	}
	if tbl.Column(0).Align() != AlignRight {
		t.Errorf("align = %v, want AlignRight", tbl.Column(0).Align())
	}

	// Direct SetAlign should be observed by HasAlign() / Align() the
	// same way, confirming the two paths share storage.
	tbl.Column(1).SetAlign(AlignCenter)
	if !tbl.Column(1).HasAlign() || tbl.Column(1).Align() != AlignCenter {
		t.Errorf("SetAlign route: got %v (has=%v)",
			tbl.Column(1).Align(), tbl.Column(1).HasAlign())
	}
}

func TestColumnStyleMalformedIgnored(t *testing.T) {
	tbl := NewTable()
	tbl.Column(0).Style("width: abc; min-width: ; nonsense; max-width: 8")
	if tbl.Column(0).Width() != 0 {
		t.Errorf("width should be unset (abc invalid), got %d", tbl.Column(0).Width())
	}
	if tbl.Column(0).Min() != 0 {
		t.Errorf("min should be unset (empty value), got %d", tbl.Column(0).Min())
	}
	if tbl.Column(0).Max() != 8 {
		t.Errorf("max = %d, want 8 (valid decl after garbage)", tbl.Column(0).Max())
	}
}

func TestColumnStyleColorCascadesToCells(t *testing.T) {
	forceColor(t)
	h := th{t}
	tbl := NewTable(WithTargetWidth(30))
	tbl.Column(1).Style("color: red")

	r := h.row(tbl.AddRow())
	h.cell(r.AddCell(WithContent("a")))
	h.cell(r.AddCell(WithContent("b"))) // should inherit red
	h.cell(r.AddCell(WithContent("c")))

	out := tbl.String()
	// Red = 31. Must appear at least once (for col 1's cell).
	if !strings.Contains(out, "\x1b[31") {
		t.Errorf("expected red SGR in output, got:\n%q", out)
	}
}

func TestCascadeRowOverridesColumnOverridesTable(t *testing.T) {
	forceColor(t)
	h := th{t}
	tbl := NewTable(
		WithTargetWidth(30),
		WithTableStyle("color: blue"),
	)
	tbl.Column(0).Style("color: green") // overrides table-blue for col 0

	hdr := h.header(tbl.AddHeader(WithRowStyle("color: red"))) // row overrides col+table
	h.cell(hdr.AddCell(WithContent("H1")))
	h.cell(hdr.AddCell(WithContent("H2")))

	body := h.row(tbl.AddRow())
	h.cell(body.AddCell(WithContent("B1"))) // col 0 cell: inherits col green
	h.cell(body.AddCell(WithContent("B2"))) // col 1 cell: inherits table blue

	out := tbl.String()
	// All three codes must appear somewhere.
	if !strings.Contains(out, "\x1b[31") {
		t.Error("row red (31) missing")
	}
	if !strings.Contains(out, "\x1b[32") {
		t.Error("column green (32) missing")
	}
	if !strings.Contains(out, "\x1b[34") {
		t.Error("table blue (34) missing")
	}
}

func TestCellAlignmentBeatsRowBeatsColumn(t *testing.T) {
	h := th{t}
	tbl := NewTable(WithTargetWidth(30))
	tbl.Column(0).SetAlign(AlignRight)
	tbl.Column(1).SetAlign(AlignRight)
	tbl.Column(2).SetAlign(AlignRight)

	hdr := h.header(tbl.AddHeader(WithRowStyle("text-align: center")))
	h.cell(hdr.AddCell(WithContent("A")))                       // col→right, row→center → center
	h.cell(hdr.AddCell(WithContent("B"), WithAlign(AlignLeft))) // cell→left
	h.cell(hdr.AddCell(WithContent("C")))                       // row→center

	out := tbl.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	content := lines[1]
	// cells across 3 cols (target=30): assigned [7, 7, 6] content
	// widths → cell areas 9, 9, 8.
	//   col 0 A: row→center, " " + "    A    " + " "? No. Center in
	//     7-wide: 3 left, 3 right → "   A   ", plus pads → "    A    "
	//   col 1 B: explicit left, " B      " → pads → " B       "
	//   col 2 C: row→center. Center in 6-wide: 2 left, 3 right →
	//     "  C   ", plus pads → "   C    "
	want := "│    A    │ B       │   C    │"
	if content != want {
		t.Errorf("got:\n%q\nwant:\n%q", content, want)
	}
}

func TestColumnStyleFlexAndWidthTogether(t *testing.T) {
	h := th{t}
	tbl := NewTable(WithTargetWidth(60))
	// One col pinned, two flex at different weights.
	tbl.Column(0).Style("width: 10")
	tbl.Column(1).Style("flex: 1")
	tbl.Column(2).Style("flex: 3")

	r := h.row(tbl.AddRow())
	h.cell(r.AddCell(WithContent("a")))
	h.cell(r.AddCell(WithContent("b")))
	h.cell(r.AddCell(WithContent("c")))

	l := Layout(tbl, Measure(tbl))
	// Col 0 pinned = 10.
	if l.colAssigned[0] != 10 {
		t.Errorf("col 0 = %d, want 10 (pinned)", l.colAssigned[0])
	}
	// Col 2 should be ~3x col 1 after each gets content min (1).
	// Flex remaining = 60 - 10(overhead) - 10(col 0) - 1 - 1 = 38;
	// cols 1 and 2 flex [1,3] → col 1 ~10, col 2 ~28 plus mins.
	if l.colAssigned[2] <= l.colAssigned[1]*2 {
		t.Errorf("col 2 should dominate col 1: %v", l.colAssigned)
	}
}

func TestColumnStyleInvalidWidthFalls(t *testing.T) {
	tbl := NewTable()
	tbl.Column(0).Style("width: -3; width: 0; width: xyz")
	if tbl.Column(0).Width() != 0 {
		t.Errorf("width should remain unset from all-invalid inputs, got %d",
			tbl.Column(0).Width())
	}
}
