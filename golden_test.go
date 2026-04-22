// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"bytes"
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
	// Normalise CRLF → LF in case the fixture was checked out on
	// Windows without the repo's .gitattributes. tbl.String() always
	// emits LF; comparing byte-exact after that step keeps the
	// expectation platform-agnostic.
	want = bytes.ReplaceAll(want, []byte("\r\n"), []byte("\n"))
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
	tbl := NewTable(WithTargetWidth(40))

	banner := tbl.AddHeader()
	banner.AddCell(
		WithContent("Evaluation Results"),
		WithColSpan(3),
		WithAlign(AlignCenter),
	)

	cols := tbl.AddHeader()
	cols.AddCell(WithContent("Check"))
	cols.AddCell(WithContent("Status"))
	cols.AddCell(WithContent("Message"))

	r1 := tbl.AddRow()
	r1.AddCell(WithContent("OSPS-BR-05"))
	r1.AddCell(WithContent("PASS"), WithAlign(AlignCenter))
	r1.AddCell(WithContent("all good"))

	r2 := tbl.AddRow()
	r2.AddCell(WithContent("OSPS-DO-02"))
	r2.AddCell(WithContent("FAIL"), WithAlign(AlignCenter))
	r2.AddCell(WithContent("review deps"))

	assertGolden(t, "multi_header_colspan.golden", tbl.String())
}

// TestGoldenSpanNested locks a table where a single cell spans both
// two rows AND two columns. The interior seams of the big cell's
// rectangle are suppressed; its right and bottom borders join the
// surrounding grid correctly with a ┤ on row 0/1 separator and a ┼
// on the separator below.
func TestGoldenSpanNested(t *testing.T) {
	tbl := NewTable(WithTargetWidth(40))

	r0 := tbl.AddRow()
	r0.AddCell(WithContent("big\nspan"), WithRowSpan(2), WithColSpan(2))
	r0.AddCell(WithContent("alpha"))

	r1 := tbl.AddRow()
	r1.AddCell(WithContent("beta"))

	r2 := tbl.AddRow()
	r2.AddCell(WithContent("gamma"))
	r2.AddCell(WithContent("delta"))
	r2.AddCell(WithContent("omega"))

	assertGolden(t, "span_nested.golden", tbl.String())
}

// TestGoldenUnicodeMix locks a table containing CJK, hiragana, and
// a ZWJ emoji family alongside ASCII. The test pins the table to
// EmojiWidthConservative so output stays byte-identical regardless
// of the runner's terminal detection — and the fixture stays
// visually aligned on every terminal, including ones without ZWJ
// ligature support.
func TestGoldenUnicodeMix(t *testing.T) {
	tbl := NewTable(
		WithTargetWidth(40),
		WithEmojiWidth(EmojiWidthConservative),
	)

	hdr := tbl.AddHeader()
	hdr.AddCell(WithContent("Name"))
	hdr.AddCell(WithContent("CJK"))
	hdr.AddCell(WithContent("Emoji"))

	r1 := tbl.AddRow()
	r1.AddCell(WithContent("ascii"))
	r1.AddCell(WithContent("中文测试")) //nolint:gosmopolitan // CJK render test
	r1.AddCell(WithContent("🔥🚀"))

	r2 := tbl.AddRow()
	r2.AddCell(WithContent("more"))
	r2.AddCell(WithContent("こんにちは"))
	r2.AddCell(WithContent("👨‍👩‍👧"))

	assertGolden(t, "unicode_mix.golden", tbl.String())
}
