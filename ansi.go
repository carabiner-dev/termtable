// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

// This file implements ANSI escape-sequence scanning. Two concerns meet
// here: width measurement must ignore the bytes that produce no
// visible output (so "\x1b[31mA\x1b[0m" is width 1, not 9), and
// wrapping must preserve those bytes verbatim so colored text survives
// a line break. scanANSI returns the input's text / escape segmentation
// so downstream code can handle each concern independently.

// ansiSegKind tags a byte range as visible text or an ANSI escape
// sequence.
type ansiSegKind uint8

const (
	segText ansiSegKind = iota
	segEsc
)

// ansiSegment is a half-open byte range [start, end) and its kind.
type ansiSegment struct {
	start, end int
	kind       ansiSegKind
}

// scanANSI splits s into alternating text and escape-sequence segments.
// It recognizes Fe escapes (CSI with \x1b[, OSC with \x1b], plus the
// string-terminating variants \x1bP / \x1bX / \x1b^ / \x1b_), charset
// designators (\x1b followed by one of ()*+-./), and single-byte Fe
// escapes in the 0x40..0x5F range. Malformed or truncated escapes are
// captured as escape segments and stripped by StripANSI.
//
// The function never allocates for input without escapes beyond the
// returned slice.
func scanANSI(s string) []ansiSegment {
	if !hasESC(s) {
		return []ansiSegment{{0, len(s), segText}}
	}
	var segs []ansiSegment
	i, n := 0, len(s)
	for i < n {
		if s[i] != 0x1B {
			start := i
			for i < n && s[i] != 0x1B {
				i++
			}
			segs = append(segs, ansiSegment{start, i, segText})
			continue
		}
		start := i
		i = consumeEscape(s, i)
		segs = append(segs, ansiSegment{start, i, segEsc})
	}
	return segs
}

// consumeEscape advances past a single ANSI escape starting at s[i],
// where s[i] is known to be 0x1B. Returns the index immediately after
// the escape (clamped at len(s) for truncated sequences).
func consumeEscape(s string, i int) int {
	n := len(s)
	i++ // skip ESC
	if i >= n {
		return i
	}
	c := s[i]
	i++ // skip byte after ESC
	switch c {
	case '[':
		return consumeCSI(s, i)
	case ']', 'P', 'X', '^', '_':
		return consumeString(s, i)
	case '(', ')', '*', '+', '-', '.', '/':
		if i < n {
			i++ // charset designator has one more byte
		}
		return i
	default:
		return i
	}
}

// consumeCSI parses a CSI payload: zero or more parameter bytes
// (0x30..0x3F), zero or more intermediate bytes (0x20..0x2F), and a
// single final byte (0x40..0x7E).
func consumeCSI(s string, i int) int {
	n := len(s)
	for i < n && s[i] >= 0x30 && s[i] <= 0x3F {
		i++
	}
	for i < n && s[i] >= 0x20 && s[i] <= 0x2F {
		i++
	}
	if i < n && s[i] >= 0x40 && s[i] <= 0x7E {
		i++
	}
	return i
}

// consumeString consumes bytes until a string-terminator is found:
// either BEL (0x07) or ST (\x1b\). Unterminated sequences run to EOF.
func consumeString(s string, i int) int {
	n := len(s)
	for i < n {
		if s[i] == 0x07 {
			return i + 1
		}
		if s[i] == 0x1B && i+1 < n && s[i+1] == '\\' {
			return i + 2
		}
		i++
	}
	return i
}

// hasESC is a fast-path check that avoids segmenting inputs that
// contain no escape bytes at all.
func hasESC(s string) bool {
	for i := range len(s) {
		if s[i] == 0x1B {
			return true
		}
	}
	return false
}

// StripANSI returns s with all ANSI escape sequences removed. Visible
// text bytes are preserved byte-for-byte.
func StripANSI(s string) string {
	if !hasESC(s) {
		return s
	}
	b := make([]byte, 0, len(s))
	for _, seg := range scanANSI(s) {
		if seg.kind == segText {
			b = append(b, s[seg.start:seg.end]...)
		}
	}
	return string(b)
}
