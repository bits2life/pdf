package report

import (
	"go.bits2life.com/pdf"
)

type Report struct {
	doc            *pdf.Document
	layout         *DocumentLayout
	contentPadding pdf.BoxSpacing
	headerFunc     func(maxWidth Unit, pageNum int) ContentBlock
	footerFunc     func(maxWidth Unit, pageNum int) ContentBlock
}

func NewReport(format pdf.PageFormat, orientation pdf.PageOrientation) *Report {
	doc := pdf.NewDocument(format, orientation)
	noBlock := func(maxWidth Unit, pageNum int) ContentBlock {
		return EmptyBlock
	}
	return &Report{
		doc:            doc,
		layout:         nil,
		contentPadding: pdf.Padding(0),
		headerFunc:     noBlock,
		footerFunc:     noBlock,
	}
}

func (r *Report) Styles(fn func(r *pdf.Registrar)) {
	r.doc.Styles(fn)
}

func (r *Report) RegisterImageFromBytes(name, imageType string, data []byte) (*pdf.ImageInfo, error) {
	return r.doc.RegisterImageFromBytes(name, imageType, data)
}

func (r *Report) HeaderFunc(headerFunc func(maxWidth Unit, pageNum int) ContentBlock) {
	r.headerFunc = headerFunc
}

func (r *Report) FooterFunc(footerFunc func(maxWidth Unit, pageNum int) ContentBlock) {
	r.footerFunc = footerFunc
}

func (r *Report) WithPadding(padding ...Unit) *Report {
	r.contentPadding = pdf.Padding(padding...)
	return r
}

// Layout plans the pages for the blocks that source returns, until it returns nil. It fails when
// a block cannot be placed even on an empty page.
func (r *Report) Layout(source func() ContentBlock) error {
	r.layout = NewDocumentLayout(
		r.doc,
		r.contentPadding,
		r.headerFunc, r.footerFunc,
	)

	return r.layout.LayoutPages(source)
}

func (r *Report) PageCount() int {
	return r.layout.PageCount()
}

func (r *Report) Render() {
	r.layout.RenderPages()
}

// Save saves the PDF to a file
func (r *Report) Save(filename string) {
	r.doc.WithFilename(filename).Close()
}

func (r *Report) Bytes() ([]byte, error) {
	return r.doc.Bytes()
}

// ------------------------------------------------------------------------------------------------
// SpacerBlock
// ------------------------------------------------------------------------------------------------
type SpacerBlock Unit

func (r *Report) Spacer(height Unit) SpacerBlock {
	return SpacerBlock(height)
}

func (b SpacerBlock) Measure(maxWidth, maxHeight Unit) (usedW, usedH Unit, remain ContentBlock) {
	return Mm(0), maxHeight.Min(Unit(b)), nil
}

func (b SpacerBlock) Draw(canvas pdf.Canvas, x, y, maxWidth, maxHeight Unit) error {
	return nil
}

var EmptyBlock ContentBlock = SpacerBlock(Unit(0))

// ------------------------------------------------------------------------------------------------
// TextBlock
// ------------------------------------------------------------------------------------------------
type TextBlock struct {
	Text *pdf.RichText
}

func (r *Report) Text(text string) ContentBlock {
	return &TextBlock{Text: r.doc.NewRichText(text)}
}

func (b *TextBlock) Measure(maxWidth, maxHeight Unit) (usedW, usedH Unit, remain ContentBlock) {
	w, h, r := b.Text.Measure(Unit(maxWidth), Unit(maxHeight))
	if r != nil {
		return Unit(w), Unit(h), &TextBlock{Text: r}
	}
	return Unit(w), Unit(h), nil
}

func (b *TextBlock) Draw(canvas pdf.Canvas, x, y, maxWidth, maxHeight Unit) error {
	return b.Text.Draw(canvas, Unit(x), Unit(y), Unit(maxWidth), Unit(maxHeight))
}

// ------------------------------------------------------------------------------------------------
// CompositeBlock
// ------------------------------------------------------------------------------------------------
type CompositeBlock struct {
	Blocks []ContentBlock

	// Heights of the blocks that fit in the last Measure, which Draw draws.
	heights []Unit
}

