// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

// Alignment controls horizontal placement of wrapped cell content within
// its allotted width.
type Alignment uint8

const (
	// AlignLeft is the default: content hugs the left edge, right padded
	// to fill the column.
	AlignLeft Alignment = iota
	// AlignCenter distributes padding on both sides; any odd remainder
	// goes to the right side.
	AlignCenter
	// AlignRight hugs the right edge, left padded to fill the column.
	AlignRight
)

func (a Alignment) String() string {
	switch a {
	case AlignLeft:
		return "left"
	case AlignCenter:
		return "center"
	case AlignRight:
		return "right"
	default:
		return "unknown"
	}
}

// VerticalAlignment controls where a cell's wrapped content sits
// within a row (or rowspan block) that is taller than the content
// itself — a common situation when one cell wraps to multiple lines
// and its neighbours have only one.
type VerticalAlignment uint8

const (
	// VAlignTop is the default: content hugs the top of the cell,
	// any extra vertical space sits below.
	VAlignTop VerticalAlignment = iota
	// VAlignMiddle distributes extra space evenly above and below
	// the content; any odd remainder goes to the bottom.
	VAlignMiddle
	// VAlignBottom pushes content to the bottom; any extra vertical
	// space sits above.
	VAlignBottom
)

func (v VerticalAlignment) String() string {
	switch v {
	case VAlignTop:
		return "top"
	case VAlignMiddle:
		return "middle"
	case VAlignBottom:
		return "bottom"
	default:
		return "unknown"
	}
}
