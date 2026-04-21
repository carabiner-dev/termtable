// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

// Column is a virtual element representing a grid column. Columns are
// created automatically as cells populate new column positions; they may
// also be retrieved explicitly via Table.Column. Per-column configuration
// (width, alignment override, styling) is reserved for a later phase; the
// type exists now so the API shape is stable.
type Column struct {
	id    string
	index int
	opts  columnOptions
}

type columnOptions struct{}

func newColumn(index int) *Column {
	return &Column{index: index}
}

// ID returns the column's user-assigned ID, or the empty string.
func (c *Column) ID() string { return c.id }

// Index returns the zero-based column index.
func (c *Column) Index() int { return c.index }

func (c *Column) elementID() string { return c.id }
