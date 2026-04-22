// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"strings"
	"testing"
)

// longDescTable builds a two-column reference table with the long
// cell option list applied to the "Description" cell. Every
// wrap-mode test uses the same body so differences between tests
// reflect only the mode under test.
func longDescTable(t *testing.T, configure func(*Table), cellOpts ...CellOption) string {
	t.Helper()
	h := th{t}
	tbl := NewTable(WithTargetWidth(40))
	if configure != nil {
		configure(tbl)
	}
	hdr := h.header(tbl.AddHeader())
	h.cell(hdr.AddCell(WithContent("Name")))
	h.cell(hdr.AddCell(WithContent("Description")))
	r := h.row(tbl.AddRow())
	h.cell(r.AddCell(WithContent("widget")))
	h.cell(r.AddCell(append(
		[]CellOption{WithContent("a long description that would otherwise wrap across multiple lines")},
		cellOpts...,
	)...))
	return tbl.String()
}

func contentLines(out string) []string {
	var rows []string
	for _, ln := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if strings.HasPrefix(ln, "│") {
			rows = append(rows, ln)
		}
	}
	return rows
}

func TestDefaultWrapMultiLine(t *testing.T) {
	out := longDescTable(t, nil)
	rows := contentLines(out)
	// Header + at least 2 body sub-lines when wrapping is on.
	if len(rows) < 3 {
		t.Errorf("expected multi-line wrap to produce >=3 content lines, got %d:\n%s",
			len(rows), out)
	}
}

func TestWithSingleLineClipsToOneLine(t *testing.T) {
	out := longDescTable(t, nil, WithSingleLine())
	rows := contentLines(out)
	// Header + 1 body line = 2.
	if len(rows) != 2 {
		t.Errorf("expected 2 content lines with WithSingleLine, got %d:\n%s",
			len(rows), out)
	}
	if !strings.Contains(rows[1], "…") {
		t.Errorf("expected ellipsis in truncated single-line cell: %q", rows[1])
	}
}

func TestCSSWhiteSpaceNowrapEquivalent(t *testing.T) {
	imperative := longDescTable(t, nil, WithSingleLine())
	css := longDescTable(t, nil, WithCellStyle("white-space: nowrap"))
	if imperative != css {
		t.Errorf("WithSingleLine and white-space: nowrap should produce identical output\nimperative:\n%s\ncss:\n%s",
			imperative, css)
	}
}

func TestCSSTextOverflowClip(t *testing.T) {
	out := longDescTable(t, nil, WithCellStyle("white-space: nowrap; text-overflow: clip"))
	rows := contentLines(out)
	if len(rows) != 2 {
		t.Errorf("expected 2 content lines, got %d", len(rows))
	}
	if strings.Contains(rows[1], "…") {
		t.Errorf("text-overflow: clip should suppress ellipsis: %q", rows[1])
	}
}

func TestCSSLineClamp(t *testing.T) {
	out := longDescTable(t, nil, WithCellStyle("line-clamp: 2"))
	rows := contentLines(out)
	// Header + 2 body lines = 3.
	if len(rows) != 3 {
		t.Errorf("expected 3 content lines with line-clamp: 2, got %d:\n%s",
			len(rows), out)
	}
	if !strings.Contains(rows[2], "…") {
		t.Errorf("expected ellipsis on clamped last line: %q", rows[2])
	}
}

func TestCSSLineClampNone(t *testing.T) {
	// Setting line-clamp: none after a prior clamp should reset to
	// unbounded.
	out := longDescTable(t, nil, WithCellStyle("line-clamp: 1; line-clamp: none"))
	rows := contentLines(out)
	if len(rows) < 3 {
		t.Errorf("expected unbounded wrap after line-clamp: none: %d rows", len(rows))
	}
}

func TestCSSWebkitLineClampAccepted(t *testing.T) {
	out := longDescTable(t, nil, WithCellStyle("-webkit-line-clamp: 2"))
	rows := contentLines(out)
	if len(rows) != 3 {
		t.Errorf("expected -webkit-line-clamp to behave like line-clamp: got %d rows", len(rows))
	}
}

func TestColumnWhiteSpaceCascadesToCells(t *testing.T) {
	out := longDescTable(t, func(tbl *Table) {
		tbl.Column(1).Style("white-space: nowrap")
	})
	rows := contentLines(out)
	if len(rows) != 2 {
		t.Errorf("expected column white-space: nowrap to cascade to cells, got %d rows:\n%s",
			len(rows), out)
	}
}

func TestCellMultiLineOverridesColumn(t *testing.T) {
	// Column forces single-line; a cell explicitly asks for multi.
	out := longDescTable(t, func(tbl *Table) {
		tbl.Column(1).Style("white-space: nowrap")
	}, WithMultiLine())
	rows := contentLines(out)
	if len(rows) < 3 {
		t.Errorf("cell WithMultiLine should override column nowrap: got %d rows:\n%s",
			len(rows), out)
	}
}

func TestTableWhiteSpaceNowrapCascades(t *testing.T) {
	h := th{t}
	tbl := NewTable(
		WithTargetWidth(40),
		WithTableStyle("white-space: nowrap"),
	)
	hdr := h.header(tbl.AddHeader())
	h.cell(hdr.AddCell(WithContent("Name")))
	h.cell(hdr.AddCell(WithContent("Description")))
	r := h.row(tbl.AddRow())
	h.cell(r.AddCell(WithContent("widget")))
	h.cell(r.AddCell(WithContent("a long description that would otherwise wrap across multiple lines")))

	out := tbl.String()
	rows := contentLines(out)
	if len(rows) != 2 {
		t.Errorf("expected table-wide nowrap to cascade: got %d rows:\n%s",
			len(rows), out)
	}
}
