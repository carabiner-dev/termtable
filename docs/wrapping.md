# Wrapping and overflow

Cells can render content in one of two modes:

- **Multi-line** (the default) — text wraps at whitespace, honours
  embedded `\n`, and the row grows as tall as the tallest wrapped
  cell needs.
- **Single-line** — content stays on one row and is trimmed when it
  doesn't fit the column.

The knobs are CSS-standard: `white-space`, `text-overflow`, and
`line-clamp`. All three participate in the Style cascade (table →
column → row → cell), so you can flip the mode at any level and
have it ripple down.

## Properties

| CSS                        | Options                                  | Default | Meaning                                                      |
|:---------------------------|:-----------------------------------------|:--------|:-------------------------------------------------------------|
| `white-space: normal`      | `WithWrap(true)`, `WithMultiLine()`      | ✓       | Wrap at whitespace; honour `\n`.                             |
| `white-space: nowrap`      | `WithWrap(false)`, `WithSingleLine()`    |         | Render as one line; rely on overflow handling to fit.        |
| `text-overflow: ellipsis`  | `WithTrim(true)`                         | ✓       | Append `…` when content is truncated.                        |
| `text-overflow: clip`      | `WithTrim(false)`                        |         | Hard-cut at the column edge.                                 |
| `line-clamp: N`            | `WithMaxLines(N)`                        | 0       | Cap at N lines; unbounded by default. `none` resets.         |

`-webkit-line-clamp` is accepted as an alias for `line-clamp`. The
aliases `pre-line` (for `normal`) and `pre` (for `nowrap`) are
recognized by `white-space`.

## Samples

All samples share the same content: a two-column table where the
"Description" row has text that would wrap to five lines at this
width.

### Default — multi-line wrap

Equivalent to `white-space: normal`.

```
┌────────────────┬─────────────────────┐
│ Name           │ Description         │
├────────────────┼─────────────────────┤
│ widget         │ a long description  │
│                │ that would          │
│                │ otherwise wrap      │
│                │ across multiple     │
│                │ lines               │
└────────────────┴─────────────────────┘
```

### `WithSingleLine()` — one line with ellipsis

Equivalent to CSS `white-space: nowrap` (trim remains on by default).

```go
r.AddCell(termtable.WithContent(desc), termtable.WithSingleLine())
// or
r.AddCell(termtable.WithContent(desc),
    termtable.WithCellStyle("white-space: nowrap"))
```

```
┌────────────────┬─────────────────────┐
│ Name           │ Description         │
├────────────────┼─────────────────────┤
│ widget         │ a long description… │
└────────────────┴─────────────────────┘
```

### `text-overflow: clip` — hard-cut, no ellipsis

```go
r.AddCell(termtable.WithContent(desc),
    termtable.WithCellStyle("white-space: nowrap; text-overflow: clip"))
```

```
┌────────────────┬─────────────────────┐
│ Name           │ Description         │
├────────────────┼─────────────────────┤
│ widget         │ a long description  │
└────────────────┴─────────────────────┘
```

### `line-clamp: N` — cap wrapped lines

Keeps wrapping on, but stops after N lines and appends an ellipsis
(or clips, if `text-overflow: clip` is also set).

```go
r.AddCell(termtable.WithContent(desc),
    termtable.WithCellStyle("line-clamp: 2"))
```

```
┌────────────────┬─────────────────────┐
│ Name           │ Description         │
├────────────────┼─────────────────────┤
│ widget         │ a long description  │
│                │ that would…         │
└────────────────┴─────────────────────┘
```

### Column cascade

Apply nowrap to a whole column with one call; every cell inherits.

```go
t.Column(1).Style("white-space: nowrap")
```

```
┌────────────────┬─────────────────────┐
│ Name           │ Description         │
├────────────────┼─────────────────────┤
│ widget         │ a long description… │
└────────────────┴─────────────────────┘
```

A specific cell can still opt back into wrapping with
`WithMultiLine()` or `white-space: normal`.

## When to use which

- **Status/ID columns** (`PASS`/`FAIL`, item IDs) — `white-space:
  nowrap` + `text-overflow: clip`. These columns should never
  grow their row's height.
- **Message columns** — the default (multi-line). Let them wrap to
  show full context.
- **Preview/digest columns** — `white-space: nowrap` (ellipsis
  default). Only show the first line; users can click through or
  expand elsewhere.
- **Fixed-height tables** — `line-clamp: N` on body cells with a
  target row count.

## Edge cases

- **Content with explicit `\n` in single-line mode:** the
  newlines are still honoured by `NaturalLines`, but the single-
  line branch takes only the *first* natural line and then
  trims/clips to the column width. Downstream content past the
  first line is dropped without an ellipsis marker.
- **Single-cluster-wider-than-column:** if a single grapheme
  (like a wide emoji) is wider than the column, it's rendered as
  its own line in multi-line mode regardless of the wrap policy,
  because we can't split a grapheme. In single-line mode it's
  clipped away if `trim` is on (no room for both the cluster and
  the ellipsis).
- **`line-clamp: 1` vs `WithSingleLine`:** they look similar but
  differ in where the break falls. `line-clamp: 1` wraps at
  whitespace and shows the first wrapped line (typically ends on
  a word boundary). `WithSingleLine` truncates at exactly the
  column width, wherever that falls.
