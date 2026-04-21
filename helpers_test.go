// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import "testing"

// th is a tiny test helper that threads *testing.T through construction
// sites that return (X, error). Usage:
//
//	h := th{t}
//	r := h.row(tbl.AddRow())
//	c := h.cell(r.AddCell(WithContent("x")))
//
// Each method follows Go's multi-value spread rule (a two-return
// expression is spreadable as the sole argument after the receiver), so
// construction stays tight at call sites without discarding errors.
type th struct{ t *testing.T }

func (h th) row(r *Row, err error) *Row {
	h.t.Helper()
	if err != nil {
		h.t.Fatalf("unexpected error: %v", err)
	}
	return r
}

func (h th) header(hd *Header, err error) *Header {
	h.t.Helper()
	if err != nil {
		h.t.Fatalf("unexpected error: %v", err)
	}
	return hd
}

func (h th) footer(f *Footer, err error) *Footer {
	h.t.Helper()
	if err != nil {
		h.t.Fatalf("unexpected error: %v", err)
	}
	return f
}

func (h th) cell(c *Cell, err error) *Cell {
	h.t.Helper()
	if err != nil {
		h.t.Fatalf("unexpected error: %v", err)
	}
	return c
}
