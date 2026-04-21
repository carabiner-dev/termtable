// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestMeasureSingleSpanColumnWidths(t *testing.T) {
	h := th{t}
	tbl := NewTable()
	r := h.row(tbl.AddRow())
	h.cell(r.AddCell(WithContent("ab")))          // col 0: min=2, des=2
	h.cell(r.AddCell(WithContent("hello world"))) // col 1: min=5, des=11
	h.cell(r.AddCell(WithContent("a\nbcd")))      // col 2: min=3, des=3

	m := Measure(tbl)
	wantMin := []int{2, 5, 3}
	wantDes := []int{2, 11, 3}
	for i := range wantMin {
		if m.colMin[i] != wantMin[i] {
			t.Errorf("colMin[%d] = %d, want %d", i, m.colMin[i], wantMin[i])
		}
		if m.colDesired[i] != wantDes[i] {
			t.Errorf("colDesired[%d] = %d, want %d", i, m.colDesired[i], wantDes[i])
		}
	}
	if len(m.multiSpans) != 0 {
		t.Errorf("multiSpans = %d, want 0", len(m.multiSpans))
	}
}

func TestMeasureMultiSpanRecorded(t *testing.T) {
	h := th{t}
	tbl := NewTable()
	r := h.row(tbl.AddRow())
	h.cell(r.AddCell(WithContent("banner text here"), WithColSpan(3)))

	m := Measure(tbl)
	// Single-column contributions should be zero — the multi-span
	// constraint is the only source of width for this cell.
	for i, v := range m.colMin {
		if v != 0 {
			t.Errorf("colMin[%d] = %d, want 0 (multi-span doesn't feed per-column)", i, v)
		}
	}
	if len(m.multiSpans) != 1 {
		t.Fatalf("multiSpans = %d, want 1", len(m.multiSpans))
	}
	cons := m.multiSpans[0]
	if cons.colStart != 0 || cons.colSpan != 3 {
		t.Errorf("span = [%d..%d), want [0..3)", cons.colStart, cons.colStart+cons.colSpan)
	}
	if cons.minWidth != MinUnbreakableWidth("banner text here") {
		t.Errorf("minWidth = %d", cons.minWidth)
	}
}

func TestMeasureConsumesReader(t *testing.T) {
	h := th{t}
	const want = "hello"
	tbl := NewTable()
	r := h.row(tbl.AddRow())
	c := h.cell(r.AddCell(WithReader(strings.NewReader(want))))

	Measure(tbl)
	if !c.resolved {
		t.Error("cell should be marked resolved after Measure")
	}
	if c.Content() != want {
		t.Errorf("Content = %q, want %q", c.Content(), want)
	}
	if !c.hasContent {
		t.Error("hasContent should be true after reader consumption")
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestMeasureReaderErrorRecorded(t *testing.T) {
	h := th{t}
	tbl := NewTable()
	r := h.row(tbl.AddRow())
	h.cell(r.AddCell(WithReader(errReader{})))

	m := Measure(tbl)
	if len(m.readerErrs) != 1 {
		t.Fatalf("readerErrs = %d, want 1", len(m.readerErrs))
	}
}

func TestMeasureReaderContentAvailableForLaterCalls(t *testing.T) {
	h := th{t}
	tbl := NewTable()
	r := h.row(tbl.AddRow())
	c := h.cell(r.AddCell(WithReader(io.NopCloser(strings.NewReader("cached")))))

	Measure(tbl)
	// Calling Measure again should not re-read (reader is drained).
	Measure(tbl)
	if c.Content() != "cached" {
		t.Errorf("Content = %q, want %q", c.Content(), "cached")
	}
}

func TestMeasureWalksAllSections(t *testing.T) {
	h := th{t}
	tbl := NewTable()
	hd := h.header(tbl.AddHeader())
	h.cell(hd.AddCell(WithContent("H")))

	r := h.row(tbl.AddRow())
	h.cell(r.AddCell(WithContent("B")))

	f := h.footer(tbl.AddFooter())
	h.cell(f.AddCell(WithContent("F")))

	m := Measure(tbl)
	if len(m.colMin) != 1 {
		t.Fatalf("colMin len = %d, want 1", len(m.colMin))
	}
	if m.colMin[0] != 1 || m.colDesired[0] != 1 {
		t.Errorf("single column widths = min=%d des=%d, want 1/1", m.colMin[0], m.colDesired[0])
	}
}
