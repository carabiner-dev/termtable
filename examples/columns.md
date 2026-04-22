# Column configuration

Three knobs compose: `SetMin` for a floor, `SetWidth` to pin,
`SetWeight` to distribute leftover proportionally. Here the first
column is anchored at min=12, the second is pinned at exactly 6
and center-aligned, and the third flexes three times as much as
everything else.

```go
t := termtable.NewTable(termtable.WithTargetWidth(60))

t.Column(0).SetMin(12)
t.Column(1).SetWidth(6).SetAlign(termtable.AlignCenter)
t.Column(2).SetWeight(3)

h := t.AddHeader()
h.AddCell(termtable.WithContent("Check"))
h.AddCell(termtable.WithContent("Sts"))
h.AddCell(termtable.WithContent("Message"))

r1 := t.AddRow()
r1.AddCell(termtable.WithContent("OSPS-BR-05"))
r1.AddCell(termtable.WithContent("PASS"))
r1.AddCell(termtable.WithContent("all criteria satisfied"))

r2 := t.AddRow()
r2.AddCell(termtable.WithContent("OSPS-DO-02"))
r2.AddCell(termtable.WithContent("FAIL"))
r2.AddCell(termtable.WithContent("needs attention soon"))

fmt.Print(t.String())
```

```
┌────────────────────┬────────┬────────────────────────────┐
│ Check              │  Sts   │ Message                    │
├────────────────────┼────────┼────────────────────────────┤
│ OSPS-BR-05         │  PASS  │ all criteria satisfied     │
├────────────────────┼────────┼────────────────────────────┤
│ OSPS-DO-02         │  FAIL  │ needs attention soon       │
└────────────────────┴────────┴────────────────────────────┘
```

Same layout using CSS on columns:

```go
t.Column(0).Style("min-width: 12")
t.Column(1).Style("width: 6; text-align: center")
t.Column(2).Style("flex: 3")
```

Related docs: [../docs/columns.md](../docs/columns.md) has the full
solver description and a gallery for each primitive
(width / min / max / weight / alignment).
