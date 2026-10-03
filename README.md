# pdf

`pdf` thinly wraps [gofpdf](https://github.com/jung-kurt/gofpdf) with an API
suited to building report generators: units, styles, a small rich-text markup,
canvases with margins, and images. The `report` package adds a flow layout on
top, where content blocks are measured, assigned to pages and drawn in order,
with per-page headers and footers.

```go
import (
	"go.bits2life.com/pdf"
	"go.bits2life.com/pdf/report"
)

r := report.NewReport(pdf.A4, pdf.Portrait).WithPadding(report.Mm(15))
r.Styles(func(s *pdf.Registrar) {
	s.Block("base").WithSize(9).WithLineHeight(1.2)
	s.Block("h1").WithSize(16).WithBold(true)
})

sent := false
r.Layout(func() report.ContentBlock {
	if sent {
		return nil
	}
	sent = true
	return r.Composite(
		r.Text("<h1>Hello</h1>"),
		r.Spacer(report.Mm(5)),
		r.ReferenceData([]string{"<b>Date:</b>", "2025-01-15"}),
	)
})
r.Render()
data, err := r.Bytes()
```

Only the PDF core fonts (Helvetica, Times, Courier, Symbol, ZapfDingbats) are
used; no font files are bundled. Text is encoded as Windows-1252.

See [report/README.md](report/README.md) for the layout model.

## Development

```bash
go test ./...
```

Tests write sample PDFs next to the tests (`output/`, `*.pdf`); these are
ignored by git.
