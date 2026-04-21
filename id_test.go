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
	tbl := NewTable()
	r, _ := tbl.AddRow()
	if _, err := r.AddCell(WithCellID("dup"), WithContent("a")); err != nil {
		t.Fatalf("first add: %v", err)
	}
	_, err := r.AddCell(WithCellID("dup"), WithContent("b"))
	if !errors.Is(err, ErrDuplicateID) {
		t.Fatalf("second add err = %v, want ErrDuplicateID", err)
	}
	// Row should still have exactly one cell; the failing add must not
	// leak grid state.
	if got := len(r.Cells()); got != 1 {
		t.Errorf("cells after conflict = %d, want 1", got)
	}
}

func TestGetElementByIDTypes(t *testing.T) {
	tbl := NewTable(WithTableID("tbl"))

	h, err := tbl.AddHeader(WithRowID("h"))
	if err != nil {
		t.Fatalf("AddHeader: %v", err)
	}
	hc, err := h.AddCell(WithCellID("hc"), WithContent("head"))
	if err != nil {
		t.Fatalf("header cell: %v", err)
	}

	r, err := tbl.AddRow(WithRowID("r"))
	if err != nil {
		t.Fatalf("AddRow: %v", err)
	}
	rc, err := r.AddCell(WithCellID("rc"), WithContent("body"))
	if err != nil {
		t.Fatalf("row cell: %v", err)
	}

	f, err := tbl.AddFooter(WithRowID("f"))
	if err != nil {
		t.Fatalf("AddFooter: %v", err)
	}
	fc, err := f.AddCell(WithCellID("fc"), WithContent("foot"))
	if err != nil {
		t.Fatalf("footer cell: %v", err)
	}

	cases := []struct {
		id   string
		want Element
	}{
		{"tbl", tbl},
		{"h", h},
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
	tbl := NewTable()
	r, _ := tbl.AddRow(WithRowID("r1"))
	c, _ := r.AddCell(WithCellID("c1"), WithContent("x"))

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
