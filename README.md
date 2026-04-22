# termtable

A library for compsing and rendering text tables on a terminal featuring
a DOM-like model. termtable handles Unicode width (emoji, CJK, combining marks),
ANSI escape sequences, word-wrapping, trimming, and column/row
spans with configurable Unicode box borders. All controllable 
programatically and through CSS declarations.

## Install

```sh
go get github.com/carabiner-dev/termtable
```

## Quick start

```go
package main

import (
	"fmt"

	"github.com/carabiner-dev/termtable"
)

func main() {
	t := termtable.NewTable(termtable.WithTargetWidth(50))

	banner := t.AddHeader()
	banner.AddCell(
		termtable.WithContent("Evaluation Results"),
		termtable.WithColSpan(3),
		termtable.WithAlign(termtable.AlignCenter),
	)

	head := t.AddHeader()
	head.AddCell(termtable.WithContent("Check"))
	head.AddCell(termtable.WithContent("Status"))
	head.AddCell(termtable.WithContent("Message"))

	r1 := t.AddRow()
	r1.AddCell(termtable.WithContent("OSPS-BR-05"))
	r1.AddCell(termtable.WithContent("PASS"),
		termtable.WithAlign(termtable.AlignCenter))
	r1.AddCell(termtable.WithContent("all criteria met"))

	r2 := t.AddRow()
	r2.AddCell(termtable.WithContent("OSPS-DO-02"))
	r2.AddCell(termtable.WithContent("FAIL"),
		termtable.WithAlign(termtable.AlignCenter))
	r2.AddCell(termtable.WithContent("review dependencies"))

	fmt.Print(t.String())
}
```

Output:

```
┌────────────────────────────────────────────────┐
│               Evaluation Results               │
├────────────────┬────────────┬──────────────────┤
│ Check          │ Status     │ Message          │
├────────────────┼────────────┼──────────────────┤
│ OSPS-BR-05     │    PASS    │ all criteria met │
├────────────────┼────────────┼──────────────────┤
│ OSPS-DO-02     │    FAIL    │ review           │
│                │            │ dependencies     │
└────────────────┴────────────┴──────────────────┘
```

## Examples

Feature-by-feature runnable snippets live in
[examples/](examples/). Start with
[examples/basic.md](examples/basic.md) and browse from there.

## Docs

Start with the [guide](docs/guide.md) for a walkthrough of the
mental model, a step-by-step tutorial, and pointers to the rest.
Per-subsystem reference:

- [Border styles](docs/borders.md) — the six built-in glyph sets
  and how to select them imperatively or via CSS.
- [Column configuration](docs/columns.md) — sizing (width / min /
  max / weight), alignment cascade, `Column.Style` CSS.
- [Styling](docs/styling.md) — table → column → row → cell
  cascade, CSS property reference, colour grammar, `NoColor`
  handling.
- [Wrapping and overflow](docs/wrapping.md) — single-line vs
  multi-line modes, `white-space` / `text-overflow` /
  `line-clamp`, column and table cascade for line-mode control.
- [Emoji width](docs/emoji.md) — why tables can misalign on some
  terminals, the conservative vs grapheme modes, auto-detection
  whitelist, and the `TERMTABLE_EMOJI_WIDTH` env var.
- [Warnings](docs/warnings.md) — authoring vs render events,
  `Table.Warnings`, `Table.LastRenderError`.

## Contributing

PRs welcome! If you're adding a feature, pair it with:

- A focused test next to the existing ones for the subsystem you're
  touching.
- A short entry under [`examples/`](examples/) if the feature is
  user-visible.
- A note in the matching page under [`docs/`](docs/) for anything
  affecting the API or CSS surface.

Before opening a PR, please run:

```sh
go test -race ./...
golangci-lint run
```

Both should be clean. Stable snapshots of the rendered tables are
locked via `testdata/golden/` — regenerate with
`TERMTABLE_UPDATE_GOLDEN=1 go test ./...` if your change
intentionally alters the baseline.

## License

termtable is Copyright by Carabiner Systems, Inc and released under the
Apache-2.0 license. See [`LICENSE`](LICENSE) for details, as with all our
open source projects feel free to open pull requests and issues, we love feedback!
