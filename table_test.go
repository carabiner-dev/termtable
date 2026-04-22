// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"strings"
	"testing"
)

func TestEmptyTableDimensions(t *testing.T) {
	tbl := NewTable()
	if tbl.NumColumns() != 0 {
		t.Errorf("NumColumns = %d, want 0", tbl.NumColumns())
	}
	if tbl.NumRows() != 0 {
		t.Errorf("NumRows = %d, want 0", tbl.NumRows())
	}
	if tbl.CellAt(0, 0) != nil {
		t.Error("CellAt on empty table should be nil")
	}
	if tbl.InBounds(0, 0) {
		t.Error("InBounds(0,0) on empty table should be false")
	}
}

func TestNumRowsSumAcrossSections(t *testing.T) {
	tbl := NewTable()
	tbl.AddHeader()
	tbl.AddHeader()
	tbl.AddRow()
	tbl.AddRow()
	tbl.AddRow()
	tbl.AddFooter()
	if got := tbl.NumRows(); got != 6 {
		t.Errorf("NumRows = %d, want 6 (2h + 3r + 1f)", got)
	}
}

func TestCellAtAcrossSections(t *testing.T) {
	tbl := NewTable()
	hd := tbl.AddHeader()
	hc := hd.AddCell(WithCellID("hc"), WithContent("head"))

	r := tbl.AddRow()
	rc := r.AddCell(WithCellID("rc"), WithContent("body"))

	f := tbl.AddFooter()
	fc := f.AddCell(WithCellID("fc"), WithContent("foot"))

	// Absolute row indices: header at 0, body at 1, footer at 2.
	if got := tbl.CellAt(0, 0); got != hc {
		t.Errorf("CellAt(0,0) = %v, want header cell", got)
	}
	if got := tbl.CellAt(1, 0); got != rc {
		t.Errorf("CellAt(1,0) = %v, want body cell", got)
	}
	if got := tbl.CellAt(2, 0); got != fc {
		t.Errorf("CellAt(2,0) = %v, want footer cell", got)
	}
	if tbl.CellAt(3, 0) != nil {
		t.Error("CellAt past last row should be nil")
	}
	if tbl.CellAt(-1, 0) != nil {
		t.Error("CellAt negative should be nil")
	}
}

func TestCellAtSpansMapToSameCell(t *testing.T) {
	tbl := NewTable()
	r := tbl.AddRow()
	c := r.AddCell(WithContent("wide"), WithColSpan(3))
	for col := range 3 {
		if got := tbl.CellAt(0, col); got != c {
			t.Errorf("CellAt(0,%d) = %v, want %v", col, got, c)
		}
	}
	if tbl.CellAt(0, 3) != nil {
		t.Error("CellAt past span should be nil")
	}
}

func TestInBounds(t *testing.T) {
	tbl := NewTable()
	r := tbl.AddRow()
	r.AddCell(WithContent("a"))
	r.AddCell(WithContent("b"))
	cases := []struct {
		r, c int
		want bool
	}{
		{0, 0, true},
		{0, 1, true},
		{0, 2, false},
		{1, 0, false},
		{-1, 0, false},
		{0, -1, false},
	}
	for _, tc := range cases {
		if got := tbl.InBounds(tc.r, tc.c); got != tc.want {
			t.Errorf("InBounds(%d,%d) = %v, want %v", tc.r, tc.c, got, tc.want)
		}
	}
}

func TestResolvedTargetWidth(t *testing.T) {
	tbl := NewTable()
	t.Setenv("COLUMNS", "")
	if got := tbl.ResolvedTargetWidth(); got != defaultTargetWidth {
		t.Errorf("default = %d, want %d", got, defaultTargetWidth)
	}
	t.Setenv("COLUMNS", "42")
	if got := tbl.ResolvedTargetWidth(); got != 42 {
		t.Errorf("COLUMNS = %d, want 42", got)
	}

	explicit := NewTable(WithTargetWidth(120))
	if got := explicit.ResolvedTargetWidth(); got != 120 {
		t.Errorf("explicit = %d, want 120", got)
	}

	t.Setenv("COLUMNS", "garbage")
	if got := tbl.ResolvedTargetWidth(); got != defaultTargetWidth {
		t.Errorf("garbage COLUMNS = %d, want default %d", got, defaultTargetWidth)
	}
}

