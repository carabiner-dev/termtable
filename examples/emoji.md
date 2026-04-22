# Emoji width

termtable defaults to **conservative** emoji width — each visible
codepoint in a grapheme cluster is counted at its standalone
width, so ZWJ families, flag pairs, and skin-tone modifiers always
fit their columns, even on terminals that don't render the
composite glyphs. On modern terminals (iTerm2, WezTerm, Kitty,
Alacritty, recent VS Code / GNOME Terminal / Windows Terminal)
the auto-detection upgrades to tight Unicode widths.

```go
t := termtable.NewTable(termtable.WithTargetWidth(40))

h, _ := t.AddHeader()
h.AddCell(termtable.WithContent("Kind"))
h.AddCell(termtable.WithContent("Glyph"))

for _, row := range [][]string{
    {"family", "👨‍👩‍👧"},
    {"flag", "🇯🇵"},
    {"tone", "👋🏽"},
    {"plain", "🔥"},
} {
    r, _ := t.AddRow()
    r.AddCell(termtable.WithContent(row[0]))
    r.AddCell(termtable.WithContent(row[1]))
}

fmt.Print(t.String())
```

Conservative mode (the rendered row widths account for up to 6
columns for the ZWJ family):

```
┌───────────────────┬──────────────────┐
│ Kind              │ Glyph            │
├───────────────────┼──────────────────┤
│ family            │ 👨‍👩‍👧           │
├───────────────────┼──────────────────┤
│ flag              │ 🇯🇵             │
├───────────────────┼──────────────────┤
│ tone              │ 👋🏽             │
├───────────────────┼──────────────────┤
│ plain             │ 🔥               │
└───────────────────┴──────────────────┘
```

Override at the table level or via the `TERMTABLE_EMOJI_WIDTH`
environment variable:

```go
termtable.NewTable(
    termtable.WithEmojiWidth(termtable.EmojiWidthGrapheme),
)
```

```sh
TERMTABLE_EMOJI_WIDTH=grapheme my-tool
```

Related docs: [../docs/emoji.md](../docs/emoji.md) for the full
auto-detection whitelist, env-var grammar, and when each mode is
the right choice.
