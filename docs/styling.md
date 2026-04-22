# Styling

Styling in `termtable` is CSS-like. Every element type (table, row,
column, cell) carries an optional `Style` that contributes to the
**effective style** computed per cell at render time.

## The cascade

```
┌─────────┐       lower-level fields win over upper-level ones;
│ Table   │       fields never set at any level fall back to the
├─────────┤       zero value (AlignLeft, no color, no bold, etc.)
│ Column  │
├─────────┤
│ Row     │
├─────────┤
│ Cell    │
└─────────┘
```

The cascade order is **table → column → row → cell**. A bold
header row overrides column-level coloring for those cells; a cell
with its own color overrides the row. Row wins over column because
headers commonly want uniform styling that dominates type-based
column styling.

Only fields that are actually **set** at a level participate in the
cascade. Unset fields are transparent — they inherit whatever the
parent levels produced.

## Setting styles

Three entry points, one for each element type:

```go
termtable.NewTable(
    termtable.WithTableStyle("color: white; border-color: cyan"),
)
tbl.AddHeader(
    termtable.WithRowStyle("font-weight: bold; background: blue"),
)
hdr.AddCell(
    termtable.WithContent("PASS"),
    termtable.WithCellStyle("color: green; font-weight: bold"),
)
```

Cell styling has a set of convenience options for the common single
attributes:

```go
row.AddCell(termtable.WithContent("FAIL"),
    termtable.WithTextColor("red"),
    termtable.WithBackgroundColor("black"),
    termtable.WithBold(),
    termtable.WithItalic(),
    termtable.WithUnderline(),
    termtable.WithStrikethrough(),
)
```

These are additive with `WithCellStyle` — use whichever is clearer
at the call site.

Columns use imperative setters, mirroring their size/alignment
config:

```go
t.Column(1).Style("color: bright-cyan; font-weight: bold")
// or piecewise via the existing helpers plus raw CSS
t.Column(2).SetAlign(termtable.AlignRight).Style("color: yellow")
```

## Property reference

| CSS property        | Accepted values                               | Notes                                                       |
|:--------------------|:----------------------------------------------|:------------------------------------------------------------|
| `color`             | name / `#rrggbb` / `rgb(r,g,b)`               | Foreground text colour.                                     |
| `background`        | name / `#rrggbb` / `rgb(r,g,b)`               | Alias: `background-color`.                                  |
| `border-color`      | name / `#rrggbb` / `rgb(r,g,b)`               | Table-level only — ignored on rows, columns, and cells.     |
| `border-style`      | `single` \| `double` \| `heavy` \| `rounded` \| `ascii` \| `none` | Table-level only. See [borders.md](borders.md).              |
| `font-weight`       | `bold` \| `normal`                            | Bold is the SGR bold attribute (code 1).                    |
| `font-style`        | `italic` \| `normal`                          | Some terminals don't render italics.                        |
| `text-decoration`   | `underline` \| `line-through` \| `none`       | Multiple values combinable: `underline line-through`.       |
| `text-align`        | `left` \| `center` \| `right`                 | Works at any level; defaults to `left`.                     |
| `vertical-align`    | `top` \| `middle` \| `bottom`                 | Where content sits when its row is taller than its height.  |
| `white-space`       | `normal` \| `nowrap` (also `pre` / `pre-line`) | Multi-line wrap vs single-line (see [wrapping.md](wrapping.md)). |
| `text-overflow`     | `ellipsis` \| `clip`                          | Truncation marker; default is `ellipsis`.                   |
| `line-clamp`        | non-negative integer, `none`, `auto`          | Cap wrapped height at N lines; `none`/`0` means unbounded. `-webkit-line-clamp` accepted. |

Column CSS additionally accepts the sizing properties documented in
[columns.md](columns.md). Everything else the parser doesn't
recognize is silently ignored — the spec is permissive so that
existing CSS snippets can pass unchanged.

## Colour grammar

The same grammar applies to every colour-valued property.

**Named colours** — the eight standard ANSI colours and their
`bright-` variants:

```
black      red      green      yellow      blue      magenta      cyan      white
bright-black  bright-red  bright-green  bright-yellow
bright-blue   bright-magenta  bright-cyan  bright-white
```

**Hex** — six digits after a `#`:

```css
color: #ff0088
background: #1a1a1a
```

**RGB functional** — three decimal channels in `0..255`:

