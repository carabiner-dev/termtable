// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

// Package termtable provides a DOM-like model for composing and rendering
// text tables on a terminal. It handles Unicode width (including emoji and
// CJK), ANSI escape sequences, word-wrapping, trimming, and column/row
// spanning with single-line Unicode box-drawing borders.
//
// A table is composed of headers, body rows, and footers. Each row contains
// cells. Cells can span multiple columns (ColSpan) and multiple rows within
// the same section (RowSpan). Headers, body, and footers form separate
// sections in the grid; rowspans do not cross section boundaries.
//
// Elements can be addressed two ways:
//
//   - Logical: row.Cell(i) returns the i-th declared cell in that row.
//   - Grid: table.CellAt(r, c) returns the cell covering the absolute grid
//     coordinate (r, c); multiple (r, c) pairs map to the same cell when it
//     spans.
//
// Any element may be tagged with a unique ID via its With*ID option and
// looked up with table.GetElementByID.
//
// Rendering proceeds in three passes: measurement, layout, and paint.
// The table is fully buffered; there is no streaming output.
//
// Known limitations (Phase 1–3):
//
//   - Right-to-left / bidirectional text is not supported.
//   - Styling (colors, bold, alternate border styles) is reserved for a later
//     phase; option hooks exist but have no effect yet.
//   - Terminal width is read from the COLUMNS environment variable; it
//     falls back to 80 when unset.
package termtable
