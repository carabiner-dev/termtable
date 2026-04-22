# Border styles

`termtable` ships six ready-made border glyph sets. Each is a
`BorderSet` value constructed by a no-argument function.

Set the border on a table either imperatively via `WithBorder`, or
declaratively via `WithTableStyle("border-style: <name>")` — both
paths lead to the same storage.

```go
// Imperative
t := termtable.NewTable(
    termtable.WithTargetWidth(40),
    termtable.WithBorder(termtable.RoundedLine()),
)

// CSS-driven
t := termtable.NewTable(
    termtable.WithTargetWidth(40),
    termtable.WithTableStyle("border-style: rounded"),
)
```

Unknown `border-style` keywords are silently ignored and the table
keeps its current set (defaulting to `SingleLine`).

## Catalogue

All six samples render the same three-column content so you can
compare the glyphs at a glance.

### `SingleLine()` — `border-style: single` (default)

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

### `DoubleLine()` — `border-style: double`

```
╔═══════════════╦══════════╦═══════════╗
║ Check         ║ Status   ║ Message   ║
╠═══════════════╬══════════╬═══════════╣
║ OSPS-BR-05    ║   PASS   ║ all good  ║
╠═══════════════╬══════════╬═══════════╣
║ OSPS-DO-02    ║   FAIL   ║ review    ║
║               ║          ║ deps      ║
╚═══════════════╩══════════╩═══════════╝
```

### `HeavyLine()` — `border-style: heavy`

```
┏━━━━━━━━━━━━━━━┳━━━━━━━━━━┳━━━━━━━━━━━┓
┃ Check         ┃ Status   ┃ Message   ┃
┣━━━━━━━━━━━━━━━╋━━━━━━━━━━╋━━━━━━━━━━━┫
┃ OSPS-BR-05    ┃   PASS   ┃ all good  ┃
┣━━━━━━━━━━━━━━━╋━━━━━━━━━━╋━━━━━━━━━━━┫
┃ OSPS-DO-02    ┃   FAIL   ┃ review    ┃
┃               ┃          ┃ deps      ┃
┗━━━━━━━━━━━━━━━┻━━━━━━━━━━┻━━━━━━━━━━━┛
```

### `RoundedLine()` — `border-style: rounded`

Single-line runs and joins, with rounded outer corners. Unicode has
no rounded T-joins or cross, so the interior uses the same glyphs as
`SingleLine`.

```
╭───────────────┬──────────┬───────────╮
│ Check         │ Status   │ Message   │
├───────────────┼──────────┼───────────┤
│ OSPS-BR-05    │   PASS   │ all good  │
├───────────────┼──────────┼───────────┤
│ OSPS-DO-02    │   FAIL   │ review    │
│               │          │ deps      │
╰───────────────┴──────────┴───────────╯
```

### `ASCIILine()` — `border-style: ascii`

Pure ASCII. Useful for logs, emails, legacy terminals, and any
environment that cannot display Unicode box-drawing characters. All
corners, T-joins, and crosses render as `+`.

```
+---------------+----------+-----------+
| Check         | Status   | Message   |
+---------------+----------+-----------+
| OSPS-BR-05    |   PASS   | all good  |
+---------------+----------+-----------+
| OSPS-DO-02    |   FAIL   | review    |
|               |          | deps      |
+---------------+----------+-----------+
```

### `NoBorder()` — `border-style: hidden`

Every border glyph is replaced with U+0020 (space). The grid spacing
is preserved, so columns stay aligned, but no visible dividers are
drawn. Combine with `WithTablePadding(termtable.Padding{})` to
collapse padding too.

```
                                        
  Check           Status     Message    
                                        
  OSPS-BR-05        PASS     all good   
                                        
  OSPS-DO-02        FAIL     review     
                             deps       
```

## Colouring borders

Border colour is set independently of the border style via
`border-color`. The colour applies uniformly to every glyph and fill
segment in the table.

```go
t := termtable.NewTable(
    termtable.WithTargetWidth(40),
    termtable.WithTableStyle("border-style: double; border-color: cyan"),
)
```

Per-column `border-color` declarations are accepted by the CSS
parser but ignored at render time — borders are a table-wide concern.

## Per-edge borders

The `border` shorthand and the four longhands
(`border-top`, `border-right`, `border-bottom`, `border-left`)
control whether a given edge is drawn. Each accepts one of:

| Value    | Effect                                                       |
|:---------|:-------------------------------------------------------------|
| `solid`  | Draw the edge using the table's BorderSet glyphs (default).  |
| `hidden` | Emit the line but fill with spaces — preserves grid spacing. |
| `none`   | Omit the edge entirely. If every cell at a boundary agrees, the line is dropped from the output (no blank line). |

Resolution at each boundary follows the **Solid > Hidden > None**
precedence. If *any* adjacent cell says `solid`, the line is drawn
and cells that opted for `none` render their portion as spaces. If
*every* adjacent cell says `none`, the line is skipped altogether.

Cascades table → column → row → cell, same as the visual style
properties. Rows honour `border-top` and `border-bottom`; cells
honour all four edges; `border` sets every edge at once.

The canonical "header rule only" layout:

```go
t := termtable.NewTable(termtable.WithTableStyle("border: none"))
hdr := t.AddHeader(termtable.WithRowBorderBottom(termtable.BorderEdgeSolid))
hdr.AddCell(termtable.WithContent("Name"))
hdr.AddCell(termtable.WithContent("Age"))
// …body rows…
```

yields:

```
Name           Age
────────────────────
alice          30
bob            29
```

A single cell can opt out of its row's border:

```go
hdr := t.AddHeader(termtable.WithRowBorderBottom(termtable.BorderEdgeSolid))
hdr.AddCell(termtable.WithContent("Keep"))
hdr.AddCell(termtable.WithContent("Skip"), termtable.WithCellBorderBottom(termtable.BorderEdgeNone))
hdr.AddCell(termtable.WithContent("Keep"))
```

The rule is still drawn (the row wants it) but the middle column's
portion renders as spaces.

## `border-style: none` vs `hidden`

The two are different defaults for the whole table:

- `border-style: hidden` — loads `NoBorder()` as the glyph set.
  Boundaries are still emitted; they just render as spaces. Use this
  when you want the grid spacing but no visible dividers.
- `border: none` (or `border-style: none`) — sets every default edge
  to `none`. Boundaries where nothing opts in are **dropped
  entirely** — no blank line between rows, content sits flush. Use
  this as the starting point for per-row/cell border opt-ins.

## Custom border sets

Any `BorderSet` value can be passed to `WithBorder`. The `Joins`
array is indexed by a 4-bit arm mask (N=1, E=2, S=4, W=8) covering
the 11 valid junction shapes: four corners, four T-joins, two runs
(vertical and horizontal), and the full cross. See the source of
`ASCIILine` for a full example.

Any entry of `Joins` that isn't set falls back to U+0020 (space) at
render time, so incomplete sets degrade gracefully rather than
emitting zero bytes.