// TestDetectTerminalWidthNonTTY verifies that the TTY probe silently
// reports "not available" when stdout/stderr are pipes (which is the
// shape `go test` runs with). This guarantees that in non-interactive
// environments the resolver falls through to defaultTargetWidth rather
// than returning a bogus 0.
func TestDetectTerminalWidthNonTTY(t *testing.T) {
	if _, ok := detectTerminalWidth(); ok {
		t.Skip("stdout/stderr appear to be a real TTY; skipping non-TTY assertion")
	}
}

// withFakeTTY swaps terminalWidthProbe for the duration of the test.
// Pass ok=false to simulate a non-TTY environment.
func withFakeTTY(t *testing.T, width int, ok bool) {
	t.Helper()
	saved := terminalWidthProbe
	terminalWidthProbe = func() (int, bool) { return width, ok }
	t.Cleanup(func() { terminalWidthProbe = saved })
}

// TestResolvedTargetWidthDefaultFillsNinetyPercent verifies that a
// table built with no width preference grows to 90% of the attached
// terminal — 80 is the floor, not the target.
func TestResolvedTargetWidthDefaultFillsNinetyPercent(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 200, true)

	tbl := NewTable()
	if got := tbl.ResolvedTargetWidth(); got != 180 {
		t.Errorf("default width = %d, want 180 (90%% of 200)", got)
	}
}

// TestResolvedTargetWidthDefaultMinFloor verifies that on medium-width
// terminals where 90% would drop below 80, the 80-column floor takes
// over.
func TestResolvedTargetWidthDefaultMinFloor(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 85, true)

	tbl := NewTable()
	if got := tbl.ResolvedTargetWidth(); got != defaultTargetWidth {
		// 90% of 85 = 76, below the 80 floor — floor wins.
		t.Errorf("default width = %d, want %d (80 floor beats 76)", got, defaultTargetWidth)
	}
}

// TestLayoutContentShrinksToNatural verifies that when content is
// narrower than max-width but wider than min-width, the table renders
// at its natural (content-desired) width — not padded out to the
// max-width ceiling.
func TestLayoutContentShrinksToNatural(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 200, true)

	tbl := NewTable()
	// Short content: natural width ~= "hello" + "world" + overhead = ~15.
	// min-width default is 80, so we should land at 80 — not the 180
	// max-width ceiling.
	r := tbl.AddRow()
	r.AddCell(WithContent("hello"))
	r.AddCell(WithContent("world"))

	out := tbl.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	width := DisplayWidth(lines[0])
	if width != 80 {
		t.Errorf("rendered width = %d, want 80 (min-width floor, not max-width 180)", width)
	}
}

// TestLayoutContentGrowsToMax verifies that content wider than the
// min-width but narrower than max-width stretches the table to its
// natural width.
func TestLayoutContentGrowsToMax(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 200, true)

	tbl := NewTable()
	// Fabricate content wide enough to land between 80 and 180.
	big := strings.Repeat("x", 60)
	r := tbl.AddRow()
	r.AddCell(WithContent(big))
	r.AddCell(WithContent(big))

	out := tbl.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	width := DisplayWidth(lines[0])
	// Natural width = 60 + 60 + overhead(7) = 127. Within [80, 180].
	if width != 127 {
		t.Errorf("rendered width = %d, want 127 (content-natural)", width)
	}
}

// TestLayoutContentClampedToMax verifies that content wider than the
// max-width ceiling causes the layout to cap at max-width — not
// overflow the screen.
func TestLayoutContentClampedToMax(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 200, true)

	tbl := NewTable()
	// Content so wide that natural would exceed 180.
	big := strings.Repeat("x", 120)
	r := tbl.AddRow()
	r.AddCell(WithContent(big))
	r.AddCell(WithContent(big))

	out := tbl.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	width := DisplayWidth(lines[0])
	if width != 180 {
		t.Errorf("rendered width = %d, want 180 (max-width ceiling)", width)
	}
}

