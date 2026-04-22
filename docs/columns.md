# Column configuration

Columns are virtual objects. `termtable` creates them on demand as
cells land in new grid positions, but you can also retrieve a
column directly via `Table.Column(i)` — that will extend the column
list if needed so it's safe to call before any rows are added.

Configuration is imperative and chainable:

```go
t.Column(1).SetWidth(8).SetAlign(termtable.AlignCenter)
```

or declarative via CSS-style declarations:

```go
t.Column(1).Style("width: 8; text-align: center")
```

Both paths share the same storage — mix and match freely.

## The layout solver in one paragraph

The table's target width is allocated to columns in four passes.
First, each column's **effective minimum** is set to
`max(contentMinimum, userMin)`. Second, columns with an explicit
`SetWidth` are **pinned** (min and max both equal the requested
width). Third, the remaining budget is **water-filled** by
`weight` into the flex columns, capped at each column's `SetMax`.
Fourth, any multi-span cell's minimum-width constraint is satisfied
by borrowing from outside-span slack columns.

The upshot:

- `SetWidth(n)` **pins** — column is always exactly `n` wide.
- `SetMin(n)` **floors** — column is never narrower than `n`.
- `SetMax(n)` **caps** — column is never wider than `n`.
- `SetWeight(w)` **distributes** — larger weights claim more of the
  leftover budget. Default is `1.0`; `0` opts the column out of the
  leftover entirely.

## Methods and CSS equivalents

| Method                | CSS equivalent          | Effect                                               |
|:----------------------|:------------------------|:-----------------------------------------------------|
| `SetWidth(n)`         | `width: n`              | Pin content width to exactly `n`.                    |
| `SetMin(n)`           | `min-width: n`          | Floor — combined with content minimum via `max`.     |
| `SetMax(n)`           | `max-width: n`          | Cap — content wraps when it would exceed.            |
| `SetWeight(w)`        | `flex: w`               | Share of leftover budget.                            |
| `SetAlign(a)`         | `text-align: left\|center\|right` | Default horizontal alignment for cells in the column. |
| `SetVAlign(v)`        | `vertical-align: top\|middle\|bottom` | Default vertical alignment for cells in the column.   |
| `SetID(id)`           | —                       | Register the column with `Table.GetElementByID`.     |

CSS also forwards the style-only properties (`color`, `background`,
`font-weight`, `font-style`, `text-decoration`) into a column-level
`Style` that cascades to every cell in the column. `border-color`
at column level is accepted by the parser but ignored at render
time — border glyphs are a table-wide concern.

Passing `n <= 0` to any of the numeric setters **clears** the
override, restoring the solver's defaults.

## Gallery

Every sample uses `WithTargetWidth(60)` and the same 3-column body,
so the effects are directly comparable. The configuration shown
above each sample is the **only** difference between runs.

### Default — equal split

All three columns have `weight=1` and no bounds, so the available
budget is divided equally.

```go
t := termtable.NewTable(termtable.WithTargetWidth(60))
```

```
┌─────────────────────┬────────────────┬───────────────────┐
│ Check               │ Status         │ Message           │
├─────────────────────┼────────────────┼───────────────────┤
│ OSPS-BR-05          │ PASS           │ all criteria      │
│                     │                │ satisfied         │
├─────────────────────┼────────────────┼───────────────────┤
│ OSPS-DO-02          │ FAIL           │ needs attention   │
│                     │                │ soon              │
└─────────────────────┴────────────────┴───────────────────┘
```

### `SetWidth(8)` on column 1 — pinned

Column 1 is locked at 8 content columns regardless of the target
width; columns 0 and 2 absorb the remainder equally.

```go
t.Column(1).SetWidth(8)
```

```
┌────────────────────────┬──────────┬──────────────────────┐
│ Check                  │ Status   │ Message              │
├────────────────────────┼──────────┼──────────────────────┤
│ OSPS-BR-05             │ PASS     │ all criteria         │
│                        │          │ satisfied            │
├────────────────────────┼──────────┼──────────────────────┤
│ OSPS-DO-02             │ FAIL     │ needs attention soon │
└────────────────────────┴──────────┴──────────────────────┘
```

### `SetMin(14)` on column 0 — floor

Column 0's content ("OSPS-BR-05" = 10 columns) is not enough to
exercise the floor naturally; the user-min pushes it to 14 and the
solver pays the debt from the widest slack column.

```go
t.Column(0).SetMin(14)
```

```
┌───────────────────────┬───────────────┬──────────────────┐
│ Check                 │ Status        │ Message          │
├───────────────────────┼───────────────┼──────────────────┤
│ OSPS-BR-05            │ PASS          │ all criteria     │
│                       │               │ satisfied        │
├───────────────────────┼───────────────┼──────────────────┤
│ OSPS-DO-02            │ FAIL          │ needs attention  │
│                       │               │ soon             │
└───────────────────────┴───────────────┴──────────────────┘
```

### `SetMax(6)` on column 1 — cap

Column 1's natural content would easily fit more space, but the cap
holds it narrow. The leftover budget flows to the remaining flex
columns.

```go
t.Column(1).SetMax(6)
```

