// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/rivo/uniseg"
)

// EmojiWidthMode controls how termtable counts display columns for
// grapheme clusters whose rendering varies across terminals — most
// commonly ZWJ emoji families like 👨‍👩‍👧, regional-indicator flag
// pairs like 🇯🇵, and skin-tone modifier sequences like 👋🏽.
//
// The Unicode standard considers each of these a single grapheme
// cluster and defines a "collapsed" display width (typically 2). In
// practice the rendering width depends on whether the user's font
// has a glyph for the joined form. When the font doesn't, terminals
// fall back to rendering each constituent codepoint as its own
// glyph — which can double, triple, or sextuple the visual width
// and misalign the whole table.
type EmojiWidthMode uint8

const (
	// EmojiWidthAuto (the zero value) resolves to EmojiWidthGrapheme
	// when termtable detects a terminal known to render composite
	// emoji correctly, and EmojiWidthConservative everywhere else.
	// The TERMTABLE_EMOJI_WIDTH environment variable overrides the
	// detection.
	EmojiWidthAuto EmojiWidthMode = iota

	// EmojiWidthConservative counts every visible codepoint in a
	// grapheme cluster at its standalone width, skipping zero-width
	// composers (ZWJ, variation selectors, combining marks). The
	// result is an upper bound on the rendered width — tables stay
	// aligned even on terminals that don't implement emoji
	// ligatures.
	EmojiWidthConservative

	// EmojiWidthGrapheme uses the Unicode-standard cluster width
	// reported by uniseg. Produces tight layouts on modern
	// terminals; may misalign rows on ones that fall back to
	// rendering composite emoji piecewise.
	EmojiWidthGrapheme
)

func (m EmojiWidthMode) String() string {
	switch m {
	case EmojiWidthAuto:
		return "auto"
	case EmojiWidthConservative:
		return "conservative"
	case EmojiWidthGrapheme:
		return "grapheme"
	default:
		return unknownName
	}
}

// resolveEmojiWidth returns the concrete mode (Conservative or
// Grapheme) in force for this table. The precedence is:
//
//  1. An explicit WithEmojiWidth(mode) option (where mode is not
//     Auto) wins unconditionally.
//  2. The TERMTABLE_EMOJI_WIDTH environment variable, if set to
//     "conservative", "grapheme", or their aliases, is consulted
//     next.
//  3. detectModernEmojiTerminal returns true for a whitelist of
//     terminals known to handle ZWJ ligatures — those get
//     Grapheme.
//  4. Everything else gets Conservative.
func (t *Table) resolveEmojiWidth() EmojiWidthMode {
	switch t.opts.emojiWidth {
	case EmojiWidthConservative, EmojiWidthGrapheme:
		return t.opts.emojiWidth
	case EmojiWidthAuto:
		// fall through
	}
	if mode, ok := emojiModeFromEnv(); ok {
		return mode
	}
	if detectModernEmojiTerminal() {
		return EmojiWidthGrapheme
	}
	return EmojiWidthConservative
}

// emojiModeFromEnv reads TERMTABLE_EMOJI_WIDTH and maps a few common
// spellings to the concrete modes. Returns ok=false when the env
// var is unset or has an unrecognized value.
func emojiModeFromEnv() (EmojiWidthMode, bool) {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("TERMTABLE_EMOJI_WIDTH"))) {
	case "conservative", "safe", "wide":
		return EmojiWidthConservative, true
	case "grapheme", "unicode", "tight":
		return EmojiWidthGrapheme, true
	case "auto":
		// Explicit auto resets to detection.
		return 0, false
	}
	return 0, false
}

// detectModernEmojiTerminal returns true when the host terminal is
// in a hand-maintained whitelist of emulators known to render ZWJ
// emoji, flag sequences, and skin-tone modifiers correctly. The
// whitelist is deliberately small — it's safer to be conservative
// (wastes a little space) than to trust a terminal that will
// misalign output.
func detectModernEmojiTerminal() bool {
	switch os.Getenv("TERM_PROGRAM") {
	case "iTerm.app", "WezTerm", "vscode", "Hyper", "Apple_Terminal", "ghostty":
		return true
	}
	switch os.Getenv("TERM") {
	case "xterm-kitty", "wezterm", "xterm-ghostty",
		"alacritty", "alacritty-direct":
		return true
	}
	// Windows Terminal (and some modern MS envs) sets WT_SESSION.
	if os.Getenv("WT_SESSION") != "" {
		return true
	}
	// GNOME Terminal / VTE-based. VTE_VERSION is 4-6 digits; 6000+
	// (VTE 0.60, released 2019) is a safe cutoff for ZWJ support.
	if v := os.Getenv("VTE_VERSION"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 6000 {
			return true
		}
	}
	return false
}

// clusterWidth returns the display width of a single grapheme
// cluster under mode. Only Conservative and Grapheme are accepted —
// callers resolve Auto before dispatching here.
func clusterWidth(cluster string, mode EmojiWidthMode) int {
	if mode == EmojiWidthConservative {
		return conservativeClusterWidth(cluster)
	}
	return uniseg.StringWidth(cluster)
}

// conservativeClusterWidth returns the width the cluster would
// occupy if each of its emoji-capable codepoints rendered as its
// own standalone glyph. Zero-width composers (ZWJ, variation
// selectors, combining marks) contribute nothing; the result is at
// least the uniseg width so pure CJK and ASCII tokens are
// unaffected.
func conservativeClusterWidth(cluster string) int {
	base := uniseg.StringWidth(cluster)
	if cluster == "" {
		return 0
	}
	var parts int
	for _, r := range cluster {
		if isZeroWidthComposer(r) {
			continue
		}
		parts++
	}
	if parts <= 1 {
		return base
	}
	// Each renderable part could end up as a separate glyph. Take
	// the wider of (collapsed width reported by uniseg) and
	// (worst-case sum where each part is two columns).
	worst := parts * 2
	if worst > base {
		return worst
	}
	return base
}

// isZeroWidthComposer reports whether r contributes no visual
// width even when a cluster fails to ligature — zero-width joiners,
// variation selectors, and non-spacing / enclosing combining marks.
func isZeroWidthComposer(r rune) bool {
	switch r {
	case 0x200D: // ZERO WIDTH JOINER
		return true
	case 0xFE0E, 0xFE0F: // VARIATION SELECTOR-15 / VARIATION SELECTOR-16
		return true
	}
	return unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r)
}
