// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import "testing"

func TestStripANSIBasic(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"plain", "plain"},
		{"\x1b[31mred\x1b[0m", "red"},
		{"a\x1b[1mb\x1b[0mc", "abc"},
		{"\x1b[38;2;255;0;0mtrue color\x1b[0m", "true color"},
		// OSC 8 hyperlink, BEL terminator.
		{"\x1b]8;;http://example.com\x07link\x1b]8;;\x07", "link"},
		// OSC 8 hyperlink, ST terminator.
		{"\x1b]8;;http://example.com\x1b\\link\x1b]8;;\x1b\\", "link"},
		// Bracketed paste on/off.
		{"\x1b[?2004hvisible\x1b[?2004l", "visible"},
		// Charset designator: ESC ( B selects US ASCII; consumes 2 bytes.
		{"\x1b(Bplain", "plain"},
		// Truncated escape at EOF: dropped.
		{"data\x1b[31", "data"},
		{"\x1b", ""},
	}
	for _, tc := range cases {
		if got := StripANSI(tc.in); got != tc.want {
			t.Errorf("StripANSI(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestScanANSIPreservesBoundaries(t *testing.T) {
	in := "a\x1b[31mbc\x1b[0md"
	segs := scanANSI(in)
	// Expected: "a" text, "\x1b[31m" esc, "bc" text, "\x1b[0m" esc, "d" text.
	if len(segs) != 5 {
		t.Fatalf("segments = %d, want 5: %+v", len(segs), segs)
	}
	want := []ansiSegment{
		{0, 1, segText},
		{1, 6, segEsc},
		{6, 8, segText},
		{8, 12, segEsc},
		{12, 13, segText},
	}
	for i, w := range want {
		if segs[i] != w {
			t.Errorf("seg[%d] = %+v, want %+v", i, segs[i], w)
		}
	}
}

func TestScanANSINoEscapes(t *testing.T) {
	segs := scanANSI("plain text")
	if len(segs) != 1 || segs[0].kind != segText {
		t.Errorf("no-escape input should produce one text segment, got %+v", segs)
	}
}
