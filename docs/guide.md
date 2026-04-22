# Guide

This page explains how termtable is shaped and how to build tables
with it. It's the one to read first; the other docs under this
folder are reference material for individual subsystems.

## What termtable is

A library for emitting grid-aligned text tables to terminals.
Tables stay aligned across Unicode width quirks (CJK, emoji, ZWJ
families), honour ANSI colour inside cell content, and support the
usual spreadsheet primitives — column and row spans, cell
alignment, per-column widths, wrapping, trimming.

termtable is not a spreadsheet engine and not an interactive
widget library. It takes your data, lays it out, and emits bytes.

## The data model

```
Table
 ├── Header section   (zero or more header Rows)
 ├── Body section     (zero or more Rows)
 ├── Footer section   (zero or more footer Rows)
 └── Columns          (virtual — shared across all sections)

Each Row contains Cells (in declaration order).
Each Cell has content, a span (rows × columns), and optional Style.
Each Column has sizing (width / min / max / weight) and optional Style.
```

A **section** is just a bucket of rows. Headers render above the
body, footers below, and rowspans stay within their own section
(clamped with a warning if they'd cross a boundary).

A **column** is a virtual object that termtable creates on demand
as cells populate new grid positions. You can also grab one
explicitly via `Table.Column(i)` to configure it. A column isn't
a collection of cells; it's a descriptor that per-column settings
(width, alignment, style) hang off.

Every element — Table, Column, Row, Cell — can optionally carry
an **ID** and a **Style**. Styles cascade from the outside in:
**table → column → row → cell**, with set fields at lower levels
overriding upper-level ones.

## Building a table, step by step

Each step below adds one idea to the previous one. Code is shown
as deltas; full programs are in [`../examples/`](../examples/).

### Step 1 — a body of rows

The smallest useful table is just some rows of cells. Every row
starts empty; `AddCell` anchors cells in the next available grid
position.

```go
t := termtable.NewTable(termtable.WithTargetWidth(40))

r1 := t.AddRow()
r1.AddCell(termtable.WithContent("OSPS-BR-05"))
r1.AddCell(termtable.WithContent("PASS"))

r2 := t.AddRow()
r2.AddCell(termtable.WithContent("OSPS-DO-02"))
r2.AddCell(termtable.WithContent("FAIL"))

fmt.Print(t.String())
```

```
┌──────────────────────┬───────────────┐
│ OSPS-BR-05           │ PASS          │
├──────────────────────┼───────────────┤
│ OSPS-DO-02           │ FAIL          │
└──────────────────────┴───────────────┘
```

### Step 2 — add a header

Headers share the API with rows. Add one or more before the body
for labelled columns.

```go
h := t.AddHeader()
h.AddCell(termtable.WithContent("Check"))
h.AddCell(termtable.WithContent("Status"))
```

```
┌─────────────────────┬────────────────┐
│ Check               │ Status         │
├─────────────────────┼────────────────┤
│ OSPS-BR-05          │ PASS           │
├─────────────────────┼────────────────┤
│ OSPS-DO-02          │ FAIL           │
└─────────────────────┴────────────────┘
```

### Step 3 — a third column, column-level alignment

Column 1 gets centre alignment via `SetAlign`; every cell in it
inherits. The message column wraps naturally when content is
longer than the column's share of the budget.

```go
t.Column(1).SetAlign(termtable.AlignCenter)

// ...existing header plus a third cell:
h.AddCell(termtable.WithContent("Message"))

// ...existing body rows plus third cells:
r1.AddCell(termtable.WithContent("all good"))
r2.AddCell(termtable.WithContent("review deps"))
```

```
┌───────────────┬──────────┬───────────┐
│ Check         │  Status  │ Message   │
├───────────────┼──────────┼───────────┤
│ OSPS-BR-05    │   PASS   │ all good  │
├───────────────┼──────────┼───────────┤
│ OSPS-DO-02    │   FAIL   │ review    │
│               │          │ deps      │
└───────────────┴──────────┴───────────┘
```

### Step 4 — banner header, footer, colspan

A colspan cell in a header creates a banner. Another section
below the body is added with `AddFooter`.

```go
banner := t.AddHeader()          // inserted BEFORE the column header
banner.AddCell(
    termtable.WithContent("Evaluation Results"),
    termtable.WithColSpan(3),
    termtable.WithAlign(termtable.AlignCenter),
)

f := t.AddFooter()
f.AddCell(
    termtable.WithContent("1 passed, 1 failed"),
    termtable.WithColSpan(3),
    termtable.WithAlign(termtable.AlignCenter),
)
```

```
┌──────────────────────────────────────┐
│          Evaluation Results          │
├───────────────┬──────────┬───────────┤
│ Check         │  Status  │ Message   │
├───────────────┼──────────┼───────────┤
│ OSPS-BR-05    │   PASS   │ all good  │
├───────────────┼──────────┼───────────┤
│ OSPS-DO-02    │   FAIL   │ review    │
│               │          │ deps      │
├───────────────┴──────────┴───────────┤
│          1 passed, 1 failed          │
└──────────────────────────────────────┘
```

Notice how the border joins resolve automatically: the banner
suppresses the `┬` under it, and the footer row seamlessly
closes the sub-columns.

## Options

Every constructor takes a variadic list of functional options
named `With*`. The naming convention tells you where they attach:

- `NewTable(WithTargetWidth(80) | WithTargetWidthPercent(75), WithTableStyle("…"), WithTablePadding(…), WithBorder(…), WithSpanOverwrite(false) /* strict mode */, WithEmojiWidth(…), WithTableID("…"))`
- `AddHeader` / `AddRow` / `AddFooter` accept `WithRowID`, `WithRowStyle`, `WithCell(*Cell)`
- `AddCell` / `AttachCell` / `NewCell` accept `WithCellID`, `WithContent`, `WithReader`, `WithColSpan`, `WithRowSpan`, `WithAlign`, `WithVAlign`, `WithWrap` / `WithSingleLine` / `WithMultiLine`, `WithTrim`, `WithMaxLines`, `WithTrimPosition`, `WithPadding` *(table-level only — see below)*, `WithCellStyle`, `WithTextColor`, `WithBackgroundColor`, `WithBold`, `WithItalic`, `WithUnderline`, `WithStrikethrough`

Options are composable and order-independent. If two options touch
the same field, the later one wins. A few options are deliberately
only valid on one element type — padding is table-wide (so columns
align), border glyphs are table-wide (a whole-grid concern).

Columns use **imperative setters** rather than options:
`t.Column(1).SetWidth(8).SetAlign(AlignRight)`. They don't own
content themselves, so a mutation-style API reads better than
threading options through a constructor.

All of these also work through **CSS-style declarations**:

```go
t.Column(1).Style("width: 8; text-align: right")
t.AddRow(termtable.WithRowStyle("color: red; font-weight: bold"))
cell.Style("white-space: nowrap; text-overflow-position: middle")
```

See the individual docs for each property grammar.

## Styling, layout, and borders

Three kinds of attributes cascade through the same
**table → column → row → cell** chain:

1. **Visual** — colour, bold/italic/underline, background. See
   [styling.md](styling.md).
2. **Alignment** — horizontal (`text-align`) and vertical
   (`vertical-align`). Also in [styling.md](styling.md).
3. **Layout** — `white-space`, `text-overflow`, `line-clamp`,
   `text-overflow-position`. See [wrapping.md](wrapping.md).

Separate knobs, because they apply at different layers:

- **Column sizing** (`width`, `min-width`, `max-width`, `flex`) —
  see [columns.md](columns.md).
- **Border glyphs** (`border-style`) — see
  [borders.md](borders.md). Border colour (`border-color`) lives
  in the style cascade and is table-level.
- **Emoji width** — see [emoji.md](emoji.md).

## Authoring errors

The authoring API (`AddHeader`, `AddRow`, `AddFooter`, `AddCell`,
`AttachCell`) is **panic-free by default** and returns plain values
— no `(T, error)` pair. Edge cases that were previously errors are
now handled like so:

- **Duplicate IDs**: the second element is attached with an empty
  ID, and a `DuplicateIDEvent` lands on `tbl.Warnings()`.
- **Content and reader together**: last-writer-wins; the prior
  source is cleared and a `ContentSourceReplacedEvent` is recorded.
- **Cell attached to two rows**: the cell migrates — it's detached
  from its old row and reattached to the new one.
- **Span clamping**: `WithColSpan(n)` / `WithRowSpan(n)` silently
  clamp to 1 when `n < 1`.
- **Span conflicts**: by default the new cell wins; the earlier
  cell is dropped or truncated and an `OverwriteEvent` is recorded.

If you want old-school "refuse the operation" behaviour, two knobs
opt you in:

- `WithSpanOverwrite(false)` on the table enables **strict mode**:
  `AddCell` panics on span conflict instead of overwriting. Pair
  with `AddCellWithError` / `AttachCellWithError` (on `Row`,
  `Header`, `Footer`) to receive `ErrSpanConflict` as a value.
- The `*WithError` variants also expose any non-span error surface
  the defaults chose to swallow, for callers that want to inspect
  rather than trust.

## Rendering

Two entry points:

- `tbl.String() string` — returns the rendered output. Layout
  errors are captured on the Table; call `tbl.LastRenderError()`
  to inspect.
- `tbl.WriteTo(w io.Writer) (int64, error)` — writes directly to
  any `io.Writer` and returns the layout error synchronously.
  Prefer this when you're integrating with a logger or pipe.

### Target width resolution

Width resolution follows CSS table semantics: the table has an
**intrinsic** width from its content, and three optional bounds —
`width`, `min-width`, `max-width` — that constrain it.

**Defaults** (applied only when the corresponding option is unset):

- `min-width: 80` — the table never renders narrower than 80 columns
  unless the screen itself can't fit 80.
- `max-width: 90%` — the table never exceeds 90 % of the attached
  screen; content beyond that wraps or clips inside cells.

**Pinning an exact size** — `WithTargetWidth(n)` /
`WithTargetWidthPercent(p)` / `width: N | N%` — pins the layout to
that target; the content-intrinsic width is ignored. Default
bounds do **not** clamp an explicit target. Explicit bounds
(`WithMinWidth`, `WithMaxWidth`, `min-width`, `max-width` in CSS)
still apply on top.

**No pin** — the layout picks the content-intrinsic width and
clamps it into `[min-width, max-width]`.

**Screen cap** — whatever value the resolver produces is clamped
to the attached terminal's width (stdout or stderr, via
`golang.org/x/term`). Pipes and other non-interactive sinks leave
the value uncapped.

