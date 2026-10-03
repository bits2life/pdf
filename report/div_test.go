package report

import (
	"testing"

	"go.bits2life.com/pdf"
)

func TestDivInlineBlockBehavior(t *testing.T) {
	report := NewReport(pdf.A4, pdf.Portrait)

	// Create a test text content that's narrower than the page
	textBlock := report.Text("Short text")

	// Test inline behavior (should size to content)
	inlineDiv := report.Div(textBlock).Inline()
	inlineW, inlineH, _ := inlineDiv.Measure(Mm(100), Mm(50))

	// Test block behavior (should use full available width)
	blockDiv := report.Div(textBlock).Block()
	blockW, blockH, _ := blockDiv.Measure(Mm(100), Mm(50))

	// Test default behavior (should be block)
	defaultDiv := report.Div(textBlock)
	defaultW, defaultH, _ := defaultDiv.Measure(Mm(100), Mm(50))

	// Inline div should be narrower than the available width
	if inlineW >= Mm(100) {
		t.Errorf("Inline div should be narrower than available width. Got %v, max was %v", inlineW, Mm(100))
	}

	// Block div should use the full available width
	if blockW != Mm(100) {
		t.Errorf("Block div should use full available width. Got %v, expected %v", blockW, Mm(100))
	}

	// Default behavior should be block (full width)
	if defaultW != Mm(100) {
		t.Errorf("Default div should use full available width. Got %v, expected %v", defaultW, Mm(100))
	}

	// Heights should be the same regardless of width behavior
	if inlineH != blockH || blockH != defaultH {
		t.Errorf("Heights should be the same. Inline: %v, Block: %v, Default: %v", inlineH, blockH, defaultH)
	}

	t.Logf("Inline width: %v, Block width: %v, Default width: %v", inlineW, blockW, defaultW)
	t.Logf("Heights (all should be equal): %v", inlineH)
}
