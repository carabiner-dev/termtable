# Column and row spans

Cells can span multiple columns (`WithColSpan`), multiple rows
(`WithRowSpan`), or both. Border joins are suppressed where a span
crosses them — the renderer resolves the adjacent glyphs
automatically.

```go
t := termtable.NewTable(termtable.WithTargetWidth(40))

r0 := t.AddRow()
r0.AddCell(
    termtable.WithContent("big\nspan"),
    termtable.WithRowSpan(2),
    termtable.WithColSpan(2),
)
r0.AddCell(termtable.WithContent("alpha"))

// Row 1: column 0 and 1 are reserved by the rowspan above, so the
// first AddCell lands at column 2.
r1 := t.AddRow()
r1.AddCell(termtable.WithContent("beta"))

r2 := t.AddRow()
r2.AddCell(termtable.WithContent("gamma"))
r2.AddCell(termtable.WithContent("delta"))
r2.AddCell(termtable.WithContent("omega"))

fmt.Print(t.String())
```

```
┌─────────────────────────┬────────────┐
│ big                     │ alpha      │
│ span                    ├────────────┤
│                         │ beta       │
├────────────┬────────────┼────────────┤
│ gamma      │ delta      │ omega      │
└────────────┴────────────┴────────────┘
```

When a declared rowspan would reach beyond its section, termtable
clamps the effective span and emits a `CrossSectionSpanEvent` you
can inspect via `Table.Warnings()` — see
[warnings.md](warnings.md).

Related docs: [../docs/borders.md](../docs/borders.md) for how
border joins resolve around spans.
