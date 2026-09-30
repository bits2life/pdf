package report

import (
	"fmt"

	"go.bits2life.com/pdf"
)

// ------------------------------------------------------------------------------------------------
// TableBlock
//
// A table component that renders structured data with headers, rows, and optional footers.
// Uses the CalculateColumnWidths function for flexible column sizing.
// ------------------------------------------------------------------------------------------------

// TableColumn represents a table column definition
type TableColumn struct {
	ID    string // Unique identifier for the column
	Label string // Display label (supports rich text)
	Width string // Width specification (e.g., "15mm", "10%", "n", "2n")
	Align string // Text alignment ("left", "center", "right")
}

// TableBlock represents a table with columns, rows, and optional footer
type TableBlock struct {
	report  *Report
	columns []TableColumn
	rows    []map[string]interface{}
	footer  []map[string]interface{}
}

// Table creates a new table block with the specified columns and rows
func (r *Report) Table(columns []TableColumn, rows []map[string]interface{}) *TableBlock {
	return &TableBlock{
		report:  r,
		columns: columns,
		rows:    rows,
		footer:  nil,
	}
}

// WithFooter adds footer rows to the table (chainable method)
func (tb *TableBlock) WithFooter(footer []map[string]interface{}) *TableBlock {
	tb.footer = footer
	return tb
}

func (tb *TableBlock) Measure(maxWidth, maxHeight Unit) (usedW, usedH Unit, remain ContentBlock) {
	if len(tb.columns) == 0 {
		return Unit(0), Unit(0), nil
	}

	// Collect width specifications from column definitions
	widthSpecs := make([]string, len(tb.columns))
	for i, col := range tb.columns {
		widthSpecs[i] = col.Width
		// Convert legacy "*" format to "n" for backward compatibility
		if col.Width == "*" {
			widthSpecs[i] = "n"
		}
		// For numeric-only specs, treat as mm
		if isNumericOnly(col.Width) {
			widthSpecs[i] = col.Width + "mm"
		}
	}

	// Calculate column widths
	columnWidths, err := pdf.CalculateColumnWidths(widthSpecs, maxWidth)
	if err != nil {
		// Fallback to equal widths
		equalWidth := maxWidth.Div(float64(len(tb.columns)))
		columnWidths = make([]Unit, len(tb.columns))
		for i := range columnWidths {
			columnWidths[i] = equalWidth
		}
	}

	var totalHeight Unit
	cellPadding := pdf.Padding(Mm(2))
	headerPadding := pdf.Padding(Mm(2))

	// Measure header height
	headerHeight := Unit(0)
	for i, col := range tb.columns {
		colWidth := columnWidths[i]
		headerText := tb.report.doc.NewRichText(col.Label).WithPadding(headerPadding).WithStyle("th").WithStyle(col.Align)
		_, h, _ := headerText.Measure(colWidth, maxHeight)
		headerHeight = headerHeight.Max(h)
	}
	totalHeight += headerHeight

	// Measure row heights
	for rowIdx, row := range tb.rows {
		maxRowHeight := Unit(0)
		for i, col := range tb.columns {
			colWidth := columnWidths[i]

			// Get cell value
			cellValue := ""
			if val, exists := row[col.ID]; exists {
				cellValue = fmt.Sprintf("%v", val)
			}

			// Measure cell text
			cellText := tb.report.doc.NewRichText(cellValue).WithStyle("td").WithStyle(col.Align).WithPadding(cellPadding)
			_, h, _ := cellText.Measure(colWidth, maxHeight-totalHeight)
			maxRowHeight = maxRowHeight.Max(h)
		}

		// Check if this row would exceed available height
		if totalHeight+maxRowHeight > maxHeight && rowIdx > 0 {
			// Return partial content with remaining rows
			remainingRows := make([]map[string]interface{}, len(tb.rows)-rowIdx)
			copy(remainingRows, tb.rows[rowIdx:])

			remainingTable := &TableBlock{
				report:  tb.report,
				columns: tb.columns,
				rows:    remainingRows,
				footer:  tb.footer, // Footer goes with remaining content
			}

			return maxWidth, totalHeight, remainingTable
		}

		totalHeight += maxRowHeight
	}

	// Measure footer height if present
	if len(tb.footer) > 0 {
		for _, footerRow := range tb.footer {
			maxFooterRowHeight := Unit(0)
			for i, col := range tb.columns {
				colWidth := columnWidths[i]

				// Get footer cell value
				cellValue := ""
				if val, exists := footerRow[col.ID]; exists {
					cellValue = fmt.Sprintf("%v", val)
				}

				// Measure footer text
				footerText := tb.report.doc.NewRichText(cellValue).WithStyle("td").WithStyle(col.Align).WithPadding(cellPadding)
				_, h, _ := footerText.Measure(colWidth, maxHeight-totalHeight)
				maxFooterRowHeight = maxFooterRowHeight.Max(h)
			}

			// Check if footer would exceed available height
			if totalHeight+maxFooterRowHeight > maxHeight {
				// Move footer to remaining content
				remainingTable := &TableBlock{
					report:  tb.report,
					columns: tb.columns,
					rows:    nil, // No rows left, just footer
					footer:  tb.footer,
				}

				return maxWidth, totalHeight, remainingTable
			}

			totalHeight += maxFooterRowHeight
		}
	}

	return maxWidth, totalHeight, nil
}

