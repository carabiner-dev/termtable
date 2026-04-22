# Wrapping modes

Three rows of the same long description: default multi-line wrap,
single-line truncation, and line-clamp at 2.

```go
t := termtable.NewTable(termtable.WithTargetWidth(40))

h, _ := t.AddHeader()
h.AddCell(termtable.WithContent("Name"))
h.AddCell(termtable.WithContent("Description"))

r1, _ := t.AddRow()
r1.AddCell(termtable.WithContent("multi"))
r1.AddCell(termtable.WithContent(
    "this is a long description that wraps across several lines"))

r2, _ := t.AddRow()
r2.AddCell(termtable.WithContent("single"))
r2.AddCell(
    termtable.WithContent("this is a long description that wraps across several lines"),
    termtable.WithSingleLine(),
)

r3, _ := t.AddRow()
r3.AddCell(termtable.WithContent("clamp"))
r3.AddCell(
    termtable.WithContent("this is a long description that wraps across several lines"),
    termtable.WithCellStyle("line-clamp: 2"),
)

fmt.Print(t.String())
```

```
┌────────────────┬─────────────────────┐
│ Name           │ Description         │
├────────────────┼─────────────────────┤
│ multi          │ this is a long      │
│                │ description that    │
│                │ wraps across        │
│                │ several lines       │
├────────────────┼─────────────────────┤
│ single         │ this is a long des… │
├────────────────┼─────────────────────┤
│ clamp          │ this is a long      │
│                │ description that…   │
└────────────────┴─────────────────────┘
```

All three are expressible as CSS too:

```css
white-space: normal            /* multi (default) */
white-space: nowrap             /* single */
white-space: normal; line-clamp: 2  /* clamp */
```

Related docs: [../docs/wrapping.md](../docs/wrapping.md) has the
full property table, recipes, and edge cases.
