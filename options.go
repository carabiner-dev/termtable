// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"io"
	"strings"
)

// TableOption configures a *Table.
type TableOption func(*Table)

// RowOption configures a row being added via AddRow, AddHeader, or
// AddFooter. Internally it operates on the shared rowBody; users never
// interact with rowBody directly.
type RowOption func(*rowBody)

// CellOption configures a *Cell during NewCell, AddCell, or AttachCell.
type CellOption func(*Cell)

// ---------------------------------------------------------------------
// Table options
// ---------------------------------------------------------------------

// WithTableID assigns a unique ID to the table itself, retrievable via
// Table.GetElementByID.
func WithTableID(id string) TableOption {
	return func(t *Table) { t.id = id }
}

// WithTargetWidth pins the layout target width to w terminal columns.
// When unset, the table reads the COLUMNS environment variable, then
// falls back to 80.
func WithTargetWidth(w int) TableOption {
	return func(t *Table) {
		t.opts.targetWidth = w
		t.opts.targetWidthSet = true
	}
}

// WithBorder replaces the table's border glyph set. Defaults to
// DefaultSingleLine.
func WithBorder(b BorderSet) TableOption {
	return func(t *Table) { t.opts.border = b }
}

// WithSpanOverwrite controls span-conflict behavior. The default
// (true) lets later cells overwrite earlier spans: fully-covered
// cells are dropped and partially overlapped cells are truncated,
// with an OverwriteEvent recorded on Table.Warnings for each. This
// means AddCell and AttachCell never fail for layout reasons under
// default settings — conflicts are absorbed.
//
// Passing false switches to strict mode: a colliding span is an
// author error. AddCell / AttachCell panic with a wrapped
// ErrSpanConflict, and the new cell is not placed. Callers who
// want explicit error handling instead of panics can use the
// AddCellWithError / AttachCellWithError methods on the row.
func WithSpanOverwrite(enable bool) TableOption {
	return func(t *Table) { t.opts.spanOverwrite = enable }
}

// WithTablePadding overrides the table-wide cell padding. Padding is
// uniform across every cell — configuring it per-cell would let
// columns misalign. Default is DefaultPadding() (one column of
// horizontal padding, no vertical).
func WithTablePadding(p Padding) TableOption {
	return func(t *Table) { t.opts.padding = p }
}

// WithEmojiWidth pins the emoji-width counting mode for the table.
// The default (EmojiWidthAuto) picks EmojiWidthConservative unless
// termtable detects a terminal known to render composite emoji
// correctly, in which case it picks EmojiWidthGrapheme. The
// TERMTABLE_EMOJI_WIDTH environment variable overrides the
// detection; an explicit non-auto value here overrides both.
//
// See EmojiWidthMode for the semantics.
func WithEmojiWidth(mode EmojiWidthMode) TableOption {
	return func(t *Table) { t.opts.emojiWidth = mode }
}

// WithTableStyle sets table-wide style defaults via a CSS-like
// declaration block, e.g.
//
//	WithTableStyle("color: white; background: blue; border-style: double; border-color: cyan")
//
// Supported properties:
//
//   - color, background (background-color), border-color: color values
//     as named colors ("red", "bright-cyan"), hex ("#rrggbb"), or
//     rgb(r,g,b).
//   - font-weight: bold | normal
//   - font-style: italic | normal
//   - text-decoration: underline | line-through | none
//   - border-style: single | double | heavy | rounded | ascii | none —
//     selects the BorderSet used for the table, equivalent to calling
//     WithBorder with the corresponding constructor (SingleLine,
//     DoubleLine, HeavyLine, RoundedLine, ASCIILine, NoBorder).
//
// Unknown properties and unrecognized values are silently ignored.
func WithTableStyle(css string) TableOption {
	return func(t *Table) {
		iterateCSS(css, func(prop, val string) {
			if prop == "border-style" {
				if b, ok := borderSetByName(strings.ToLower(val)); ok {
					t.opts.border = b
				}
				return
			}
			if t.style == nil {
				t.style = &Style{}
			}
			applyDecl(t.style, prop, val)
		})
	}
}

// ---------------------------------------------------------------------
// Row / Header / Footer options
// ---------------------------------------------------------------------

