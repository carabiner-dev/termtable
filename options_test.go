// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"errors"
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

func TestContentAndReaderMutex(t *testing.T) {
	h := th{t}
	tbl := NewTable()
	r := h.row(tbl.AddRow())
	_, err := r.AddCell(
		WithContent("hi"),
		WithReader(strings.NewReader("also hi")),
	)
	if !errors.Is(err, ErrContentAndReader) {
		t.Fatalf("err = %v, want ErrContentAndReader", err)
	}
}

func TestInvalidSpan(t *testing.T) {
	h := th{t}
	tbl := NewTable()
	r := h.row(tbl.AddRow())
	if _, err := r.AddCell(WithColSpan(0)); !errors.Is(err, ErrInvalidSpan) {
		t.Errorf("colSpan=0: err = %v, want ErrInvalidSpan", err)
	}
	if _, err := r.AddCell(WithRowSpan(0)); !errors.Is(err, ErrInvalidSpan) {
		t.Errorf("rowSpan=0: err = %v, want ErrInvalidSpan", err)
	}
}

func TestAddRowWithPendingCells(t *testing.T) {
	h := th{t}
	tbl := NewTable()
	c1 := NewCell(WithCellID("c1"), WithContent("a"))
	c2 := NewCell(WithCellID("c2"), WithContent("b"))

	r := h.row(tbl.AddRow(WithRowID("r"), WithCell(c1), WithCell(c2)))
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

func TestAttachCellTwiceFails(t *testing.T) {
	h := th{t}
	tbl := NewTable()
	r1 := h.row(tbl.AddRow())
	c := NewCell(WithContent("x"))
	if _, err := r1.AttachCell(c); err != nil {
		t.Fatalf("AttachCell: %v", err)
	}
	r2 := h.row(tbl.AddRow())
	if _, err := r2.AttachCell(c); !errors.Is(err, ErrCellAlreadyAdopted) {
		t.Errorf("second attach: err = %v, want ErrCellAlreadyAdopted", err)
	}
}
