// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"strings"
	"testing"
)

// renderBasic renders a 2x2 reference table with the given border
// set. Used by border tests to assert on glyphs without re-building
// the table each time.
func renderBasic(t *testing.T, b BorderSet) string {
	t.Helper()
	tbl := NewTable(WithTargetWidth(30), WithBorder(b))
	r0 := tbl.AddRow()
	r0.AddCell(WithContent("a"))
	r0.AddCell(WithContent("b"))
	r1 := tbl.AddRow()
	r1.AddCell(WithContent("c"))
	r1.AddCell(WithContent("d"))
	return tbl.String()
}

func TestSingleLineGlyphs(t *testing.T) {
	out := renderBasic(t, SingleLine())
	for _, want := range []rune{'─', '│', '┌', '┐', '└', '┘', '┬', '┴', '├', '┤', '┼'} {
		if !strings.ContainsRune(out, want) {
			t.Errorf("SingleLine output missing %q:\n%s", want, out)
		}
	}
}

func TestDefaultSingleLineAliasesSingleLine(t *testing.T) {
	if SingleLine() != DefaultSingleLine() {
		t.Error("DefaultSingleLine and SingleLine should return identical sets")
	}
}

func TestDoubleLineGlyphs(t *testing.T) {
	out := renderBasic(t, DoubleLine())
	for _, want := range []rune{'═', '║', '╔', '╗', '╚', '╝', '╦', '╩', '╠', '╣', '╬'} {
		if !strings.ContainsRune(out, want) {
			t.Errorf("DoubleLine output missing %q:\n%s", want, out)
		}
	}
}

func TestHeavyLineGlyphs(t *testing.T) {
	out := renderBasic(t, HeavyLine())
	for _, want := range []rune{'━', '┃', '┏', '┓', '┗', '┛', '┳', '┻', '┣', '┫', '╋'} {
		if !strings.ContainsRune(out, want) {
			t.Errorf("HeavyLine output missing %q:\n%s", want, out)
		}
	}
}

func TestRoundedLineHasRoundedCornersOnly(t *testing.T) {
	out := renderBasic(t, RoundedLine())
	for _, want := range []rune{'╭', '╮', '╰', '╯'} {
		if !strings.ContainsRune(out, want) {
			t.Errorf("RoundedLine output missing %q:\n%s", want, out)
		}
	}
	// Interior joins and runs remain single-line.
	for _, want := range []rune{'─', '│', '┬', '┴', '├', '┤', '┼'} {
		if !strings.ContainsRune(out, want) {
			t.Errorf("RoundedLine should keep single-line interior, missing %q:\n%s", want, out)
		}
	}
	// And should not contain the sharp single-line corners.
	for _, bad := range []rune{'┌', '┐', '└', '┘'} {
		if strings.ContainsRune(out, bad) {
			t.Errorf("RoundedLine leaked sharp corner %q:\n%s", bad, out)
		}
	}
}

func TestASCIILineStaysInASCIIRange(t *testing.T) {
	out := renderBasic(t, ASCIILine())
	for _, r := range out {
		if r > 127 {
			t.Errorf("ASCIILine emitted non-ASCII rune %q (0x%x)", r, r)
		}
	}
	// Corners / T-joins / cross all render as '+'.
	if !strings.Contains(out, "+") {
		t.Errorf("ASCIILine output missing '+' joins:\n%s", out)
	}
}

func TestNoBorderRendersOnlySpacesForBorderGlyphs(t *testing.T) {
	out := renderBasic(t, NoBorder())
	// Output lines: top border, content, sep, content, bottom. Border
	// lines must consist entirely of spaces.
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 5 {
		t.Fatalf("lines = %d, want 5:\n%s", len(lines), out)
	}
	for _, idx := range []int{0, 2, 4} {
		if strings.TrimSpace(lines[idx]) != "" {
			t.Errorf("NoBorder line %d should be blank, got %q", idx, lines[idx])
		}
	}
	// Content rows still have their text, surrounded by spaces where
	// border glyphs would otherwise be.
	if !strings.Contains(lines[1], "a") || !strings.Contains(lines[1], "b") {
		t.Errorf("NoBorder content row should carry values: %q", lines[1])
	}
}

func TestBorderStyleCSSRoutesToBorderSet(t *testing.T) {
	const noneStyle = "none"
	cases := []struct {
		name  string
		decl  string
		probe rune // a glyph unique to that style
	}{
		{"double", "border-style: double", '═'},
		{"heavy", "border-style: heavy", '━'},
		{"rounded", "border-style: rounded", '╭'},
		{"ascii", "border-style: ascii", '+'},
		{noneStyle, "border-style: none", ' '}, // all glyphs are spaces
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tbl := NewTable(
				WithTargetWidth(20),
				WithTableStyle(tc.decl),
			)
			r := tbl.AddRow()
			r.AddCell(WithContent("x"))
			r.AddCell(WithContent("y"))
			out := tbl.String()
			if tc.name == noneStyle {
				lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
				if strings.TrimSpace(lines[0]) != "" {
					t.Errorf("border-style:none should produce blank top border, got %q", lines[0])
				}
				return
			}
			if !strings.ContainsRune(out, tc.probe) {
				t.Errorf("border-style:%s missing probe %q:\n%s", tc.name, tc.probe, out)
			}
		})
	}
}

func TestBorderStyleCSSCoexistsWithOtherStyle(t *testing.T) {
	forceColor(t)
	tbl := NewTable(
		WithTargetWidth(20),
		WithTableStyle("border-style: double; border-color: cyan"),
	)
	r := tbl.AddRow()
	r.AddCell(WithContent("x"))

	out := tbl.String()
	if !strings.ContainsRune(out, '═') {
		t.Errorf("border-style:double not applied: %q", out)
	}
	if !strings.Contains(out, "\x1b[36m") {
		t.Errorf("border-color:cyan not applied alongside border-style: %q", out)
	}
}

func TestBorderStyleCSSUnknownKeywordIgnored(t *testing.T) {
	tbl := NewTable(
		WithTargetWidth(20),
		WithTableStyle("border-style: nonsense"),
	)
	r := tbl.AddRow()
	r.AddCell(WithContent("x"))

	// Unknown keyword leaves the default (single-line) set in place.
	if !strings.ContainsRune(tbl.String(), '┌') {
		t.Error("unknown border-style should leave single-line default")
	}
}
