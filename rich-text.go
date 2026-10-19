package pdf

import (
	"math"
	"unicode/utf8"

	"github.com/jung-kurt/gofpdf"
	"golang.org/x/text/encoding/charmap"
)

/**
 * Text rendering is the main value-add of this library. We'll support a simple
 * tag-based approach to text styling in an attempt to make it easier to create
 * components and layouts.
 *
 * For details on styling, see @styles.go. In short, the document has a registry
 * of styles that combine into the text formats we render. There are two types
 * of styles: block and inline. Block styles are applied to the entire block of
 * text, while inline styles are applied to the individual characters.
 *
 * In line with HTML semantics, block styles get their own line(s) and cannot
 * be nested. Inline styles can be nested and will be rendered inline with the
 * text. Unlike HTML, only "simple" tags are supported - no attributes.
 *
 * A <br/> tag will break the current line and start a new one.
 *
 * For now we're fine without support for in-content page breaks. We'll render
 * "small enough" text blocks to just fit them whole on a single page.
 *
 * Note that line-break behavior is part of the styles. Content that is cropped
 * is considered rendered if cropping was configured in the style. So rendering
 * "Enouhg text not to fit in a table cell" in a narrow area with Break Ellipsis
 * will show as (i.e.) "Enough text not to..." but return overflow as false.
 */

// Basic text rendering methods on Canvas:
// func (c Canvas) SetFont(family, style string, size Unit) {
// func (c Canvas) TextWidth(text string) Unit {
// func (c Canvas) DrawText(x, y Unit, text string) {

type RichText struct {
	document      *Document
	styleStack    *StyleStack
	Padding       BoxSpacing
	blocks        []Block
	currentBlock  int
	overflowBlock int

	// Cache measurements for reuse
	measurements []BlockMeasurement
	maxWidth     Unit // Track what width we measured for
}

type BlockMeasurement struct {
	Width    Unit
	Height   Unit
	Overflow bool
}

func (d Document) NewRichText(text string) *RichText {
	blocks := ParseText(text)
	return &RichText{
		document:      &d,
		styleStack:    d.styles.NewStack(),
		blocks:        blocks,
		currentBlock:  0,
		overflowBlock: -1,
		measurements:  make([]BlockMeasurement, len(blocks)),
		maxWidth:      Unit(-1), // Invalid initial value
	}
}

func (rt *RichText) WithStyle(style string) *RichText {
	rt.styleStack.Push(style)
	rt.styleStack.SetRoot()
	return rt
}

func (rt *RichText) WithPadding(p BoxSpacing) *RichText {
	rt.Padding = p
	return rt
}

// Helper methods
func (rt *RichText) HasMore() bool {
	return rt.currentBlock < len(rt.blocks)
}

func (rt *RichText) Reset() {
	rt.currentBlock = 0
	rt.overflowBlock = -1
}

func (rt *RichText) Overflow() bool {
	return rt.overflowBlock != -1
}

// Measure - determines which blocks fit and sets overflow state
func (rt *RichText) Measure(maxWidth, maxHeight Unit) (w, h Unit, overflow *RichText) {
	totalW := Unit(0)
	totalH := Unit(0)
	padW := rt.Padding.Left + rt.Padding.Right
	padH := rt.Padding.Top + rt.Padding.Bottom

	// Reset overflow state
	rt.overflowBlock = -1

	// Re-measure blocks if maxWidth changed
	if rt.maxWidth != maxWidth {
		rt.measureBlocks(maxWidth - padW)
		rt.maxWidth = maxWidth
	}

	for i := 0; i < len(rt.blocks); i++ {
		measurement := rt.measurements[i]

		// Blocks are measured at unlimited height, so an overflowing block is a NoWrap line wider
		// than maxWidth. More space below can't fix that, so it isn't split off to the next page
		// (which would never end): the line takes the full width and is drawn past the edge.
		if totalH+measurement.Height > maxHeight {
			rt.overflowBlock = i

			// When nothing fits, report no size at all (not even padding), so that callers can
			// tell that placing this text makes no progress.
			if i == 0 {
				totalW, totalH, padW, padH = 0, 0, 0, 0
			}

			// Create new RichText instance with remaining content
			if i < len(rt.blocks) {
				remainingBlocks := make([]Block, len(rt.blocks)-i)
				copy(remainingBlocks, rt.blocks[i:])

				overflowRT := &RichText{
					document:      rt.document,
					styleStack:    rt.document.styles.NewStack(), // Fresh style stack for overflow content
					Padding:       rt.Padding,
					blocks:        remainingBlocks,
					currentBlock:  0,
					overflowBlock: -1,
					measurements:  make([]BlockMeasurement, len(remainingBlocks)),
					maxWidth:      Unit(-1), // Will be measured when needed
				}

				return totalW + padW, totalH + padH, overflowRT
			}

			return totalW + padW, totalH + padH, nil
		}

		totalW = totalW.Max(measurement.Width.Min(maxWidth - padW))
		totalH += measurement.Height
	}

	return totalW + rt.Padding.Left + rt.Padding.Right, totalH + rt.Padding.Top + rt.Padding.Bottom, nil
}