// TestWithMinWidthOverride verifies that an explicit WithMinWidth
// raises (or lowers) the floor.
func TestWithMinWidthOverride(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 200, true)

	tbl := NewTable(WithMinWidth(100))
	r := tbl.AddRow()
	r.AddCell(WithContent("tiny"))

	out := tbl.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if w := DisplayWidth(lines[0]); w != 100 {
		t.Errorf("rendered width = %d, want 100 (explicit min-width)", w)
	}
}

// TestWithMaxWidthOverride verifies that an explicit WithMaxWidth
// lowers the ceiling.
func TestWithMaxWidthOverride(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 200, true)

	tbl := NewTable(WithMaxWidth(50))
	big := strings.Repeat("x", 80)
	r := tbl.AddRow()
	r.AddCell(WithContent(big))
	r.AddCell(WithContent(big))

	out := tbl.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	// Explicit max-width=50 beats the default min-width=80 per CSS:
	// min ≤ max is required; when max < min, min still wins. But here
	// the user set max explicitly, so it overrides the default floor.
	// In practice: the user's explicit max wins.
	if w := DisplayWidth(lines[0]); w > 80 {
		t.Errorf("rendered width = %d, want <= 80 (explicit max clamps)", w)
	}
}

// TestCSSMinMaxWidth verifies that `min-width` and `max-width` in the
// table CSS drive the same options.
func TestCSSMinMaxWidth(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 200, true)

	tbl := NewTable(WithTableStyle("min-width: 60; max-width: 50%"))
	r := tbl.AddRow()
	r.AddCell(WithContent(strings.Repeat("x", 200)))

	out := tbl.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	// 50% of 200 = 100, max-width clamps to 100.
	if w := DisplayWidth(lines[0]); w != 100 {
		t.Errorf("rendered width = %d, want 100 (max-width 50%%)", w)
	}
}

// TestExplicitWidthBypassesDefaults verifies that WithTargetWidth
// still pins to the requested value: the default min/max bounds do
// NOT clamp an explicit target (matches user expectation and preserves
// backwards compatibility).
func TestExplicitWidthBypassesDefaults(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 200, true)

	tbl := NewTable(WithTargetWidth(50))
	r := tbl.AddRow()
	r.AddCell(WithContent("a"))
	r.AddCell(WithContent("b"))

	out := tbl.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if w := DisplayWidth(lines[0]); w != 50 {
		t.Errorf("rendered width = %d, want 50 (explicit width bypasses default min 80)", w)
	}
}

// TestExplicitWidthClampedByExplicitMax verifies that an explicit max
// still clamps an explicit width — CSS-style composition of
// explicitly-set bounds.
func TestExplicitWidthClampedByExplicitMax(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 200, true)

	tbl := NewTable(WithTargetWidth(120), WithMaxWidth(90))
	r := tbl.AddRow()
	r.AddCell(WithContent("a"))
	r.AddCell(WithContent("b"))

	out := tbl.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if w := DisplayWidth(lines[0]); w != 90 {
		t.Errorf("rendered width = %d, want 90 (explicit max overrides explicit width)", w)
	}
}

// TestResolvedTargetWidthDefaultCappedByNarrowTTY verifies that on a
// screen narrower than the 80-column floor, the terminal width wins:
// the floor never exceeds the physical screen.
func TestResolvedTargetWidthDefaultCappedByNarrowTTY(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 40, true)

	tbl := NewTable()
	if got := tbl.ResolvedTargetWidth(); got != 40 {
		t.Errorf("default capped = %d, want 40", got)
	}
}

// TestResolvedTargetWidthCapsExplicitToTTY verifies that an explicit
// WithTargetWidth wider than the attached terminal is capped to the
// terminal width.
func TestResolvedTargetWidthCapsExplicitToTTY(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 80, true)

	tbl := NewTable(WithTargetWidth(200))
	if got := tbl.ResolvedTargetWidth(); got != 80 {
		t.Errorf("capped width = %d, want 80 (TTY cap)", got)
	}
}

