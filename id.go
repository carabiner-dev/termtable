// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable

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

// register records the mapping id -> e. Returns true when the id
// was accepted (either because it was newly inserted, unchanged,
// or empty — empty IDs are a no-op). Returns false when id is
// already claimed by a different element; the existing mapping is
// preserved and the caller is expected to surface a
// DuplicateIDEvent warning.
func (r *idRegistry) register(id string, e Element) bool {
	if id == "" {
		return true
	}
	if existing, ok := r.m[id]; ok && existing != e {
		return false
	}
	r.m[id] = e
	return true
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