// Draw - draws blocks from currentBlock up to overflowBlock (or end)
func (rt *RichText) Draw(c Canvas, x, y, maxWidth, maxHeight Unit) error {
	endBlock := len(rt.blocks)

	// Only draw up to overflow boundary
	if rt.overflowBlock != -1 {
		endBlock = rt.overflowBlock
	}

	// Ensure we have measurements for this maxWidth
	if rt.maxWidth != maxWidth {
		rt.measureBlocks(maxWidth)
		rt.maxWidth = maxWidth
	}

	// Apply padding
	x += rt.Padding.Left
	y += rt.Padding.Top
	cy := y
	maxWidth -= (rt.Padding.Left + rt.Padding.Right)
	maxHeight -= (rt.Padding.Top + rt.Padding.Bottom)

	for i := rt.currentBlock; i < endBlock; i++ {
		rt.document.DrawRichTextBlock(c, rt.blocks[i], x, cy, maxWidth, maxHeight-(cy-y), rt.styleStack)
		cy += rt.measurements[i].Height
	}

	// Advance currentBlock to where we stopped
	rt.currentBlock = endBlock

	return nil
}

// measureBlocks - cache measurements for all blocks at given maxWidth
func (rt *RichText) measureBlocks(maxWidth Unit) {
	for i, block := range rt.blocks {
		w, h, overflow := rt.document.MeasureRichTextBlock(block, maxWidth, Unit(math.Inf(1)), rt.styleStack)
		rt.measurements[i] = BlockMeasurement{
			Width:    w,
			Height:   h,
			Overflow: overflow,
		}
	}
}

// TextLayout represents the laid-out structure of a rich text block
type TextLayout struct {
	Lines       []LayoutLine
	TotalWidth  Unit
	TotalHeight Unit
	Overflow    bool
}

type LayoutLine struct {
	Segments []LayoutSegment
	Width    Unit
	Y        Unit // Y position for this line
}

type LayoutSegment struct {
	Text  string
	Style *Style
	Width Unit
	X     Unit // X position within the line
}

// layoutRichTextBlock performs the shared layout logic for both measuring and drawing
func (d *Document) layoutRichTextBlock(block Block, maxWidth, maxHeight Unit, styleStack *StyleStack) *TextLayout {
	if len(block.Tokens) == 0 {
		return &TextLayout{}
	}

	stack := styleStack

	// Push block style if the first token is a block opening tag
	if block.Tokens[0].Type == OpenTag {
		if style := d.styles.Get(block.Tokens[0].Tag); style != nil && style.Block {
			stack.Push(block.Tokens[0].Tag)
		}
	}

	// Get style properties for line positioning (will be base style or block style)
	style := stack.Current()
	lineHeightPt := style.LineHeightPt
	breakMode := style.Break

	// Reset stack after getting block style
	if block.Tokens[0].Type == OpenTag {
		if style := d.styles.Get(block.Tokens[0].Tag); style != nil && style.Block {
			stack.Pop()
		}
	}

	var lines []LayoutLine
	var currentLine LayoutLine
	var totalWidth Unit
	var totalHeight Unit

	for _, token := range block.Tokens {
		switch token.Type {
		case OpenTag:
			// Push style (both block and inline)
			if style := d.styles.Get(token.Tag); style != nil {
				stack.Push(token.Tag)
			}

		case CloseTag:
			// Pop style (both block and inline)
			if style := d.styles.Get(token.Tag); style != nil {
				stack.Pop()
			}

		case LineBreak:
			// Force line break
			if len(currentLine.Segments) > 0 || currentLine.Width > 0 {
				lines = append(lines, currentLine)
				totalWidth = totalWidth.Max(currentLine.Width)
			}
			currentLine = LayoutLine{}

		case TextNode:
			content := token.Content
			if content == "" {
				continue
			}

			// Get current style for this text
			currentStyle := stack.Current()

			// Handle different break modes
			switch breakMode {
			case Wrap:
				// Word wrapping
				d.layoutTextWithWrapping(content, currentStyle, maxWidth, &currentLine, &lines, &totalWidth)

			case NoWrap:
				// No wrapping, single line
				segment := d.createTextSegment(content, currentStyle)
				d.addSegmentToLine(segment, &currentLine)

			case Ellipsis:
				// Add text with ellipsis if too long
				d.layoutTextWithEllipsis(content, currentStyle, maxWidth, &currentLine)

			case Clip:
				// Clip text to maxWidth
				segment := d.createTextSegment(content, currentStyle)
				if currentLine.Width+segment.Width > maxWidth {
					// Truncate the segment to fit
					availableWidth := maxWidth - currentLine.Width
					segment = d.createTruncatedSegment(content, currentStyle, availableWidth)
				}
				d.addSegmentToLine(segment, &currentLine)
			}
		}
	}

	// Add final line if it has content
	if len(currentLine.Segments) > 0 || currentLine.Width > 0 {
		lines = append(lines, currentLine)
		totalWidth = totalWidth.Max(currentLine.Width)
	}

	// Calculate line positions and total height using consistent lineHeightPt
	currentY := Unit(0)
	for i := range lines {
		lines[i].Y = currentY
		currentY += lineHeightPt
	}

	if len(lines) > 0 {
		totalHeight = Unit(len(lines)) * lineHeightPt
	}

	// Check for overflow
	overflow := totalHeight > maxHeight || (breakMode == NoWrap && totalWidth > maxWidth)

	return &TextLayout{
		Lines:       lines,
		TotalWidth:  totalWidth,
		TotalHeight: totalHeight,
		Overflow:    overflow,
	}
}

