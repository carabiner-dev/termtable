# CSS-driven configuration

Everything styleable has a CSS entry point: table, row, column, and
cell all accept `Style` / `WithTableStyle` / `WithRowStyle` /
`WithCellStyle` declaration blocks. This sample wires every major
knob through CSS rather than imperatively.

```go
t := termtable.NewTable(
    termtable.WithTargetWidth(50),
    termtable.WithTableStyle("border-style: rounded; border-color: cyan"),
)

t.Column(0).Style("min-width: 12")
t.Column(1).Style("width: 8; text-align: center")
t.Column(2).Style("flex: 3")

hdr, _ := t.AddHeader(termtable.WithRowStyle(
    "color: white; background: blue; font-weight: bold",
))
hdr.AddCell(termtable.WithContent("Check"))
hdr.AddCell(termtable.WithContent("Status"))
hdr.AddCell(termtable.WithContent("Message"))

r, _ := t.AddRow()
r.AddCell(termtable.WithContent("OSPS-BR-05"))
r.AddCell(
    termtable.WithContent("PASS"),
    termtable.WithCellStyle("color: green; font-weight: bold"),
)
r.AddCell(termtable.WithContent("all criteria satisfied"))

fmt.Print(t.String())
```

```
╭─────────────────┬──────────┬───────────────────╮
│ Check           │  Status  │ Message           │
├─────────────────┼──────────┼───────────────────┤
│ OSPS-BR-05      │   PASS   │ all criteria      │
│                 │          │ satisfied         │
╰─────────────────┴──────────┴───────────────────╯
```

All supported properties and their value grammars are listed in
[../docs/styling.md](../docs/styling.md). Column-specific
extensions (`width`, `min-width`, `max-width`, `flex`,
`text-align`) live in [../docs/columns.md](../docs/columns.md);
layout properties (`white-space`, `text-overflow`,
`text-overflow-position`, `line-clamp`) are in
[../docs/wrapping.md](../docs/wrapping.md); `border-style` in
[../docs/borders.md](../docs/borders.md).
