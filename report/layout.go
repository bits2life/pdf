package report

import (
	"fmt"

	"go.bits2life.com/pdf"
)

type ContentBlock interface {
	Measure(maxWidth, maxHeight Unit) (w, h Unit, remain ContentBlock)
	Draw(canvas pdf.Canvas, x, y, maxWidth, maxHeight Unit) error
	// Consider adding KeepTogetherHint() for additional layout control
}

type LayoutFunc func(maxWidth Unit, pageNum int) ContentBlock

type DocumentLayout struct {
	doc            *pdf.Document
	pages          []PagePlan
	contentPadding pdf.BoxSpacing
	headerFunc     LayoutFunc
	footerFunc     LayoutFunc
}

type Measurement struct {
	Block               ContentBlock
	MaxWidth, MaxHeight Unit
	Width, Height       Unit
	Remain              ContentBlock
}

func MeasureBlock(block ContentBlock, maxWidth, maxHeight Unit) Measurement {
	w, h, remain := block.Measure(maxWidth, maxHeight)
	return Measurement{block, maxWidth, maxHeight, w, h, remain}
}

func (m Measurement) Draw(canvas pdf.Canvas, x, y Unit) error {
	return m.Block.Draw(canvas, x, y, m.Width, m.Height)
}

type PagePlan struct {
	header, footer Measurement
	content        []Measurement
	pageNum        int
}

func NewDocumentLayout(doc *pdf.Document, contentPadding pdf.BoxSpacing, headerFunc, footerFunc LayoutFunc) *DocumentLayout {
	return &DocumentLayout{
		doc:            doc,
		contentPadding: contentPadding,
		headerFunc:     headerFunc,
		footerFunc:     footerFunc,
		pages:          []PagePlan{},
	}
}

func (dl *DocumentLayout) LayoutPages(source func() ContentBlock) error {
	pageWidth, pageHeight := dl.doc.PageSize()
	contentWidth := pageWidth - dl.contentPadding.Width()

	var page *PagePlan
	space := Unit(0.0)
	fullPageSpace := Unit(0.0)

	AddPage := func() {
		plan := PagePlan{
			header: MeasureBlock(
				dl.headerFunc(pageWidth, len(dl.pages)+1),
				pageWidth,
				pageHeight,
			),
			footer: MeasureBlock(
				dl.footerFunc(pageWidth, len(dl.pages)+1),
				pageWidth,
				pageHeight,
			),
			pageNum: len(dl.pages) + 1,
			content: []Measurement{},
		}

		dl.pages = append(dl.pages, plan)
		page = &dl.pages[len(dl.pages)-1]
		space = pageHeight - plan.header.Height - plan.footer.Height - dl.contentPadding.Height()
		fullPageSpace = space
	}

	AddPage()

	blk := source()
	for blk != nil {
		measurement := MeasureBlock(blk, contentWidth, space)

		if measurement.Height > space {
			if space == fullPageSpace {
				return fmt.Errorf("Unbreakable (and too tall) content on page %d", page.pageNum)
			} else {
				AddPage()
			}
		} else {
			space -= measurement.Height
			page.content = append(page.content, measurement)
			blk = measurement.Remain
		}

		if blk == nil {
			blk = source()
		}
	}

	return nil
}

func (dl *DocumentLayout) RenderPages() error {
	// Render pages
	for _, plan := range dl.pages {
		err := dl.renderPage(&plan)
		if err != nil {
			return err
		}
	}

	return nil
}

func (dl *DocumentLayout) renderPage(p *PagePlan) error {
	page, err := dl.doc.NewPage()
	if err != nil {
		return err
	}
	canvas := page.Canvas()

	// Draw header
	err = p.header.Draw(canvas, 0, 0)
	if err != nil {
		return err
	}

	// Draw footer
	err = p.footer.Draw(canvas, 0, canvas.Height()-p.footer.Height)
	if err != nil {
		return err
	}

	// Draw page content
	pad := dl.contentPadding
	contentCanvas := canvas.SubCanvas(pad.Left, p.header.Height+pad.Top, canvas.Width()-pad.Width(), canvas.Height()-p.header.Height-p.footer.Height-pad.Height())
	y := Unit(0.0)

	for _, m := range p.content {
		err := m.Draw(contentCanvas, 0, y)
		if err != nil {
			return err
		}
		y += m.Height
	}

	return nil
}

func (dl *DocumentLayout) PageCount() int {
	return len(dl.pages)
}
