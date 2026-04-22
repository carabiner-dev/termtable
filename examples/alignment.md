# Alignment

Horizontal alignment (`WithAlign`, `SetAlign`, or `text-align`) and
vertical alignment (`WithVAlign`, `SetVAlign`, or `vertical-align`)
both cascade table → column → row → cell. In the sample below the
middle column inherits `AlignCenter` from the column, the Notes
cell overrides to right-align, and the Notes body cell centers
vertically within the row.

```go
t := termtable.NewTable(termtable.WithTargetWidth(40))
t.Column(1).SetAlign(termtable.AlignCenter)

h, _ := t.AddHeader()
h.AddCell(termtable.WithContent("Check"))
h.AddCell(termtable.WithContent("Status")) // inherits column center
h.AddCell(termtable.WithContent("Notes"),
    termtable.WithAlign(termtable.AlignRight),
)

r, _ := t.AddRow()
r.AddCell(termtable.WithContent("lookup"))
r.AddCell(termtable.WithContent("PASS")) // inherits column center
r.AddCell(
    termtable.WithContent("retried twice\nthen succeeded"),
    termtable.WithVAlign(termtable.VAlignMiddle),
)

fmt.Print(t.String())
```

```
┌───────────┬───────────┬──────────────┐
│ Check     │  Status   │        Notes │
├───────────┼───────────┼──────────────┤
│ lookup    │   PASS    │ retried      │
│           │           │ twice        │
│           │           │ then         │
│           │           │ succeeded    │
└───────────┴───────────┴──────────────┘
```

Related docs: [../docs/styling.md](../docs/styling.md) for the full
cascade diagram; [../docs/columns.md](../docs/columns.md) for
column-level helpers.
