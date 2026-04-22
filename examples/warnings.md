# Warnings

Non-fatal events — span overwrites, layout overflows, reader
failures, cross-section spans — accumulate on `Table.Warnings()`.
They never abort rendering; best-effort output is always produced.

```go
type boomReader struct{}

func (boomReader) Read([]byte) (int, error) {
    return 0, errors.New("network down")
}

t := termtable.NewTable(
    termtable.WithTargetWidth(30),
    termtable.WithSpanOverwrite(true),
)

// Row 1 will have its cell dropped by the rowspan overwrite
// from row 0.
r1, _ := t.AddRow()
r2, _ := t.AddRow()
r2.AddCell(termtable.WithCellID("victim"), termtable.WithContent("v"))
r1.AddCell(termtable.WithContent("over"), termtable.WithRowSpan(2))

// A cell whose reader fails on consumption.
r3, _ := t.AddRow()
r3.AddCell(termtable.WithCellID("broken"), termtable.WithReader(boomReader{}))

fmt.Print(t.String())
for _, w := range t.Warnings() {
    fmt.Printf("  - %s\n", w)
}
```

```
┌────────────────────────────┐
│ over                       │
│                            │
├────────────────────────────┤
│                            │
└────────────────────────────┘
  - overwrite: dropped cell id="victim"
  - reader error: cell id="broken": network down
```

Authoring events (span overwrites, ID collisions) persist across
renders; render events (span overflow, reader errors,
cross-section clamps) are reset on every `String`/`WriteTo` call
so repeated renders don't double-count.

Related docs: [../docs/warnings.md](../docs/warnings.md) for the
full event-type catalogue, inspection recipes, and the
`Table.LastRenderError` accessor for `String` callers.
