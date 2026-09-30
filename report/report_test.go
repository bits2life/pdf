package report

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"testing"

	"go.bits2life.com/pdf"
)

// testLogo returns a small placeholder logo as PNG bytes.
func testLogo() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 100, 80))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{40, 70, 140, 255}}, image.Point{}, draw.Src)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// newInvoiceReport sets up an example theme: A4 portrait with a logo and title header and a
// page-numbered footer.
func newInvoiceReport(title string) *Report {
	report := NewReport(pdf.A4, pdf.Portrait).WithPadding(Mm(0), Mm(15))
	report.Styles(func(r *pdf.Registrar) {
		r.Block("h1").WithSize(16).WithBold(true).WithAlign(pdf.Right).WithLineHeight(1.2)
		r.Block("base").WithSize(8).WithLineHeight(1.2)
		r.Block("title").WithSize(16).WithBold(true).WithAlign(pdf.Right)
		r.Block("normal").WithFont("Helvetica")
		r.Block("ref").WithLineHeight(2.8)
		r.Block("small").WithSize(8).WithLineHeight(1.2)
		r.Block("footer").WithSize(8).WithAlign(pdf.Right).WithColor(color.RGBA{66, 66, 66, 255})
	})

	if _, err := report.RegisterImageFromBytes("logo.png", "png", testLogo()); err != nil {
		panic(err)
	}

	report.HeaderFunc(func(maxWidth Unit, pageNum int) ContentBlock {
		return report.Div(
			report.LeftRight(
				report.Image("logo.png", Mm(25), pdf.Automatic),
				report.Div(report.Text(title)).WithPadding(Mm(2)).Inline(),
			),
			report.Spacer(Mm(10)),
			report.Line(Pt(0.3)),
		).WithPadding(Mm(15))
	})

	report.FooterFunc(func(maxWidth Unit, pageNum int) ContentBlock {
		return report.Dynamic(func() ContentBlock {
			t := fmt.Sprintf("<footer>Page %d of %d</footer>", pageNum, report.PageCount())
			return report.Div(report.Text(t)).WithPadding(Mm(6), Mm(15))
		})
	})

	return report
}

func TestGenerateReport(t *testing.T) {
	report := newInvoiceReport("<h1>Proforma Invoice</h1><small><i><b><br/> <br/>Example Supplier AB</b><br/>Exempelgatan 1<br/>123 45 Exempelstad<br/>Sweden<br/><em>VAT: SE000000000001</em></i></small>")

	// Layout the content using the Layout method as a generator
	contentSent := false
	report.Layout(func() ContentBlock {
		if contentSent {
			return nil // No more content
		}
		contentSent = true
		return report.Composite(
			report.ReferenceData(
				[]string{
					"<b>Invoice No.:</b>", "100000001",
					"<b>Invoice Date:</b>", "2025-01-15",
					"<b>Shipping Condition:</b>", "Truck - Consolidated",
					"<b>Shipment ID:</b>", "2000001",
					"<b>Date Shipped:</b>", "2025-01-15",
					"<b>Delivery Number:</b>", "3000001",
					"<b>Incoterms:</b>", "FCA Voorbeeld",
					"<b>Customs Clearance:</b>", "Undeclared",
					"<b>Currency:</b>", "USD",
				},
			),
			report.Spacer(Mm(10)),
			report.Band(
				report.Columns(Mm(20),
					report.Text("<small><b>Consignor:</b><br/>Example Manufacturing B.V.<br/>Voorbeeldstraat 10<br/>1234 AB Voorbeeld<br/>Netherlands<br/><em>VAT: NL000000000B01</em></small>"),
					report.Text("<small><b>Sold To:</b><br/>Example Customer AB<br/>Provgatan 2<br/>543 21 Provstad<br/>Sweden<br/><em>VAT: SE000000000002</em></small>"),
					report.Text("<small><b>Ship To:</b><br/>Example Supplier AB<br/>Exempelgatan 1<br/>123 45 Exempelstad<br/>Sweden<br/><em>VAT: SE000000000001</em></small>"),
				).WithIndependentSizes(),
				"F0F0FF",
			),
			report.Spacer(Mm(10)),
			report.Table(
				[]TableColumn{
					{ID: "po", Label: "<b>P.O.</b>", Width: "1.2cm", Align: "left"},
					{ID: "order_no", Label: "<b>Order No.</b>", Width: "1.7cm", Align: "left"},
					{ID: "material_no", Label: "<b>Material No.<br/><em>Part No.</em></b>", Width: "2cm", Align: "left"},
					{ID: "description", Label: "<b>Description</b>", Width: "n", Align: "left"},
					{ID: "hs_custom_code", Label: "<b>HS Custom<br/>Code</b>", Width: "2cm", Align: "center"},
					{ID: "country_of_origin", Label: "<b>Country<br/>of Origin</b>", Width: "1.8cm", Align: "center"},
					{ID: "qty", Label: "<b>Qty</b>", Width: "1cm", Align: "center"},
					{ID: "unit_price", Label: "<b>Unit Price</b>", Width: "2cm", Align: "right"},
					{ID: "net_price", Label: "<b>Net Price</b>", Width: "2cm", Align: "right"},
				},
				[]map[string]interface{}{
					{"po": "1001", "order_no": "500001", "material_no": "ABC1001<br/><em>1234567890</em>", "description": "GEARBOX ASSEMBLY", "hs_custom_code": "1234567890", "country_of_origin": "US", "qty": 4, "unit_price": "123,00", "net_price": "1492,00"},
					{"po": "1002", "order_no": "500001", "material_no": "ABC1002<br/><em>1234567890</em>", "description": "GEARBOX ASSEMBLY", "hs_custom_code": "1234567890", "country_of_origin": "US", "qty": 4, "unit_price": "123,00", "net_price": "1492,00"},
				},
			).WithFooter([]map[string]interface{}{
				{"qty": "8", "net_price": "2984,00"},
			}),
			report.Spacer(Mm(10)),
			report.Columns(Mm(10),
				report.Text("<small><b>Material List</b><br/><em>M000001</em></small>"),
				report.Text("<small>9000000001<br/>9000000002<br/>9000000003</small>"),
				report.Text("<small>9000000004<br/>9000000005<br/>9000000006</small>"),
				report.Text("<small>9000000001<br/>9000000002<br/>9000000003</small>"),
				report.Text("<small>9000000004<br/>9000000005</small>"),
			).WithIndependentSizes(),
		)
	})

	// Render the report to generate the pages
	report.Render()

	// Save the PDF to a file
	report.Save("test_output.pdf")

	// Basic validation that the report was created successfully
	if report.PageCount() < 1 {
		t.Error("Expected at least 1 page in the report")
	}

	t.Logf("Successfully generated report with %d page(s)", report.PageCount())
}