| Setup                                             | Result        |
|:--------------------------------------------------|:--------------|
| No options, short content, 200-col terminal       | `80` (min)    |
| No options, medium content (~70 wide), 200 TTY    | natural (≤ max) |
| No options, long content, 200-col terminal        | `180` (max)   |
| No options, 85-col terminal                       | `80` (floor)  |
| No options, 40-col terminal                       | `40` (capped) |
| `WithTargetWidth(200)`, 80-col terminal           | `80` (TTY)    |
| `WithTargetWidth(40)`, 120-col terminal           | `40`          |
| `WithTargetWidth(50)`, 200-col terminal           | `50` (explicit ignores default min) |
| `WithTargetWidth(120) + WithMaxWidth(90)`         | `90`          |
| `WithMinWidth(100)`, short content                | `100`         |
| `WithMaxWidth(50)`, long content                  | `50`          |
| `WithTableStyle("width: 80%")`, 100 TTY           | `80`          |
| `WithTableStyle("min-width: 60; max-width: 50%")` | clamped into `[60, 0.5 × screen]` |

**CSS grammar**. All three bounds accept absolute or percent
values, and the last declaration on the table wins:

```go
termtable.WithTableStyle("width: 120")
termtable.WithTableStyle("min-width: 40; max-width: 75%")
termtable.WithTableStyle("width: 80%")   // last; overrides prior width
```

