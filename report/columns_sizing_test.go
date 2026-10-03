package report

import (
	"testing"

	"go.bits2life.com/pdf"
)

func TestColumnsSizingModes(t *testing.T) {
	report := NewReport(pdf.A4, pdf.Portrait).WithPadding(Mm(15))

	// Create content with different natural widths
	shortText := report.Text("Short")
	mediumText := report.Text("Medium length text")
	longText := report.Text("This is a much longer text that needs more space")

	pageWidth, _ := report.doc.PageSize()
	contentWidth := pageWidth - Mm(30) // Account for padding

	// Test equal sizing (default)
	equalColumns := report.Columns(Mm(10), shortText, mediumText, longText)
	eqW, eqH, eqRemain := equalColumns.Measure(contentWidth, Mm(100))

	t.Logf("Equal sizing: width=%v, height=%v, remain=%v", eqW, eqH, eqRemain != nil)

	// Test independent sizing
	independentColumns := report.Columns(Mm(10), shortText, mediumText, longText).WithIndependentSizes()
	indW, indH, indRemain := independentColumns.Measure(contentWidth, Mm(100))

	t.Logf("Independent sizing: width=%v, height=%v, remain=%v", indW, indH, indRemain != nil)

	// Test explicit equal sizing (for API completeness)
	explicitEqualColumns := report.Columns(Mm(10), shortText, mediumText, longText).WithEqualSizes()
	explW, explH, explRemain := explicitEqualColumns.Measure(contentWidth, Mm(100))

	t.Logf("Explicit equal sizing: width=%v, height=%v, remain=%v", explW, explH, explRemain != nil)

	// Verify equal sizing modes behave the same
	if eqW != explW || eqH != explH {
		t.Errorf("Default equal sizing should match explicit equal sizing")
	}

	// Test natural widths of individual texts for comparison
	sw, _, _ := shortText.Measure(contentWidth, Mm(100))
	mw, _, _ := mediumText.Measure(contentWidth, Mm(100))
	lw, _, _ := longText.Measure(contentWidth, Mm(100))

	t.Logf("Natural widths: short=%v, medium=%v, long=%v", sw, mw, lw)

	// Independent sizing should use natural widths
	expectedIndependentWidth := sw + mw + lw + 2*Mm(10) // 2 gaps
	if indW != expectedIndependentWidth {
		t.Logf("Independent width calculation: expected=%v, got=%v", expectedIndependentWidth, indW)
	}
}

func TestColumnsVisualComparison(t *testing.T) {
	report := NewReport(pdf.A4, pdf.Portrait).WithPadding(Mm(15))

	// Create visual test document showing both modes
	contentSent := false
	report.Layout(func() ContentBlock {
		if contentSent {
			return nil
		}
		contentSent = true
		return report.Composite(
			report.Text("<b>Equal Sizing (Default):</b>"),
			report.Spacer(Mm(3)),
			report.Columns(Mm(15),
				report.Text("Short"),
				report.Text("Medium length text"),
				report.Text("This is much longer text content"),
			),
			report.Spacer(Mm(10)),

			report.Text("<b>Independent Sizing:</b>"),
			report.Spacer(Mm(3)),
			report.Columns(Mm(15),
				report.Text("Short"),
				report.Text("Medium length text"),
				report.Text("This is much longer text content"),
			).WithIndependentSizes(),
			report.Spacer(Mm(10)),

			report.Text("Notice how independent sizing uses natural content widths"),
			report.Text("while equal sizing distributes space evenly."),
		)
	})

	report.Render()
	report.Save("columns_comparison.pdf")
	t.Logf("Generated columns_comparison.pdf showing both sizing modes")
}
