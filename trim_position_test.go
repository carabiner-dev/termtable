// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"strings"
	"testing"
)

// urlTable builds a single-row, single-cell table containing one URL
// that is too wide for the column, so trimming must kick in.
func urlTable(t *testing.T, opts ...CellOption) string {
	t.Helper()
	tbl := NewTable(WithTargetWidth(25))
	tbl.Column(0).Style("white-space: nowrap")
	r := tbl.AddRow()
	r.AddCell(append(
		[]CellOption{WithContent("https://example.com/page.html")},
		opts...,
	)...)
	return tbl.String()
}

func firstContentLine(out string) string {
	for _, ln := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if strings.HasPrefix(ln, "│") {
			return ln
		}
	}
	return ""
}

func TestTrimPositionDefaultIsEnd(t *testing.T) {
	out := urlTable(t)
	content := firstContentLine(out)
	if !strings.HasPrefix(content, "│ https://") {
		t.Errorf("default trim should keep prefix: %q", content)
	}
	if !strings.Contains(content, "…") {
		t.Errorf("default trim should append ellipsis: %q", content)
	}
}

func TestTrimPositionEnd(t *testing.T) {
	out := urlTable(t, WithTrimPosition(TrimEnd))
	content := firstContentLine(out)
	if !strings.Contains(content, "https://") {
		t.Errorf("TrimEnd should keep prefix: %q", content)
	}
	if !strings.Contains(content, "…") {
		t.Errorf("TrimEnd should contain ellipsis: %q", content)
	}
	if strings.Contains(content, "page.html") {
		t.Errorf("TrimEnd should drop the suffix: %q", content)
	}
}

func TestTrimPositionStart(t *testing.T) {
	out := urlTable(t, WithTrimPosition(TrimStart))
	content := firstContentLine(out)
	if !strings.Contains(content, "…") {
		t.Fatalf("TrimStart should contain ellipsis: %q", content)
	}
	// Prefix dropped, suffix preserved.
	if strings.Contains(content, "https://") {
		t.Errorf("TrimStart should drop the prefix: %q", content)
	}
	if !strings.Contains(content, "page.html") {
		t.Errorf("TrimStart should keep content suffix: %q", content)
	}
}

func TestTrimPositionMiddle(t *testing.T) {
	out := urlTable(t, WithTrimPosition(TrimMiddle))
	content := firstContentLine(out)
	if !strings.Contains(content, "…") {
		t.Fatalf("TrimMiddle should contain ellipsis: %q", content)
	}
	// Both ends preserved — that's the whole point.
	if !strings.Contains(content, "https") {
		t.Errorf("TrimMiddle should keep start: %q", content)
	}
	if !strings.Contains(content, "page.html") {
		t.Errorf("TrimMiddle should keep end: %q", content)
	}
}

func TestTrimPositionCSSEquivalent(t *testing.T) {
	imperative := urlTable(t, WithTrimPosition(TrimMiddle))
	css := urlTable(t, WithCellStyle("text-overflow-position: middle"))
	if imperative != css {
		t.Errorf("CSS should match imperative:\nimperative:\n%s\ncss:\n%s",
			imperative, css)
	}
}

func TestTrimPositionCSSSynonyms(t *testing.T) {
	cases := []struct {
		css  string
		want TrimPosition
	}{
		{"text-overflow-position: start", TrimStart},
		{"text-overflow-position: left", TrimStart},
		{"text-overflow-position: head", TrimStart},
		{"text-overflow-position: middle", TrimMiddle},
		{"text-overflow-position: center", TrimMiddle},
		{"text-overflow-position: end", TrimEnd},
		{"text-overflow-position: right", TrimEnd},
		{"text-overflow-position: tail", TrimEnd},
		{"text-overflow-side: middle", TrimMiddle}, // alias
	}
	for _, tc := range cases {
		var s Style
		parseCSS(tc.css, &s)
		if s.set&sTrimPos == 0 {
			t.Errorf("%s: sTrimPos not set", tc.css)
			continue
		}
		if s.trimPosition != tc.want {
			t.Errorf("%s: got %v, want %v", tc.css, s.trimPosition, tc.want)
		}
	}
}

func TestTrimPositionCascadesFromColumn(t *testing.T) {
	tbl := NewTable(WithTargetWidth(25))
	tbl.Column(0).Style("white-space: nowrap; text-overflow-position: start")
	r := tbl.AddRow()
	r.AddCell(WithContent("https://example.com/page.html"))

	out := tbl.String()
	content := firstContentLine(out)
	if !strings.Contains(content, "…") {
		t.Fatalf("expected ellipsis: %q", content)
	}
	// TrimStart cascaded from the column should drop the prefix
	// and keep the content's suffix.
	if strings.Contains(content, "https://") {
		t.Errorf("column TrimStart cascade should drop prefix: %q", content)
	}
	if !strings.Contains(content, "page.html") {
		t.Errorf("column TrimStart cascade should keep suffix: %q", content)
	}
}

func TestLineClampEllipsisAlwaysAtEnd(t *testing.T) {
	// Vertical truncation (line-clamp) always signals "more below"
	// by placing the ellipsis at the end of the final kept line,
	// regardless of text-overflow-position. The trim position only
	// matters for horizontal clipping.
	tbl := NewTable(WithTargetWidth(25))
	r := tbl.AddRow()
	r.AddCell(
		WithContent("one two three four five six seven eight"),
		WithMaxLines(1),
		WithTrimPosition(TrimStart),
	)

	out := tbl.String()
	content := firstContentLine(out)
	if !strings.Contains(content, "one") {
		t.Errorf("line-clamp should preserve the first line's content: %q", content)
	}
	if !strings.HasSuffix(strings.TrimRight(content, " │"), "…") {
		t.Errorf("line-clamp ellipsis should sit at the end: %q", content)
	}
}

func TestTrimPositionClipNoEllipsis(t *testing.T) {
	// text-overflow: clip + text-overflow-position: middle should
	// drop the middle portion entirely (no ellipsis marker).
	out := urlTable(t, WithCellStyle("text-overflow: clip; text-overflow-position: middle"))
	content := firstContentLine(out)
	if strings.Contains(content, "…") {
		t.Errorf("text-overflow: clip should not emit ellipsis: %q", content)
	}
	if !strings.Contains(content, "https") {
		t.Errorf("TrimMiddle clip should keep prefix: %q", content)
	}
	if !strings.Contains(content, "page.html") {
		t.Errorf("TrimMiddle clip should keep suffix: %q", content)
	}
}

func TestTrimPositionIrrelevantWhenFits(t *testing.T) {
	tbl := NewTable(WithTargetWidth(40))
	r := tbl.AddRow()
	r.AddCell(WithContent("short"), WithSingleLine(), WithTrimPosition(TrimMiddle))

	out := tbl.String()
	if strings.Contains(out, "…") {
		t.Errorf("content that fits should render without ellipsis: %q", out)
	}
	if !strings.Contains(out, "short") {
		t.Errorf("content should be present: %q", out)
	}
}
