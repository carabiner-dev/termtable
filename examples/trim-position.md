# Trim position

When a URL, path, or ID doesn't fit a single-line column, choose
where the ellipsis lands. `TrimEnd` (default) shows the prefix,
`TrimStart` the suffix, `TrimMiddle` both ends.

```go
t := termtable.NewTable(termtable.WithTargetWidth(40))
t.Column(1).Style("white-space: nowrap")

h, _ := t.AddHeader()
h.AddCell(termtable.WithContent("Mode"))
h.AddCell(termtable.WithContent("URL"))

urls := []string{
    "https://www.example.com/page1.html",
    "https://www.example.com/page2.html",
    "https://api.example.com/v1/items/42",
}

for _, row := range []struct {
    label string
    pos   termtable.TrimPosition
}{
    {"end", termtable.TrimEnd},
    {"start", termtable.TrimStart},
    {"middle", termtable.TrimMiddle},
} {
    for i, u := range urls {
        r, _ := t.AddRow()
        if i == 0 {
            r.AddCell(
                termtable.WithContent(row.label),
                termtable.WithRowSpan(3),
                termtable.WithVAlign(termtable.VAlignMiddle),
            )
        }
        r.AddCell(
            termtable.WithContent(u),
            termtable.WithTrimPosition(row.pos),
        )
    }
}

fmt.Print(t.String())
```

```
┌─────────────────────┬────────────────┐
│ Mode                │ URL            │
├─────────────────────┼────────────────┤
│                     │ https://www.e… │
│                     ├────────────────┤
│ end                 │ https://www.e… │
│                     ├────────────────┤
│                     │ https://api.e… │
├─────────────────────┼────────────────┤
│                     │ …om/page1.html │
│                     ├────────────────┤
│ start               │ …om/page2.html │
│                     ├────────────────┤
│                     │ …m/v1/items/42 │
├─────────────────────┼────────────────┤
│                     │ https:…e1.html │
│                     ├────────────────┤
│ middle              │ https:…e2.html │
│                     ├────────────────┤
│                     │ https:…tems/42 │
└─────────────────────┴────────────────┘
```

Equivalent via CSS: `text-overflow-position: start | middle | end`
(termtable extension — see
[../docs/wrapping.md](../docs/wrapping.md) for the full grammar).