`WithTargetWidth` and `WithTargetWidthPercent` (and the `width:`
declaration) are mutually exclusive — whichever is set last wins.
The same holds for the min/max pairs.

Both call the same three-pass pipeline:

1. **Measure** — walks every cell, consumes any `WithReader`
   sources, computes per-column minimum and desired widths.
2. **Layout** — distributes the target width across columns using
   the equal-split-with-weights solver (see
   [columns.md](columns.md) for the exact algorithm), then wraps
   each cell's content to its allotted width and sums row
   heights.
3. **Paint** — emits border glyphs, content lines, and ANSI
   styling, with span-aware border join resolution.

### Best-effort output

Rendering never panics. When the target width is too narrow to
fit even the content minimums, termtable falls back to each
column's minimum width, produces the best output it can, and
surfaces the error:

```go
t := termtable.NewTable(termtable.WithTargetWidth(10))
r := t.AddRow()
r.AddCell(termtable.WithContent("longword"))
r.AddCell(termtable.WithContent("another"))

var buf bytes.Buffer
_, err := t.WriteTo(&buf)
fmt.Print(buf.String())
if err != nil {
    fmt.Println("err:", err)
}
```

```
┌──────────┬─────────┐
│ longword │ another │
└──────────┴─────────┘
err: target width 10 leaves 3 for content but content minimum sums to 15: target width too narrow for content minimums
```

You get a wider-than-requested table plus a meaningful error. The
table caller decides whether to accept, retry with a wider
target, or propagate.

### Warnings

Non-fatal events — span overwrites, reader failures, multi-span
overflows, cross-section rowspans — accumulate on
`tbl.Warnings()`. Authoring events persist for the table's life;
render events refresh on every call. Full event catalogue in
[warnings.md](warnings.md).

## Inspection and IDs

Any element accepts an ID; `tbl.GetElementByID` looks them up via
a type switch:

```go
r := t.AddRow()
r.AddCell(termtable.WithCellID("status"), termtable.WithContent("PASS"))
r.AddCell(termtable.WithContent("done"))

if c, ok := t.GetElementByID("status").(*termtable.Cell); ok {
    fmt.Printf("content=%q grid=(%d,%d)\n", c.Content(), c.GridRow(), c.GridCol())
}
fmt.Print(t.String())
```

```
content="PASS" grid=(0,0)
┌──────────────┬─────────────┐
│ PASS         │ done        │
└──────────────┴─────────────┘
```

IDs are unique across the whole table — when a second element
tries to register an in-use ID, its own ID is cleared and a
`DuplicateIDEvent` lands on `tbl.Warnings()`. The element itself
is still attached. Empty IDs aren't registered, so you only pay
for ones you actually reference.

## Where to go next

- **Runnable snippets** for every major feature live in
  [../examples/](../examples/).
- **Per-subsystem reference** is the rest of this `docs/`
  folder: [borders.md](borders.md), [columns.md](columns.md),
  [styling.md](styling.md), [wrapping.md](wrapping.md),
  [emoji.md](emoji.md), [warnings.md](warnings.md).
- **API surface**: [pkg.go.dev](https://pkg.go.dev/github.com/carabiner-dev/termtable)
  has every exported identifier with its godoc comment plus the
  `Example*` functions that double as integration tests.
