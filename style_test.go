// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"reflect"
	"strings"
	"testing"

	"github.com/fatih/color"
)

// forceColor enables ANSI emission for the duration of a test,
// regardless of TTY detection or environment settings. fatih/color's
// global NoColor flag otherwise suppresses output when stdout is not
// a terminal.
func forceColor(t *testing.T) {
	t.Helper()
	saved := color.NoColor
	color.NoColor = false
	t.Cleanup(func() { color.NoColor = saved })
}

func TestParseCSSNamedColors(t *testing.T) {
	var s Style
	parseCSS("color: red; background: blue", &s)
	if s.set&sFg == 0 {
		t.Error("fg not set")
	}
	if !reflect.DeepEqual(s.fgAttrs, []color.Attribute{color.FgRed}) {
		t.Errorf("fgAttrs = %v", s.fgAttrs)
	}
	if s.set&sBg == 0 {
		t.Error("bg not set")
	}
	if !reflect.DeepEqual(s.bgAttrs, []color.Attribute{color.BgBlue}) {
		t.Errorf("bgAttrs = %v", s.bgAttrs)
	}
}

func TestParseCSSBrightVariants(t *testing.T) {
	var s Style
	parseCSS("color: bright-green; background-color: bright-magenta", &s)
	if !reflect.DeepEqual(s.fgAttrs, []color.Attribute{color.FgHiGreen}) {
		t.Errorf("fgAttrs = %v", s.fgAttrs)
	}
	if !reflect.DeepEqual(s.bgAttrs, []color.Attribute{color.BgHiMagenta}) {
		t.Errorf("bgAttrs = %v", s.bgAttrs)
	}
}

func TestParseCSSHexColor(t *testing.T) {
	var s Style
	parseCSS("color: #ff0088", &s)
	want := []color.Attribute{38, 2, 255, 0, 136}
	if !reflect.DeepEqual(s.fgAttrs, want) {
		t.Errorf("fgAttrs = %v, want %v", s.fgAttrs, want)
	}
}

func TestParseCSSRGBColor(t *testing.T) {
	var s Style
	parseCSS("background: rgb(10, 20, 255)", &s)
	want := []color.Attribute{48, 2, 10, 20, 255}
	if !reflect.DeepEqual(s.bgAttrs, want) {
		t.Errorf("bgAttrs = %v, want %v", s.bgAttrs, want)
	}
}

func TestParseCSSTextAttributes(t *testing.T) {
	var s Style
	parseCSS("font-weight: bold; font-style: italic; text-decoration: underline line-through", &s)
	if !s.bold || s.set&sBold == 0 {
		t.Error("bold not set")
	}
	if !s.italic || s.set&sItalic == 0 {
		t.Error("italic not set")
	}
	if !s.underline || s.set&sUnderline == 0 {
		t.Error("underline not set")
	}
	if !s.strike || s.set&sStrike == 0 {
		t.Error("strike not set")
	}
}

func TestParseCSSTextDecorationNone(t *testing.T) {
	var s Style
	s.underline = true
	s.strike = true
	s.set |= sUnderline | sStrike
	parseCSS("text-decoration: none", &s)
	if s.underline || s.strike {
		t.Error("text-decoration:none should clear underline/strike")
	}
}

func TestParseCSSBorderColor(t *testing.T) {
	var s Style
	parseCSS("border-color: cyan", &s)
	if s.set&sBorder == 0 {
		t.Error("border not set")
	}
	if !reflect.DeepEqual(s.borderAttrs, []color.Attribute{color.FgCyan}) {
		t.Errorf("borderAttrs = %v", s.borderAttrs)
	}
}

func TestParseCSSUnknownPropertiesIgnored(t *testing.T) {
	var s Style
	parseCSS("nonsense: value; color: red; also-wrong: 3px", &s)
	if s.set&sFg == 0 {
		t.Error("valid property should still be applied around unknowns")
	}
	if !reflect.DeepEqual(s.fgAttrs, []color.Attribute{color.FgRed}) {
		t.Errorf("fgAttrs = %v", s.fgAttrs)
	}
}

func TestParseCSSMalformed(t *testing.T) {
	var s Style
	parseCSS("color; missing-colon; ; background: ; color: nonsense", &s)
	// None of the above should produce a set property: two are
	// malformed, one has an empty value, one has an unrecognized
	// value.
	if s.set != 0 {
		t.Errorf("set = %b, want 0 (all inputs invalid)", s.set)
	}
}

func TestStyleMergeCascade(t *testing.T) {
	// Simulate table → row → cell cascade.
	var table Style
	parseCSS("color: white; background: blue", &table)

	var row Style
	parseCSS("font-weight: bold", &row)

	var cell Style
	parseCSS("color: red", &cell)

	var eff Style
	eff.merge(&table)
	eff.merge(&row)
	eff.merge(&cell)

	if !reflect.DeepEqual(eff.fgAttrs, []color.Attribute{color.FgRed}) {
		t.Errorf("fg: expected cell override to red, got %v", eff.fgAttrs)
	}
	if !reflect.DeepEqual(eff.bgAttrs, []color.Attribute{color.BgBlue}) {
		t.Errorf("bg: expected table inheritance of blue, got %v", eff.bgAttrs)
	}
	if !eff.bold {
		t.Error("bold: expected row inheritance of true")
	}
}

