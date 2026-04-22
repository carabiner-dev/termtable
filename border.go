// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

// BorderSet is the glyph set used to draw table borders. The Joins
// array is indexed by a four-bit arm mask with bit layout
// (least-significant first): N=1, E=2, S=4, W=8. Only the 11 valid
// entries (runs, corners, T-joins, crosses) are consulted during
// rendering; the remaining entries are left zero.
//
// termtable ships several ready-made sets (SingleLine, DoubleLine,
// HeavyLine, RoundedLine, ASCIILine, NoBorder). Callers may also
// construct a BorderSet manually — see the ASCIILine source for the
// full shape.
type BorderSet struct {
	// Horizontal is the glyph repeated along horizontal border runs.
	Horizontal rune
	// Vertical is the glyph repeated along vertical border runs.
	Vertical rune
	// Joins maps arm bitmasks to the glyph drawn at corners and
	// junctions.
	Joins [16]rune
}

// Arm bits for BorderSet.Joins.
const (
	armN = 1 << 0 // north (up)
	armE = 1 << 1 // east (right)
	armS = 1 << 2 // south (down)
	armW = 1 << 3 // west (left)
)

// SingleLine returns the Unicode single-line box-drawing BorderSet:
// ─ │ ┌ ┐ └ ┘ ├ ┤ ┬ ┴ ┼ . This is the default used by NewTable
// when no WithBorder option is supplied.
func SingleLine() BorderSet {
	var b BorderSet
	b.Horizontal = '─'
	b.Vertical = '│'
	b.Joins[armN|armS] = '│'
	b.Joins[armE|armW] = '─'
	b.Joins[armS|armE] = '┌'
	b.Joins[armS|armW] = '┐'
	b.Joins[armN|armE] = '└'
	b.Joins[armN|armW] = '┘'
	b.Joins[armN|armS|armE] = '├'
	b.Joins[armN|armS|armW] = '┤'
	b.Joins[armE|armS|armW] = '┬'
	b.Joins[armN|armE|armW] = '┴'
	b.Joins[armN|armE|armS|armW] = '┼'
	return b
}

// DefaultSingleLine is an alias for SingleLine retained for
// compatibility with Phase 1 code. New callers should use SingleLine
// directly.
//
// Deprecated: use SingleLine.
func DefaultSingleLine() BorderSet { return SingleLine() }

// DoubleLine returns the Unicode double-line BorderSet:
// ═ ║ ╔ ╗ ╚ ╝ ╠ ╣ ╦ ╩ ╬ . Useful for emphasis or to visually
// distinguish outer borders from any future inner styling.
func DoubleLine() BorderSet {
	var b BorderSet
	b.Horizontal = '═'
	b.Vertical = '║'
	b.Joins[armN|armS] = '║'
	b.Joins[armE|armW] = '═'
	b.Joins[armS|armE] = '╔'
	b.Joins[armS|armW] = '╗'
	b.Joins[armN|armE] = '╚'
	b.Joins[armN|armW] = '╝'
	b.Joins[armN|armS|armE] = '╠'
	b.Joins[armN|armS|armW] = '╣'
	b.Joins[armE|armS|armW] = '╦'
	b.Joins[armN|armE|armW] = '╩'
	b.Joins[armN|armE|armS|armW] = '╬'
	return b
}

// HeavyLine returns the Unicode heavy (bold) box-drawing BorderSet:
// ━ ┃ ┏ ┓ ┗ ┛ ┣ ┫ ┳ ┻ ╋ .
func HeavyLine() BorderSet {
	var b BorderSet
	b.Horizontal = '━'
	b.Vertical = '┃'
	b.Joins[armN|armS] = '┃'
	b.Joins[armE|armW] = '━'
	b.Joins[armS|armE] = '┏'
	b.Joins[armS|armW] = '┓'
	b.Joins[armN|armE] = '┗'
	b.Joins[armN|armW] = '┛'
	b.Joins[armN|armS|armE] = '┣'
	b.Joins[armN|armS|armW] = '┫'
	b.Joins[armE|armS|armW] = '┳'
	b.Joins[armN|armE|armW] = '┻'
	b.Joins[armN|armE|armS|armW] = '╋'
	return b
}

// RoundedLine returns a BorderSet using single-line runs and joins
// but rounded outer corners (╭ ╮ ╰ ╯). Unicode has no rounded
// equivalents for T-joins or the full cross, so those remain the
// standard single-line glyphs.
func RoundedLine() BorderSet {
	b := SingleLine()
	b.Joins[armS|armE] = '╭'
	b.Joins[armS|armW] = '╮'
	b.Joins[armN|armE] = '╰'
	b.Joins[armN|armW] = '╯'
	return b
}

// ASCIILine returns an ASCII-only BorderSet (- | +) suitable for
// environments that cannot render Unicode box-drawing characters —
// logs, legacy terminals, email clients. All T-joins, corners, and
// crosses render as '+'.
func ASCIILine() BorderSet {
	var b BorderSet
	b.Horizontal = '-'
	b.Vertical = '|'
	b.Joins[armN|armS] = '|'
	b.Joins[armE|armW] = '-'
	b.Joins[armS|armE] = '+'
	b.Joins[armS|armW] = '+'
	b.Joins[armN|armE] = '+'
	b.Joins[armN|armW] = '+'
	b.Joins[armN|armS|armE] = '+'
	b.Joins[armN|armS|armW] = '+'
	b.Joins[armE|armS|armW] = '+'
	b.Joins[armN|armE|armW] = '+'
	b.Joins[armN|armE|armS|armW] = '+'
	return b
}

// NoBorder returns a BorderSet whose glyphs are all U+0020 (space).
// The resulting table has no visible dividers but preserves the
// horizontal and vertical spacing, producing an invisibly-gridded
// layout. Combine with WithTablePadding(Padding{}) to collapse
// padding as well.
func NoBorder() BorderSet {
	var b BorderSet
	b.Horizontal = ' '
	b.Vertical = ' '
	b.Joins[armN|armS] = ' '
	b.Joins[armE|armW] = ' '
	b.Joins[armS|armE] = ' '
	b.Joins[armS|armW] = ' '
	b.Joins[armN|armE] = ' '
	b.Joins[armN|armW] = ' '
	b.Joins[armN|armS|armE] = ' '
	b.Joins[armN|armS|armW] = ' '
	b.Joins[armE|armS|armW] = ' '
	b.Joins[armN|armE|armW] = ' '
	b.Joins[armN|armE|armS|armW] = ' '
	return b
}

// borderSetByName resolves a CSS border-style keyword to its
// BorderSet constructor. Returns the zero value and false for
// unknown names. Note that CSS's "border-style: none" is NOT
// handled here — it sets the table's default edge directive to
// BorderEdgeNone rather than swapping in a BorderSet. Callers that
// want the spaces-everywhere behavior should use "hidden".
func borderSetByName(name string) (BorderSet, bool) {
	switch name {
	case "single":
		return SingleLine(), true
	case "double":
		return DoubleLine(), true
	case "heavy":
		return HeavyLine(), true
	case "rounded":
		return RoundedLine(), true
	case "ascii":
		return ASCIILine(), true
	case cssHidden:
		return NoBorder(), true
	}
	return BorderSet{}, false
}
