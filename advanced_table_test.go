package pdf

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"testing"
)

func TestAdvancedTableRendering(t *testing.T) {
	// Load the advanced table JSON specification
	data, err := os.ReadFile("testdata/advanced_table.json")
	if err != nil {
		t.Fatalf("Failed to read testdata/advanced_table.json: %v", err)
	}

	var invoice ProformaInvoice
	err = json.Unmarshal(data, &invoice)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Create a new A4 portrait document
	doc := NewDocument(A4, Portrait).WithFilename("output/advanced_table_demo.pdf")

	// Set up styles for the document
	doc.Styles(func(r *Registrar) {
		r.Block("base").WithSize(9)
		r.Block("title").WithFont("Arial").WithSize(18).WithBold(true).WithAlign(Center)
		r.Block("header").WithFont("Arial").WithSize(12).WithBold(true)
		r.Block("normal").WithFont("Arial")
		r.Block("ref").WithFont("Arial").WithLineHeight(2.8)
		r.Block("wrap").WithBreak(Wrap)
		r.Inline("em").WithItalic(true)
		r.Block("small").WithFont("Arial").WithSize(8).WithLineHeight(1.2)
		r.Block("td").WithSize(7)
		r.Block("th").WithSize(7).WithBold(true)
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

	// Start content from the top
	currentY := Unit(0)

	// Render the title
	titleText := doc.NewRichText(fmt.Sprintf("<title>%s</title>", invoice.Title))
	titleW, titleH, _ := titleText.Measure(canvas.Width(), canvas.Height()-currentY)
	titleText.Draw(canvas, (canvas.Width() - titleW).Div(2), currentY, canvas.Width(), titleH)
	currentY += titleH + Mm(10)

	// Render each content section
	for _, rawContent := range invoice.Content {
		// Try to parse as string first
		var textContent string
		if err := json.Unmarshal(rawContent, &textContent); err == nil {
			fmt.Println("textContent", textContent)

			materialText := doc.NewRichText(textContent)
			_, h, _ := materialText.Measure(canvas.Width(), NoLimit)
			materialText.Draw(canvas, Unit(0), currentY, canvas.Width(), h)
			currentY += h + Mm(3)
			continue
		}

		// Try to parse as a map to determine the type
		var typeMap map[string]interface{}
		if err := json.Unmarshal(rawContent, &typeMap); err != nil {
			continue
		}

		contentType, ok := typeMap["type"].(string)
		if !ok {
			continue
		}

		switch contentType {
		case "reference-data":
			var refData ReferenceData
			if err := json.Unmarshal(rawContent, &refData); err == nil {
				currentY = renderReferenceData(doc, canvas, refData.Rows, currentY)
			}
		case "table":
			var tableData TableData
			if err := json.Unmarshal(rawContent, &tableData); err == nil {
				currentY = renderTable(doc, canvas, tableData, currentY)
			}
		}
		currentY += Mm(8) // Add spacing between sections
	}

	// Save the document
	doc.Close()

	t.Log("Advanced table demo PDF generated successfully at output/advanced_table_demo.pdf")
}

func TestColumnWidthCalculationDemo(t *testing.T) {
	// Demonstrate various column width calculations
	testCases := []struct {
		name       string
		specs      []string
		totalWidth Unit
	}{
		{
			name:       "User's Original Example",
			specs:      []string{"15mm", "65pt", "10%", "n", "2n"},
			totalWidth: Pt(500),
		},
		{
			name:       "Advanced Table Example",
			specs:      []string{"20mm", "n", "15%", "50pt", "1.5cm", "2n"},
			totalWidth: Pt(595), // A4 width minus margins
		},
		{
			name:       "Responsive Design Example",
			specs:      []string{"5%", "20%", "n", "15mm", "2n"},
			totalWidth: Pt(400),
		},
		{
			name:       "Complex Mixed Example",
			specs:      []string{"1in", "25%", "0.5n", "30pt", "2n", "10%", "n"},
			totalWidth: Pt(800),
		},
	}

	fmt.Printf("\n=== Column Width Calculation Demonstration ===\n")

	for _, tc := range testCases {
		fmt.Printf("\n--- %s ---\n", tc.name)
		fmt.Printf("Specifications: %v\n", tc.specs)
		fmt.Printf("Total Width: %.2f pt\n", tc.totalWidth.Pt())

		widths, err := CalculateColumnWidths(tc.specs, tc.totalWidth)
		if err != nil {
			t.Errorf("Error calculating widths for %s: %v", tc.name, err)
			continue
		}

		fmt.Printf("Calculated Widths:\n")
		totalCalculated := Unit(0)
		for i, width := range widths {
			fmt.Printf("  %s -> %.2f pt (%.2f mm)\n", tc.specs[i], width.Pt(), width.Mm())
			totalCalculated += width
		}

		fmt.Printf("Total: %.2f pt (difference: %.2f pt)\n",
			totalCalculated.Pt(),
			math.Abs(totalCalculated.Pt()-tc.totalWidth.Pt()))

		// Verify the total matches (within floating point precision)
		if math.Abs(totalCalculated.Pt()-tc.totalWidth.Pt()) > 0.01 {
			t.Errorf("Width calculation mismatch for %s: expected %.2f, got %.2f",
				tc.name, tc.totalWidth.Pt(), totalCalculated.Pt())
		}
	}
}
