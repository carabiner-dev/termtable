// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"strings"
	"testing"
)

// withCleanEnv saves and clears every env var that influences the
// emoji-width resolver, then restores them at test end. Tests that
// assert on auto-detection or env-var handling use this to ensure
// the ambient environment doesn't leak in.
func withCleanEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"TERMTABLE_EMOJI_WIDTH",
		"TERM_PROGRAM", "TERM", "WT_SESSION", "VTE_VERSION",
	} {
		t.Setenv(k, "")
	}
}

func TestConservativeClusterWidth(t *testing.T) {
	cases := []struct {
		in         string
		wantCons   int
		wantGraph  int
		descriptor string
	}{
		{"a", 1, 1, "plain ASCII"},
		{"漢", 2, 2, "CJK"}, //nolint:gosmopolitan // CJK width test
		{"🔥", 2, 2, "single emoji"},
		{"🔥🚀", 4, 4, "two emoji"},
		{"👨‍👩‍👧", 6, 2, "ZWJ family (man + woman + girl)"},
		{"🇯🇵", 4, 2, "flag (regional indicators)"},
		{"👋🏽", 4, 2, "skin-tone modifier"},
		{"❤️", 2, 2, "VS16 emoji presentation"},
	}
	for _, tc := range cases {
		gotCons := conservativeClusterWidth(tc.in)
		if gotCons != tc.wantCons {
			t.Errorf("conservative %s %q = %d, want %d", tc.descriptor, tc.in, gotCons, tc.wantCons)
		}
		gotGraph := clusterWidth(tc.in, EmojiWidthGrapheme)
		if gotGraph != tc.wantGraph {
			t.Errorf("grapheme %s %q = %d, want %d", tc.descriptor, tc.in, gotGraph, tc.wantGraph)
		}
	}
}

func TestDisplayWidthForModes(t *testing.T) {
	s := "abc 👨‍👩‍👧 xyz"
	cons := displayWidthFor(s, EmojiWidthConservative)
	grap := displayWidthFor(s, EmojiWidthGrapheme)
	if cons <= grap {
		t.Errorf("conservative width (%d) should exceed grapheme (%d) for ZWJ content", cons, grap)
	}
}

func TestPublicDisplayWidthUsesGrapheme(t *testing.T) {
	// Keep the contract that DisplayWidth (public) reports Unicode
	// grapheme semantics regardless of the resolver.
	if got := DisplayWidth("👨‍👩‍👧"); got != 2 {
		t.Errorf("DisplayWidth = %d, want 2 (uniseg grapheme)", got)
	}
}

func TestResolveEmojiWidthDefault(t *testing.T) {
	withCleanEnv(t)
	tbl := NewTable()
	if mode := tbl.resolveEmojiWidth(); mode != EmojiWidthConservative {
		t.Errorf("default mode = %v, want Conservative", mode)
	}
}

func TestResolveEmojiWidthAutoDetect(t *testing.T) {
	withCleanEnv(t)
	t.Setenv("TERM_PROGRAM", "iTerm.app")

	tbl := NewTable()
	if mode := tbl.resolveEmojiWidth(); mode != EmojiWidthGrapheme {
		t.Errorf("TERM_PROGRAM=iTerm.app: mode = %v, want Grapheme", mode)
	}
}

func TestResolveEmojiWidthEnvOverride(t *testing.T) {
	withCleanEnv(t)
	t.Setenv("TERM_PROGRAM", "iTerm.app") // would otherwise pick Grapheme
	t.Setenv("TERMTABLE_EMOJI_WIDTH", "conservative")

	tbl := NewTable()
	if mode := tbl.resolveEmojiWidth(); mode != EmojiWidthConservative {
		t.Errorf("env override: mode = %v, want Conservative", mode)
	}
}

func TestResolveEmojiWidthExplicitBeatsEnv(t *testing.T) {
	withCleanEnv(t)
	t.Setenv("TERMTABLE_EMOJI_WIDTH", "conservative")

	tbl := NewTable(WithEmojiWidth(EmojiWidthGrapheme))
	if mode := tbl.resolveEmojiWidth(); mode != EmojiWidthGrapheme {
		t.Errorf("explicit WithEmojiWidth should beat env: mode = %v", mode)
	}
}

func TestResolveEmojiWidthExplicitAutoStillAutoDetects(t *testing.T) {
	withCleanEnv(t)
	t.Setenv("TERM_PROGRAM", "WezTerm")

	tbl := NewTable(WithEmojiWidth(EmojiWidthAuto))
	if mode := tbl.resolveEmojiWidth(); mode != EmojiWidthGrapheme {
		t.Errorf("explicit Auto should still auto-detect: mode = %v", mode)
	}
}

func TestDetectModernEmojiTerminalWhitelist(t *testing.T) {
	cases := []struct {
		envKey, envVal string
		want           bool
	}{
		{"TERM_PROGRAM", "iTerm.app", true},
		{"TERM_PROGRAM", "WezTerm", true},
		{"TERM_PROGRAM", "vscode", true},
		{"TERM_PROGRAM", "ghostty", true},
		{"TERM_PROGRAM", "Hyper", true},
		{"TERM_PROGRAM", "Apple_Terminal", true},
		{"TERM_PROGRAM", "SomethingObscure", false},
		{"TERM", "xterm-kitty", true},
		{"TERM", "alacritty", true},
		{"TERM", "wezterm", true},
		{"TERM", "xterm-ghostty", true},
		{"TERM", "xterm-256color", false},
		{"TERM", "screen", false},
		{"WT_SESSION", "any-nonempty-value", true},
		{"VTE_VERSION", "6001", true},
		{"VTE_VERSION", "5999", false},
		{"VTE_VERSION", "not-a-number", false},
	}
	for _, tc := range cases {
		withCleanEnv(t)
		t.Setenv(tc.envKey, tc.envVal)
		if got := detectModernEmojiTerminal(); got != tc.want {
			t.Errorf("%s=%q: got %v, want %v", tc.envKey, tc.envVal, got, tc.want)
		}
	}
}

func TestEmojiModeAffectsLayout(t *testing.T) {
	buildZWJ := func(mode EmojiWidthMode) string {
		tbl := NewTable(WithTargetWidth(30), WithEmojiWidth(mode))
		r := tbl.AddRow()
		r.AddCell(WithContent("name"))
		r.AddCell(WithContent("👨‍👩‍👧"))
		return tbl.String()
	}

	cons := buildZWJ(EmojiWidthConservative)
	grap := buildZWJ(EmojiWidthGrapheme)
	if cons == grap {
		t.Errorf("conservative and grapheme should produce different output for ZWJ content\nconservative:\n%s\ngrapheme:\n%s", cons, grap)
	}

	// Within each mode, all rendered lines should agree on width
	// when measured with that same mode's rules — that's the actual
	// alignment property the renderer is supposed to uphold.
	for _, tc := range []struct {
		mode EmojiWidthMode
		out  string
	}{
		{EmojiWidthConservative, cons},
		{EmojiWidthGrapheme, grap},
	} {
		lines := strings.Split(strings.TrimRight(tc.out, "\n"), "\n")
		target := displayWidthFor(lines[0], tc.mode)
		for i, ln := range lines {
			if w := displayWidthFor(ln, tc.mode); w != target {
				t.Errorf("%v: uneven line %d width %d (want %d): %q", tc.mode, i, w, target, ln)
			}
		}
	}
}
