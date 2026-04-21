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
