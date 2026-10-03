package report_test

import (
	"fmt"

	"go.bits2life.com/pdf"
	"go.bits2life.com/pdf/report"
)

func Example() {
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
	fmt.Println(err == nil && len(data) > 0, r.PageCount())
	// Output: true 1
}