func TestStyleApplyContentEmitsSGR(t *testing.T) {
	forceColor(t)
	s := &Style{}
	parseCSS("color: red; font-weight: bold", s)
	out := s.applyContent("hello")
	if !strings.Contains(out, "\x1b[") {
		t.Errorf("expected ANSI sequence, got %q", out)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("content missing: %q", out)
	}
	if DisplayWidth(out) != 5 {
		t.Errorf("display width = %d, want 5", DisplayWidth(out))
	}
}

func TestStyleApplyContentEmptyIsNoop(t *testing.T) {
	forceColor(t)
	s := &Style{}
	if got := s.applyContent("hello"); got != "hello" {
		t.Errorf("empty style should pass content through, got %q", got)
	}
}

func TestStyleApplyBorderOnlyUsesBorderAttrs(t *testing.T) {
	forceColor(t)
	s := &Style{}
	parseCSS("color: red; background: blue; border-color: cyan", s)
	out := s.applyBorder("─")
	// Must contain cyan (36), must not contain red (31) or blue bg (44).
	if !strings.Contains(out, "\x1b[36m") {
		t.Errorf("border output %q missing cyan", out)
	}
	if strings.Contains(out, "\x1b[31m") || strings.Contains(out, "\x1b[44m") {
		t.Errorf("border output %q leaked fg/bg attrs", out)
	}
}

func TestRenderAppliesCellStyle(t *testing.T) {
	forceColor(t)
	h := th{t}
	tbl := NewTable(WithTargetWidth(20))
	r := h.row(tbl.AddRow())
	h.cell(r.AddCell(WithContent("PASS"),
		WithTextColor("green"), WithBold()))

	out := tbl.String()
	if !strings.Contains(out, "\x1b[32") && !strings.Contains(out, "\x1b[1;32") &&
		!strings.Contains(out, "\x1b[32;1") {
		t.Errorf("expected green+bold SGR in output, got:\n%q", out)
	}
	// Grid alignment must survive despite ANSI bytes.
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	target := DisplayWidth(lines[0])
	for i, ln := range lines {
		if DisplayWidth(ln) != target {
			t.Errorf("line %d width %d, want %d: %q",
				i, DisplayWidth(ln), target, ln)
		}
	}
}

func TestRenderAppliesTableBorderColor(t *testing.T) {
	forceColor(t)
	h := th{t}
	tbl := NewTable(
		WithTargetWidth(20),
		WithTableStyle("border-color: cyan"),
	)
	r := h.row(tbl.AddRow())
	h.cell(r.AddCell(WithContent("x")))
	h.cell(r.AddCell(WithContent("y")))

	out := tbl.String()
	if !strings.Contains(out, "\x1b[36m") {
		t.Errorf("expected cyan SGR on borders, got:\n%q", out)
	}
}

func TestRenderRowStyleCascadesToCells(t *testing.T) {
	forceColor(t)
	h := th{t}
	tbl := NewTable(WithTargetWidth(20))
	hdr := h.header(tbl.AddHeader(WithRowStyle("font-weight: bold")))
	h.cell(hdr.AddCell(WithContent("hdr1")))
	h.cell(hdr.AddCell(WithContent("hdr2")))

	out := tbl.String()
	// SGR code for bold is 1. Look for "1m" in an escape following
	// an opening \x1b[.
	if !strings.Contains(out, "\x1b[1m") && !strings.Contains(out, ";1m") {
		t.Errorf("expected bold SGR in header row, got:\n%q", out)
	}
}

func TestRenderNoColorYieldsPlainOutput(t *testing.T) {
	saved := color.NoColor
	color.NoColor = true
	defer func() { color.NoColor = saved }()

	h := th{t}
	tbl := NewTable(
		WithTargetWidth(20),
		WithTableStyle("border-color: cyan"),
	)
	r := h.row(tbl.AddRow())
	h.cell(r.AddCell(WithContent("x"),
		WithTextColor("red"), WithBold(),
		WithBackgroundColor("blue")))

	out := tbl.String()
	if strings.Contains(out, "\x1b[") {
		t.Errorf("NoColor=true should suppress ANSI, got:\n%q", out)
	}
}

func TestCellStyleOverridesRowStyle(t *testing.T) {
	forceColor(t)
	h := th{t}
	tbl := NewTable(WithTargetWidth(20))
	hdr := h.header(tbl.AddHeader(WithRowStyle("color: blue")))
	h.cell(hdr.AddCell(WithContent("a"))) // inherits blue
	h.cell(hdr.AddCell(WithContent("b"), WithTextColor("red")))

	out := tbl.String()
	// Both codes must be present somewhere.
	if !strings.Contains(out, "\x1b[34") {
		t.Errorf("blue (34) missing from output: %q", out)
	}
	if !strings.Contains(out, "\x1b[31") {
		t.Errorf("red (31) missing from output: %q", out)
	}
}