// WithRowID assigns a unique ID to the row being added.
func WithRowID(id string) RowOption {
	return func(r *rowBody) { r.id = id }
}

// WithCell queues a previously constructed cell for adoption into the
// row. Multiple WithCell options may be supplied; they are attached in
// the order given after the row itself has been inserted.
func WithCell(c *Cell) RowOption {
	return func(r *rowBody) {
		r.pendingCells = append(r.pendingCells, c)
	}
}

// WithRowStyle sets a style that applies to every cell in this row
// unless the cell overrides the corresponding properties. See
// WithTableStyle for the supported CSS property grammar.
func WithRowStyle(css string) RowOption {
	return func(r *rowBody) {
		if r.style == nil {
			r.style = &Style{}
		}
		parseCSS(css, r.style)
	}
}

// ---------------------------------------------------------------------
// Cell options
// ---------------------------------------------------------------------

// WithCellID assigns a unique ID to the cell.
func WithCellID(id string) CellOption {
	return func(c *Cell) { c.id = id }
}

// WithContent sets the cell's textual content. Honors "\n" as a hard
// line break; combines with automatic wrapping when the cell is
// wider than its assigned column width. If a reader source was
// previously set on the cell via WithReader, it is discarded (a
// ContentSourceReplacedEvent warning is emitted when the cell is
// attached to a row).
func WithContent(s string) CellOption {
	return func(c *Cell) {
		if c.reader != nil {
			c.reader = nil
			c.resolved = false
			c.resolveErr = nil
			c.contentSourceSwapped = true
		}
		c.content = s
		c.hasContent = true
	}
}

// WithReader sets the cell's content source to an io.Reader
// consumed lazily on the first render pass. If a string source was
// previously set on the cell via WithContent, it is discarded (a
// ContentSourceReplacedEvent warning is emitted when the cell is
// attached to a row).
func WithReader(r io.Reader) CellOption {
	return func(c *Cell) {
		if c.hasContent {
			c.content = ""
			c.hasContent = false
			c.contentSourceSwapped = true
		}
		c.reader = r
		c.resolved = false
		c.resolveErr = nil
	}
}

// WithColSpan sets the number of columns the cell occupies. Values
// of n <= 0 clamp to 1 (the default) so the option never produces
// an invalid span.
func WithColSpan(n int) CellOption {
	return func(c *Cell) {
		if n < 1 {
			n = 1
		}
		c.colSpan = n
	}
}

// WithRowSpan sets the number of rows the cell occupies within its
// section. Values of n <= 0 clamp to 1 (the default). Rowspans that
// would extend past the last row of the section are clamped by the
// renderer and a CrossSectionSpanEvent is emitted.
func WithRowSpan(n int) CellOption {
	return func(c *Cell) {
		if n < 1 {
			n = 1
		}
		c.rowSpan = n
	}
}

// WithAlign sets the cell's horizontal alignment. Default is AlignLeft.
// Stored on the cell's Style so it participates in the table → column
// → row → cell cascade; cells that never call WithAlign inherit from
// their row, then column, then table, defaulting to AlignLeft.
func WithAlign(a Alignment) CellOption {
	return func(c *Cell) {
		ensureStyle(c).align = a
		c.style.set |= sAlign
	}
}

// WithVAlign sets the cell's vertical alignment within its row
// (which may be taller than the cell's own wrapped content when a
// neighbour wrapped to more lines). Default is VAlignTop. Like
// WithAlign, the value cascades via Style — cells without an
// explicit vertical alignment inherit row, column, and table
// defaults in that order.
func WithVAlign(v VerticalAlignment) CellOption {
	return func(c *Cell) {
		ensureStyle(c).valign = v
		c.style.set |= sVAlign
	}
}

// WithWrap toggles automatic word-wrapping on whitespace. Default is
// true (multi-line). Equivalent to setting CSS
// white-space: normal (wrap=true) or nowrap (wrap=false).
// Participates in the Style cascade — a row or column setting the
// same property forces every inheriting cell.
func WithWrap(enable bool) CellOption {
	return func(c *Cell) {
		ensureStyle(c).wrap = enable
		c.style.set |= sWrap
	}
}

