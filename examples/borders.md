# Border glyph sets

termtable ships six ready-made border sets. Select one with
`WithBorder` or `WithTableStyle("border-style: ...")`.

```go
t := termtable.NewTable(
    termtable.WithTargetWidth(30),
    termtable.WithBorder(termtable.RoundedLine()),
)

h := t.AddHeader()
h.AddCell(termtable.WithContent("Col A"))
h.AddCell(termtable.WithContent("Col B"))

r := t.AddRow()
r.AddCell(termtable.WithContent("one"))
r.AddCell(termtable.WithContent("two"))

fmt.Print(t.String())
```

`DoubleLine()` (`border-style: double`):

```
╔══════════════╦═════════════╗
║ Col A        ║ Col B       ║
╠══════════════╬═════════════╣
║ one          ║ two         ║
╚══════════════╩═════════════╝
```

`HeavyLine()` (`border-style: heavy`):

```
┏━━━━━━━━━━━━━━┳━━━━━━━━━━━━━┓
┃ Col A        ┃ Col B       ┃
┣━━━━━━━━━━━━━━╋━━━━━━━━━━━━━┫
┃ one          ┃ two         ┃
┗━━━━━━━━━━━━━━┻━━━━━━━━━━━━━┛
```

`RoundedLine()` (`border-style: rounded`):

```
╭──────────────┬─────────────╮
│ Col A        │ Col B       │
├──────────────┼─────────────┤
│ one          │ two         │
╰──────────────┴─────────────╯
```

`ASCIILine()` (`border-style: ascii`) — for environments without
Unicode box-drawing support:

```
+--------------+-------------+
| Col A        | Col B       |
+--------------+-------------+
| one          | two         |
+--------------+-------------+
```

`NoBorder()` (`border-style: none`) — invisible borders but
preserved grid alignment:

```
                              
  Col A          Col B        
                              
  one            two          
                              
```

Related docs: [../docs/borders.md](../docs/borders.md) for every
set, including `SingleLine()` (the default) and instructions for
building your own.
