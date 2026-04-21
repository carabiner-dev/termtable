// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrorSentinelsWrap(t *testing.T) {
	cases := []error{
		ErrSpanConflict,
		ErrDuplicateID,
		ErrContentAndReader,
		ErrReaderAlreadyConsumed,
		ErrTargetTooNarrow,
		ErrCellAlreadyAdopted,
		ErrInvalidSpan,
		ErrCrossSectionSpan,
	}
	for _, sentinel := range cases {
		wrapped := fmt.Errorf("context: %w", sentinel)
		if !errors.Is(wrapped, sentinel) {
			t.Errorf("errors.Is(wrapped, %v) = false", sentinel)
		}
	}
}

func TestWarningStringers(t *testing.T) {
	drop := OverwriteEvent{DroppedID: "x", At: [2]int{1, 2}}
	if drop.String() == "" {
		t.Error("drop String should be non-empty")
	}
	trunc := OverwriteEvent{TruncatedID: "y", NewColSpan: 1, NewRowSpan: 1, At: [2]int{0, 0}}
	if trunc.String() == "" {
		t.Error("trunc String should be non-empty")
	}
	over := SpanOverflowEvent{CellID: "z", Required: 10, Got: 6}
	if over.String() == "" {
		t.Error("overflow String should be non-empty")
	}
}