// WithTrim toggles ellipsis-based trimming when a cell's content
// must be cut (either because single-line content overflows the
// column, or because a line-clamp limit was exceeded). Default is
// true. Equivalent to CSS text-overflow: ellipsis (trim=true) or
// clip (trim=false).
func WithTrim(enable bool) CellOption {
	return func(c *Cell) {
		ensureStyle(c).trim = enable
		c.style.set |= sTrim
	}
}

// WithSingleLine is a shorthand for WithWrap(false). Long content
// renders on one line; if trim is enabled (the default) it is
// truncated with an ellipsis.
func WithSingleLine() CellOption { return WithWrap(false) }

// WithMultiLine is a shorthand for WithWrap(true). Useful when a
// row or column has forced single-line mode and a particular cell
// needs to opt back into wrapping.
func WithMultiLine() CellOption { return WithWrap(true) }

// WithMaxLines caps the cell's wrapped content to at most n lines.
// A value of 0 means unbounded (the default). Equivalent to CSS
// line-clamp: N. When the limit fires and trim is enabled, the
// final kept line ends in an ellipsis.
func WithMaxLines(n int) CellOption {
	return func(c *Cell) {
		ensureStyle(c).maxLines = n
		c.style.set |= sMaxLines
	}
}

// WithTrimPosition controls where the ellipsis (or clip) lands when
// a cell's content must be truncated to fit. Default is TrimEnd —
// the content's prefix is kept and the marker sits at the right
// edge. TrimStart keeps the suffix (marker on the left), TrimMiddle
// keeps both ends. Equivalent to termtable's CSS extension
// text-overflow-position: end | start | middle.
//
// This only affects horizontal single-line truncation (wrap=false,
// or the last line of a line-clamped multi-line cell when it
// doesn't fit its column width). Vertical dropping under
// line-clamp always happens from the end.
func WithTrimPosition(pos TrimPosition) CellOption {
	return func(c *Cell) {
		ensureStyle(c).trimPosition = pos
		c.style.set |= sTrimPos
	}
}

// WithCellStyle sets style properties on the cell, cascaded over the
// row's and table's style. See WithTableStyle for the CSS grammar.
// Convenience options WithTextColor, WithBackgroundColor, WithBold,
// WithItalic, WithUnderline, and WithStrikethrough set individual
// properties and may be combined with WithCellStyle.
func WithCellStyle(css string) CellOption {
	return func(c *Cell) {
		if c.style == nil {
			c.style = &Style{}
		}
		parseCSS(css, c.style)
	}
}

// WithTextColor sets the cell's foreground color. Accepts named
// colors, a hex string, or rgb(r,g,b). Unrecognized values are
// ignored.
func WithTextColor(value string) CellOption {
	return func(c *Cell) {
		attrs, ok := parseFgColor(value)
		if !ok {
			return
		}
		ensureStyle(c).fgAttrs = attrs
		c.style.set |= sFg
	}
}

// WithBackgroundColor sets the cell's background color. Accepts the
// same value grammar as WithTextColor.
func WithBackgroundColor(value string) CellOption {
	return func(c *Cell) {
		attrs, ok := parseBgColor(value)
		if !ok {
			return
		}
		ensureStyle(c).bgAttrs = attrs
		c.style.set |= sBg
	}
}

// WithBold enables the bold text attribute on the cell.
func WithBold() CellOption {
	return func(c *Cell) {
		ensureStyle(c).bold = true
		c.style.set |= sBold
	}
}

// WithItalic enables the italic text attribute on the cell.
func WithItalic() CellOption {
	return func(c *Cell) {
		ensureStyle(c).italic = true
		c.style.set |= sItalic
	}
}

// WithUnderline enables the underline text attribute on the cell.
func WithUnderline() CellOption {
	return func(c *Cell) {
		ensureStyle(c).underline = true
		c.style.set |= sUnderline
	}
}

// WithStrikethrough enables the line-through text attribute on the
// cell. (Not every terminal renders this; supported by most modern
// emulators.)
func WithStrikethrough() CellOption {
	return func(c *Cell) {
		ensureStyle(c).strike = true
		c.style.set |= sStrike
	}
}

// ensureStyle returns c.style, creating a fresh Style if the cell
// does not yet have one.
func ensureStyle(c *Cell) *Style {
	if c.style == nil {
		c.style = &Style{}
	}
	return c.style
}

// Column configuration is imperative: retrieve a column via
// Table.Column(i) and call its Set* methods or Style(css).
