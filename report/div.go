package report

import (
	"go.bits2life.com/pdf"
)

// ------------------------------------------------------------------------------------------------
// Div
//
// A container block that will draw its children with a given margin and padding.
// ------------------------------------------------------------------------------------------------

// DivBlock represents a container with margin, padding, background, and children
type DivBlock struct {
	Margin     Spacing
	Padding    Spacing
	Background *DivBackground // RGB color for background, nil means no background
	Children   []ContentBlock
	isInline   bool // if true, size to content; if false, use full available width
}

// DivBackground represents the background color for a div
type DivBackground struct {
	Color string // hex color string
}

// Div creates a new div block with the specified properties
func (r *Report) Div(children ...ContentBlock) *DivBlock {
	return &DivBlock{
		Children: children,
		isInline: false, // default to block behavior (full width)
	}
}

func (b *DivBlock) WithBackground(hex string) *DivBlock {
	b.Background = &DivBackground{Color: hex}
	return b
}

func (b *DivBlock) WithMargin(margin ...Unit) *DivBlock {
	b.Margin = pdf.Spacing(margin...)
	return b
}

func (b *DivBlock) WithPadding(padding ...Unit) *DivBlock {
	b.Padding = pdf.Spacing(padding...)
	return b
}

// Inline sets the div to size according to its content (inline behavior)
func (b *DivBlock) Inline() *DivBlock {
	b.isInline = true
	return b
}

// Block sets the div to use full available width (block behavior, default)
func (b *DivBlock) Block() *DivBlock {
	b.isInline = false
	return b
}

func (b *DivBlock) Measure(maxWidth, maxHeight Unit) (usedW, usedH Unit, remain ContentBlock) {
	// Start with full available space
	availableWidth := maxWidth
	availableHeight := maxHeight

	// Apply margin constraints
	availableWidth -= b.Margin.Width()
	availableHeight -= b.Margin.Height()

	// Apply padding constraints
	availableWidth -= b.Padding.Width()
	availableHeight -= b.Padding.Height()

	// Measure children like CompositeBlock
	var totalHeight Unit
	for _, child := range b.Children {
		w, h, r := child.Measure(availableWidth, availableHeight-totalHeight)
		usedW = usedW.Max(w)
		totalHeight += h
		if r != nil {
			// If any child doesn't fit, return what we've used so far plus margins/padding
			usedW += (b.Margin.Width() + b.Padding.Width())
			usedH = totalHeight + (b.Margin.Height() + b.Padding.Height())

			// For block behavior, use full available width; for inline, use content width
			if b.isInline {
				return usedW, usedH, r
			} else {
				return maxWidth, usedH, r
			}
		}
	}

	// Add back margin and padding to final dimensions
	usedW += (b.Margin.Width() + b.Padding.Width())
	usedH = totalHeight + (b.Margin.Height() + b.Padding.Height())

	// For block behavior (default), use full available width
	// For inline behavior, use content width
	if b.isInline {
		return usedW, usedH, nil
	} else {
		return maxWidth, usedH, nil
	}
}

func (b *DivBlock) Draw(canvas pdf.Canvas, x, y, maxWidth, maxHeight Unit) error {
	// Calculate content area after applying margin
	contentX := x
	contentY := y
	contentWidth := maxWidth
	contentHeight := maxHeight

	contentX += b.Margin.Left
	contentY += b.Margin.Top
	contentWidth -= b.Margin.Width()
	contentHeight -= b.Margin.Height()

	// Calculate children area after applying padding
	childrenX := contentX
	childrenY := contentY
	childrenWidth := contentWidth

	childrenX += b.Padding.Left
	childrenY += b.Padding.Top
	childrenWidth -= b.Padding.Width()
	// NOTE: Don't reduce children height by padding - we'll handle this differently

	// First, measure all children to get the total content height
	// Use generous height for measurement to get accurate content size
	var totalContentHeight Unit
	for _, child := range b.Children {
		_, h, _ := child.Measure(childrenWidth, maxHeight) // Use maxHeight, not constrained by padding
		totalContentHeight += h
	}

	// Draw background only for the actual content height (plus padding)
	if b.Background != nil {
		actualBackgroundHeight := totalContentHeight + b.Padding.Height()
		canvas.PushColor(b.Background.Color)
		canvas.FillRect(contentX, contentY, contentWidth, actualBackgroundHeight)
		canvas.PopColor()
	}

	// Now draw the children
	// Give children adequate space to render, but position them within the padded area
	var currentY Unit
	for _, child := range b.Children {
		// Calculate available height: ensure children have enough space to render
		// The available height should be at least the remaining content height, but not exceed maxHeight
		remainingContentHeight := totalContentHeight - currentY
		availableHeight := remainingContentHeight.Max(maxHeight - (childrenY + currentY - y))

		if err := child.Draw(canvas, childrenX, childrenY+currentY, childrenWidth, availableHeight); err != nil {
			return err
		}
		_, h, _ := child.Measure(childrenWidth, availableHeight)
		currentY += h
	}

	return nil
}
