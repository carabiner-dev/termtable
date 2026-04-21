// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"strings"
	"unicode"

	"github.com/rivo/uniseg"
)

// GraphemeRun carries a single grapheme cluster's text, its display
// width, and any ANSI escape bytes that immediately preceded it in the
// source (so a wrap-aware renderer can re-emit them). Cluster text
// never contains ANSI escapes and never contains '\n' — NaturalLines
// splits hard breaks before segmentation.
type GraphemeRun struct {
	Text      string
	Width     int
	EscPrefix string
}

// DisplayWidth returns the number of terminal columns s would occupy
// when rendered, ignoring ANSI escape sequences. Grapheme clusters are
// counted by their East Asian Width: most characters are 1 column,
// CJK and emoji are 2, combining marks are 0.
func DisplayWidth(s string) int {
	return uniseg.StringWidth(StripANSI(s))
}

// MinUnbreakableWidth returns the display width of the widest run of
// consecutive non-whitespace grapheme clusters in s (ANSI ignored).
// It is the smallest column width at which s can render without hard-
// breaking a word.
func MinUnbreakableWidth(s string) int {
	stripped := StripANSI(s)
	var maxW, curW int
	state := -1
	rest := stripped
	for rest != "" {
		var (
			cluster string
			w       int
		)
		cluster, rest, w, state = uniseg.FirstGraphemeClusterInString(rest, state)
		if isWhitespaceCluster(cluster) {
			if curW > maxW {
				maxW = curW
			}
			curW = 0
			continue
		}
		curW += w
	}
	if curW > maxW {
		maxW = curW
	}
	return maxW
}

// NaturalLines splits s on '\n' (hard breaks) and returns each line as
// a sequence of grapheme runs with preserved ANSI escape prefixes. A
// trailing '\r' on each split line is removed so CRLF inputs behave
// the same as LF inputs.
//
// An input of "" produces a single empty line. An input of "\n"
// produces two empty lines.
func NaturalLines(s string) [][]GraphemeRun {
	rawLines := strings.Split(s, "\n")
	out := make([][]GraphemeRun, 0, len(rawLines))
	for _, line := range rawLines {
		out = append(out, graphemeRunsOf(strings.TrimSuffix(line, "\r")))
	}
	return out
}

// graphemeRunsOf segments a single line (containing no '\n') into
// grapheme runs, attaching any intervening ANSI escape sequences as
// EscPrefix of the next visible cluster. Trailing escapes with no
// following cluster are dropped.
func graphemeRunsOf(line string) []GraphemeRun {
	if line == "" {
		return nil
	}
	segs := scanANSI(line)
	var runs []GraphemeRun
	var pendingEsc strings.Builder
	state := -1
	for _, seg := range segs {
		if seg.kind == segEsc {
			pendingEsc.WriteString(line[seg.start:seg.end])
			continue
		}
		rest := line[seg.start:seg.end]
		for rest != "" {
			var (
				cluster string
				w       int
			)
			cluster, rest, w, state = uniseg.FirstGraphemeClusterInString(rest, state)
			runs = append(runs, GraphemeRun{
				Text:      cluster,
				Width:     w,
				EscPrefix: pendingEsc.String(),
			})
			pendingEsc.Reset()
		}
	}
	return runs
}

// isWhitespaceCluster reports whether every rune in the cluster is
// Unicode whitespace. Zero-width modifiers combined with whitespace
// keep the cluster whitespace-like; any non-whitespace rune makes it
// non-whitespace overall.
func isWhitespaceCluster(cluster string) bool {
	if cluster == "" {
		return false
	}
	for _, r := range cluster {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}
