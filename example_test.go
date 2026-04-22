// SPDX-FileCopyrightText: Copyright 2026 Carabiner Systems, Inc
// SPDX-License-Identifier: Apache-2.0

package termtable_test

import (
	"fmt"

	"github.com/fatih/color"

	"github.com/carabiner-dev/termtable"
)

// must is a small helper used by the examples to keep them readable.
// Example functions assume construction succeeds; any unexpected
// error is surfaced loudly rather than swallowed.
func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// Example demonstrates the minimal usage pattern: build a table,
// add headers and rows, print the result.
func Example() {
	// Suppress ANSI codes so the expected output below is stable
	// regardless of where `go test` is run.
	color.NoColor = true

	t := termtable.NewTable(termtable.WithTargetWidth(30))

	h := must(t.AddHeader())
	must(h.AddCell(termtable.WithContent("Name")))
	must(h.AddCell(termtable.WithContent("Count")))

	r1 := must(t.AddRow())
	must(r1.AddCell(termtable.WithContent("alpha")))
	must(r1.AddCell(termtable.WithContent("1")))

	r2 := must(t.AddRow())
	must(r2.AddCell(termtable.WithContent("beta")))
	must(r2.AddCell(termtable.WithContent("2")))

	fmt.Print(t.String())
	// Output:
	// ┌──────────────┬─────────────┐
	// │ Name         │ Count       │
	// ├──────────────┼─────────────┤
	// │ alpha        │ 1           │
	// ├──────────────┼─────────────┤
	// │ beta         │ 2           │
	// └──────────────┴─────────────┘
}

// Example_columns shows configuring column widths and alignment with
// the CSS-style Column.Style helper.
func Example_columns() {
	color.NoColor = true

	t := termtable.NewTable(termtable.WithTargetWidth(40))
	t.Column(0).Style("min-width: 10")
	t.Column(1).Style("width: 6; text-align: center")
	t.Column(2).Style("flex: 2")

	h := must(t.AddHeader())
	must(h.AddCell(termtable.WithContent("Check")))
	must(h.AddCell(termtable.WithContent("Sts")))
	must(h.AddCell(termtable.WithContent("Message")))

	r := must(t.AddRow())
	must(r.AddCell(termtable.WithContent("lookup")))
	must(r.AddCell(termtable.WithContent("OK")))
	must(r.AddCell(termtable.WithContent("all good")))

	fmt.Print(t.String())
	// Output:
	// ┌───────────────┬────────┬─────────────┐
	// │ Check         │  Sts   │ Message     │
	// ├───────────────┼────────┼─────────────┤
	// │ lookup        │   OK   │ all good    │
	// └───────────────┴────────┴─────────────┘
}

// Example_borderStyle shows selecting an alternate border glyph set
// through the table's CSS-style configuration.
func Example_borderStyle() {
	color.NoColor = true

	t := termtable.NewTable(
		termtable.WithTargetWidth(30),
		termtable.WithTableStyle("border-style: rounded"),
	)

	h := must(t.AddHeader())
	must(h.AddCell(termtable.WithContent("Col A")))
	must(h.AddCell(termtable.WithContent("Col B")))

	r := must(t.AddRow())
	must(r.AddCell(termtable.WithContent("one")))
	must(r.AddCell(termtable.WithContent("two")))

	fmt.Print(t.String())
	// Output:
	// ╭──────────────┬─────────────╮
	// │ Col A        │ Col B       │
	// ├──────────────┼─────────────┤
	// │ one          │ two         │
	// ╰──────────────┴─────────────╯
}