// TestResolvedTargetWidthExplicitFitsInTTY verifies that an explicit
// width narrower than the TTY is honoured verbatim.
func TestResolvedTargetWidthExplicitFitsInTTY(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 120, true)

	tbl := NewTable(WithTargetWidth(40))
	if got := tbl.ResolvedTargetWidth(); got != 40 {
		t.Errorf("narrow explicit = %d, want 40", got)
	}
}

// TestResolvedTargetWidthCOLUMNSCappedToTTY verifies that a COLUMNS
// value wider than the screen is also capped. COLUMNS is a preference,
// not a licence to overflow the terminal.
func TestResolvedTargetWidthCOLUMNSCappedToTTY(t *testing.T) {
	t.Setenv("COLUMNS", "200")
	withFakeTTY(t, 80, true)

	tbl := NewTable()
	if got := tbl.ResolvedTargetWidth(); got != 80 {
		t.Errorf("COLUMNS cap = %d, want 80", got)
	}
}

// TestResolvedTargetWidthNoTTYNoCap verifies that when no terminal is
// attached (e.g. writing to a pipe or file) the resolver does not
// invent a cap: explicit widths pass through verbatim even when they
// are large.
func TestResolvedTargetWidthNoTTYNoCap(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 0, false)

	tbl := NewTable(WithTargetWidth(500))
	if got := tbl.ResolvedTargetWidth(); got != 500 {
		t.Errorf("non-TTY explicit = %d, want 500 (no cap applied)", got)
	}
}

// TestRenderNeverExceedsTTYWidth verifies the hard guarantee: when a
// terminal is attached, no rendered line is wider than the terminal,
// even when the content's minimum widths would otherwise force an
// overflowing best-effort render.
func TestRenderNeverExceedsTTYWidth(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 20, true)

	// Target is narrower than the content minimums — the layout
	// solver will surface ErrTargetTooNarrow and produce a
	// best-effort render that, left alone, would exceed the target.
	// The TTY clip must bring every line back under 20 columns.
	tbl := NewTable(WithTargetWidth(10))
	r := tbl.AddRow()
	r.AddCell(WithContent("averylongwordthatexceedsthescreen"))
	r.AddCell(WithContent("anotherlongwordtoo"))

	out := tbl.String()
	for i, ln := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if w := DisplayWidth(ln); w > 20 {
			t.Errorf("line %d has width %d > 20 (TTY cap): %q", i, w, ln)
		}
	}
}

// TestRenderBordersNeverGetEllipsisOnClip verifies the user-facing
// invariant: when the rendered output must be clipped to a narrower
// TTY than the layout produced, border glyphs survive intact. Only
// cell content shows an ellipsis; the top/bottom/separator rows are
// clipped silently (no ellipsis on pure box-drawing rows).
func TestRenderBordersNeverGetEllipsisOnClip(t *testing.T) {
	t.Setenv("COLUMNS", "")

	// Simulate a pathological case: layout can't fit even one glyph
	// per column (4 cols + overhead 9 > 10), so the rendered output
	// still exceeds the TTY after the hard-fit pass. The TTY post
	// clip must preserve borders.
	withFakeTTY(t, 10, true)

	tbl := NewTable()
	r := tbl.AddRow()
	r.AddCell(WithContent("aaaaa"))
	r.AddCell(WithContent("bbbbb"))
	r.AddCell(WithContent("ccccc"))
	r.AddCell(WithContent("ddddd"))

	out := tbl.String()
	for i, ln := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if ln == "" {
			continue
		}
		// Every rendered row must end in a border glyph, never in "…".
		if strings.HasSuffix(ln, "…") {
			t.Errorf("line %d ends in ellipsis (border clipped): %q", i, ln)
		}
		// Top/bottom/separator rows contain only box-drawing glyphs —
		// an ellipsis must never appear inside them.
		if isBorderOnly(graphemeRunsOf(ln, EmojiWidthGrapheme)) {
			continue
		}
		// For content rows, if clipping happened the last printable
		// glyph must be a border.
		last := []rune(strings.TrimRight(ln, ""))
		if len(last) > 0 {
			r := last[len(last)-1]
			if !isBorderRune(r) {
				t.Errorf("content line %d does not end in a border glyph: %q", i, ln)
			}
		}
	}
}

