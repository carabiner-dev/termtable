# Element IDs

Every element type (table, column, row, cell) can carry a unique
ID. `Table.GetElementByID` looks them up via a type switch at the
call site.

```go
t := termtable.NewTable(
    termtable.WithTargetWidth(30),
    termtable.WithTableID("checks"),
)

hdr, _ := t.AddHeader(termtable.WithRowID("head"))
hdr.AddCell(termtable.WithCellID("hc-check"), termtable.WithContent("Check"))
hdr.AddCell(termtable.WithCellID("hc-status"), termtable.WithContent("Status"))

r, _ := t.AddRow(termtable.WithRowID("r1"))
r.AddCell(termtable.WithContent("lookup"))
r.AddCell(termtable.WithCellID("r1-status"), termtable.WithContent("PASS"))

// Columns get IDs imperatively.
_ = t.Column(1).SetID("status-col")

// Look up and narrow with a type switch.
switch e := t.GetElementByID("r1-status").(type) {
case *termtable.Cell:
    fmt.Printf("found cell %q with content %q\n", e.ID(), e.Content())
}

fmt.Print(t.String())
```

```
found cell "r1-status" with content "PASS"
┌──────────────┬─────────────┐
│ Check        │ Status      │
├──────────────┼─────────────┤
│ lookup       │ PASS        │
└──────────────┴─────────────┘
```

Unique IDs are enforced — a second element registering an
already-used ID gets `ErrDuplicateID`. Empty IDs are never
registered, so you only pay for IDs on elements you actually care
about.

Related docs: the [top-level README](../README.md) and
[../docs/warnings.md](../docs/warnings.md) (for the ID-collision
path).
