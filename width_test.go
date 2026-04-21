// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import "testing"

func TestDisplayWidthBasic(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"hello", 5},
		{"中文", 4},  //nolint:gosmopolitan // CJK width test data
		{"🔥", 2},   // single emoji
		{"a🔥b", 4}, // mixed
		{"a\x1b[31mb\x1b[0mc", 3},
		// Zero-width joiner family: single grapheme cluster, width 2.
		{"á", 1}, // "é" as NFD (a + combining acute)
	}
	for _, tc := range cases {
		if got := DisplayWidth(tc.in); got != tc.want {
			t.Errorf("DisplayWidth(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestDisplayWidthEmojiFamilies(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"👨‍👩‍👧‍👦", 2}, // ZWJ family
		{"🇯🇵", 2},      // flag
		{"👋🏽", 2},      // skin-tone modifier
		{"❤️", 2},      // heart + VS16 (emoji presentation)
	}
	for _, tc := range cases {
		got := DisplayWidth(tc.in)
		if got != tc.want {
			t.Errorf("DisplayWidth(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestMinUnbreakableWidth(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"hello", 5},
		{"hello world", 5},
		{"abc xyzw 12", 4},
		{"nospaces", 8},
		{"   ", 0},
		{"中文 abcdef", 6},             //nolint:gosmopolitan // widest word is "abcdef"
		{"短 xx\tlong-word", 9},       //nolint:gosmopolitan // mixed CJK and ASCII
		{"a\x1b[31mbcde\x1b[0mf", 6}, // ANSI doesn't split words
	}
	for _, tc := range cases {
		if got := MinUnbreakableWidth(tc.in); got != tc.want {
			t.Errorf("MinUnbreakableWidth(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestNaturalLinesHardBreaks(t *testing.T) {
	cases := []struct {
		in   string
		want []string // joined cluster text per line
	}{
		{"", []string{""}},
		{"a", []string{"a"}},
		{"a\nb", []string{"a", "b"}},
		{"a\r\nb", []string{"a", "b"}},
		{"\n", []string{"", ""}},
		{"abc\n\ndef", []string{"abc", "", "def"}},
	}
	for _, tc := range cases {
		lines := NaturalLines(tc.in)
		if len(lines) != len(tc.want) {
			t.Errorf("NaturalLines(%q) lines = %d, want %d", tc.in, len(lines), len(tc.want))
			continue
		}
		for i, runs := range lines {
			got := joinRuns(runs)
			if got != tc.want[i] {
				t.Errorf("NaturalLines(%q)[%d] = %q, want %q", tc.in, i, got, tc.want[i])
			}
		}
	}
}

func TestNaturalLinesANSIPrefixAttached(t *testing.T) {
	runs := NaturalLines("\x1b[31mred\x1b[0m")[0]
	if len(runs) != 3 {
		t.Fatalf("runs = %d, want 3", len(runs))
	}
	if runs[0].EscPrefix != "\x1b[31m" {
		t.Errorf("runs[0].EscPrefix = %q, want %q", runs[0].EscPrefix, "\x1b[31m")
	}
	if runs[1].EscPrefix != "" || runs[2].EscPrefix != "" {
		t.Error("only the first cluster should carry the opening escape")
	}
	// Trailing \x1b[0m has no following cluster; current behavior is to
	// drop it. Future phases may attach it as trailing state.
}

func TestNaturalLinesANSISpanningBreak(t *testing.T) {
	// Color opens on line 1, continues on line 2. Line 2's cluster has
	// no pending esc because scanning resets between lines. Documented
	// limitation; rendering re-emits the last-seen escape.
	lines := NaturalLines("\x1b[31ma\nb")
	if len(lines) != 2 {
		t.Fatalf("lines = %d", len(lines))
	}
	if lines[0][0].EscPrefix != "\x1b[31m" {
		t.Errorf("line 0 prefix = %q", lines[0][0].EscPrefix)
	}
	if len(lines[1]) != 1 || lines[1][0].EscPrefix != "" {
		t.Errorf("line 1 runs = %v", lines[1])
	}
}

// joinRuns concatenates the Text fields of runs for simple
// comparisons.
func joinRuns(runs []GraphemeRun) string {
	var b []byte
	for _, r := range runs {
		b = append(b, r.Text...)
	}
	return string(b)
}
