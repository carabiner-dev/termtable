// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

import "fmt"

// Element is the sealed interface implemented by every addressable object
// in a table: *Table, *Header, *Footer, *Row, *Cell, *Column. Callers
// resolve to a concrete type with a type switch.
type Element interface {
	elementID() string
}

// idRegistry maps element IDs to the owning element. IDs are unique
// across the whole table; empty IDs are never registered.
type idRegistry struct {
	m map[string]Element
}

func newIDRegistry() *idRegistry {
	return &idRegistry{m: make(map[string]Element)}
}

func (r *idRegistry) register(id string, e Element) error {
	if id == "" {
		return nil
	}
	if existing, ok := r.m[id]; ok && existing != e {
		return fmt.Errorf("id %q already registered: %w", id, ErrDuplicateID)
	}
	r.m[id] = e
	return nil
}

func (r *idRegistry) unregister(id string) {
	if id == "" {
		return
	}
	delete(r.m, id)
}

func (r *idRegistry) lookup(id string) Element {
	if id == "" {
		return nil
	}
	return r.m[id]
}
