// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"strconv"
	"strings"

	"github.com/fatih/color"
)

// Style collects the visual formatting attributes that can be applied
// to a Table, row, or Cell. Each field is optional; unset fields are
// inherited from an enclosing element via style merge. Display-width
// math is unaffected by styling — attributes are emitted as ANSI
// escape sequences that contribute zero terminal columns.
//
// When cell content already contains its own ANSI escape sequences
// and a background is also set via this Style, the content's inner
// reset sequence will drop the outer background. Use one or the
// other to avoid artifacts.
type Style struct {
	fgAttrs     []color.Attribute
	bgAttrs     []color.Attribute
	borderAttrs []color.Attribute

	align     Alignment
	valign    VerticalAlignment
	bold      bool
	italic    bool
	underline bool
	strike    bool

	set styleField
}

// styleField is a bitmask indicating which Style fields have been set
// (as opposed to inherited or defaulted).
type styleField uint32

const (
	sFg styleField = 1 << iota
	sBg
	sBorder
	sBold
	sItalic
	sUnderline
	sStrike
	sAlign
	sVAlign
)

// isEmpty reports whether s contributes nothing to output.
func (s *Style) isEmpty() bool {
	return s == nil || s.set == 0
}

// merge overlays src onto s: any field that src has set replaces s's.
// Used to compute an effective style by cascading table → row → cell.
func (s *Style) merge(src *Style) {
	if src == nil || src.set == 0 {
		return
	}
	if src.set&sFg != 0 {
		s.fgAttrs = src.fgAttrs
		s.set |= sFg
	}
	if src.set&sBg != 0 {
		s.bgAttrs = src.bgAttrs
		s.set |= sBg
	}
	if src.set&sBorder != 0 {
		s.borderAttrs = src.borderAttrs
		s.set |= sBorder
	}
	if src.set&sBold != 0 {
		s.bold = src.bold
		s.set |= sBold
	}
	if src.set&sItalic != 0 {
		s.italic = src.italic
		s.set |= sItalic
	}
	if src.set&sUnderline != 0 {
		s.underline = src.underline
		s.set |= sUnderline
	}
	if src.set&sStrike != 0 {
		s.strike = src.strike
		s.set |= sStrike
	}
	if src.set&sAlign != 0 {
		s.align = src.align
		s.set |= sAlign
	}
	if src.set&sVAlign != 0 {
		s.valign = src.valign
		s.set |= sVAlign
	}
}

// applyContent returns text wrapped with the style's foreground,
// background, and text-attribute SGR codes (bold/italic/underline/
// line-through). Border color is NOT applied here — use applyBorder
// for border glyphs.
func (s *Style) applyContent(text string) string {
	if s.isEmpty() {
		return text
	}
	attrs := s.contentAttrs()
	if len(attrs) == 0 {
		return text
	}
	return color.New(attrs...).Sprint(text)
}

// applyBorder returns text wrapped with the style's border-color SGR
// codes. Unrelated style fields (fg, bg, bold, …) are ignored.
func (s *Style) applyBorder(text string) string {
	if s == nil || s.set&sBorder == 0 || len(s.borderAttrs) == 0 {
		return text
	}
	return color.New(s.borderAttrs...).Sprint(text)
}

// contentAttrs returns the SGR attributes that apply to cell content
// (fg + bg + text attributes). Border attributes are intentionally
// omitted; see applyBorder.
func (s *Style) contentAttrs() []color.Attribute {
	if s.isEmpty() {
		return nil
	}
	var attrs []color.Attribute
	if s.set&sFg != 0 {
		attrs = append(attrs, s.fgAttrs...)
	}
	if s.set&sBg != 0 {
		attrs = append(attrs, s.bgAttrs...)
	}
	if s.set&sBold != 0 && s.bold {
		attrs = append(attrs, color.Bold)
	}
	if s.set&sItalic != 0 && s.italic {
		attrs = append(attrs, color.Italic)
	}
	if s.set&sUnderline != 0 && s.underline {
		attrs = append(attrs, color.Underline)
	}
	if s.set&sStrike != 0 && s.strike {
		attrs = append(attrs, color.CrossedOut)
	}
	return attrs
}

// iterateCSS invokes visit for every well-formed "prop: val"
// declaration in a CSS-like block. Property names are lower-cased and
// trimmed; values are trimmed. Declarations missing a colon or with
// an empty property are skipped silently, letting callers tolerate
// malformed input gracefully.
func iterateCSS(css string, visit func(prop, val string)) {
	for _, decl := range strings.Split(css, ";") {
		decl = strings.TrimSpace(decl)
		if decl == "" {
			continue
		}
		colon := strings.Index(decl, ":")
		if colon < 0 {
			continue
		}
		prop := strings.ToLower(strings.TrimSpace(decl[:colon]))
		if prop == "" {
			continue
		}
		val := strings.TrimSpace(decl[colon+1:])
		visit(prop, val)
	}
}

// parseCSS parses a CSS-like declaration block (e.g.
// "color: red; background: blue; font-weight: bold") into s. Unknown
// properties are silently ignored so future additions do not break
// existing callers. Colons inside unknown values are tolerated.
func parseCSS(css string, s *Style) {
	iterateCSS(css, func(prop, val string) {
		applyDecl(s, prop, val)
	})
}

