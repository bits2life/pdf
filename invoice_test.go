package pdf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"testing"
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

// ProformaInvoice represents the JSON structure of the proforma invoice
type ProformaInvoice struct {
	Layout  string            `json:"layout"`
	Title   string            `json:"title"`
	Sender  string            `json:"sender"`
	Content []json.RawMessage `json:"content"`
}

type ReferenceData struct {
	Type string     `json:"type"`
	Rows [][]string `json:"rows"`
}

type ColumnsData struct {
	Type    string   `json:"type"`
	Gap     float64  `json:"gap"`
	Columns []string `json:"columns"`
}

type TableData struct {
	Type   string                   `json:"type"`
	Cols   []TableColumn            `json:"cols"`
	Rows   []map[string]interface{} `json:"rows"`
	Footer []map[string]interface{} `json:"footer,omitempty"`
}

type TableColumn struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Width string `json:"width"`
	Align string `json:"align"`
}

type BlueData struct {
	Type    string          `json:"type"`
	Padding []float64       `json:"padding"`
	Content json.RawMessage `json:"content"`
}

func TestProformaInvoiceRendering(t *testing.T) {
	// Load the JSON specification
	data, err := os.ReadFile("testdata/invoice.json")
	if err != nil {
		t.Fatalf("Failed to read testdata/invoice.json: %v", err)
	}

	var invoice ProformaInvoice
	err = json.Unmarshal(data, &invoice)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Create a new A4 portrait document
	doc := NewDocument(A4, Portrait).WithFilename("output/proforma_invoice.pdf")

	// Set up styles for the document
	doc.Styles(func(r *Registrar) {
		r.Block("base").WithSize(8).WithLineHeight(1.2)
		r.Block("title").WithFont("Arial").WithSize(16).WithBold(true).WithAlign(Right)
		r.Block("header").WithFont("Arial").WithSize(12).WithBold(true)
		r.Block("normal").WithFont("Arial")
		r.Block("ref").WithFont("Arial").WithLineHeight(2.8)
		r.Block("wrap").WithBreak(Wrap)
		r.Inline("em").WithItalic(true).WithBold(false)
		r.Inline("b").WithBold(true)
		r.Inline("i").WithItalic(true)
		r.Block("small").WithFont("Arial").WithSize(8).WithLineHeight(1.2)
		r.Block("td").WithSize(7)
		r.Block("th").WithSize(7).WithBold(true).WithColor(color.White)
		r.Block("right").WithAlign(Right)
		r.Block("left").WithAlign(Left)
		r.Block("center").WithAlign(Center)
	})

	// Create a new page
	page, err := doc.NewPage()
	if err != nil {
		t.Fatalf("Failed to create page: %v", err)
	}

	// Get the page canvas with margins
	canvas := page.Canvas().Margins(Mm(15), Mm(15), Mm(15), Mm(15))

	// Add a placeholder logo to the upper left corner (2.5x2 cm)
	logo, err := doc.RegisterImageFromBytes("logo.png", "png", testLogo())
	if err != nil {
		t.Fatalf("Failed to load logo: %v", err)
	}
	logo.SetWidth(Cm(2.5))
	canvas.DrawImage(logo, Unit(0), Unit(0))

	// Start content below the logo with some spacing
	currentY := Mm(2)

	// Render the title
	titleText := doc.NewRichText(fmt.Sprintf("<title>%s</title>", invoice.Title))
	titleW, titleH, _ := titleText.Measure(canvas.Width(), canvas.Height()-currentY)
	titleText.Draw(canvas, Unit(0), currentY, canvas.Width(), titleH)
	currentY += titleH + Mm(3)

	// Render the sender
	senderText := doc.NewRichText(invoice.Sender)
	senderW, senderH, _ := senderText.Measure(canvas.Width(), NoLimit)
	titleW.Max(senderW)
	senderText.Draw(canvas, canvas.Width()-titleW.Max(senderW), currentY, senderW, senderH)
	currentY += senderH + Mm(5)

	canvas.PushColor("404040")
	canvas.DrawLine(Unit(0), currentY, canvas.Width(), currentY)
	canvas.PopColor()
	currentY += Unit(16)

	// Add Page 1/1 to the bottom right corner
	pageText := doc.NewRichText(fmt.Sprintf("Page 1 / 1"))
	pageWidth, pageHeight, _ := pageText.Measure(canvas.Width(), NoLimit)
	pageText.Draw(canvas, canvas.Width()-pageWidth, canvas.Height()-pageHeight, pageWidth, pageHeight)

	// Render each content section
	for _, rawContent := range invoice.Content {
		currentY += renderContent(doc, canvas, currentY, rawContent)
	}

	// Save the document
	doc.Close()

	t.Log("Proforma invoice PDF generated successfully at output/proforma_invoice.pdf")
}

