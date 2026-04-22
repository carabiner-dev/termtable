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

### `NoBorder()` — `border-style: none`

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

## Custom border sets

Any `BorderSet` value can be passed to `WithBorder`. The `Joins`
array is indexed by a 4-bit arm mask (N=1, E=2, S=4, W=8) covering
the 11 valid junction shapes: four corners, four T-joins, two runs
(vertical and horizontal), and the full cross. See the source of
`ASCIILine` for a full example.

Any entry of `Joins` that isn't set falls back to U+0020 (space) at
render time, so incomplete sets degrade gracefully rather than
emitting zero bytes.
