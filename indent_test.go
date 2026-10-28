// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"strings"
	"testing"
)

// renderedLines returns the non-empty lines of a rendered table.
func renderedLines(t *testing.T, tbl *Table) []string {
	t.Helper()
	out := tbl.String()
	if out == "" {
		t.Fatal("table rendered empty")
	}
	if !strings.HasSuffix(out, "\n") {
		t.Fatalf("output should end in a newline, got %q", out)
	}
	return strings.Split(strings.TrimRight(out, "\n"), "\n")
}

// TestIndentPrefixesEveryLine verifies that the margin is emitted in
// front of borders and content alike, and that the trailing newline
// does not grow a whitespace-only line.
func TestIndentPrefixesEveryLine(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 0, false)

	tbl := NewTable(WithIndent(3), WithTargetWidth(30))
	r := tbl.AddRow()
	r.AddCell(WithContent("a"))
	r.AddCell(WithContent("b"))

	for i, ln := range renderedLines(t, tbl) {
		if !strings.HasPrefix(ln, "   ") || strings.HasPrefix(ln, "    ") {
			t.Errorf("line %d = %q, want exactly 3 leading spaces", i, ln)
		}
		if got := DisplayWidth(ln); got != 33 {
			t.Errorf("line %d width = %d, want 33 (30 + indent)", i, got)
		}
	}
	if strings.HasSuffix(tbl.String(), "\n   ") {
		t.Error("indent must not be applied after the final newline")
	}
}

// TestIndentReservesScreenWidth verifies that a table filling 100 % of
// the screen stays inside the terminal once the margin is counted.
func TestIndentReservesScreenWidth(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 120, true)

	tbl := NewTable(WithIndent(2), WithMaxWidthPercent(100))
	r := tbl.AddRow()
	r.AddCell(WithContent(strings.Repeat("x", 200)))
	r.AddCell(WithContent(strings.Repeat("y", 200)))

	if got := tbl.ResolvedTargetWidth(); got != 118 {
		t.Errorf("ResolvedTargetWidth = %d, want 118 (120 − 2)", got)
	}
	for i, ln := range renderedLines(t, tbl) {
		if got := DisplayWidth(ln); got != 120 {
			t.Errorf("line %d width = %d, want exactly 120", i, got)
		}
	}
}

// TestIndentCapsDefaultFloor verifies that the 80-column floor is
// capped to the screen after the margin, not before.
func TestIndentCapsDefaultFloor(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 80, true)

	tbl := NewTable(WithIndent(4))
	if got := tbl.ResolvedTargetWidth(); got != 76 {
		t.Errorf("ResolvedTargetWidth = %d, want 76 (80 − 4)", got)
	}
}

// TestIndentAppliesToColumnsEnv verifies that COLUMNS, used both as an
// explicit target and as a percentage base, has the margin reserved.
func TestIndentAppliesToColumnsEnv(t *testing.T) {
	t.Setenv("COLUMNS", "100")
	withFakeTTY(t, 0, false)

	if got := NewTable(WithIndent(10)).ResolvedTargetWidth(); got != 90 {
		t.Errorf("COLUMNS target = %d, want 90", got)
	}
	if got := NewTable(WithIndent(10), WithTargetWidthPercent(50)).ResolvedTargetWidth(); got != 45 {
		t.Errorf("50%% of COLUMNS = %d, want 45", got)
	}
}

// TestIndentNeverStarvesTheTable verifies that a margin wider than the
// screen still leaves one column for the layout solver.
func TestIndentNeverStarvesTheTable(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 5, true)

	if got := NewTable(WithIndent(9)).ResolvedTargetWidth(); got != 1 {
		t.Errorf("ResolvedTargetWidth = %d, want 1", got)
	}
}

// TestIndentOptions verifies the option, its CSS spelling and the
// rejection of non-positive values.
func TestIndentOptions(t *testing.T) {
	if got := NewTable(WithIndent(0)).opts.indent; got != 0 {
		t.Errorf("WithIndent(0) set indent = %d", got)
	}
	if got := NewTable(WithIndent(-2)).opts.indent; got != 0 {
		t.Errorf("WithIndent(-2) set indent = %d", got)
	}
	if got := NewTable(WithTableStyle("margin-left: 6")).opts.indent; got != 6 {
		t.Errorf("margin-left: 6 gave indent = %d", got)
	}
	if got := NewTable(WithTableStyle("margin-left: 10%")).opts.indent; got != 0 {
		t.Errorf("percent margin-left should be ignored, got %d", got)
	}
	if got := NewTable(WithTableStyle("margin-left: 2"), WithIndent(5)).opts.indent; got != 5 {
		t.Errorf("last setter should win, got %d", got)
	}
}
