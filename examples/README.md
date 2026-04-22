# Examples

One minimal, focused sample per major feature. Each file has the
smallest runnable Go snippet that exercises the feature plus the
rendered output, so you can read or copy.

If you want a narrative walkthrough first, read
[../docs/guide.md](../docs/guide.md). For full reference
documentation see [../docs/](../docs/).

| Example | Feature |
|:---|:---|
| [basic.md](basic.md) | Minimal table — header + body |
| [sections.md](sections.md) | Multi-header + body + footer with a colspan banner |
| [spans.md](spans.md) | Column- and row-spanning cells |
| [alignment.md](alignment.md) | Horizontal and vertical alignment, column cascade |
| [columns.md](columns.md) | Column widths: pinned, min, weighted flex |
| [borders.md](borders.md) | Alternate border glyph sets |
| [styling.md](styling.md) | Colour, bold, header row styling |
| [wrapping.md](wrapping.md) | Multi-line, single-line, and `line-clamp` |
| [trim-position.md](trim-position.md) | Left / middle / right ellipsis placement |
| [emoji.md](emoji.md) | Conservative width mode for ZWJ emoji |
| [css.md](css.md) | CSS-driven configuration end-to-end |
| [ids.md](ids.md) | Element IDs and `GetElementByID` |
| [readers.md](readers.md) | Cells backed by `io.Reader` |
| [warnings.md](warnings.md) | Inspecting non-fatal render events |

All snippets assume `import "github.com/carabiner-dev/termtable"`.
Rendered output is shown with colours stripped (as if
`color.NoColor` were true) so the plain glyphs are visible in the
page. On a real terminal the styled examples also emit ANSI colour
escapes described in [../docs/styling.md](../docs/styling.md).
