// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

// unknownName is returned by String methods for enum values outside
// the documented range. Kept as a package-level constant so adding
// new enums doesn't require remembering the exact spelling.
const unknownName = "unknown"

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
		return unknownName
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
		return unknownName
	}
}

// TrimPosition controls where an ellipsis (or clip) lands when a
// cell's content needs to be truncated horizontally. It is only
// consulted when content is actually being truncated — cells that
// fit leave their content unchanged regardless of this setting.
type TrimPosition uint8

const (
	// TrimEnd (the default) keeps the content's prefix and places
	// the truncation marker at the right — e.g. "www.exampl…".
	TrimEnd TrimPosition = iota
	// TrimStart keeps the content's suffix and places the marker
	// at the left — e.g. "…/page.html".
	TrimStart
	// TrimMiddle keeps both ends and places the marker between —
	// e.g. "www.exam…/page.html".
	TrimMiddle
)

func (p TrimPosition) String() string {
	switch p {
	case TrimEnd:
		return "end"
	case TrimStart:
		return "start"
	case TrimMiddle:
		return "middle"
	default:
		return unknownName
	}
}
