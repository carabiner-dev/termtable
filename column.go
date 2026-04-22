// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"strconv"
	"strings"
)

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

	// table is the owning Table. Set in newColumn via Table.growColumnTo.
	// Retained so SetID can register the column with the table's ID
	// registry.
	table *Table

	width  int
	minW   int
	maxW   int
	weight float64

	// style is the column's default style. Alignment (text-align),
	// color, background, and other attributes set here cascade to the
	// cells that occupy this column.
	style *Style

	set columnField
}

// columnField is a bitmask marking which sizing fields of a Column
// were set explicitly. Style-related fields (alignment, color, etc.)
// live on the column's Style and use the Style's own bitmask.
type columnField uint8

const (
	cWidth columnField = 1 << iota
	cMin
	cMax
	cWeight
)

func newColumn(index int) *Column {
	return &Column{index: index, weight: 1.0}
}

// ID returns the column's user-assigned ID, or the empty string.
func (c *Column) ID() string { return c.id }

// SetID assigns the column an ID that can be resolved via
// Table.GetElementByID. Passing an empty string unsets a previously
// assigned ID. Returns an error only if the new ID collides with an
// existing element's ID in the same table.
func (c *Column) SetID(id string) error {
	if c.table == nil {
		c.id = id
		return nil
	}
	if c.id != "" {
		c.table.registry.unregister(c.id)
	}
	if id == "" {
		c.id = ""
		return nil
	}
	if err := c.table.registry.register(id, c); err != nil {
		// Re-register the old id on failure so state stays consistent.
		// The old id was registered before; re-registering the same
		// (id, element) pair cannot fail because the registry treats
		// an identical mapping as a no-op.
		if c.id != "" {
			c.table.registry.m[c.id] = c
		}
		return err
	}
	c.id = id
	return nil
}

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
// an explicit alignment, and rows without a row-level text-align,
// inherit from the column.
func (c *Column) SetAlign(a Alignment) *Column {
	if c.style == nil {
		c.style = &Style{}
	}
	c.style.align = a
	c.style.set |= sAlign
	return c
}

// SetVAlign sets the default vertical alignment for cells in this
// column. Cells with their own WithVAlign override this; otherwise
// the column's value participates in the table → column → row →
// cell cascade.
func (c *Column) SetVAlign(v VerticalAlignment) *Column {
	if c.style == nil {
		c.style = &Style{}
	}
	c.style.valign = v
	c.style.set |= sVAlign
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
func (c *Column) Align() Alignment {
	if c.style != nil && c.style.set&sAlign != 0 {
		return c.style.align
	}
	return AlignLeft
}

// HasAlign reports whether the column has an alignment override in
// force, either from Column.SetAlign or from Column.Style with
// text-align.
func (c *Column) HasAlign() bool {
	return c.style != nil && c.style.set&sAlign != 0
}

// Style parses a CSS-like declaration block and applies it to the
// column. Sizing properties route to the imperative Set* methods
// (so Column.Style and Column.SetWidth are interchangeable); style
// properties populate a column-level Style that cascades to every
// cell in the column.
//
// Supported sizing properties:
//
//	width: N           pins the column to exactly N content columns
//	min-width: N       lower bound on content width
//	max-width: N       upper bound on content width
//	flex: N            weight for distributing leftover budget
//	text-align: L|C|R  default alignment for cells in the column
//
// Style properties are the same set accepted by WithTableStyle
// (color, background, font-weight, font-style, text-decoration).
// border-color at column level is ignored — border glyphs are
// table-wide and configured via WithTableStyle.
//
// Unrecognized properties and unparseable values are silently ignored.
func (c *Column) Style(css string) *Column {
	iterateCSS(css, c.applyCSSDecl)
	return c
}

func (c *Column) applyCSSDecl(prop, val string) {
	switch prop {
	case "width":
		if n, ok := parsePositiveInt(val); ok {
			c.SetWidth(n)
		}
	case "min-width":
		if n, ok := parsePositiveInt(val); ok {
			c.SetMin(n)
		}
	case "max-width":
		if n, ok := parsePositiveInt(val); ok {
			c.SetMax(n)
		}
	case "flex":
		if w, ok := parseNonNegFloat(val); ok {
			c.SetWeight(w)
		}
	case "border-color":
		// Intentionally ignored: border glyphs are table-wide and not
		// per-column. Documented in Column.Style.
	default:
		// Fall through to the style parser for color, background,
		// font-weight, text-align, etc. Column.SetAlign routes
		// through the same storage so text-align here is equivalent
		// to calling SetAlign directly.
		if c.style == nil {
			c.style = &Style{}
		}
		applyDecl(c.style, prop, val)
	}
}

func parsePositiveInt(s string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 1 {
		return 0, false
	}
	return n, true
}

func parseNonNegFloat(s string) (float64, bool) {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || f < 0 {
		return 0, false
	}
	return f, true
}
