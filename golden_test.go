// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"os"
	"path/filepath"
	"testing"
)

// assertGolden compares got to the contents of testdata/golden/<name>.
// Set TERMTABLE_UPDATE_GOLDEN=1 to overwrite the fixture instead of
// comparing; use this after deliberate rendering changes.
func assertGolden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", "golden", name)

	if os.Getenv("TERMTABLE_UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatalf("update golden %s: %v", name, err)
		}
		t.Logf("updated golden %s", name)
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v (did you forget to set TERMTABLE_UPDATE_GOLDEN=1?)", name, err)
	}
	if got != string(want) {
		t.Errorf("golden %s mismatch:\n--- want ---\n%s--- got ---\n%s", name, want, got)
	}
}

// TestGoldenMultiHeaderColspan locks the rendering of a table with a
// full-width banner header above a three-column sub-header. The
// interesting bits are the top border having no T-joins under the
// banner (the colspan suppresses them) and the separator between the
// two header rows introducing ┬ where column boundaries start.
func TestGoldenMultiHeaderColspan(t *testing.T) {
	h := th{t}
	tbl := NewTable(WithTargetWidth(40))

	banner := h.header(tbl.AddHeader())
	h.cell(banner.AddCell(
		WithContent("Evaluation Results"),
		WithColSpan(3),
		WithAlign(AlignCenter),
	))

	cols := h.header(tbl.AddHeader())
	h.cell(cols.AddCell(WithContent("Check")))
	h.cell(cols.AddCell(WithContent("Status")))
	h.cell(cols.AddCell(WithContent("Message")))

	r1 := h.row(tbl.AddRow())
	h.cell(r1.AddCell(WithContent("OSPS-BR-05")))
	h.cell(r1.AddCell(WithContent("PASS"), WithAlign(AlignCenter)))
	h.cell(r1.AddCell(WithContent("all good")))

	r2 := h.row(tbl.AddRow())
	h.cell(r2.AddCell(WithContent("OSPS-DO-02")))
	h.cell(r2.AddCell(WithContent("FAIL"), WithAlign(AlignCenter)))
	h.cell(r2.AddCell(WithContent("review deps")))

	assertGolden(t, "multi_header_colspan.golden", tbl.String())
}

// TestGoldenSpanNested locks a table where a single cell spans both
// two rows AND two columns. The interior seams of the big cell's
// rectangle are suppressed; its right and bottom borders join the
// surrounding grid correctly with a ┤ on row 0/1 separator and a ┼
// on the separator below.
func TestGoldenSpanNested(t *testing.T) {
	h := th{t}
	tbl := NewTable(WithTargetWidth(40))

	r0 := h.row(tbl.AddRow())
	h.cell(r0.AddCell(WithContent("big\nspan"), WithRowSpan(2), WithColSpan(2)))
	h.cell(r0.AddCell(WithContent("alpha")))

	r1 := h.row(tbl.AddRow())
	h.cell(r1.AddCell(WithContent("beta")))

	r2 := h.row(tbl.AddRow())
	h.cell(r2.AddCell(WithContent("gamma")))
	h.cell(r2.AddCell(WithContent("delta")))
	h.cell(r2.AddCell(WithContent("omega")))

	assertGolden(t, "span_nested.golden", tbl.String())
}

// TestGoldenUnicodeMix locks a table containing CJK, hiragana, and
// wide emoji alongside ASCII — the grid must stay aligned across
// mixed widths.
//
// Note: we deliberately use single-codepoint emoji (each rendered
// as width 2 everywhere) rather than ZWJ sequences like 👨‍👩‍👧. Per
// Unicode the whole family is one grapheme cluster of width 2 and
// uniseg reports that correctly, but terminals without ZWJ
// ligature support (some SSH contexts, tmux/screen with a
// no-emoji font, certain Windows setups) render the family as its
// three constituent emojis at width 6. termtable sides with the
// Unicode reading; keeping the fixture portable is a separate
// concern from validating the width math.
func TestGoldenUnicodeMix(t *testing.T) {
	h := th{t}
	tbl := NewTable(WithTargetWidth(40))

	hdr := h.header(tbl.AddHeader())
	h.cell(hdr.AddCell(WithContent("Name")))
	h.cell(hdr.AddCell(WithContent("CJK")))
	h.cell(hdr.AddCell(WithContent("Emoji")))

	r1 := h.row(tbl.AddRow())
	h.cell(r1.AddCell(WithContent("ascii")))
	h.cell(r1.AddCell(WithContent("中文测试"))) //nolint:gosmopolitan // CJK render test
	h.cell(r1.AddCell(WithContent("🔥🚀")))

	r2 := h.row(tbl.AddRow())
	h.cell(r2.AddCell(WithContent("more")))
	h.cell(r2.AddCell(WithContent("こんにちは")))
	h.cell(r2.AddCell(WithContent("🎉📦")))

	assertGolden(t, "unicode_mix.golden", tbl.String())
}
