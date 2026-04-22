# Basic table

Two columns, one header row, two body rows, default border.

```go
package main

import (
	"fmt"

	"github.com/carabiner-dev/termtable"
)

func main() {
	t := termtable.NewTable(termtable.WithTargetWidth(30))

	h, _ := t.AddHeader()
	h.AddCell(termtable.WithContent("Name"))
	h.AddCell(termtable.WithContent("Count"))

	r1, _ := t.AddRow()
	r1.AddCell(termtable.WithContent("alpha"))
	r1.AddCell(termtable.WithContent("1"))

	r2, _ := t.AddRow()
	r2.AddCell(termtable.WithContent("beta"))
	r2.AddCell(termtable.WithContent("2"))

	fmt.Print(t.String())
}
```

```
┌──────────────┬─────────────┐
│ Name         │ Count       │
├──────────────┼─────────────┤
│ alpha        │ 1           │
├──────────────┼─────────────┤
│ beta         │ 2           │
└──────────────┴─────────────┘
```

See also the [README](../README.md) for the full getting-started
overview.