func renderContent(doc *Document, c Canvas, startY Unit, content json.RawMessage) Unit {
	currentY := startY

	switch content[0] {
	case '"':
		var textContent string
		if err := json.Unmarshal(content, &textContent); err != nil {
			panic(err)
		}

		materialText := doc.NewRichText(textContent)
		_, h, _ := materialText.Measure(c.Width(), NoLimit)
		materialText.Draw(c, Unit(0), currentY, c.Width(), h)
		currentY += h + Mm(2)

	case '{':
		var typeMap map[string]interface{}
		if err := json.Unmarshal(content, &typeMap); err != nil {
			return Unit(0)
		}

		contentType, ok := typeMap["type"].(string)
		if !ok {
			return Unit(0)
		}

		switch contentType {
		case "reference-data":
			var refData ReferenceData
			if err := json.Unmarshal(content, &refData); err == nil {
				currentY = renderReferenceData(doc, c, refData.Rows, currentY)
			}
		case "columns":
			var colData ColumnsData
			if err := json.Unmarshal(content, &colData); err == nil {
				currentY = renderColumns(doc, c, colData.Columns, colData.Gap, currentY)
			}
		case "table":
			var tableData TableData
			if err := json.Unmarshal(content, &tableData); err == nil {
				currentY = renderTable(doc, c, tableData, currentY)
			}
		case "blue":
			var blueData BlueData
			if err := json.Unmarshal(content, &blueData); err == nil {
				currentY = renderBlue(doc, c, blueData, currentY)
			}
		}

	default:
		var n float64
		if err := json.Unmarshal(content, &n); err == nil {
			currentY += Unit(n)
		} else {
			panic(err)
		}
	}

	return currentY - startY
}

func renderBlue(doc *Document, c Canvas, content BlueData, startY Unit) Unit {
	currentY := startY

	// c.PushColor(220, 220, 255)
	// c.FillRect(0, currentY, c.Width(), c.Height())
	// c.PopColor()

	return currentY
}

func renderReferenceData(doc *Document, canvas Canvas, rows [][]string, startY Unit) Unit {
	currentY := startY
	lineHeight := Mm(4)

	// First we'll create RichText for each label and value
	labelTexts := make([]*RichText, len(rows))
	valueTexts := make([]*RichText, len(rows))
	for i, row := range rows {
		labelTexts[i] = doc.NewRichText(row[0])
		valueTexts[i] = doc.NewRichText(row[1])
	}

	// Then measure labels to find the widest one
	maxLabelWidth := Unit(0)
	for _, labelText := range labelTexts {
		labelWidth, _, _ := labelText.Measure(canvas.Width(), canvas.Height())
		if labelWidth > maxLabelWidth {
			maxLabelWidth = labelWidth
		}
	}

	// We'll draw values at maxLabelWidth + Mm(5)
	valueX := maxLabelWidth + Mm(5)

	for i, row := range rows {
		if len(row) >= 2 {
			// Draw label (bold)
			labelTexts[i].Draw(canvas, Unit(0), currentY, valueX, lineHeight)

			// Draw value
			valueTexts[i].Draw(canvas, valueX, currentY, canvas.Width()-valueX, lineHeight)

			currentY += lineHeight
		}
	}

	return currentY
}