func applyDecl(s *Style, prop, val string) {
	switch prop {
	case "color":
		if attrs, ok := parseFgColor(val); ok {
			s.fgAttrs = attrs
			s.set |= sFg
		}
	case "background", "background-color":
		if attrs, ok := parseBgColor(val); ok {
			s.bgAttrs = attrs
			s.set |= sBg
		}
	case "border-color":
		if attrs, ok := parseFgColor(val); ok {
			s.borderAttrs = attrs
			s.set |= sBorder
		}
	case "font-weight":
		switch strings.ToLower(val) {
		case "bold":
			s.bold = true
			s.set |= sBold
		case "normal":
			s.bold = false
			s.set |= sBold
		}
	case "font-style":
		switch strings.ToLower(val) {
		case "italic":
			s.italic = true
			s.set |= sItalic
		case "normal":
			s.italic = false
			s.set |= sItalic
		}
	case "text-decoration":
		applyTextDecoration(s, val)
	case "text-align":
		switch strings.ToLower(val) {
		case "left":
			s.align = AlignLeft
			s.set |= sAlign
		case "center":
			s.align = AlignCenter
			s.set |= sAlign
		case "right":
			s.align = AlignRight
			s.set |= sAlign
		}
	case "vertical-align":
		switch strings.ToLower(val) {
		case "top":
			s.valign = VAlignTop
			s.set |= sVAlign
		case "middle":
			s.valign = VAlignMiddle
			s.set |= sVAlign
		case "bottom":
			s.valign = VAlignBottom
			s.set |= sVAlign
		}
	}
}

func applyTextDecoration(s *Style, val string) {
	for _, tok := range strings.Fields(strings.ToLower(val)) {
		switch tok {
		case "underline":
			s.underline = true
			s.set |= sUnderline
		case "line-through":
			s.strike = true
			s.set |= sStrike
		case "none":
			s.underline = false
			s.strike = false
			s.set |= sUnderline | sStrike
		}
	}
}

// Named color tables. Users can also supply #rrggbb or rgb(r,g,b).
// Bright variants map to the Hi* ANSI codes.

var fgNamedColors = map[string]color.Attribute{
	"black":          color.FgBlack,
	"red":            color.FgRed,
	"green":          color.FgGreen,
	"yellow":         color.FgYellow,
	"blue":           color.FgBlue,
	"magenta":        color.FgMagenta,
	"cyan":           color.FgCyan,
	"white":          color.FgWhite,
	"bright-black":   color.FgHiBlack,
	"bright-red":     color.FgHiRed,
	"bright-green":   color.FgHiGreen,
	"bright-yellow":  color.FgHiYellow,
	"bright-blue":    color.FgHiBlue,
	"bright-magenta": color.FgHiMagenta,
	"bright-cyan":    color.FgHiCyan,
	"bright-white":   color.FgHiWhite,
}

var bgNamedColors = map[string]color.Attribute{
	"black":          color.BgBlack,
	"red":            color.BgRed,
	"green":          color.BgGreen,
	"yellow":         color.BgYellow,
	"blue":           color.BgBlue,
	"magenta":        color.BgMagenta,
	"cyan":           color.BgCyan,
	"white":          color.BgWhite,
	"bright-black":   color.BgHiBlack,
	"bright-red":     color.BgHiRed,
	"bright-green":   color.BgHiGreen,
	"bright-yellow":  color.BgHiYellow,
	"bright-blue":    color.BgHiBlue,
	"bright-magenta": color.BgHiMagenta,
	"bright-cyan":    color.BgHiCyan,
	"bright-white":   color.BgHiWhite,
}

// parseFgColor resolves a color value to a slice of ANSI attributes
// suitable for foreground. Returns (nil, false) if the value is
// unrecognized.
func parseFgColor(val string) ([]color.Attribute, bool) {
	v := strings.ToLower(strings.TrimSpace(val))
	if a, ok := fgNamedColors[v]; ok {
		return []color.Attribute{a}, true
	}
	if r, g, b, ok := parseRGB(v); ok {
		return []color.Attribute{38, 2, color.Attribute(r), color.Attribute(g), color.Attribute(b)}, true
	}
	return nil, false
}

// parseBgColor is the background counterpart of parseFgColor.
func parseBgColor(val string) ([]color.Attribute, bool) {
	v := strings.ToLower(strings.TrimSpace(val))
	if a, ok := bgNamedColors[v]; ok {
		return []color.Attribute{a}, true
	}
	if r, g, b, ok := parseRGB(v); ok {
		return []color.Attribute{48, 2, color.Attribute(r), color.Attribute(g), color.Attribute(b)}, true
	}
	return nil, false
}

// parseRGB accepts "#rrggbb" or "rgb(r,g,b)" and returns the 0..255
// channel values. The ok flag reports whether the input matched.
func parseRGB(val string) (r, g, b int, ok bool) {
	if strings.HasPrefix(val, "#") {
		return parseHex(val[1:])
	}
	const (
		prefix = "rgb("
		suffix = ")"
	)
	if !strings.HasPrefix(val, prefix) || !strings.HasSuffix(val, suffix) {
		return 0, 0, 0, false
	}
	inner := val[len(prefix) : len(val)-len(suffix)]
	parts := strings.Split(inner, ",")
	if len(parts) != 3 {
		return 0, 0, 0, false
	}
	rr, rok := parseChannel(parts[0])
	gg, gok := parseChannel(parts[1])
	bb, bok := parseChannel(parts[2])
	if !rok || !gok || !bok {
		return 0, 0, 0, false
	}
	return rr, gg, bb, true
}

func parseChannel(s string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 0 || n > 255 {
		return 0, false
	}
	return n, true
}

func parseHex(s string) (r, g, b int, ok bool) {
	if len(s) != 6 {
		return 0, 0, 0, false
	}
	rr, err := strconv.ParseUint(s[0:2], 16, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	gg, err := strconv.ParseUint(s[2:4], 16, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	bb, err := strconv.ParseUint(s[4:6], 16, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	return int(rr), int(gg), int(bb), true
}
