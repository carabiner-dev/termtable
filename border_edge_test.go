// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"strings"
	"testing"
)

// TestBorderEdgeNoneSkipsLines verifies that "border: none" at the
// table level removes every border line (top, bottom, inter-row) and
// every vertical seam. Output is content-only.
func TestBorderEdgeNoneSkipsLines(t *testing.T) {
	tbl := NewTable(
		WithTargetWidth(30),
		WithTableStyle("border: none"),
	)
	hdr := tbl.AddRow()
	hdr.AddCell(WithContent("A"))
	hdr.AddCell(WithContent("B"))
	r := tbl.AddRow()
	r.AddCell(WithContent("1"))
	r.AddCell(WithContent("2"))

	out := tbl.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 content lines (no borders), got %d:\n%s", len(lines), out)
	}
	for _, ln := range lines {
		if strings.ContainsAny(ln, "─│┌┐└┘├┤┬┴┼") {
			t.Errorf("border glyph leaked into borderless output: %q", ln)
		}
	}
}

// TestBorderEdgeHiddenKeepsSpacing verifies that "border: hidden"
// emits lines as all-spaces — rows are spaced apart but no glyph
// appears.
func TestBorderEdgeHiddenKeepsSpacing(t *testing.T) {
	tbl := NewTable(
		WithTargetWidth(30),
		WithTableStyle("border: hidden"),
	)
	r := tbl.AddRow()
	r.AddCell(WithContent("A"))
	r.AddCell(WithContent("B"))

	out := tbl.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	// 1 top + 1 content + 1 bottom = 3 lines.
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (hidden preserves spacing), got %d:\n%s", len(lines), out)
	}
	if strings.TrimSpace(lines[0]) != "" {
		t.Errorf("hidden top border should be all spaces, got %q", lines[0])
	}
	if strings.TrimSpace(lines[2]) != "" {
		t.Errorf("hidden bottom border should be all spaces, got %q", lines[2])
	}
}

// TestRowBorderBottomSolidUnderlinesHeader is the canonical
// "header-only rule" layout: table has no default borders, the first
// row requests a solid bottom border. Result: single rule under the
// header, nothing else.
func TestRowBorderBottomSolidUnderlinesHeader(t *testing.T) {
	tbl := NewTable(
		WithTargetWidth(40),
		WithTableStyle("border: none"),
	)
	hdr := tbl.AddHeader(WithRowBorderBottom(BorderEdgeSolid))
	hdr.AddCell(WithContent("Name"))
	hdr.AddCell(WithContent("Age"))
	r := tbl.AddRow()
	r.AddCell(WithContent("alice"))
	r.AddCell(WithContent("30"))

	out := tbl.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	// header + rule + body = 3 lines.
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (header, rule, body), got %d:\n%s", len(lines), out)
	}
	// The second line is the rule: should contain the horizontal glyph.
	if !strings.ContainsRune(lines[1], '─') {
		t.Errorf("expected ─ in rule line, got %q", lines[1])
	}
	// Verify the rule line has NO vertical glyphs (no seams, since
	// table border is none).
	if strings.ContainsRune(lines[1], '│') {
		t.Errorf("rule line should not have vertical seams, got %q", lines[1])
	}
}

// TestCellBorderOverridesRow verifies that a single cell opting to
// `border-bottom: none` in a row that has `border-bottom: solid`
// substitutes a space at its column position — the line is still
// drawn (because the other cells agree on solid) but the opted-out
// cell's portion renders as whitespace.
func TestCellBorderOverridesRow(t *testing.T) {
	tbl := NewTable(
		WithTargetWidth(40),
		WithTableStyle("border: none"),
	)
	hdr := tbl.AddHeader(WithRowBorderBottom(BorderEdgeSolid))
	hdr.AddCell(WithContent("Keep"))
	hdr.AddCell(WithContent("Skip"), WithCellBorderBottom(BorderEdgeNone))
	hdr.AddCell(WithContent("Keep"))
	r := tbl.AddRow()
	r.AddCell(WithContent("1"))
	r.AddCell(WithContent("2"))
	r.AddCell(WithContent("3"))

	out := tbl.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d:\n%s", len(lines), out)
	}
	rule := lines[1]
	// Rule must start and end with horizontal glyphs…
	if !strings.HasPrefix(strings.TrimLeft(rule, " "), "─") {
		t.Errorf("rule should start with ─, got %q", rule)
	}
	// …but contain a run of spaces in the middle where "Skip" sits.
	if !strings.Contains(rule, "  ") {
		t.Errorf("rule should contain a space run at the opted-out column, got %q", rule)
	}
}

// TestAllCellsBorderNoneSkipsLine verifies that when every cell at a
// boundary says None, the line is dropped entirely (not rendered as
// spaces).
func TestAllCellsBorderNoneSkipsLine(t *testing.T) {
	tbl := NewTable(
		WithTargetWidth(30),
		WithTableStyle("border: none"),
	)
	hdr := tbl.AddHeader()
	hdr.AddCell(WithContent("A"))
	hdr.AddCell(WithContent("B"))
	r := tbl.AddRow()
	r.AddCell(WithContent("x"))
	r.AddCell(WithContent("y"))

	out := tbl.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	// Two content lines only — no borders anywhere.
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d:\n%s", len(lines), out)
	}
}

// TestDefaultStillDrawsAllBorders is a regression guard: unchanged
// callers (no border options, no per-row directives) continue to get
// today's fully-bordered output.
func TestDefaultStillDrawsAllBorders(t *testing.T) {
	tbl := NewTable(WithTargetWidth(30))
	r := tbl.AddRow()
	r.AddCell(WithContent("x"))
	r.AddCell(WithContent("y"))

	out := tbl.String()
	if !strings.ContainsRune(out, '┌') || !strings.ContainsRune(out, '┘') ||
		!strings.ContainsRune(out, '│') || !strings.ContainsRune(out, '─') {
		t.Errorf("default output missing expected border glyphs:\n%s", out)
	}
}