```
┌─────────────────────────┬────────┬───────────────────────┐
│ Check                   │ Status │ Message               │
├─────────────────────────┼────────┼───────────────────────┤
│ OSPS-BR-05              │ PASS   │ all criteria          │
│                         │        │ satisfied             │
├─────────────────────────┼────────┼───────────────────────┤
│ OSPS-DO-02              │ FAIL   │ needs attention soon  │
└─────────────────────────┴────────┴───────────────────────┘
```

### `SetWeight(3)` on column 2 — flex

Columns 0 and 1 keep `weight=1`. Column 2 claims three times as
much of the leftover budget, producing a wide message area with
shorter sibling columns.

```go
t.Column(2).SetWeight(3)
```

```
┌─────────────────┬─────────────┬──────────────────────────┐
│ Check           │ Status      │ Message                  │
├─────────────────┼─────────────┼──────────────────────────┤
│ OSPS-BR-05      │ PASS        │ all criteria satisfied   │
├─────────────────┼─────────────┼──────────────────────────┤
│ OSPS-DO-02      │ FAIL        │ needs attention soon     │
└─────────────────┴─────────────┴──────────────────────────┘
```

### Column alignment cascade

Cells inherit their column's alignment unless they set their own
via `WithAlign`. Here column 1 pins its width **and** cascades a
center alignment to its body cells (the header "Status" is also
centered because it too inherits).

```go
t.Column(1).SetWidth(8).SetAlign(termtable.AlignCenter)
```

```
┌────────────────────────┬──────────┬──────────────────────┐
│ Check                  │  Status  │ Message              │
├────────────────────────┼──────────┼──────────────────────┤
│ OSPS-BR-05             │   PASS   │ all criteria         │
│                        │          │ satisfied            │
├────────────────────────┼──────────┼──────────────────────┤
│ OSPS-DO-02             │   FAIL   │ needs attention soon │
└────────────────────────┴──────────┴──────────────────────┘
```

### Vertical alignment — `SetVAlign` / `vertical-align`

When a row is taller than a cell's own wrapped content (commonly
because a neighbour wrapped to more lines), the cell's content sits
at the top by default. `SetVAlign` places it middle or bottom;
cascade works the same way as horizontal alignment.

```go
t.Column(0).SetVAlign(termtable.VAlignMiddle) // or VAlignBottom / VAlignTop
```

Top (default):

```
┌──────────────────┬───────────────────┐
│ short            │ this is a much    │
│                  │ longer message    │
│                  │ that must wrap    │
└──────────────────┴───────────────────┘
```

Middle:

```
┌──────────────────┬───────────────────┐
│                  │ this is a much    │
│ short            │ longer message    │
│                  │ that must wrap    │
└──────────────────┴───────────────────┘
```

Bottom:

```
┌──────────────────┬───────────────────┐
│                  │ this is a much    │
│                  │ longer message    │
│ short            │ that must wrap    │
└──────────────────┴───────────────────┘
```

### CSS combination

The imperative and CSS paths produce identical layouts. This is the
same table as the cascade example above plus a wide message column.

```go
t.Column(0).Style("min-width: 12")
t.Column(1).Style("width: 8; text-align: center")
t.Column(2).Style("flex: 3")
```

```
┌────────────────────┬──────────┬──────────────────────────┐
│ Check              │  Status  │ Message                  │
├────────────────────┼──────────┼──────────────────────────┤
│ OSPS-BR-05         │   PASS   │ all criteria satisfied   │
├────────────────────┼──────────┼──────────────────────────┤
│ OSPS-DO-02         │   FAIL   │ needs attention soon     │
└────────────────────┴──────────┴──────────────────────────┘
```

## Interactions worth knowing

**Content minimum wins over `SetWidth`.** If a column has content
whose unbreakable minimum exceeds the requested pin, the column
overflows rather than truncating content. Use `SetMax` if you
actively want content clipped to a cap; the wrapper will trim with
an ellipsis.

**`SetWeight(0)` anchors the column.** A zero weight opts the
column out of the leftover-distribution pass. Combined with the
default content minimum, that effectively pins the column at the
width its content demands — useful for ID columns that you want as
narrow as possible while letting the message column absorb slack.

**Caps can shrink the whole table.** If every column's `SetMax`
leaves the solver unable to consume the full target width, the
rendered output is narrower than the terminal. That's by design —
the alternative would be to silently ignore the user's caps.

**Alignment override precedence.** Cell `WithAlign` > row
`text-align` > column `SetAlign`/`text-align` > table
`text-align`. In a header row with `WithRowStyle("text-align:
center")`, every header cell centers regardless of column
alignment. Body cells in the same column keep inheriting from the
column.

## Registering columns by ID

```go
t.Column(1).SetID("status")
// ...elsewhere:
switch e := t.GetElementByID("status").(type) {
case *termtable.Column:
    e.SetMax(8)
}
```

IDs are unique across the whole table — a collision records a
`DuplicateIDEvent` on `tbl.Warnings()` and the column's ID is left
empty. `SetID` returns the column so calls can chain. Clear a
column's ID by passing the empty string; reassigning a new ID
unregisters the old one first.