// Helper methods for text layout

func (d *Document) createTextSegment(text string, style *Style) LayoutSegment {
	// Set font for measurement
	d.pdf.SetFont(style.Font, style.getVariant(), style.Size.Pt())

	return LayoutSegment{
		Text:  text,
		Style: style,
		Width: stringWidth(d.pdf, text),
	}
}

func (d *Document) createTruncatedSegment(text string, style *Style, maxWidth Unit) LayoutSegment {
	// Set font for measurement
	d.pdf.SetFont(style.Font, style.getVariant(), style.Size.Pt())

	if maxWidth <= Unit(0) {
		return LayoutSegment{Style: style}
	}

	truncated := truncateTextToWidth(text, maxWidth, d.pdf)

	return LayoutSegment{
		Text:  truncated,
		Style: style,
		Width: stringWidth(d.pdf, truncated),
	}
}

func (d *Document) addSegmentToLine(segment LayoutSegment, line *LayoutLine) {
	segment.X = line.Width
	line.Segments = append(line.Segments, segment)
	line.Width += segment.Width
}

func (d *Document) layoutTextWithWrapping(text string, style *Style, maxWidth Unit, currentLine *LayoutLine, lines *[]LayoutLine, totalWidth *Unit) {
	// Set font for measurement
	d.pdf.SetFont(style.Font, style.getVariant(), style.Size.Pt())

	words := splitIntoWords(text)
	spaceWidth := stringWidth(d.pdf, " ")

	for i, word := range words {
		wordSegment := d.createTextSegment(word, style)

		// Add space before word if there's already content on the line and this isn't the first word
		needsSpace := currentLine.Width > 0 || i > 0
		spaceNeeded := Unit(0)
		if needsSpace {
			spaceNeeded = spaceWidth
		}

		// Check if word fits on current line
		if currentLine.Width+spaceNeeded+wordSegment.Width <= maxWidth || currentLine.Width == 0 {
			if needsSpace {
				spaceSegment := LayoutSegment{
					Text:  " ",
					Style: style,
					Width: Unit(spaceWidth),
					X:     currentLine.Width,
				}
				currentLine.Segments = append(currentLine.Segments, spaceSegment)
				currentLine.Width += spaceWidth
			}
			d.addSegmentToLine(wordSegment, currentLine)
		} else {
			// Start new line
			if len(currentLine.Segments) > 0 {
				*lines = append(*lines, *currentLine)
				*totalWidth = totalWidth.Max(currentLine.Width)
			}
			*currentLine = LayoutLine{}
			d.addSegmentToLine(wordSegment, currentLine)
		}
	}
}

