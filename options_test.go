// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"strings"
	"testing"
)

func TestNewCellDefaults(t *testing.T) {
	c := NewCell()
	if c.ColSpan() != 1 {
		t.Errorf("ColSpan = %d, want 1", c.ColSpan())
	}
	if c.RowSpan() != 1 {
		t.Errorf("RowSpan = %d, want 1", c.RowSpan())
	}
	if c.Align() != AlignLeft {
		t.Errorf("Align = %v, want AlignLeft", c.Align())
	}
	// Layout knobs live on Style now. A default-constructed cell has
	// no Style set, so every field is inherited (effective values
	// come from the Table cascade at render time).
	if c.style != nil {
		t.Errorf("default cell should have no Style, got %+v", c.style)
	}
}

func TestCellOptionsApply(t *testing.T) {
	c := NewCell(
		WithCellID("x"),
		WithContent("hello"),
		WithColSpan(3),
		WithRowSpan(2),
		WithAlign(AlignRight),
		WithWrap(false),
		WithTrim(false),
		WithMaxLines(5),
	)
	if c.ID() != "x" {
		t.Errorf("ID = %q, want x", c.ID())
	}
	if c.Content() != "hello" {
		t.Errorf("Content = %q", c.Content())
	}
	if c.ColSpan() != 3 || c.RowSpan() != 2 {
		t.Errorf("spans = %dx%d, want 3x2", c.ColSpan(), c.RowSpan())
	}
	if c.Align() != AlignRight {
		t.Errorf("Align = %v", c.Align())
	}
	if c.style == nil {
		t.Fatal("options should have populated the cell's Style")
	}
	if c.style.wrap || c.style.set&sWrap == 0 {
		t.Errorf("wrap should be false (set=%v wrap=%v)",
			c.style.set&sWrap != 0, c.style.wrap)
	}
	if c.style.trim || c.style.set&sTrim == 0 {
		t.Errorf("trim should be false (set=%v trim=%v)",
			c.style.set&sTrim != 0, c.style.trim)
	}
	if c.style.maxLines != 5 || c.style.set&sMaxLines == 0 {
		t.Errorf("maxLines should be 5 (set=%v val=%d)",
			c.style.set&sMaxLines != 0, c.style.maxLines)
	}
}

func TestContentAndReaderLastWriterWins(t *testing.T) {
	// Reader applied after content: reader wins.
	tbl := NewTable()
	r := tbl.AddRow()
	c := r.AddCell(
		WithCellID("rc"),
		WithContent("hi"),
		WithReader(strings.NewReader("from reader")),
	)
	if c.reader == nil {
		t.Errorf("reader should win when applied after content")
	}
	if c.hasContent {
		t.Errorf("content flag should be cleared when reader wins")
	}
	// Warning is recorded for the swap.
	var sawReader bool
	for _, w := range tbl.Warnings() {
		ev, ok := w.(ContentSourceReplacedEvent)
		if ok && ev.CellID == "rc" {
			sawReader = true
		}
	}
	if !sawReader {
		t.Errorf("expected ContentSourceReplacedEvent for reader-wins, warnings=%v", tbl.Warnings())
	}

	// Content applied after reader: content wins.
	tbl2 := NewTable()
	r2 := tbl2.AddRow()
	c2 := r2.AddCell(
		WithCellID("cc"),
		WithReader(strings.NewReader("from reader")),
		WithContent("hi"),
	)
	if c2.reader != nil {
		t.Errorf("content should win when applied after reader")
	}
	if !c2.hasContent || c2.Content() != "hi" {
		t.Errorf("content should be set to %q, got %q", "hi", c2.Content())
	}
	var sawContent bool
	for _, w := range tbl2.Warnings() {
		ev, ok := w.(ContentSourceReplacedEvent)
		if ok && ev.CellID == "cc" {
			sawContent = true
		}
	}
	if !sawContent {
		t.Errorf("expected ContentSourceReplacedEvent for content-wins, warnings=%v", tbl2.Warnings())
	}
}

func TestSpanClampsBelowOne(t *testing.T) {
	c0 := NewCell(WithColSpan(0))
	if c0.ColSpan() != 1 {
		t.Errorf("ColSpan(0) should clamp to 1, got %d", c0.ColSpan())
	}
	cNeg := NewCell(WithColSpan(-3))
	if cNeg.ColSpan() != 1 {
		t.Errorf("ColSpan(-3) should clamp to 1, got %d", cNeg.ColSpan())
	}
	r0 := NewCell(WithRowSpan(0))
	if r0.RowSpan() != 1 {
		t.Errorf("RowSpan(0) should clamp to 1, got %d", r0.RowSpan())
	}
	rNeg := NewCell(WithRowSpan(-3))
	if rNeg.RowSpan() != 1 {
		t.Errorf("RowSpan(-3) should clamp to 1, got %d", rNeg.RowSpan())
	}
}

func TestAddRowWithPendingCells(t *testing.T) {
	tbl := NewTable()
	c1 := NewCell(WithCellID("c1"), WithContent("a"))
	c2 := NewCell(WithCellID("c2"), WithContent("b"))

	r := tbl.AddRow(WithRowID("r"), WithCell(c1), WithCell(c2))
	if r.ID() != "r" {
		t.Errorf("row id = %q", r.ID())
	}
	if got := r.Cells(); len(got) != 2 || got[0] != c1 || got[1] != c2 {
		t.Errorf("cells = %v", got)
	}
	if !c1.adopted || !c2.adopted {
		t.Error("pending cells should be marked adopted")
	}
	if c1.GridCol() != 0 || c2.GridCol() != 1 {
		t.Errorf("grid cols = %d, %d", c1.GridCol(), c2.GridCol())
	}
}

func TestAttachCellMigratesBetweenRows(t *testing.T) {
	tbl := NewTable()
	r1 := tbl.AddRow()
	c := NewCell(WithContent("x"))
	r1.AttachCell(c)
	if !c.adopted {
		t.Fatal("cell should be adopted after first attach")
	}
	if len(r1.Cells()) != 1 {
		t.Fatalf("r1 cells after first attach = %d, want 1", len(r1.Cells()))
	}

	// Attaching to a second row should migrate the cell: it leaves r1
	// and lives in r2.
	r2 := tbl.AddRow()
	r2.AttachCell(c)
	if len(r1.Cells()) != 0 {
		t.Errorf("r1 cells after migration = %d, want 0", len(r1.Cells()))
	}
	if len(r2.Cells()) != 1 || r2.Cells()[0] != c {
		t.Errorf("r2 should own the migrated cell, got cells=%v", r2.Cells())
	}
	if !c.adopted {
		t.Error("cell should remain adopted after migration")
	}
}