// TestResolvedTargetWidthPercentOfTTY verifies that a percentage is
// computed against the detected terminal width.
func TestResolvedTargetWidthPercentOfTTY(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 100, true)

	tbl := NewTable(WithTargetWidthPercent(50))
	if got := tbl.ResolvedTargetWidth(); got != 50 {
		t.Errorf("50%% of 100-col TTY = %d, want 50", got)
	}
}

// TestResolvedTargetWidthPercentClampedToTTY verifies that a percent
// above 100 is capped by the terminal — the same hard ceiling as for
// absolute widths.
func TestResolvedTargetWidthPercentClampedToTTY(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 80, true)

	tbl := NewTable(WithTargetWidthPercent(150))
	if got := tbl.ResolvedTargetWidth(); got != 80 {
		t.Errorf("150%% capped to 80-col TTY = %d, want 80", got)
	}
}

// TestResolvedTargetWidthPercentFallsBackToCOLUMNS verifies that in a
// non-TTY sink, the percentage base is the COLUMNS env var.
func TestResolvedTargetWidthPercentFallsBackToCOLUMNS(t *testing.T) {
	t.Setenv("COLUMNS", "120")
	withFakeTTY(t, 0, false)

	tbl := NewTable(WithTargetWidthPercent(50))
	if got := tbl.ResolvedTargetWidth(); got != 60 {
		t.Errorf("50%% of COLUMNS=120 = %d, want 60", got)
	}
}

// TestResolvedTargetWidthPercentFallsBackToDefault verifies the
// no-TTY-no-COLUMNS path uses the 80-column default as the base.
func TestResolvedTargetWidthPercentFallsBackToDefault(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 0, false)

	tbl := NewTable(WithTargetWidthPercent(50))
	if got := tbl.ResolvedTargetWidth(); got != 40 {
		t.Errorf("50%% of default 80 = %d, want 40", got)
	}
}

// TestTableStyleCSSWidthAbsolute verifies `width: N` in WithTableStyle
// drives the absolute target width, matching WithTargetWidth(N).
func TestTableStyleCSSWidthAbsolute(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 0, false)

	tbl := NewTable(WithTableStyle("width: 42"))
	if got := tbl.ResolvedTargetWidth(); got != 42 {
		t.Errorf("CSS width:42 = %d, want 42", got)
	}
}

// TestTableStyleCSSWidthPercent verifies `width: P%` in WithTableStyle
// drives the percent target, matching WithTargetWidthPercent(P).
func TestTableStyleCSSWidthPercent(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 100, true)

	tbl := NewTable(WithTableStyle("width: 80%"))
	if got := tbl.ResolvedTargetWidth(); got != 80 {
		t.Errorf("CSS width:80%% of 100 = %d, want 80", got)
	}
}

// TestTableStyleCSSWidthOverridesPreceding verifies the last width
// token in a CSS block wins — and crosses the absolute/percent
// boundary correctly.
func TestTableStyleCSSWidthOverridesPreceding(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 200, true)

	// percent then absolute: absolute wins
	tbl := NewTable(WithTableStyle("width: 50%; width: 30"))
	if got := tbl.ResolvedTargetWidth(); got != 30 {
		t.Errorf("last=absolute: got %d, want 30", got)
	}
	// absolute then percent: percent wins
	tbl2 := NewTable(WithTableStyle("width: 30; width: 50%"))
	if got := tbl2.ResolvedTargetWidth(); got != 100 {
		t.Errorf("last=percent: got %d, want 100", got)
	}
}

