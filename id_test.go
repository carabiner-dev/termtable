// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"testing"
)

func TestEmptyIDNotRegistered(t *testing.T) {
	reg := newIDRegistry()
	c := NewCell()
	if ok := reg.register("", c); !ok {
		t.Fatalf("register empty returned false")
	}
	if reg.lookup("") != nil {
		t.Error("empty id should never resolve")
	}
}

func TestDuplicateIDWarnsAndClears(t *testing.T) {
	tbl := NewTable()
	r := tbl.AddRow()
	first := r.AddCell(WithCellID("dup"), WithContent("a"))
	second := r.AddCell(WithCellID("dup"), WithContent("b"))

	// Both cells are attached; the duplicate has its ID cleared.
	if got := len(r.Cells()); got != 2 {
		t.Errorf("cells after duplicate = %d, want 2", got)
	}
	if first.ID() != "dup" {
		t.Errorf("first cell ID = %q, want %q", first.ID(), "dup")
	}
	if second.ID() != "" {
		t.Errorf("second cell ID = %q, want cleared", second.ID())
	}
	// The original cell remains resolvable by the ID.
	if tbl.GetElementByID("dup") != first {
		t.Errorf("GetElementByID(dup) should resolve to first cell")
	}
	// A DuplicateIDEvent must be recorded.
	var saw bool
	for _, w := range tbl.Warnings() {
		ev, ok := w.(DuplicateIDEvent)
		if ok && ev.ID == "dup" && ev.Kind == "cell" {
			saw = true
		}
	}
	if !saw {
		t.Errorf("expected DuplicateIDEvent, got warnings=%v", tbl.Warnings())
	}
}

func TestGetElementByIDTypes(t *testing.T) {
	tbl := NewTable(WithTableID("tbl"))

	hd := tbl.AddHeader(WithRowID("h"))
	hc := hd.AddCell(WithCellID("hc"), WithContent("head"))

	r := tbl.AddRow(WithRowID("r"))
	rc := r.AddCell(WithCellID("rc"), WithContent("body"))

	f := tbl.AddFooter(WithRowID("f"))
	fc := f.AddCell(WithCellID("fc"), WithContent("foot"))

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
	tbl := NewTable()
	r := tbl.AddRow(WithRowID("r1"))
	c := r.AddCell(WithCellID("c1"), WithContent("x"))

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
