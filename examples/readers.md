# io.Reader content

A cell's content can come from an `io.Reader` instead of an
in-memory string. termtable consumes the reader lazily on the
first render pass and caches the result, so subsequent renders
don't re-read.

```go
import (
    "io"
    "strings"

    "github.com/carabiner-dev/termtable"
)

t := termtable.NewTable(termtable.WithTargetWidth(50))

h, _ := t.AddHeader()
h.AddCell(termtable.WithContent("Source"))
h.AddCell(termtable.WithContent("Body"))

r1, _ := t.AddRow()
r1.AddCell(termtable.WithContent("inline"))
r1.AddCell(termtable.WithContent("authored in-place"))

r2, _ := t.AddRow()
r2.AddCell(termtable.WithContent("reader"))
var src io.Reader = strings.NewReader("consumed from io.Reader")
r2.AddCell(termtable.WithReader(src))

fmt.Print(t.String())
```

```
┌──────────────────────┬─────────────────────────┐
│ Source               │ Body                    │
├──────────────────────┼─────────────────────────┤
│ inline               │ authored in-place       │
├──────────────────────┼─────────────────────────┤
│ reader               │ consumed from io.Reader │
└──────────────────────┴─────────────────────────┘
```

If the reader returns an error during consumption, the cell renders
as empty and a `ReaderErrorEvent` lands on `Table.Warnings()` —
see [warnings.md](warnings.md).

Don't pair `WithContent` and `WithReader` on the same cell — that
returns `ErrContentAndReader` at `AddCell` time.
