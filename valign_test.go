// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"strings"
	"testing"
)

// shortLongTable builds a two-column table where col 0 has a single
// short word and col 1 wraps to three lines; the resulting row has
// height 3 and col 0 has two blank lines to distribute.
func shortLongTable(t *testing.T, opts ...CellOption) string {
	t.Helper()
	tbl := NewTable(WithTargetWidth(40))
	r := tbl.AddRow()
	r.AddCell(append([]CellOption{WithContent("short")}, opts...)...)
	r.AddCell(WithContent("this is a much longer message that must wrap"))
	return tbl.String()
}

func TestVAlignTopDefault(t *testing.T) {
	out := shortLongTable(t)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	// 1 top border + 3 content + 1 bottom = 5 lines. Content sub-lines
	// are indices 1, 2, 3.
	if len(lines) != 5 {
		t.Fatalf("lines = %d, want 5", len(lines))
	}
	if !strings.Contains(lines[1], "short") {
		t.Errorf("top-aligned content should be on sub-line 0: %q", lines[1])
	}
	for _, i := range []int{2, 3} {
		if strings.Contains(lines[i], "short") {
			t.Errorf("top-aligned content leaked to sub-line %d: %q", i-1, lines[i])
		}
	}
}

func TestVAlignMiddle(t *testing.T) {
	out := shortLongTable(t, WithVAlign(VAlignMiddle))
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	// With vspan=3 and h=1, middle offset = 1 → content on sub-line 1.
	if !strings.Contains(lines[2], "short") {
		t.Errorf("middle-aligned content should be on sub-line 1: %q", lines[2])
	}
	for _, i := range []int{1, 3} {
		if strings.Contains(lines[i], "short") {
			t.Errorf("middle-aligned content leaked to sub-line %d: %q", i-1, lines[i])
		}
	}
}

func TestVAlignBottom(t *testing.T) {
	out := shortLongTable(t, WithVAlign(VAlignBottom))
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	// offset = vspan - h = 2 → content on sub-line 2.
	if !strings.Contains(lines[3], "short") {
		t.Errorf("bottom-aligned content should be on sub-line 2: %q", lines[3])
	}
	for _, i := range []int{1, 2} {
		if strings.Contains(lines[i], "short") {
			t.Errorf("bottom-aligned content leaked to sub-line %d: %q", i-1, lines[i])
		}
	}
}

func TestVAlignCSSEquivalent(t *testing.T) {
	tbl := NewTable(WithTargetWidth(40))
	r := tbl.AddRow()
	r.AddCell(
		WithContent("short"),
		WithCellStyle("vertical-align: bottom"),
	)
	r.AddCell(WithContent("this is a much longer message that must wrap"))

	out := tbl.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if !strings.Contains(lines[3], "short") {
		t.Errorf("CSS bottom-aligned content should be on sub-line 2: %q", lines[3])
	}
}

func TestColumnVAlignCascadesToCells(t *testing.T) {
	tbl := NewTable(WithTargetWidth(40))
	tbl.Column(0).SetVAlign(VAlignMiddle)

	r := tbl.AddRow()
	r.AddCell(WithContent("short")) // inherits column
	r.AddCell(WithContent("this is a much longer message that must wrap"))

	out := tbl.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if !strings.Contains(lines[2], "short") {
		t.Errorf("cell should inherit column middle-align: %q", lines[2])
	}
}

func TestCellVAlignOverridesColumn(t *testing.T) {
	tbl := NewTable(WithTargetWidth(40))
	tbl.Column(0).SetVAlign(VAlignMiddle)

	r := tbl.AddRow()
	r.AddCell(WithContent("short"), WithVAlign(VAlignBottom))
	r.AddCell(WithContent("this is a much longer message that must wrap"))

	out := tbl.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if !strings.Contains(lines[3], "short") {
		t.Errorf("cell bottom-align should override column middle: %q", lines[3])
	}
}

func TestVAlignNoEffectWhenContentFillsRow(t *testing.T) {
	// Row height equals content height → no offset possible, valign
	// should be a no-op.
	tbl := NewTable(WithTargetWidth(40))
	r := tbl.AddRow()
	r.AddCell(WithContent("a"), WithVAlign(VAlignBottom))
	r.AddCell(WithContent("b"))

	out := tbl.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	// Top border + 1 content + bottom border = 3 lines. Content on
	// sub-line 0 (lines[1]).
	if !strings.Contains(lines[1], "a") {
		t.Errorf("single-line row should still render content: %q", lines[1])
	}
}
