# Warnings and render errors

`termtable` never panics or aborts for content-shape issues — a cell
that overflows, a reader that fails, a rowspan that reaches past its
section. Instead, every non-fatal event is recorded as a `Warning`
that you can inspect via `Table.Warnings()`.

Fatal layout errors (the target width cannot fit content minimums)
do surface as a proper error from `Table.WriteTo`. The renderer
still produces a best-effort output at the minimum widths so a
broken table shows up visibly rather than silently.

## Two kinds of events

`Table.Warnings()` returns the concatenation of:

1. **Authoring warnings** — produced at `AddCell` / `AttachCell`
   time. They describe events that happened while you were building
   the table and persist for the table's lifetime.

2. **Render warnings** — produced at `String` / `WriteTo` time.
   They describe events that happened while laying out and
   rendering. The list is **reset on every render**, so calling
   `String` twice won't duplicate entries.

The single merged view means callers that only care "did anything
go wrong?" can use one accessor. Callers that care about the
distinction (e.g. logging authoring events exactly once at build
time) can inspect the event types themselves.

## Event types

Every event satisfies the `Warning` interface (implementing
`String() string`). Use a type switch on the concrete type when you
need the fields.

### `OverwriteEvent` — authoring

Emitted when a newly added cell's rectangle intersects an earlier
cell. The earlier cell is either **dropped** (its anchor is inside
the new rectangle) or **truncated** (its anchor sits outside but
its span reaches in). This is the default behaviour — no
`WithSpanOverwrite` call is needed.

```go
t := termtable.NewTable(termtable.WithTargetWidth(30))
r0 := t.AddRow()
r1 := t.AddRow()
r1.AddCell(termtable.WithCellID("victim"), termtable.WithContent("v"))
r0.AddCell(termtable.WithContent("over"), termtable.WithRowSpan(2))
```

```
overwrite: dropped cell id="victim"
```

Fields:

| Field                            | Populated when          |
|:---------------------------------|:------------------------|
| `DroppedID`                      | The victim was dropped. |
| `TruncatedID` / `NewColSpan` / `NewRowSpan` | The victim's span was clipped. |
| `At`                             | Anchor of the overwriting cell. |

Callers that want the old "no silent overwrites" behaviour can pass
`WithSpanOverwrite(false)` to enter **strict mode**. In strict mode
`AddCell` panics on span conflict, and the opt-in `AddCellWithError`
variant returns `ErrSpanConflict` instead.

### `DuplicateIDEvent` — authoring

Emitted when an element is attached with an ID that is already in
use. The element itself is still attached; only the duplicated ID
is dropped (the field is cleared on the losing element). The
original owner keeps the ID.

```go
t := termtable.NewTable()
r := t.AddRow()
r.AddCell(termtable.WithCellID("dup"), termtable.WithContent("a"))
r.AddCell(termtable.WithCellID("dup"), termtable.WithContent("b"))
// The second cell's ID is cleared; GetElementByID("dup") still
// resolves to the first cell.
```

```
duplicate id: "dup" (kind: cell)
```

Fields: `ID`, `Kind` — where `Kind` is `"cell"`, `"row"`,
`"header"`, `"footer"`, `"column"`, or `"table"`.

### `ContentSourceReplacedEvent` — authoring

Emitted when both `WithContent` and `WithReader` are supplied for
the same cell. The options apply in the order they appear, and the
last one wins; the prior source is cleared. The event makes the
swap visible rather than silent.

```go
r := t.AddRow()
r.AddCell(
    termtable.WithCellID("c"),
    termtable.WithContent("hi"),
    termtable.WithReader(strings.NewReader("also hi")),
) // reader wins; ContentSourceReplacedEvent recorded
```

Fields: `CellID`, `FinalSource` — where `FinalSource` is the
resulting content source (`"reader"` or `"content"`).

### `SpanOverflowEvent` — render

A multi-column cell's minimum width exceeds what its column span
can supply even after the solver borrows from outside-span slack.
Rendering continues; the cell's content wraps or overflows its
slot.

```go
t := termtable.NewTable(termtable.WithTargetWidth(14))
r := t.AddRow()
r.AddCell(
    termtable.WithCellID("banner"),
    termtable.WithContent("widebannerwideword"),
    termtable.WithColSpan(2),
)
r.AddCell(termtable.WithContent("x"))
```

