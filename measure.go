// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import "io"

// multiSpanConstraint is emitted by Measure for each cell whose colSpan
// exceeds 1. The solver uses it to enforce a joint width requirement
// across the cell's column range rather than a single-column minimum.
type multiSpanConstraint struct {
	colStart     int
	colSpan      int
	minWidth     int
	desiredWidth int
	cellID       string
}

// measureResult holds the per-column widths plus multi-span
// constraints accumulated from all cells in the table. colMin and
// colDesired are indexed by absolute column index.
type measureResult struct {
	colMin     []int
	colDesired []int
	multiSpans []multiSpanConstraint
	readerErrs []error
}

// Measure performs Pass 1 of rendering: it walks every cell in the
// table, consumes any WithReader content (caching the bytes), and
// accumulates per-column minimum and desired display widths.
//
// Single-span cells contribute directly to the column they occupy.
// Multi-span cells contribute a joint constraint that the layout
// solver balances across the cell's column range.
//
// Reader failures are recorded in readerErrs but do not abort
// measurement; affected cells are treated as empty.
func Measure(t *Table) *measureResult {
	nCols := t.NumColumns()
	out := &measureResult{
		colMin:     make([]int, nCols),
		colDesired: make([]int, nCols),
	}
	visit := func(c *Cell) {
		content, err := resolveCellContent(c)
		if err != nil {
			out.readerErrs = append(out.readerErrs, err)
		}
		minW := MinUnbreakableWidth(content)
		desW := maxLineWidth(content)
		if c.colSpan == 1 {
			if minW > out.colMin[c.gridCol] {
				out.colMin[c.gridCol] = minW
			}
			if desW > out.colDesired[c.gridCol] {
				out.colDesired[c.gridCol] = desW
			}
			return
		}
		out.multiSpans = append(out.multiSpans, multiSpanConstraint{
			colStart:     c.gridCol,
			colSpan:      c.colSpan,
			minWidth:     minW,
			desiredWidth: desW,
			cellID:       c.id,
		})
	}
	for _, h := range t.headers {
		for _, c := range h.cells {
			visit(c)
		}
	}
	for _, r := range t.rows {
		for _, c := range r.cells {
			visit(c)
		}
	}
	for _, f := range t.footers {
		for _, c := range f.cells {
			visit(c)
		}
	}
	return out
}

// resolveCellContent returns the cell's text, consuming and caching
// any attached io.Reader on the first call. Subsequent calls reuse
// the cached bytes or the cached error.
func resolveCellContent(c *Cell) (string, error) {
	if c.hasContent {
		return c.content, nil
	}
	if c.reader == nil {
		return "", nil
	}
	if c.resolved {
		return c.content, c.resolveErr
	}
	c.resolved = true
	data, err := io.ReadAll(c.reader)
	if err != nil {
		c.resolveErr = err
		return "", err
	}
	c.content = string(data)
	c.hasContent = true
	return c.content, nil
}

// maxLineWidth returns the display width of the widest natural line in
// s. "Natural" here means separated by '\n'; CRLF is handled by
// NaturalLines. The widest line dictates the column width needed to
// render the content without wrapping.
func maxLineWidth(s string) int {
	lines := NaturalLines(s)
	var maxW int
	for _, line := range lines {
		w := 0
		for _, r := range line {
			w += r.Width
		}
		if w > maxW {
			maxW = w
		}
	}
	return maxW
}
