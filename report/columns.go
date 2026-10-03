package report

import "go.bits2life.com/pdf"

// ------------------------------------------------------------------------------------------------
// Columns
//
// A block that renders a list of columns with a given gap.
// The columns are rendered as a single block with the given gap between them, starting from the
// left edge of the page. An error is returned if the columns do not fit on the page (but the
// columns are rendered anyway).
// ------------------------------------------------------------------------------------------------

// ColumnsBlock represents a set of columns with gaps between them
type ColumnsBlock struct {
	gap                Unit
	columns            []ContentBlock
	columnMeasurements []Measurement
	independentSizes   bool // if true, columns size to content; if false, equal widths
}

func (r *Report) Columns(gap Unit, columns ...ContentBlock) *ColumnsBlock {
	return &ColumnsBlock{
		gap:              gap,
		columns:          columns,
		independentSizes: false, // default to equal widths
	}
}

// WithIndependentSizes sets columns to size according to their content width
func (b *ColumnsBlock) WithIndependentSizes() *ColumnsBlock {
	b.independentSizes = true
	return b
}

// WithEqualSizes sets columns to have equal widths (default behavior)
func (b *ColumnsBlock) WithEqualSizes() *ColumnsBlock {
	b.independentSizes = false
	return b
}

func (b *ColumnsBlock) Measure(maxWidth, maxHeight Unit) (usedW, usedH Unit, remain ContentBlock) {
	if len(b.columns) == 0 {
		return Unit(0), Unit(0), nil
	}

	// Calculate total gap space needed
	totalGapWidth := Unit(len(b.columns)-1) * b.gap

	// Measure each column based on sizing mode
	b.columnMeasurements = make([]Measurement, len(b.columns))
	var maxColumnHeight Unit
	var anyRemain ContentBlock
	hasAnyContent := false

	if b.independentSizes {
		// Independent sizing: let each column use its natural width
		var totalContentWidth Unit

		// First pass: measure each column at its natural width
		for i, column := range b.columns {
			// Give each column plenty of space to determine natural width
			measurement := MeasureBlock(column, maxWidth, maxHeight)
			b.columnMeasurements[i] = measurement
			totalContentWidth += measurement.Width

			// Track the tallest column
			maxColumnHeight = maxColumnHeight.Max(measurement.Height)

			// Track if any column has content (non-zero height)
			if measurement.Height > 0 {
				hasAnyContent = true
			}

			// If any column has remaining content, we have overflow
			if measurement.Remain != nil && anyRemain == nil {
				anyRemain = measurement.Remain
			}
		}

		// Calculate total width needed
		totalWidth := totalContentWidth + totalGapWidth

		// If content doesn't fit, we still return the measurements but flag overflow
		if totalWidth > maxWidth {
			// Could implement content scaling/wrapping here in the future
			totalWidth = maxWidth // Clamp to available width
		}

		return totalWidth, maxColumnHeight, anyRemain

	} else {
		// Equal sizing: distribute width equally among columns
		availableContentWidth := maxWidth - totalGapWidth

		// If we don't have enough space even for gaps, still try to render
		if availableContentWidth <= 0 {
			availableContentWidth = maxWidth / Unit(len(b.columns))
		} else {
			// Distribute available width equally among columns
			availableContentWidth = availableContentWidth / Unit(len(b.columns))
		}

		for i, column := range b.columns {
			measurement := MeasureBlock(column, availableContentWidth, maxHeight)
			b.columnMeasurements[i] = measurement

			// Track the tallest column
			maxColumnHeight = maxColumnHeight.Max(measurement.Height)

			// Track if any column has content (non-zero height)
			if measurement.Height > 0 {
				hasAnyContent = true
			}

			// If any column has remaining content, we have overflow
			if measurement.Remain != nil && anyRemain == nil {
				anyRemain = measurement.Remain
			}
		}

		// Calculate total width used
		totalWidth := Unit(len(b.columns))*availableContentWidth + totalGapWidth

		// Don't exceed the maximum available width
		if totalWidth > maxWidth {
			totalWidth = maxWidth
		}

		// If no content could be rendered but we have columns,
		// return a small height to ensure layout progresses
		if !hasAnyContent && len(b.columns) > 0 {
			maxColumnHeight = Mm(1) // Minimal height to prevent layout from getting stuck
		}

		return totalWidth, maxColumnHeight, anyRemain
	}
}

func (b *ColumnsBlock) Draw(canvas pdf.Canvas, x, y, maxWidth, maxHeight Unit) error {
	if len(b.columns) == 0 {
		return nil
	}

	if b.independentSizes {
		// Independent sizing: position based on actual column widths
		currentX := x
		for i, measurement := range b.columnMeasurements {
			err := measurement.Draw(canvas, currentX, y)
			if err != nil {
				return err
			}

			// Move to the next column position (actual column width + gap)
			currentX += measurement.Width
			if i < len(b.columns)-1 { // Don't add gap after the last column
				currentX += b.gap
			}
		}

	} else {
		// Equal sizing: position based on equal column widths
		totalGapWidth := Unit(len(b.columns)-1) * b.gap
		availableContentWidth := maxWidth - totalGapWidth

		if availableContentWidth <= 0 {
			availableContentWidth = maxWidth / Unit(len(b.columns))
		} else {
			availableContentWidth = availableContentWidth / Unit(len(b.columns))
		}

		// Draw each column at its calculated position
		currentX := x
		for i, measurement := range b.columnMeasurements {
			err := measurement.Draw(canvas, currentX, y)
			if err != nil {
				return err
			}

			// Move to the next column position (equal column width + gap)
			currentX += availableContentWidth
			if i < len(b.columns)-1 { // Don't add gap after the last column
				currentX += b.gap
			}
		}
	}

	return nil
}
