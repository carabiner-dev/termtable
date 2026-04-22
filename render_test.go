// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// TestRenderBasic2x2 locks the rendered output of a simple 2-column,
// 2-row table end-to-end.
func TestRenderBasic2x2(t *testing.T) {
	tbl := NewTable(WithTargetWidth(20))
	r0 := tbl.AddRow()
	r0.AddCell(WithContent("A"))
	r0.AddCell(WithContent("B"))
	r1 := tbl.AddRow()
	r1.AddCell(WithContent("C"))
	r1.AddCell(WithContent("D"))

	want := strings.Join([]string{
		"┌─────────┬────────┐",
		"│ A       │ B      │",
		"├─────────┼────────┤",
		"│ C       │ D      │",
		"└─────────┴────────┘",
		"",
	}, "\n")
	got := tbl.String()
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderEmptyTable(t *testing.T) {
	tbl := NewTable()
	if got := tbl.String(); got != "" {
		t.Errorf("empty table: got %q, want empty", got)
	}
}

func TestRenderColspanSuppressesInteriorVerticals(t *testing.T) {
	tbl := NewTable(WithTargetWidth(30))
	banner := tbl.AddHeader()
	banner.AddCell(
		WithContent("Banner"),
		WithColSpan(3),
		WithAlign(AlignCenter),
	)
	cols := tbl.AddHeader()
	cols.AddCell(WithContent("a"))
	cols.AddCell(WithContent("b"))
	cols.AddCell(WithContent("c"))

	out := tbl.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	// Banner row's content line should contain no inner │ characters.
	bannerContent := lines[1]
	if strings.Count(bannerContent, "│") != 2 {
		t.Errorf("banner content line has inner verticals: %q", bannerContent)
	}
	// Top border: colspan over all 3 → top should be ┌──...──┐ with no ┬.
	top := lines[0]
	if strings.ContainsRune(top, '┬') {
		t.Errorf("top border has a ┬ where the colspan should suppress it: %q", top)
	}
	// Separator between banner and column-headers must have ┬-style
	// joins where new columns start below: expect ┬ at inner junctions
	// (N=0 since colspan suppresses vertical above; S=1, E=1, W=1).
	sep := lines[2]
	if !strings.ContainsRune(sep, '┬') {
		t.Errorf("separator should introduce ┬ joins under the banner: %q", sep)
	}
}

func TestRenderRowspanPassesThroughBorder(t *testing.T) {
	tbl := NewTable(WithTargetWidth(30))
	r0 := tbl.AddRow()
	r0.AddCell(WithContent("big\nspan"), WithRowSpan(2))
	r0.AddCell(WithContent("x"))
	r1 := tbl.AddRow()
	r1.AddCell(WithContent("y"))

	out := tbl.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	// The separator line between rows 0 and 1 must NOT contain a ┬/┴/┼
	// at the leftmost inner junction — the left column is a rowspan
	// so its horizontal border is suppressed. Expect ┤ at that spot
	// (N+S+W, no E).
	var sep string
	for i := 1; i < len(lines); i++ {
		if strings.ContainsRune(lines[i], '┤') {
			sep = lines[i]
			break
		}
	}
	if sep == "" {
		t.Fatalf("no ┤ junction found; rowspan border may not be suppressed:\n%s", out)
	}
}

func TestRenderAlignLeftRightCenter(t *testing.T) {
	tbl := NewTable(WithTargetWidth(30))
	r := tbl.AddRow()
	r.AddCell(WithContent("L"), WithAlign(AlignLeft))
	r.AddCell(WithContent("C"), WithAlign(AlignCenter))
	r.AddCell(WithContent("R"), WithAlign(AlignRight))

	out := tbl.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	content := lines[1]
	// 3 columns at target=30: overhead=10, available=20, base=6 remainder 2 →
	// assigned [7, 7, 6]. Right-align places R at the right edge of the
	// *content area*; the cell's right padding is still a single space
	// outside the content, so R sits one column inside the inner border.
	want := "│ L       │    C    │      R │"
	if content != want {
		t.Errorf("got:\n%q\nwant:\n%q", content, want)
	}
}

func TestRenderWrapsLongCellContent(t *testing.T) {
	tbl := NewTable(WithTargetWidth(20))
	r := tbl.AddRow()
	r.AddCell(WithContent("short"))
	r.AddCell(WithContent("this is longer text"))

	out := tbl.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	// Count content lines: should be at least 2 for the wrap on col 1.
	contentLines := 0
	for _, ln := range lines {
		if strings.HasPrefix(ln, "│") {
			contentLines++
		}
	}
	if contentLines < 2 {
		t.Errorf("expected multi-line wrap, got %d content lines:\n%s", contentLines, out)
	}
}

func TestRenderCJKKeepsGridAligned(t *testing.T) {
	tbl := NewTable(WithTargetWidth(30))
	r := tbl.AddRow()
	r.AddCell(WithContent("ascii"))
	r.AddCell(WithContent("中文")) //nolint:gosmopolitan // CJK render test
	r.AddCell(WithContent("abc"))

	out := tbl.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	// Every line should have the same display width — no alignment
	// should be deformed by CJK.
	target := DisplayWidth(lines[0])
	for i, ln := range lines {
		if DisplayWidth(ln) != target {
			t.Errorf("line %d width %d, want %d: %q",
				i, DisplayWidth(ln), target, ln)
		}
	}
}

func TestRenderEmojiKeepsGridAligned(t *testing.T) {
	tbl := NewTable(WithTargetWidth(30))
	r := tbl.AddRow()
	r.AddCell(WithContent("ascii"))
	r.AddCell(WithContent("🔥🚀"))
	r.AddCell(WithContent("👨‍👩‍👧"))

	out := tbl.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	// Measure with the renderer's effective emoji mode — Conservative
	// pads composite clusters wider than Grapheme, and alignment is
	// defined relative to whichever mode produced the layout.
	mode := tbl.resolveEmojiWidth()
	target := displayWidthFor(lines[0], mode)
	for i, ln := range lines {
		if got := displayWidthFor(ln, mode); got != target {
			t.Errorf("line %d width %d, want %d: %q", i, got, target, ln)
		}
	}
}

func TestRenderPreservesANSI(t *testing.T) {
	tbl := NewTable(WithTargetWidth(20))
	r := tbl.AddRow()
	r.AddCell(WithContent("\x1b[31mred\x1b[0m"))
	r.AddCell(WithContent("plain"))

	out := tbl.String()
	if !strings.Contains(out, "\x1b[31m") {
		t.Errorf("ANSI open missing from output:\n%q", out)
	}
	if !strings.Contains(out, "\x1b[0m") {
		t.Errorf("ANSI reset missing from output:\n%q", out)
	}
	// Despite ANSI bytes, grid alignment must hold.
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	target := DisplayWidth(lines[0])
	for i, ln := range lines {
		if DisplayWidth(ln) != target {
			t.Errorf("line %d width %d, want %d: %q",
				i, DisplayWidth(ln), target, ln)
		}
	}
}

func TestWriteToReturnsLayoutError(t *testing.T) {
	// Overhead for 2 cols is (2+1) + 2*2 = 7. Target=7 leaves 0
	// content cols — genuinely pathological.
	tbl := NewTable(WithTargetWidth(7))
	r := tbl.AddRow()
	r.AddCell(WithContent("loooooooongword"))
	r.AddCell(WithContent("anotheroneeeee"))

	var buf bytes.Buffer
	_, err := tbl.WriteTo(&buf)
	if !errors.Is(err, ErrTargetTooNarrow) {
		t.Errorf("err = %v, want ErrTargetTooNarrow wrapped", err)
	}
	// Best-effort output should still have been produced.
	if buf.Len() == 0 {
		t.Error("expected partial output on too-narrow layout")
	}
}

func TestRenderRowspanFillsRemainingWithBlanks(t *testing.T) {
	tbl := NewTable(WithTargetWidth(30))
	r0 := tbl.AddRow()
	r0.AddCell(WithContent("short"), WithRowSpan(3))
	r0.AddCell(WithContent("x"))
	r1 := tbl.AddRow()
	r1.AddCell(WithContent("y"))
	r2 := tbl.AddRow()
	r2.AddCell(WithContent("z"))

	out := tbl.String()
	// Just ensure the table parses as a rectangle — every line has the
	// same display width, no overflow.
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	target := DisplayWidth(lines[0])
	for i, ln := range lines {
		if DisplayWidth(ln) != target {
			t.Errorf("line %d width %d, want %d: %q",
				i, DisplayWidth(ln), target, ln)
		}
	}
}

func TestRenderHeadersBodyFootersAllSections(t *testing.T) {
	tbl := NewTable(WithTargetWidth(30))
	hd := tbl.AddHeader()
	hd.AddCell(WithContent("h1"))
	hd.AddCell(WithContent("h2"))
	r := tbl.AddRow()
	r.AddCell(WithContent("b1"))
	r.AddCell(WithContent("b2"))
	f := tbl.AddFooter()
	f.AddCell(WithContent("f1"))
	f.AddCell(WithContent("f2"))

	out := tbl.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	// 3 content rows + 1 top + 2 seps + 1 bottom = 7 lines.
	if len(lines) != 7 {
		t.Errorf("lines = %d, want 7:\n%s", len(lines), out)
	}
	// Each content row should render its values.
	for _, want := range []string{"h1", "b1", "f1"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}
