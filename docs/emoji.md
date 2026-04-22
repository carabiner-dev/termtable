# Emoji width

The short version: `termtable` defaults to the **conservative** width
mode, which reserves enough columns for each emoji even on terminals
that don't render composite emoji as a single glyph. On a whitelist
of terminals known to handle emoji correctly, it auto-upgrades to
the tighter **grapheme** mode. You can override both with
`WithEmojiWidth(...)` or the `TERMTABLE_EMOJI_WIDTH` environment
variable.

## Why modes exist

Unicode defines a **grapheme cluster** width for composite emoji —
ZWJ families like `👨‍👩‍👧`, flag pairs like `🇯🇵`, skin-tone modifiers
like `👋🏽`, variation-selector sequences like `❤️`. Per the
standard, the whole cluster has a display width of 2 (one emoji
box). That's what `uniseg.StringWidth` returns; termtable uses it
when asked.

In practice the **rendered** width depends on the font. If the
terminal's font has a glyph for the joined form, you get the tight
single-box rendering. If it doesn't, the terminal falls back to
drawing each constituent codepoint as its own emoji — a ZWJ family
splits into three 2-column glyphs for a total visual width of 6.

That discrepancy is what breaks tables. If termtable pads the row
for width 2 but the terminal actually draws width 6, the right
border gets pushed four columns out and everything from that row
onward looks misaligned.

Terminals that commonly get this wrong include tmux / screen with a
font that lacks emoji, most SSH sessions to Linux servers (which
usually lack emoji fonts entirely), Windows terminals in certain
configurations, GitLab/GitHub log viewers rendering captured output,
and anything inside a `less` pager with an older `$LANG`.

## The two modes

### `EmojiWidthConservative` — safe (default)

Counts every visible codepoint in a grapheme cluster at its
standalone width, skipping zero-width composers (ZWJ, variation
selectors, combining marks). The result is always at least the
uniseg width, so pure CJK and ASCII tokens are unaffected.

- `👨‍👩‍👧` = 6 (man + woman + girl, each 2 wide)
- `🇯🇵` = 4 (two regional indicators, each 2 wide)
- `👋🏽` = 4 (wave + tone, each 2 wide)
- `❤️` = 2 (heart + VS16 → uniseg already reports 2)
- `漢` = 2 (CJK, unchanged)
- `a` = 1 (ASCII, unchanged)

Sample output with all four composite types:

```
┌───────────────────┬──────────────────┐
│ Kind              │ Glyph            │
├───────────────────┼──────────────────┤
│ family            │ 👨‍👩‍👧           │
├───────────────────┼──────────────────┤
│ flag              │ 🇯🇵             │
├───────────────────┼──────────────────┤
│ skin tone         │ 👋🏽             │
├───────────────────┼──────────────────┤
│ plain             │ 🔥               │
└───────────────────┴──────────────────┘
```

The grid stays aligned under any rendering — from a bare xterm that
treats every component as its own glyph all the way to iTerm2
drawing the family as a single hieroglyph.

### `EmojiWidthGrapheme` — tight

Uses `uniseg.StringWidth` as-is. Correct per Unicode, and what most
modern terminals actually render:

```
┌───────────────────┬──────────────────┐
│ Kind              │ Glyph            │
├───────────────────┼──────────────────┤
│ family            │ 👨‍👩‍👧               │
├───────────────────┼──────────────────┤
│ flag              │ 🇯🇵               │
├───────────────────┼──────────────────┤
│ skin tone         │ 👋🏽               │
├───────────────────┼──────────────────┤
│ plain             │ 🔥               │
└───────────────────┴──────────────────┘
```

Same glyph column, but the emoji take only 2 columns each — the
row is tighter and looks correct on a terminal that actually
renders the composite forms. On a terminal that doesn't, the last
three data rows will overshoot the right border.

## Precedence

The mode in force for any given render is resolved in this order:

1. **Explicit `WithEmojiWidth(mode)`** — if mode is `Conservative`
   or `Grapheme`, it wins unconditionally. `WithEmojiWidth(EmojiWidthAuto)`
   is equivalent to not calling the option.
2. **`TERMTABLE_EMOJI_WIDTH`** environment variable — values
   `conservative` (or `safe`, `wide`) and `grapheme` (or `unicode`,
   `tight`) are recognized. Any other value is ignored and
   resolution falls through.
3. **Terminal auto-detection** — if any of the following env vars
   point at a known-capable terminal, mode resolves to
   `Grapheme`. Otherwise it falls through to conservative.
4. **Conservative** — the ultimate fallback.

## Terminal auto-detection whitelist

The whitelist is deliberately narrow. Entries are terminals that
ship with (or are commonly configured with) fonts capable of
rendering ZWJ emoji families, regional-indicator flags, and
skin-tone modifiers as single glyphs.

| Env var         | Value matches                                                  |
|:----------------|:---------------------------------------------------------------|
| `TERM_PROGRAM`  | `iTerm.app`, `WezTerm`, `vscode`, `Hyper`, `Apple_Terminal`, `ghostty` |
| `TERM`          | `xterm-kitty`, `wezterm`, `xterm-ghostty`, `alacritty`, `alacritty-direct` |
| `WT_SESSION`    | any non-empty value (Windows Terminal / ConPTY sets this)      |
| `VTE_VERSION`   | `>= 6000` (VTE 0.60+, i.e. GNOME Terminal released 2019 or later) |

Terminals not on this list — bare `xterm`, `screen`, anything
running under a TTY multiplexer without TTY inheritance, most CI
log viewers — fall through to conservative.

**When the heuristic is wrong**, override it. If you ship a tool
that targets, say, Kitty exclusively and want the tight layout
regardless of detection:

```go
t := termtable.NewTable(termtable.WithEmojiWidth(termtable.EmojiWidthGrapheme))
```

Or let the user decide:

```sh
TERMTABLE_EMOJI_WIDTH=grapheme my-tool
```

## `DisplayWidth` is unaffected

The public `termtable.DisplayWidth(s)` function always reports
Unicode-standard grapheme widths via uniseg. It is *not* affected
by the table's emoji-width mode — callers of the helper expect a
stable contract. Table-internal measurement goes through a
separate code path that honours the mode.

If you need the mode-aware width explicitly (for instance to lay
out text yourself outside a table), inspect the value on a table
via `tbl.LastRenderError` and related accessors, or call
`termtable.MinUnbreakableWidth` which is similarly
grapheme-normative.

## Summary

- **Default is conservative.** Tables always align, even on
  crummy terminals.
- **Auto-detection upgrades to grapheme** on a known-good
  whitelist.
- **Env var** (`TERMTABLE_EMOJI_WIDTH`) lets users override
  without rebuilding.
- **Explicit option** wins over env var for programmer intent.
- **Public helper widths stay Unicode-correct** regardless of mode.