func renderColumns(doc *Document, canvas Canvas, columns []string, gap float64, startY Unit) Unit {
	if len(columns) == 0 {
		return startY
	}

	// Define padding to be applied to each RichText
	textPadding := Padding(Mm(5), Unit(0))

	// First, measure each column to determine its preferred width
	columnWidths := make([]Unit, len(columns))
	richTexts := make([]*RichText, len(columns))
	totalMeasuredWidth := Unit(0)

	for i, column := range columns {
		rt := doc.NewRichText(column).WithPadding(textPadding)
		richTexts[i] = rt
		// Measure with a generous width to get the natural width
		w, _, _ := rt.Measure(canvas.Width(), NoLimit)
		columnWidths[i] = w
		totalMeasuredWidth += w
	}

	// Calculate total available width (minus gaps)
	totalGapWidth := Unit(gap * float64(len(columns)-1))
	availableWidth := canvas.Width() - totalGapWidth

	// If measured widths fit within available space, use them as-is
	// Otherwise, scale them proportionally
	if totalMeasuredWidth <= availableWidth {
		// Use measured widths as-is
	} else {
		// Scale widths proportionally to fit
		scaleFactor := availableWidth.Pt() / totalMeasuredWidth.Pt()
		for i := range columnWidths {
			columnWidths[i] = columnWidths[i].Mul(scaleFactor)
		}
	}

	// Measure the total height of content (including padding)
	maxHeight := Unit(0)
	for i := range columns {
		rt := richTexts[i]
		_, h, _ := rt.Measure(columnWidths[i], NoLimit)
		if h > maxHeight {
			maxHeight = h
		}
	}

	// Draw background that extends to paper edges
	marginOffset := Mm(15) // The canvas margin that we need to extend beyond

	canvas.PushColor("DCDCFF") // #dcdcff
	canvas.FillRect(
		Unit(0)-marginOffset,               // Start at left paper edge
		startY,                             // Start at content top
		canvas.Width()+marginOffset.Mul(2), // Extend to right paper edge
		maxHeight,                          // Height includes padding from RichText
	)

	// Reset color to black for text rendering
	canvas.PopColor()

	// Now render columns at calculated positions
	currentX := Unit(0)

	for i := range columns {
		// Render the column content with its calculated width
		rt := richTexts[i]
		rt.Draw(canvas, currentX, startY, columnWidths[i], maxHeight)

		// Move to next column position
		currentX += columnWidths[i] + Unit(gap)
	}

	return startY + maxHeight
}

