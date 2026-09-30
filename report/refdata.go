package report

import (
	"go.bits2life.com/pdf"
)

// ------------------------------------------------------------------------------------------------
// ReferenceDataBlock
//
// A block that renders key-value pairs as a two-column table, with the left column sized to
// the longest key and a 10cm gap between columns.
// ------------------------------------------------------------------------------------------------
type ReferenceDataBlock struct {
	report *Report
	pairs  []KeyValuePair
}

type KeyValuePair struct {
	Key   string
	Value string
}

func (r *Report) ReferenceData(data []string) ContentBlock {
	// Convert the flat array into key-value pairs
	var pairs []KeyValuePair
	for i := 0; i < len(data); i += 2 {
		if i+1 < len(data) {
			pairs = append(pairs, KeyValuePair{
				Key:   data[i],
				Value: data[i+1],
			})
		}
	}

	return &ReferenceDataBlock{
		report: r,
		pairs:  pairs,
	}
}

func (b *ReferenceDataBlock) Measure(maxWidth, maxHeight Unit) (usedW, usedH Unit, remain ContentBlock) {
	if len(b.pairs) == 0 {
		return Mm(0), Mm(0), nil
	}

	// Find the width of the longest key
	var maxKeyWidth Unit
	for _, pair := range b.pairs {
		keyText := b.report.Text(pair.Key)
		w, _, _ := keyText.Measure(maxWidth, maxHeight)
		maxKeyWidth = maxKeyWidth.Max(w)
	}

	// Calculate total used width: key column + 10mm gap + value column
	gap := Mm(5) // 10mm gap as specified
	valueColumnWidth := maxWidth - maxKeyWidth - gap

	// If there's not enough space for a reasonable value column, use all available space
	if valueColumnWidth < Mm(20) {
		valueColumnWidth = maxWidth - maxKeyWidth
		if valueColumnWidth < Mm(10) {
			valueColumnWidth = Mm(10)
		}
	}

	// Calculate total height by measuring each row
	var totalHeight Unit
	for i, pair := range b.pairs {
		keyText := b.report.Text(pair.Key)
		valueText := b.report.Text(pair.Value)

		_, keyH, _ := keyText.Measure(maxKeyWidth, maxHeight-totalHeight)
		_, valueH, _ := valueText.Measure(valueColumnWidth, maxHeight-totalHeight)

		rowHeight := keyH.Max(valueH)

		// Check if this row would exceed available height
		if totalHeight+rowHeight > maxHeight && i > 0 {
			// Return partial content with remaining pairs
			remainingPairs := []string{}
			for j := i; j < len(b.pairs); j++ {
				remainingPairs = append(remainingPairs, b.pairs[j].Key, b.pairs[j].Value)
			}
			return maxKeyWidth + gap + valueColumnWidth, totalHeight, b.report.ReferenceData(remainingPairs)
		}

		totalHeight += rowHeight
	}

	return maxKeyWidth + gap + valueColumnWidth, totalHeight, nil
}

func (b *ReferenceDataBlock) Draw(canvas pdf.Canvas, x, y, maxWidth, maxHeight Unit) error {
	if len(b.pairs) == 0 {
		return nil
	}

	// Find the width of the longest key (recalculate for consistency)
	var maxKeyWidth Unit
	for _, pair := range b.pairs {
		keyText := b.report.Text(pair.Key)
		w, _, _ := keyText.Measure(maxWidth, maxHeight)
		maxKeyWidth = maxKeyWidth.Max(w)
	}

	gap := Mm(5) // 10mm gap
	valueColumnWidth := maxWidth - maxKeyWidth - gap
	if valueColumnWidth < Mm(20) {
		valueColumnWidth = maxWidth - maxKeyWidth
		if valueColumnWidth < Mm(10) {
			valueColumnWidth = Mm(10)
		}
	}

	// Draw each row
	currentY := y
	for _, pair := range b.pairs {
		keyText := b.report.Text(pair.Key)
		valueText := b.report.Text(pair.Value)

		// Draw key in left column
		err := keyText.Draw(canvas, x, currentY, maxKeyWidth, maxHeight-(currentY-y))
		if err != nil {
			return err
		}

		// Draw value in right column (positioned after gap)
		valueX := x + maxKeyWidth + gap
		err = valueText.Draw(canvas, valueX, currentY, valueColumnWidth, maxHeight-(currentY-y))
		if err != nil {
			return err
		}

		// Calculate row height and advance Y position
		_, keyH, _ := keyText.Measure(maxKeyWidth, maxHeight-(currentY-y))
		_, valueH, _ := valueText.Measure(valueColumnWidth, maxHeight-(currentY-y))
		rowHeight := keyH.Max(valueH)
		currentY += rowHeight

		// Stop if we've exceeded available height
		if currentY-y > maxHeight {
			break
		}
	}

	return nil
}