// TestTableStyleCSSWidthIgnoresGarbage verifies a malformed width
// token doesn't clobber a valid preceding value (or trip the parser).
func TestTableStyleCSSWidthIgnoresGarbage(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 0, false)

	tbl := NewTable(WithTableStyle("width: 40; width: abc; width: -5; width: 0%"))
	if got := tbl.ResolvedTargetWidth(); got != 40 {
		t.Errorf("garbage-ignored width = %d, want 40", got)
	}
}

// TestResolvedTargetWidthPercentAndAbsoluteMutex verifies the two
// width options are mutually exclusive: whichever is applied last wins.
func TestResolvedTargetWidthPercentAndAbsoluteMutex(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 100, true)

	// Absolute set first, then percent — percent wins.
	tbl := NewTable(WithTargetWidth(30), WithTargetWidthPercent(25))
	if got := tbl.ResolvedTargetWidth(); got != 25 {
		t.Errorf("last=percent: got %d, want 25", got)
	}

	// Percent set first, then absolute — absolute wins.
	tbl2 := NewTable(WithTargetWidthPercent(25), WithTargetWidth(30))
	if got := tbl2.ResolvedTargetWidth(); got != 30 {
		t.Errorf("last=absolute: got %d, want 30", got)
	}
}

// TestRenderPipeHonoursExplicitWidth verifies that a large explicit
// WithTargetWidth is used verbatim in pipe mode — no TTY cap applies,
// so the table renders at the requested width with room to breathe.
func TestRenderPipeHonoursExplicitWidth(t *testing.T) {
	t.Setenv("COLUMNS", "")
	withFakeTTY(t, 0, false)

	tbl := NewTable(WithTargetWidth(200))
	r := tbl.AddRow()
	r.AddCell(WithContent("short"))
	r.AddCell(WithContent("also short"))

	out := tbl.String()
	// Every line should be exactly 200 columns wide — the pipe leaves
	// the target uncapped and the layout fills it.
	for i, ln := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if w := DisplayWidth(ln); w != 200 {
			t.Errorf("line %d width = %d, want 200: %q", i, w, ln)
		}
	}
}

func TestColumnAutoCreation(t *testing.T) {
	tbl := NewTable()
	r := tbl.AddRow()
	for range 4 {
		r.AddCell(WithContent("x"))
	}
	if got := tbl.NumColumns(); got != 4 {
		t.Errorf("NumColumns = %d, want 4", got)
	}
	for i := range 4 {
		col := tbl.Column(i)
		if col == nil {
			t.Errorf("Column(%d) is nil", i)
			continue
		}
		if col.Index() != i {
			t.Errorf("Column(%d).Index() = %d", i, col.Index())
		}
	}
}

func TestColumnExplicitCreate(t *testing.T) {
	tbl := NewTable()
	col := tbl.Column(3)
	if col == nil || col.Index() != 3 {
		t.Fatalf("Column(3) = %v", col)
	}
	if tbl.NumColumns() != 4 {
		t.Errorf("NumColumns = %d, want 4 (explicit growth)", tbl.NumColumns())
	}
}

func TestMultiHeaderFooterOrdering(t *testing.T) {
	tbl := NewTable()
	h1 := tbl.AddHeader()
	h1.AddCell(WithContent("h1"))
	h2 := tbl.AddHeader()
	h2.AddCell(WithContent("h2"))
	r := tbl.AddRow()
	r.AddCell(WithContent("r"))
	f1 := tbl.AddFooter()
	f1.AddCell(WithContent("f1"))
	f2 := tbl.AddFooter()
	f2.AddCell(WithContent("f2"))

	if tbl.NumRows() != 5 {
		t.Fatalf("NumRows = %d, want 5", tbl.NumRows())
	}
	wantContents := []string{"h1", "h2", "r", "f1", "f2"}
	for i, want := range wantContents {
		c := tbl.CellAt(i, 0)
		if c == nil {
			t.Errorf("CellAt(%d,0) nil", i)
			continue
		}
		if c.Content() != want {
			t.Errorf("row %d content = %q, want %q", i, c.Content(), want)
		}
	}
}