func (r *Report) Composite(blocks ...ContentBlock) ContentBlock {
	return &CompositeBlock{Blocks: blocks}
}

func (b *CompositeBlock) Measure(maxWidth, maxHeight Unit) (usedW, usedH Unit, remain ContentBlock) {
	var totalHeight Unit
	b.heights = b.heights[:0]
	for i, block := range b.Blocks {
		w, h, r := block.Measure(maxWidth, maxHeight-totalHeight)
		if i > 0 && r == nil && h > maxHeight-totalHeight {
			// Doesn't fit below the blocks before it: move it, and the rest, on.
			return usedW, totalHeight, &CompositeBlock{Blocks: b.Blocks[i:]}
		}
		usedW = usedW.Max(w)
		totalHeight += h
		b.heights = append(b.heights, h)
		if r != nil {
			// The rest of this block goes first, followed by the blocks after it.
			if rest := b.Blocks[i+1:]; len(rest) > 0 {
				r = &CompositeBlock{Blocks: append([]ContentBlock{r}, rest...)}
			}
			return usedW, totalHeight, r
		}
	}
	return usedW, totalHeight, nil
}

func (b *CompositeBlock) Draw(canvas pdf.Canvas, x, y, maxWidth, maxHeight Unit) error {
	if b.heights == nil {
		b.Measure(maxWidth, maxHeight)
	}
	var offsetY Unit
	for i, h := range b.heights {
		if err := b.Blocks[i].Draw(canvas, x, y+offsetY, maxWidth, maxHeight-offsetY); err != nil {
			return err
		}
		offsetY += h
	}
	return nil
}

// ------------------------------------------------------------------------------------------------
// KeepTogetherBlock
//
// Groups blocks so they move to the next page when they fit on a fresh page
// but not in the remaining space. If the group is taller than a page, it
// splits like CompositeBlock.
// ------------------------------------------------------------------------------------------------

// KeepTogetherBlock is a CompositeBlock that prefers not to split across pages
// when the whole group fits on one page.
type KeepTogetherBlock struct {
	CompositeBlock
	pageHeight Unit
}

// KeepTogether groups blocks so a heading-plus-first-body pair, a signature,
// or any other run stays on one page when it fits. Nested and adjacent groups
// are supported.
func (r *Report) KeepTogether(blocks ...ContentBlock) ContentBlock {
	return &KeepTogetherBlock{CompositeBlock: CompositeBlock{Blocks: blocks}}
}

func (b *KeepTogetherBlock) KeepTogetherHint() bool { return true }

func (b *KeepTogetherBlock) Measure(maxWidth, maxHeight Unit) (usedW, usedH Unit, remain ContentBlock) {
	usedW, usedH, remain = b.CompositeBlock.Measure(maxWidth, maxHeight)
	if remain == nil {
		return usedW, usedH, nil
	}
	// Does not fit here. Move the whole group if a fresh page would hold it;
	// otherwise split so a group taller than a page cannot loop or vanish.
	if b.pageHeight > 0 && maxHeight < b.pageHeight && b.fits(maxWidth, b.pageHeight) {
		return 0, 0, b
	}
	return usedW, usedH, keepTogetherRemain(remain, b.pageHeight)
}

func (b *KeepTogetherBlock) fits(maxWidth, height Unit) bool {
	probe := KeepTogetherBlock{CompositeBlock: CompositeBlock{Blocks: b.Blocks}}
	_, h, remain := probe.CompositeBlock.Measure(maxWidth, height)
	return remain == nil && h <= height
}

func keepTogetherRemain(remain ContentBlock, pageHeight Unit) ContentBlock {
	switch r := remain.(type) {
	case nil:
		return nil
	case *KeepTogetherBlock:
		r.pageHeight = pageHeight
		return r
	case *CompositeBlock:
		return &KeepTogetherBlock{CompositeBlock: *r, pageHeight: pageHeight}
	default:
		return &KeepTogetherBlock{
			CompositeBlock: CompositeBlock{Blocks: []ContentBlock{r}},
			pageHeight:     pageHeight,
		}
	}
}

