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
	if !c.opts.wrap {
		t.Error("wrap default should be true")
	}
	if !c.opts.trim {
		t.Error("trim default should be true")
	}
	if c.opts.padding != DefaultPadding() {
		t.Errorf("padding = %+v, want %+v", c.opts.padding, DefaultPadding())
	}
	if c.opts.maxLines != 0 {
		t.Errorf("maxLines default = %d, want 0 (unbounded)", c.opts.maxLines)
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
		WithPadding(Padding{Left: 2, Right: 2}),
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
	if c.opts.wrap || c.opts.trim {
		t.Error("wrap/trim should be false")
	}
	if c.opts.padding.Left != 2 || c.opts.padding.Right != 2 {
		t.Errorf("padding = %+v", c.opts.padding)
	}
	if c.opts.maxLines != 5 {
		t.Errorf("maxLines = %d", c.opts.maxLines)
	}
}

func TestContentAndReaderMutex(t *testing.T) {
	tbl := NewTable()
	r, err := tbl.AddRow()
	if err != nil {
		t.Fatalf("AddRow: %v", err)
	}
	_, err = r.AddCell(
		WithContent("hi"),
		WithReader(strings.NewReader("also hi")),
	)
	if !errors.Is(err, ErrContentAndReader) {
		t.Fatalf("err = %v, want ErrContentAndReader", err)
	}
}

func TestInvalidSpan(t *testing.T) {
	tbl := NewTable()
	r, _ := tbl.AddRow()
	if _, err := r.AddCell(WithColSpan(0)); !errors.Is(err, ErrInvalidSpan) {
		t.Errorf("colSpan=0: err = %v, want ErrInvalidSpan", err)
	}
	if _, err := r.AddCell(WithRowSpan(0)); !errors.Is(err, ErrInvalidSpan) {
		t.Errorf("rowSpan=0: err = %v, want ErrInvalidSpan", err)
	}
}

func TestAddRowWithPendingCells(t *testing.T) {
	tbl := NewTable()
	c1 := NewCell(WithCellID("c1"), WithContent("a"))
	c2 := NewCell(WithCellID("c2"), WithContent("b"))

	r, err := tbl.AddRow(WithRowID("r"), WithCell(c1), WithCell(c2))
	if err != nil {
		t.Fatalf("AddRow: %v", err)
	}
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
	tbl := NewTable()
	r1, _ := tbl.AddRow()
	c := NewCell(WithContent("x"))
	if _, err := r1.AttachCell(c); err != nil {
		t.Fatalf("AttachCell: %v", err)
	}
	r2, _ := tbl.AddRow()
	if _, err := r2.AttachCell(c); !errors.Is(err, ErrCellAlreadyAdopted) {
		t.Errorf("second attach: err = %v, want ErrCellAlreadyAdopted", err)
	}
}