func (tb *TableBlock) Draw(canvas pdf.Canvas, x, y, maxWidth, maxHeight Unit) error {
	if len(tb.columns) == 0 {
		return nil
	}

	// Collect width specifications from column definitions
	widthSpecs := make([]string, len(tb.columns))
	for i, col := range tb.columns {
		widthSpecs[i] = col.Width
		// Convert legacy "*" format to "n" for backward compatibility
		if col.Width == "*" {
			widthSpecs[i] = "n"
		}
		// For numeric-only specs, treat as mm
		if isNumericOnly(col.Width) {
			widthSpecs[i] = col.Width + "mm"
		}
	}

	// Calculate column widths
	columnWidths, err := pdf.CalculateColumnWidths(widthSpecs, maxWidth)
	if err != nil {
		// Fallback to equal widths
		equalWidth := maxWidth.Div(float64(len(tb.columns)))
		columnWidths = make([]Unit, len(tb.columns))
		for i := range columnWidths {
			columnWidths[i] = equalWidth
		}
	}

	currentY := y
	cellPadding := pdf.Padding(Mm(2))
	headerPadding := pdf.Padding(Mm(2))

	// Draw table header with dark background
	headerHeight := Unit(0)
	for i, col := range tb.columns {
		colWidth := columnWidths[i]
		headerText := tb.report.doc.NewRichText(col.Label).WithPadding(headerPadding).WithStyle("th").WithStyle(col.Align)
		_, h, _ := headerText.Measure(colWidth, maxHeight-(currentY-y))
		headerHeight = headerHeight.Max(h)
	}

	// Draw header background
	canvas.PushColor("404040")
	canvas.FillRect(x, currentY, maxWidth, headerHeight)
	canvas.PopColor()

	// Draw header text
	currentX := x
	canvas.PushColor("FFFFFF")
	for i, col := range tb.columns {
		colWidth := columnWidths[i]
		headerText := tb.report.doc.NewRichText(col.Label).WithPadding(headerPadding).WithStyle("th").WithStyle(col.Align)
		headerText.Draw(canvas, currentX, currentY, colWidth, headerHeight)
		currentX += colWidth
	}
	canvas.PopColor()
	currentY += headerHeight

	// Draw table rows
	canvas.PushColor("000000")
	for _, row := range tb.rows {
		currentX = x
		maxRowHeight := Unit(0)

		// First pass: measure row height
		for i, col := range tb.columns {
			colWidth := columnWidths[i]

			// Get cell value
			cellValue := ""
			if val, exists := row[col.ID]; exists {
				cellValue = fmt.Sprintf("%v", val)
			}

			// Measure cell text
			cellText := tb.report.doc.NewRichText(cellValue).WithStyle("td").WithStyle(col.Align).WithPadding(cellPadding)
			_, h, _ := cellText.Measure(colWidth, maxHeight-(currentY-y))
			maxRowHeight = maxRowHeight.Max(h)
		}

		// Second pass: draw cells
		currentX = x
		for i, col := range tb.columns {
			colWidth := columnWidths[i]

			// Get cell value
			cellValue := ""
			if val, exists := row[col.ID]; exists {
				cellValue = fmt.Sprintf("%v", val)
			}

			// Draw cell text
			cellText := tb.report.doc.NewRichText(cellValue).WithStyle("td").WithStyle(col.Align).WithPadding(cellPadding)
			cellText.Draw(canvas, currentX, currentY, colWidth, maxRowHeight)
			currentX += colWidth
		}
		currentY += maxRowHeight

		// Draw row separator line
		canvas.PushColor("DEDEDE")
		canvas.DrawLine(x, currentY, x+maxWidth, currentY)
		canvas.PopColor()
	}
	canvas.PopColor()

	// Draw footer if present
	if len(tb.footer) > 0 {
		for _, footerRow := range tb.footer {
			// Draw footer separator line
			canvas.PushColor("000000")
			canvas.DrawLine(x, currentY, x+maxWidth, currentY)
			canvas.PopColor()

			currentX = x
			maxFooterRowHeight := Unit(0)

			// First pass: measure footer row height
			for i, col := range tb.columns {
				colWidth := columnWidths[i]

				// Get footer cell value
				cellValue := ""
				if val, exists := footerRow[col.ID]; exists {
					cellValue = fmt.Sprintf("%v", val)
				}

				// Measure footer text
				footerText := tb.report.doc.NewRichText(cellValue).WithStyle("td").WithStyle(col.Align).WithPadding(cellPadding)
				_, h, _ := footerText.Measure(colWidth, maxHeight-(currentY-y))
				maxFooterRowHeight = maxFooterRowHeight.Max(h)
			}

			// Second pass: draw footer cells
			currentX = x
			canvas.PushColor("000000")
			for i, col := range tb.columns {
				colWidth := columnWidths[i]

				// Get footer cell value
				cellValue := ""
				if val, exists := footerRow[col.ID]; exists {
					cellValue = fmt.Sprintf("%v", val)
				}

				// Draw footer text
				footerText := tb.report.doc.NewRichText(cellValue).WithStyle("td").WithStyle(col.Align).WithPadding(cellPadding)
				footerText.Draw(canvas, currentX, currentY, colWidth, maxFooterRowHeight)
				currentX += colWidth
			}
			canvas.PopColor()
			currentY += maxFooterRowHeight
		}
	}

	return nil
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
