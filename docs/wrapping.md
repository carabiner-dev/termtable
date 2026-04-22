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
| `text-overflow-position: start\|middle\|end` | `WithTrimPosition(TrimStart\|TrimMiddle\|TrimEnd)` | `end` | Where the ellipsis lands when content is clipped horizontally. |

`-webkit-line-clamp` is accepted as an alias for `line-clamp`. The
aliases `pre-line` (for `normal`) and `pre` (for `nowrap`) are
recognized by `white-space`.

`text-overflow-position` is a termtable extension — no standard
CSS property exists for it. The alias `text-overflow-side` is
accepted too, and the value synonyms `left`/`head` for `start`,
`center` for `middle`, and `right`/`tail` for `end`.

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

### Trim position for URL-like content

When a single unbreakable token (URL, identifier, path) needs to be
truncated, controlling *which end* gets cut often matters more than
the truncation itself.

```go
t := termtable.NewTable(termtable.WithTargetWidth(30))
t.Column(0).Style("white-space: nowrap")
// three rows, each a URL that doesn't fit the column
```

With `text-overflow-position: end` (default) — keep the prefix,
useful when the suffix is the distinguishing part *only if you
can see it all*, which you can't here:

```
┌────────────────────────────┐
│ https://www.example.com/p… │
│ https://www.example.com/p… │
│ https://api.example.com/v… │
└────────────────────────────┘
```

With `text-overflow-position: start` — keep the suffix; useful
when every URL shares the same host and the path is what
distinguishes them:

```
┌────────────────────────────┐
│ …ww.example.com/page1.html │
│ …ww.example.com/page2.html │
│ …i.example.com/v1/items/42 │
└────────────────────────────┘
```

With `text-overflow-position: middle` — keep both ends so the
user can read the host AND the file/ID:

```
┌────────────────────────────┐
│ https://www.…om/page1.html │
│ https://www.…om/page2.html │
│ https://api.…m/v1/items/42 │
└────────────────────────────┘
```

The option cascades through the Style hierarchy like everything
else — set it on a column so every URL cell inherits.

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
