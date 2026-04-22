# Sections: headers, body, footers

Tables can have multiple header and footer rows in addition to the
body. Headers and footers follow the same API as body rows; headers
appear above, footers below, and rowspans stay within their
section.

```go
t := termtable.NewTable(termtable.WithTargetWidth(40))

banner, _ := t.AddHeader()
banner.AddCell(
    termtable.WithContent("Evaluation Results"),
    termtable.WithColSpan(3),
    termtable.WithAlign(termtable.AlignCenter),
)

cols, _ := t.AddHeader()
cols.AddCell(termtable.WithContent("Check"))
cols.AddCell(termtable.WithContent("Status"))
cols.AddCell(termtable.WithContent("Message"))

r1, _ := t.AddRow()
r1.AddCell(termtable.WithContent("OSPS-BR-05"))
r1.AddCell(termtable.WithContent("PASS"), termtable.WithAlign(termtable.AlignCenter))
r1.AddCell(termtable.WithContent("all good"))

r2, _ := t.AddRow()
r2.AddCell(termtable.WithContent("OSPS-DO-02"))
r2.AddCell(termtable.WithContent("FAIL"), termtable.WithAlign(termtable.AlignCenter))
r2.AddCell(termtable.WithContent("review deps"))

f, _ := t.AddFooter()
f.AddCell(
    termtable.WithContent("1 passed, 1 failed"),
    termtable.WithColSpan(3),
    termtable.WithAlign(termtable.AlignCenter),
)

fmt.Print(t.String())
```

```
┌──────────────────────────────────────┐
│          Evaluation Results          │
├───────────────┬──────────┬───────────┤
│ Check         │ Status   │ Message   │
├───────────────┼──────────┼───────────┤
│ OSPS-BR-05    │   PASS   │ all good  │
├───────────────┼──────────┼───────────┤
│ OSPS-DO-02    │   FAIL   │ review    │
│               │          │ deps      │
├───────────────┴──────────┴───────────┤
│          1 passed, 1 failed          │
└──────────────────────────────────────┘
```

Related docs: [../docs/borders.md](../docs/borders.md) covers how
the border joins resolve at section boundaries.
