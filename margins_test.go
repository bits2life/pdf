package pdf

import (
	"testing"
)

func TestMarginsWithCornerCrosses(t *testing.T) {
	// Create a new A4 portrait document
	doc := NewDocument(A4, Portrait).WithFilename("output/margins.pdf")

	// Create a new page
	page, err := doc.NewPage()
	if err != nil {
		t.Fatalf("Failed to create page: %v", err)
	}

	// Get the page canvas
	canvas := page.Canvas()

	// Apply 3cm (30mm) margins to all sides
	marginsCanvas := canvas.Margins(Mm(30), Mm(30), Mm(30), Mm(30))

	// Set drawing color to black
	marginsCanvas.PushColor("000000")

	// Get the dimensions of the margins area
	width := marginsCanvas.Width()
	height := marginsCanvas.Height()

	// Draw four 1cm x 1cm crosses in each corner
	crossSize := Cm(1) // 1cm in Unit

	// Top-left corner cross
	drawCross(marginsCanvas, Mm(0), Mm(0), crossSize)

	// Top-right corner cross
	drawCross(marginsCanvas, width-crossSize, Mm(0), crossSize)

	// Bottom-left corner cross
	drawCross(marginsCanvas, Mm(0), height-crossSize, crossSize)

	// Bottom-right corner cross
	drawCross(marginsCanvas, width-crossSize, height-crossSize, crossSize)

	// Save the document
	doc.Close()
}

// drawCross draws a cross at the specified position with the given size
func drawCross(canvas Canvas, x, y, size Unit) {
	// Draw horizontal line
	canvas.DrawLine(x, y+size.Div(2), x+size, y+size.Div(2))

	// Draw vertical line
	canvas.DrawLine(x+size.Div(2), y, x+size.Div(2), y+size)
}
