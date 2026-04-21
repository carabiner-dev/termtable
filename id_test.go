// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"errors"
	"testing"
)

func TestEmptyIDNotRegistered(t *testing.T) {
	reg := newIDRegistry()
	c := NewCell()
	if err := reg.register("", c); err != nil {
		t.Fatalf("register empty: %v", err)
	}
	if reg.lookup("") != nil {
		t.Error("empty id should never resolve")
	}
}

func TestDuplicateIDRejected(t *testing.T) {
	h := th{t}
	tbl := NewTable()
	r := h.row(tbl.AddRow())
	h.cell(r.AddCell(WithCellID("dup"), WithContent("a")))

	_, err := r.AddCell(WithCellID("dup"), WithContent("b"))
	if !errors.Is(err, ErrDuplicateID) {
		t.Fatalf("second add err = %v, want ErrDuplicateID", err)
	}
	if got := len(r.Cells()); got != 1 {
		t.Errorf("cells after conflict = %d, want 1", got)
	}
}

func TestGetElementByIDTypes(t *testing.T) {
	h := th{t}
	tbl := NewTable(WithTableID("tbl"))

	hd := h.header(tbl.AddHeader(WithRowID("h")))
	hc := h.cell(hd.AddCell(WithCellID("hc"), WithContent("head")))

	r := h.row(tbl.AddRow(WithRowID("r")))
	rc := h.cell(r.AddCell(WithCellID("rc"), WithContent("body")))

	f := h.footer(tbl.AddFooter(WithRowID("f")))
	fc := h.cell(f.AddCell(WithCellID("fc"), WithContent("foot")))

	cases := []struct {
		id   string
		want Element
	}{
		{"tbl", tbl},
		{"h", hd},
		{"hc", hc},
		{"r", r},
		{"rc", rc},
		{"f", f},
		{"fc", fc},
	}
	for _, tc := range cases {
		got := tbl.GetElementByID(tc.id)
		if got != tc.want {
			t.Errorf("GetElementByID(%q) = %v, want %v", tc.id, got, tc.want)
		}
	}
	if tbl.GetElementByID("missing") != nil {
		t.Error("missing id should resolve to nil")
	}
}

func TestGetElementByIDTypeSwitch(t *testing.T) {
	h := th{t}
	tbl := NewTable()
	r := h.row(tbl.AddRow(WithRowID("r1")))
	c := h.cell(r.AddCell(WithCellID("c1"), WithContent("x")))

	switch e := tbl.GetElementByID("r1").(type) {
	case *Row:
		if e != r {
			t.Error("row type switch returned wrong pointer")
		}
	default:
		t.Errorf("row id resolved to unexpected type %T", e)
	}
	switch e := tbl.GetElementByID("c1").(type) {
	case *Cell:
		if e != c {
			t.Error("cell type switch returned wrong pointer")
		}
	default:
		t.Errorf("cell id resolved to unexpected type %T", e)
	}
}
