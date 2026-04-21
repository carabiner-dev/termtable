// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"testing"
)

func TestEmptyTableDimensions(t *testing.T) {
	tbl := NewTable()
	if tbl.NumColumns() != 0 {
		t.Errorf("NumColumns = %d, want 0", tbl.NumColumns())
	}
	if tbl.NumRows() != 0 {
		t.Errorf("NumRows = %d, want 0", tbl.NumRows())
	}
	if tbl.CellAt(0, 0) != nil {
		t.Error("CellAt on empty table should be nil")
	}
	if tbl.InBounds(0, 0) {
		t.Error("InBounds(0,0) on empty table should be false")
	}
}

func TestNumRowsSumAcrossSections(t *testing.T) {
	tbl := NewTable()
	if _, err := tbl.AddHeader(); err != nil {
		t.Fatal(err)
	}
	if _, err := tbl.AddHeader(); err != nil {
		t.Fatal(err)
	}
	if _, err := tbl.AddRow(); err != nil {
		t.Fatal(err)
	}
	if _, err := tbl.AddRow(); err != nil {
		t.Fatal(err)
	}
	if _, err := tbl.AddRow(); err != nil {
		t.Fatal(err)
	}
	if _, err := tbl.AddFooter(); err != nil {
		t.Fatal(err)
	}
	if got := tbl.NumRows(); got != 6 {
		t.Errorf("NumRows = %d, want 6 (2h + 3r + 1f)", got)
	}
}

func TestCellAtAcrossSections(t *testing.T) {
	tbl := NewTable()
	h, _ := tbl.AddHeader()
	hc, _ := h.AddCell(WithCellID("hc"), WithContent("head"))

	r, _ := tbl.AddRow()
	rc, _ := r.AddCell(WithCellID("rc"), WithContent("body"))

	f, _ := tbl.AddFooter()
	fc, _ := f.AddCell(WithCellID("fc"), WithContent("foot"))

	// Absolute row indices: header at 0, body at 1, footer at 2.
	if got := tbl.CellAt(0, 0); got != hc {
		t.Errorf("CellAt(0,0) = %v, want header cell", got)
	}
	if got := tbl.CellAt(1, 0); got != rc {
		t.Errorf("CellAt(1,0) = %v, want body cell", got)
	}
	if got := tbl.CellAt(2, 0); got != fc {
		t.Errorf("CellAt(2,0) = %v, want footer cell", got)
	}
	if tbl.CellAt(3, 0) != nil {
		t.Error("CellAt past last row should be nil")
	}
	if tbl.CellAt(-1, 0) != nil {
		t.Error("CellAt negative should be nil")
	}
}

func TestCellAtSpansMapToSameCell(t *testing.T) {
	tbl := NewTable()
	r, _ := tbl.AddRow()
	c, _ := r.AddCell(WithContent("wide"), WithColSpan(3))
	for col := 0; col < 3; col++ {
		if got := tbl.CellAt(0, col); got != c {
			t.Errorf("CellAt(0,%d) = %v, want %v", col, got, c)
		}
	}
	if tbl.CellAt(0, 3) != nil {
		t.Error("CellAt past span should be nil")
	}
}

func TestInBounds(t *testing.T) {
	tbl := NewTable()
	r, _ := tbl.AddRow()
	if _, err := r.AddCell(WithContent("a")); err != nil {
		t.Fatal(err)
	}
	if _, err := r.AddCell(WithContent("b")); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		r, c int
		want bool
	}{
		{0, 0, true},
		{0, 1, true},
		{0, 2, false},
		{1, 0, false},
		{-1, 0, false},
		{0, -1, false},
	}
	for _, tc := range cases {
		if got := tbl.InBounds(tc.r, tc.c); got != tc.want {
			t.Errorf("InBounds(%d,%d) = %v, want %v", tc.r, tc.c, got, tc.want)
		}
	}
}

func TestResolvedTargetWidth(t *testing.T) {
	tbl := NewTable()
	t.Setenv("COLUMNS", "")
	if got := tbl.ResolvedTargetWidth(); got != defaultTargetWidth {
		t.Errorf("default = %d, want %d", got, defaultTargetWidth)
	}
	t.Setenv("COLUMNS", "42")
	if got := tbl.ResolvedTargetWidth(); got != 42 {
		t.Errorf("COLUMNS = %d, want 42", got)
	}

	explicit := NewTable(WithTargetWidth(120))
	if got := explicit.ResolvedTargetWidth(); got != 120 {
		t.Errorf("explicit = %d, want 120", got)
	}

	t.Setenv("COLUMNS", "garbage")
	if got := tbl.ResolvedTargetWidth(); got != defaultTargetWidth {
		t.Errorf("garbage COLUMNS = %d, want default %d", got, defaultTargetWidth)
	}
}

func TestColumnAutoCreation(t *testing.T) {
	tbl := NewTable()
	r, _ := tbl.AddRow()
	for i := 0; i < 4; i++ {
		if _, err := r.AddCell(WithContent("x")); err != nil {
			t.Fatal(err)
		}
	}
	if got := tbl.NumColumns(); got != 4 {
		t.Errorf("NumColumns = %d, want 4", got)
	}
	for i := 0; i < 4; i++ {
		col := tbl.Column(i)
		if col == nil {
			t.Errorf("Column(%d) is nil", i)
			continue
		}
		if col.Index() != i {
			t.Errorf("Column(%d).Index() = %d", i, col.Index())
		}
	}
}

func TestColumnExplicitCreate(t *testing.T) {
	tbl := NewTable()
	col := tbl.Column(3)
	if col == nil || col.Index() != 3 {
		t.Fatalf("Column(3) = %v", col)
	}
	if tbl.NumColumns() != 4 {
		t.Errorf("NumColumns = %d, want 4 (explicit growth)", tbl.NumColumns())
	}
}

func TestMultiHeaderFooterOrdering(t *testing.T) {
	tbl := NewTable()
	h1, _ := tbl.AddHeader()
	h1.AddCell(WithContent("h1"))
	h2, _ := tbl.AddHeader()
	h2.AddCell(WithContent("h2"))
	r, _ := tbl.AddRow()
	r.AddCell(WithContent("r"))
	f1, _ := tbl.AddFooter()
	f1.AddCell(WithContent("f1"))
	f2, _ := tbl.AddFooter()
	f2.AddCell(WithContent("f2"))

	if tbl.NumRows() != 5 {
		t.Fatalf("NumRows = %d, want 5", tbl.NumRows())
	}
	// Expected absolute row order: h1, h2, r, f1, f2.
	wantContents := []string{"h1", "h2", "r", "f1", "f2"}
	for i, want := range wantContents {
		c := tbl.CellAt(i, 0)
		if c == nil {
			t.Errorf("CellAt(%d,0) nil", i)
			continue
		}
		if c.Content() != want {
			t.Errorf("row %d content = %q, want %q", i, c.Content(), want)
		}
	}
}
