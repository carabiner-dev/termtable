// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"reflect"
	"strings"
	"testing"
)

// wrapOf is a test-only convenience that runs NaturalLines then Wrap
// with the common Phase 2 defaults (wrap=true, trim=false, no height
// cap).
func wrapOf(s string, width int) []string {
	return Wrap(NaturalLines(s), width, true, false, 0)
}

func TestWrapNoBreakNeeded(t *testing.T) {
	got := wrapOf("short", 10)
	if !reflect.DeepEqual(got, []string{"short"}) {
		t.Errorf("got %q, want [\"short\"]", got)
	}
}

func TestWrapWordBreakOnWhitespace(t *testing.T) {
	got := wrapOf("the quick brown fox", 10)
	want := []string{"the quick", "brown fox"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWrapHardBreakLongWord(t *testing.T) {
	got := wrapOf("abcdefghij", 4)
	want := []string{"abcd", "efgh", "ij"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWrapMixedBreak(t *testing.T) {
	got := wrapOf("abc supercalifragilistic ok", 5)
	// "abc" fits (3). adding " " fits (4). adding "s" exceeds (6>5 after space accumulated).
	// Actually let me just check the non-deforming property: each output line has width <= 5.
	for _, line := range got {
		if DisplayWidth(line) > 5 {
			t.Errorf("line %q exceeds width 5", line)
		}
	}
	// And all non-whitespace input is preserved.
	joined := strings.Join(got, "")
	stripped := strings.ReplaceAll("abc supercalifragilistic ok", " ", "")
	// The joined-output (with whitespace re-added between) should contain stripped.
	if !strings.Contains(strings.ReplaceAll(joined, " ", ""), stripped) {
		t.Errorf("content lost: joined=%q", joined)
	}
}

func TestWrapPreservesHardBreaks(t *testing.T) {
	got := wrapOf("line1\nline2\n", 80)
	want := []string{"line1", "line2", ""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWrapDropsLeadingWhitespaceOnContinuation(t *testing.T) {
	got := wrapOf("abc   def", 3)
	want := []string{"abc", "def"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWrapEmptyInput(t *testing.T) {
	got := wrapOf("", 10)
	want := []string{""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWrapAllWhitespaceProducesEmptyLine(t *testing.T) {
	got := wrapOf("     ", 10)
	want := []string{""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWrapSingleGraphemeWiderThanColumn(t *testing.T) {
	// Width=1 but the input is a 2-column emoji.
	got := wrapOf("🔥", 1)
	if len(got) != 1 {
		t.Fatalf("lines = %d, want 1", len(got))
	}
	if !strings.Contains(got[0], "🔥") {
		t.Errorf("output missing emoji: %q", got[0])
	}
}

func TestWrapCJKDoesNotDeform(t *testing.T) {
	got := wrapOf("中文abc", 4) //nolint:gosmopolitan // CJK wrap test
	for _, line := range got {
		if DisplayWidth(line) > 4 {
			t.Errorf("line %q exceeds width 4 (width=%d)", line, DisplayWidth(line))
		}
	}
}

func TestWrapANSIPreservedPerLine(t *testing.T) {
	// Red ABCDEF wrapped into 3-wide lines. Each line should re-open
	// red and close with reset.
	got := wrapOf("\x1b[31mABCDEF\x1b[0m", 3)
	if len(got) != 2 {
		t.Fatalf("lines = %d, want 2: %q", len(got), got)
	}
	for i, line := range got {
		if !strings.Contains(line, "\x1b[31m") {
			t.Errorf("line %d %q missing red open", i, line)
		}
		if !strings.HasSuffix(line, "\x1b[0m") {
			t.Errorf("line %d %q missing reset", i, line)
		}
		if DisplayWidth(line) > 3 {
			t.Errorf("line %d %q exceeds width 3 (visible width=%d)",
				i, line, DisplayWidth(line))
		}
	}
}

func TestWrapNoANSINoReset(t *testing.T) {
	got := wrapOf("hello", 10)
	if strings.Contains(got[0], "\x1b") {
		t.Errorf("plain input should produce plain output, got %q", got[0])
	}
}

func TestWrapNoWrapSingleLine(t *testing.T) {
	out := Wrap(NaturalLines("hello world"), 20, false, false, 0)
	if !reflect.DeepEqual(out, []string{"hello world"}) {
		t.Errorf("got %q", out)
	}
}

func TestWrapNoWrapTrimEllipsizes(t *testing.T) {
	out := Wrap(NaturalLines("hello world"), 8, false, true, 0)
	if len(out) != 1 {
		t.Fatalf("lines = %d", len(out))
	}
	if DisplayWidth(out[0]) > 8 {
		t.Errorf("line %q exceeds width 8", out[0])
	}
	if !strings.HasSuffix(out[0], ellipsis) {
		t.Errorf("line %q should end with ellipsis", out[0])
	}
}

func TestWrapMaxHeightTruncates(t *testing.T) {
	out := Wrap(NaturalLines("a\nb\nc\nd"), 5, true, true, 2)
	if len(out) != 2 {
		t.Fatalf("lines = %d, want 2: %q", len(out), out)
	}
	if !strings.HasSuffix(out[1], ellipsis) {
		t.Errorf("last line %q should end with ellipsis", out[1])
	}
}

func TestWrapMaxHeightNoTrimKeepsContent(t *testing.T) {
	out := Wrap(NaturalLines("a\nb\nc"), 5, true, false, 2)
	if len(out) != 2 {
		t.Fatalf("lines = %d", len(out))
	}
	if strings.Contains(out[1], ellipsis) {
		t.Errorf("trim=false should not append ellipsis")
	}
}
