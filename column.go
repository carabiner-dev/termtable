// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

// Column is a virtual element representing a grid column. Columns are
// created automatically as cells populate new column positions; they
// may also be retrieved explicitly via Table.Column. Configuration is
// imperative through Set* methods:
//
//	t.Column(1).SetMax(8).SetAlign(termtable.AlignCenter)
//
// Methods return the receiver so calls may be chained. Unset fields
// are inherited from content measurements or from the layout solver's
// defaults (weight = 1, alignment = AlignLeft).
type Column struct {
	id    string
	index int

	width  int
	minW   int
	maxW   int
	weight float64
	align  Alignment

	set columnField
}

// columnField is a bitmask marking which Column fields were set
// explicitly by the user (as opposed to inherited or defaulted).
type columnField uint8

const (
	cWidth columnField = 1 << iota
	cMin
	cMax
	cWeight
	cAlign
)

func newColumn(index int) *Column {
	return &Column{index: index, weight: 1.0}
}

// ID returns the column's user-assigned ID, or the empty string.
func (c *Column) ID() string { return c.id }

// Index returns the zero-based column index.
func (c *Column) Index() int { return c.index }

func (c *Column) elementID() string { return c.id }

// SetWidth pins the column to exactly n display columns of content
// (not counting padding or borders). Overrides SetMin and SetMax for
// layout purposes. A value of n <= 0 clears the explicit width so the
// solver returns to applying min/max/weight instead.
func (c *Column) SetWidth(n int) *Column {
	if n <= 0 {
		c.set &^= cWidth
		c.width = 0
		return c
	}
	c.width = n
	c.set |= cWidth
	return c
}

// SetMin sets a lower bound on the column's content width. The
// effective minimum is max(contentMinimum, userMinimum), so this
// never shrinks the column below what the content genuinely requires.
// A value of n <= 0 clears the override.
func (c *Column) SetMin(n int) *Column {
	if n <= 0 {
		c.set &^= cMin
		c.minW = 0
		return c
	}
	c.minW = n
	c.set |= cMin
	return c
}

// SetMax caps the column's content width at n. The solver honors the
// cap even when content would naturally prefer more space; content
// wraps to fit. A value of n <= 0 clears the cap.
func (c *Column) SetMax(n int) *Column {
	if n <= 0 {
		c.set &^= cMax
		c.maxW = 0
		return c
	}
	c.maxW = n
	c.set |= cMax
	return c
}

// SetWeight sets the column's share of leftover width after minimums
// and explicit widths are satisfied. Columns with larger weights
// receive proportionally more of the remainder. Default is 1.0 (equal
// share). A weight of 0 prevents the column from absorbing any
// leftover — useful for pinning a column close to its minimum while
// others grow.
func (c *Column) SetWeight(w float64) *Column {
	if w < 0 {
		w = 0
	}
	c.weight = w
	c.set |= cWeight
	return c
}

// SetAlign sets the default horizontal alignment for cells in this
// column. Cells with their own WithAlign override this; cells without
// an explicit alignment inherit it.
func (c *Column) SetAlign(a Alignment) *Column {
	c.align = a
	c.set |= cAlign
	return c
}

// Width returns the pinned width set via SetWidth, or 0 if unset.
func (c *Column) Width() int {
	if c.set&cWidth != 0 {
		return c.width
	}
	return 0
}

// Min returns the user-set minimum width, or 0 if unset.
func (c *Column) Min() int {
	if c.set&cMin != 0 {
		return c.minW
	}
	return 0
}

// Max returns the user-set maximum width, or 0 if unset.
func (c *Column) Max() int {
	if c.set&cMax != 0 {
		return c.maxW
	}
	return 0
}

// Weight returns the column's distribution weight. Defaults to 1.0
// for columns that have not called SetWeight.
func (c *Column) Weight() float64 { return c.weight }

// Align returns the column's alignment override. Check HasAlign to
// distinguish an unset value from an explicit AlignLeft.
func (c *Column) Align() Alignment { return c.align }

// HasAlign reports whether SetAlign has been called on this column.
// Used by the renderer to decide whether column alignment should
// cascade to cells that did not set their own alignment.
func (c *Column) HasAlign() bool { return c.set&cAlign != 0 }