```css
color: rgb(10, 40, 220)
```

Values outside `0..255` or malformed expressions are rejected
silently and leave the property unset. Hex shortcuts (`#f80`) and
percentage channels are not currently supported.

## NoColor handling

`termtable` layers on top of `github.com/fatih/color`, which honours
the global `color.NoColor` flag. When stdout isn't a TTY, or
`NO_COLOR` is set in the environment, ANSI sequences are stripped
automatically — the table still composes correctly, it just emits
plain text.

This means tests rendering to `bytes.Buffer` typically see plain
output. Inside a test that specifically wants to assert ANSI
escapes, flip the flag:

```go
saved := color.NoColor
color.NoColor = false
defer func() { color.NoColor = saved }()
```

## Worked example

Here is a table with three-level cascading styles. Run it in a
terminal that supports colour to see the result; the plain rendering
below shows the structure (colour information omitted).

```go
t := termtable.NewTable(
    termtable.WithTargetWidth(50),
    termtable.WithTableStyle("border-color: cyan"),
)

hdr, _ := t.AddHeader(
    termtable.WithRowStyle("color: white; background: blue; font-weight: bold"),
)
hdr.AddCell(termtable.WithContent("Check"))
hdr.AddCell(termtable.WithContent("Status"))
hdr.AddCell(termtable.WithContent("Message"))

r1, _ := t.AddRow()
r1.AddCell(termtable.WithContent("OSPS-BR-05"))
r1.AddCell(termtable.WithContent("PASS"),
    termtable.WithAlign(termtable.AlignCenter),
    termtable.WithTextColor("green"),
    termtable.WithBold(),
)
r1.AddCell(termtable.WithContent("all good"))

r2, _ := t.AddRow()
r2.AddCell(termtable.WithContent("OSPS-DO-02"))
r2.AddCell(termtable.WithContent("FAIL"),
    termtable.WithAlign(termtable.AlignCenter),
    termtable.WithCellStyle("color: red; font-weight: bold"),
)
r2.AddCell(termtable.WithContent("needs action soon"))
```

```
┌──────────────────┬──────────────┬──────────────┐
│ Check            │ Status       │ Message      │
├──────────────────┼──────────────┼──────────────┤
│ OSPS-BR-05       │     PASS     │ all good     │
├──────────────────┼──────────────┼──────────────┤
│ OSPS-DO-02       │     FAIL     │ needs        │
│                  │              │ action soon  │
└──────────────────┴──────────────┴──────────────┘
```

With colour enabled:

- Borders (`─│┌┐└┘├┤┬┴┼`) emit in cyan.
- The header row has white text on a blue background, bold.
- `PASS` is green bold.
- `FAIL` is red bold.
- Other cells inherit nothing from the row (they're body rows), so
  they render in the terminal's default colours.

## Caveat — user-supplied ANSI plus a background colour

The renderer wraps each cell's padded slot with `color.New(...).Sprint(...)`.
That produces one opening SGR sequence and one `\x1b[0m` reset.

If the cell's **content** already contains ANSI sequences — for
example the user passed `WithContent("\x1b[31mcolored\x1b[0m")`
and also set `WithBackgroundColor("blue")` — the content's internal
`\x1b[0m` reset will clear the outer background at that point. You
get the background on the padding and up to the user's reset, then
plain text for the remainder.

Two ways to avoid it:

1. Style through `termtable` (`WithTextColor`, `WithBold`, …) and
   pass plain text as content. Let the renderer compose the SGR.
2. Pre-wrap content in ANSI yourself and don't set a cell
   background; rely on the terminal's default.

Phase 4 fidelity is "reduced": colour survives line breaks via a
state-re-emit + reset strategy, and outer-plus-inner composition is
not perfect. A full SGR state machine is deferred.

## See also

Composite emoji (ZWJ families, flags, skin-tone modifiers) have
their own documentation page — see [emoji.md](emoji.md) for the
width modes and terminal-detection rules.

## Where styling doesn't apply

- **Border glyphs** are coloured via table-level `border-color` only.
  Row/column/cell styles never recolour borders, even at a
  boundary they share with the styled element.
- **Padding** belongs to the styled slot — if a cell has a background
  colour, its left/right padding shows the colour too. This is how
  table-wide padding lets background stripes look right.
- **Content inside a suppressed rowspan border** uses the cell's own
  style, not the table border colour. This is deliberate — the
  content is semantically still the cell's.