func (d *Document) layoutTextWithEllipsis(text string, style *Style, maxWidth Unit, currentLine *LayoutLine) {
	// Set font for measurement
	d.pdf.SetFont(style.Font, style.getVariant(), style.Size.Pt())

	textWidth := stringWidth(d.pdf, text)
	ellipsisWidth := stringWidth(d.pdf, "...")

	if currentLine.Width+textWidth <= maxWidth {
		segment := d.createTextSegment(text, style)
		d.addSegmentToLine(segment, currentLine)
	} else {
		// Truncate text and add ellipsis
		availableWidth := maxWidth - currentLine.Width - ellipsisWidth
		if availableWidth > 0 {
			truncated := truncateTextToWidth(text, availableWidth, d.pdf)
			if truncated != "" {
				truncatedSegment := d.createTextSegment(truncated, style)
				d.addSegmentToLine(truncatedSegment, currentLine)
			}
			ellipsisSegment := d.createTextSegment("...", style)
			d.addSegmentToLine(ellipsisSegment, currentLine)
		} else {
			// No room for ellipsis, just clip to available space
			availableWidth = maxWidth - currentLine.Width
			if availableWidth > 0 {
				truncatedSegment := d.createTruncatedSegment(text, style, availableWidth)
				d.addSegmentToLine(truncatedSegment, currentLine)
			}
		}
	}
}

func (d *Document) MeasureRichTextBlock(block Block, maxWidth, maxHeight Unit, styleStack *StyleStack) (w, h Unit, overflow bool) {
	layout := d.layoutRichTextBlock(block, maxWidth, maxHeight, styleStack)
	return layout.TotalWidth, layout.TotalHeight, layout.Overflow
}

// TODO: We're currently not considering alignment when drawing.
func (d *Document) DrawRichTextBlock(c Canvas, block Block, x, y, maxWidth, maxHeight Unit, styleStack *StyleStack) {
	layout := d.layoutRichTextBlock(block, maxWidth, maxHeight, styleStack)

	for _, line := range layout.Lines {
		lineY := y + line.Y
		align := line.Segments[0].Style.Align
		offsetX := Unit(0)
		if align == Right {
			offsetX = maxWidth - line.Width
		} else if align == Center {
			offsetX = (maxWidth - line.Width).Div(2)
		}

		// Skip lines that are outside the height bounds
		if lineY > y+maxHeight {
			break
		}

		for _, segment := range line.Segments {
			if segment.Text == "" {
				continue
			}

			// Set font and color for this segment
			c.page.doc.pdf.SetFont(segment.Style.Font, segment.Style.getVariant(), segment.Style.Size.Pt())
			if segment.Style.Color != nil {
				r, g, b, _ := segment.Style.Color.RGBA()
				c.page.doc.pdf.SetTextColor(int(r>>8), int(g>>8), int(b>>8))
			}

			// Draw the text segment
			segmentX := x + segment.X
			baselineY := lineY + segment.Style.BaseOffset
			c.page.doc.pdf.Text((c.x + segmentX + offsetX).Pt(), (c.y + baselineY).Pt(), ConvertToCP1252(segment.Text))
		}
	}
}

// ConvertToCP1252 encodes UTF-8 text as Windows-1252, the encoding gofpdf uses
// for the core fonts. Runes that Windows-1252 cannot represent become '?'.
func ConvertToCP1252(text string) string {
	out := make([]byte, 0, len(text))
	for _, r := range text {
		if r < utf8.RuneSelf {
			out = append(out, byte(r))
		} else if b, ok := charmap.Windows1252.EncodeRune(r); ok {
			out = append(out, b)
		} else {
			out = append(out, '?')
		}
	}
	return string(out)
}

// stringWidth measures UTF-8 text in the current font. gofpdf's core fonts sum
// glyph widths per byte, so the text must be measured as the Windows-1252 bytes
// that are actually drawn; measuring the UTF-8 bytes counts each å, ä or ö as
// two glyphs.
func stringWidth(pdf *gofpdf.Fpdf, text string) Unit {
	return Unit(pdf.GetStringWidth(ConvertToCP1252(text)))
}

// Helper function to split text into words
func splitIntoWords(text string) []string {
	if text == "" {
		return []string{}
	}

	var words []string
	var currentWord []rune

	for _, r := range text {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if len(currentWord) > 0 {
				words = append(words, string(currentWord))
				currentWord = nil
			}
		} else {
			currentWord = append(currentWord, r)
		}
	}

	if len(currentWord) > 0 {
		words = append(words, string(currentWord))
	}

	return words
}

// Helper function to truncate text to fit within specified width
func truncateTextToWidth(text string, maxWidth Unit, pdf *gofpdf.Fpdf) string {
	if stringWidth(pdf, text) <= maxWidth {
		return text
	}

	runes := []rune(text)
	for i := len(runes) - 1; i >= 0; i-- {
		truncated := string(runes[:i])
		if stringWidth(pdf, truncated) <= maxWidth {
			return truncated
		}
	}

	return ""
}
