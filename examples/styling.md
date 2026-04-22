# Styling

Colour, bold, and background cascade through `Style` at the table,
column, row, and cell levels. The output below is shown with ANSI
codes stripped; on a real terminal borders are cyan, the header
row is white-on-blue-bold, `PASS` is green-bold, and `FAIL` is
red-bold.

```go
t := termtable.NewTable(
    termtable.WithTargetWidth(40),
    termtable.WithTableStyle("border-color: cyan"),
)

hdr := t.AddHeader(termtable.WithRowStyle(
    "color: white; background: blue; font-weight: bold",
))
hdr.AddCell(termtable.WithContent("Check"))
hdr.AddCell(termtable.WithContent("Status"))
hdr.AddCell(termtable.WithContent("Message"))

r1 := t.AddRow()
r1.AddCell(termtable.WithContent("OSPS-BR-05"))
r1.AddCell(termtable.WithContent("PASS"),
    termtable.WithAlign(termtable.AlignCenter),
    termtable.WithTextColor("green"),
    termtable.WithBold(),
)
r1.AddCell(termtable.WithContent("all good"))

r2 := t.AddRow()
r2.AddCell(termtable.WithContent("OSPS-DO-02"))
r2.AddCell(termtable.WithContent("FAIL"),
    termtable.WithAlign(termtable.AlignCenter),
    termtable.WithCellStyle("color: red; font-weight: bold"),
)
r2.AddCell(termtable.WithContent("review deps"))

fmt.Print(t.String())
```

```
┌───────────────┬──────────┬───────────┐
│ Check         │ Status   │ Message   │
├───────────────┼──────────┼───────────┤
│ OSPS-BR-05    │   PASS   │ all good  │
├───────────────┼──────────┼───────────┤
│ OSPS-DO-02    │   FAIL   │ review    │
│               │          │ deps      │
└───────────────┴──────────┴───────────┘
```

Related docs: [../docs/styling.md](../docs/styling.md) for the full
property reference, colour grammar, and cascade precedence.