```
span overflow: cell id="banner"
```

Fields: `CellID`, `Required`, `Got` — the declared minimum content
width vs. what the solver was actually able to allocate.

### `ReaderErrorEvent` — render

The cell's `WithReader` source failed while the measurement pass
was consuming it. The cell renders as empty; the error is
preserved on the event.

```go
type boomReader struct{}
func (boomReader) Read([]byte) (int, error) { return 0, errors.New("network down") }

t := termtable.NewTable(termtable.WithTargetWidth(30))
r := t.AddRow()
r.AddCell(termtable.WithCellID("broken"), termtable.WithReader(boomReader{}))
r.AddCell(termtable.WithContent("ok"))
```

```
reader error: cell id="broken": network down
```

Fields: `CellID`, `Err`. The reader is consumed once — subsequent
renders do not re-read it (or re-emit the event).

### `CrossSectionSpanEvent` — render

A cell's `rowSpan` reaches past the last row of its section
(headers, body, or footers). Rowspans cannot cross sections, so the
renderer **clamps** the effective span to the section boundary. The
authored `rowSpan` on the `Cell` is preserved — only the render-
time behaviour is clipped.

```go
t := termtable.NewTable(termtable.WithTargetWidth(40))
h := t.AddHeader()
h.AddCell(
    termtable.WithCellID("overreach"),
    termtable.WithContent("banner"),
    termtable.WithRowSpan(3), // only 1 header exists
)
h.AddCell(termtable.WithContent("col2"))
rb := t.AddRow()
rb.AddCell(termtable.WithContent("b1"))
rb.AddCell(termtable.WithContent("b2"))
```

```
rowspan crosses section boundary: cell id="overreach"
```

Fields: `CellID`, `DeclaredSpan`, `EffectiveSpan`, `Section`.

## Layout errors vs. warnings

Rendering only returns an error in one scenario: the target width
isn't large enough to give every column at least one glyph of
content space after paying for borders and padding. That's the
`ErrTargetTooNarrow` sentinel, surfaced via `Table.WriteTo`:

```go
var buf bytes.Buffer
n, err := tbl.WriteTo(&buf)
if errors.Is(err, termtable.ErrTargetTooNarrow) {
    // The terminal is genuinely too small for the number of columns
    // in this table. buf contains a best-effort render.
}
```

Narrower-than-minimum content (long words that don't fit their
column's share of the target) is **not** an error: the layout
silently shrinks below the per-column content minimum and the wrap
pass clips each cell with an ellipsis. The output stays well-formed
at the target width. Callers don't need to handle this case.

## `Table.LastRenderError` for `String` callers

`Table.String()` has no error return (Go's `Stringer` contract), so
any layout error is captured on the table:

```go
out := tbl.String()
if err := tbl.LastRenderError(); err != nil {
    log.Printf("table render warning: %v", err)
}
```

The value is **overwritten** on every `String` call. A subsequent
successful render clears any previous error. `WriteTo` callers do
not need this accessor — they receive the error directly.

## Inspecting warnings

A typical pattern:

```go
for _, w := range tbl.Warnings() {
    switch ev := w.(type) {
    case termtable.OverwriteEvent:
        if ev.DroppedID != "" {
            log.Warnf("table: dropped cell %s during overwrite", ev.DroppedID)
        }
    case termtable.DuplicateIDEvent:
        log.Warnf("table: duplicate %s id %q cleared", ev.Kind, ev.ID)
    case termtable.ContentSourceReplacedEvent:
        log.Warnf("table: cell %s content source replaced (now %s)",
            ev.CellID, ev.FinalSource)
    case termtable.SpanOverflowEvent:
        log.Warnf("table: cell %s needs %d cols but only got %d",
            ev.CellID, ev.Required, ev.Got)
    case termtable.ReaderErrorEvent:
        log.Errorf("table: reader for cell %s failed: %v", ev.CellID, ev.Err)
    case termtable.CrossSectionSpanEvent:
        log.Warnf("table: rowspan for cell %s clipped to %d (declared %d)",
            ev.CellID, ev.EffectiveSpan, ev.DeclaredSpan)
    }
}
```

If you only need a human-readable one-liner, every event implements
`String()` — just print it.