func renderTable(doc *Document, canvas Canvas, tableData TableData, startY Unit) Unit {
	canvas = canvas.Margins(Unit(0), Mm(-2), Unit(0), Mm(-2))
	if len(tableData.Cols) == 0 {
		return startY
	}

	// Collect width specifications from column definitions
	widthSpecs := make([]string, len(tableData.Cols))
	for i, col := range tableData.Cols {
		if col.Width == "*" {
			widthSpecs[i] = "n" // Convert old "*" format to new "n" format
		} else {
			// For backward compatibility, if it's just a number, treat it as mm
			// You can also update your JSON to use explicit units like "10mm", "15pt", etc.
			widthSpecs[i] = col.Width
			if isNumericOnly(col.Width) {
				widthSpecs[i] = col.Width + "mm"
			}
		}
	}

	// Calculate column widths
	columnWidths, err := CalculateColumnWidths(widthSpecs, canvas.Width())
	if err != nil {
		// Fallback to equal widths
		equalWidth := canvas.Width().Div(float64(len(tableData.Cols)))
		columnWidths = make([]Unit, len(tableData.Cols))
		for i := range columnWidths {
			columnWidths[i] = equalWidth
		}
	}

	currentY := startY

	// Measure table header
	headerHeight := Unit(0)
	headerPadding := Padding(Mm(2))
	for i, col := range tableData.Cols {
		colWidth := columnWidths[i]
		headerText := doc.NewRichText(col.Label).WithPadding(headerPadding).WithStyle(col.Align)
		_, h, _ := headerText.Measure(colWidth, NoLimit)
		headerHeight = headerHeight.Max(h)
	}

	canvas.pdf.SetFillColor(64, 64, 64)
	canvas.FillRect(Unit(0), currentY, canvas.Width(), headerHeight)

	// Draw table headers
	x := Unit(0)
	canvas.PushColor("FFFFFF")
	for i, col := range tableData.Cols {
		colWidth := columnWidths[i]

		// Draw header text
		headerText := doc.NewRichText(col.Label).WithPadding(headerPadding).WithStyle("th").WithStyle(col.Align)
		headerText.Draw(canvas, x, currentY, colWidth, NoLimit)

		x += colWidth
	}
	canvas.PopColor()
	currentY += headerHeight

	// Draw table rows
	cellPadding := Padding(Mm(2))
	canvas.PushColor("000000")
	for _, row := range tableData.Rows {
		x = Unit(0)
		maxRowHeight := Unit(0)

		for i, col := range tableData.Cols {
			colWidth := columnWidths[i]

			// Get cell value
			cellValue := ""
			if val, exists := row[col.ID]; exists {
				cellValue = fmt.Sprintf("%v", val)
			}

			// Draw cell text
			cellText := doc.NewRichText(cellValue).WithStyle("td").WithStyle(col.Align).WithPadding(cellPadding)
			_, h, _ := cellText.Measure(colWidth, NoLimit)
			maxRowHeight = maxRowHeight.Max(h)
			cellText.Draw(canvas, x, currentY, colWidth, NoLimit)

			x += colWidth
		}
		currentY += maxRowHeight

		// TODO: Has no effect?
		canvas.PushColor("DEDEDE")
		canvas.DrawLine(Unit(0), currentY, canvas.Width(), currentY)
		canvas.PopColor()
	}
	canvas.PopColor()

	// Draw footer if present
	if len(tableData.Footer) > 0 {
		for _, footerRow := range tableData.Footer {
			canvas.PushColor("000000")
			canvas.DrawLine(Unit(0), currentY, canvas.Width(), currentY)

			// First pass: measure all cells in this footer row to get the row height
			footerRowHeight := Unit(0)
			footerTexts := make([]*RichText, len(tableData.Cols))

			for i, col := range tableData.Cols {
				colWidth := columnWidths[i]

				// Get footer cell value
				cellValue := ""
				if val, exists := footerRow[col.ID]; exists {
					cellValue = fmt.Sprintf("%v", val)
				}

				// Create and measure footer text
				footerText := doc.NewRichText(cellValue).WithStyle("td").WithStyle(col.Align).WithPadding(Padding(Mm(1), Mm(2)))
				footerTexts[i] = footerText
				_, h, _ := footerText.Measure(colWidth, NoLimit)
				footerRowHeight = footerRowHeight.Max(h)
			}

			// Second pass: draw all cells with the calculated row height
			x := Unit(0)
			for i, _ := range tableData.Cols {
				colWidth := columnWidths[i]

				// Draw footer text
				canvas.PushColor("000000")
				footerTexts[i].Draw(canvas, x, currentY, colWidth, footerRowHeight)

				x += colWidth
			}

			currentY += footerRowHeight
		}
	}

	return currentY
}

// isNumericOnly checks if a string contains only numeric characters (and decimal point)
func isNumericOnly(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			if r != '.' {
				return false
			}
		}
	}
	return true
}

func renderMaterialList(doc *Document, canvas Canvas, rows [][]string, startY Unit) Unit {
	currentY := startY

	// Render "Material List" header
	headerText := doc.NewRichText("<b>Material List</b>")
	_, headerH, _ := headerText.Measure(canvas.Width(), Mm(10))
	headerText.Draw(canvas, Unit(0), currentY, canvas.Width(), headerH)
	currentY += headerH + Mm(3)

	// Render material data
	for _, row := range rows {
		if len(row) >= 2 {
			lineHeight := Mm(6)

			// Material number
			materialText := doc.NewRichText(fmt.Sprintf("<b>%s:</b>", row[0]))
			materialText.Draw(canvas, Unit(0), currentY, Mm(30), lineHeight)

			// Material details
			detailsText := doc.NewRichText(row[1])
			detailsText.Draw(canvas, Mm(35), currentY, canvas.Width()-Mm(35), lineHeight)

			currentY += lineHeight
		}
	}

	return currentY
}