// ------------------------------------------------------------------------------------------------
// LeftRightBlock
//
// Combines two blocks into a single block that will draw the two blocks side by side, one
// at the left side and the other at the right side.
// ------------------------------------------------------------------------------------------------
type LeftRightBlock struct {
	left  ContentBlock
	right ContentBlock

	leftMeasurement  Measurement
	rightMeasurement Measurement
}

func (r *Report) LeftRight(left, right ContentBlock) ContentBlock {
	return &LeftRightBlock{left: left, right: right}
}

func (b *LeftRightBlock) Measure(maxWidth, maxHeight Unit) (usedW, usedH Unit, remain ContentBlock) {
	b.leftMeasurement = MeasureBlock(b.left, maxWidth, maxHeight)
	b.rightMeasurement = MeasureBlock(b.right, maxWidth-b.leftMeasurement.Width, maxHeight)

	return maxWidth, b.leftMeasurement.Height.Max(b.rightMeasurement.Height), nil
}

func (b *LeftRightBlock) Draw(canvas pdf.Canvas, x, y, maxWidth, maxHeight Unit) error {
	b.leftMeasurement.Draw(canvas, x, y)
	b.rightMeasurement.Draw(canvas, x+maxWidth-b.rightMeasurement.Width, y)
	return nil
}

// ------------------------------------------------------------------------------------------------
// ImageBlock
//
// Given an image, returns a block that will draw that image with the given width
// ------------------------------------------------------------------------------------------------
type ImageBlock struct {
	image *pdf.ImageInfo
}

func (r *Report) Image(name string, width, height Unit) ContentBlock {
	image, err := r.doc.Image(name)
	if err != nil {
		panic(err)
	}
	if height == pdf.Automatic {
		image.SetWidth(width)
	} else {
		image.SetHeight(height)
	}

	return &ImageBlock{image: image}
}

func (b *ImageBlock) Measure(maxWidth, maxHeight Unit) (usedW, usedH Unit, remain ContentBlock) {
	return b.image.Width(), b.image.Height(), nil
}

func (b *ImageBlock) Draw(canvas pdf.Canvas, x, y, maxWidth, maxHeight Unit) error {
	canvas.DrawImage(b.image, x, y)
	return nil
}

// ------------------------------------------------------------------------------------------------
// DynamicBlock
//
// A block that is generated from a function that is called when the block is measured or
// drawn. This gives us an easy way to create blocks that change between measurement and
// drawing - notably to include total page count in the footer.
// ------------------------------------------------------------------------------------------------
type DynamicBlock struct {
	getBlock func() ContentBlock
}

func (r *Report) Dynamic(getBlock func() ContentBlock) ContentBlock {
	return &DynamicBlock{getBlock: getBlock}
}

func (b *DynamicBlock) Measure(maxWidth, maxHeight Unit) (usedW, usedH Unit, remain ContentBlock) {
	return b.getBlock().Measure(maxWidth, maxHeight)
}

func (b *DynamicBlock) Draw(canvas pdf.Canvas, x, y, maxWidth, maxHeight Unit) error {
	return b.getBlock().Draw(canvas, x, y, maxWidth, maxHeight)
}

// ------------------------------------------------------------------------------------------------
// LineBlock
//
// A block that draws a line.
// ------------------------------------------------------------------------------------------------
type LineBlock struct {
	thickness Unit
	color     string // hex color string
}

func (r *Report) Line(thickness Unit) ContentBlock {
	return &LineBlock{thickness: thickness, color: "000000"} // default to black
}

func (b *LineBlock) WithColor(hex string) *LineBlock {
	b.color = hex
	return b
}

func (b *LineBlock) Measure(maxWidth, maxHeight Unit) (usedW, usedH Unit, remain ContentBlock) {
	return maxWidth, b.thickness, nil
}

func (b *LineBlock) Draw(canvas pdf.Canvas, x, y, maxWidth, maxHeight Unit) error {
	canvas.PushColor(b.color)
	canvas.SetLineWidth(b.thickness)
	canvas.DrawLine(x, y, x+maxWidth, y)
	canvas.PopColor()
	return nil
}
