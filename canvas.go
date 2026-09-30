package pdf

import (
	"github.com/jung-kurt/gofpdf"
)

type Canvas struct {
	pdf        *gofpdf.Fpdf
	page       *Page
	x, y, w, h Unit
}

func (c Canvas) Width() Unit {
	return c.w
}

func (c Canvas) Height() Unit {
	return c.h
}

func (c Canvas) SubCanvas(x, y, w, h Unit) Canvas {
	return Canvas{
		pdf:  c.pdf,
		page: c.page,
		x:    c.x + x,
		y:    c.y + y,
		w:    w,
		h:    h,
	}
}

func (c Canvas) Margins(top, right, bottom, left Unit) Canvas {
	return Canvas{
		pdf:  c.pdf,
		page: c.page,
		x:    c.x + left,
		y:    c.y + top,
		w:    c.w - left - right,
		h:    c.h - top - bottom,
	}
}

func (c Canvas) PushColor(hex string) {
	c.page.doc.PushColor(hex)
}

func (c Canvas) PopColor() {
	c.page.doc.PopColor()
}

func (c Canvas) SetLineWidth(width Unit) {
	c.page.doc.pdf.SetLineWidth(width.Pt())
}

func (c Canvas) DrawLine(x1, y1, x2, y2 Unit) {
	c.page.doc.pdf.Line((x1 + c.x).Pt(), (y1 + c.y).Pt(), (x2 + c.x).Pt(), (y2 + c.y).Pt())
}

func (c Canvas) DrawRect(x1, y1, w, h Unit) {
	c.page.doc.pdf.Rect((x1 + c.x).Pt(), (y1 + c.y).Pt(), w.Pt(), h.Pt(), "D")
}

func (c Canvas) FillRect(x1, y1, w, h Unit) {
	c.page.doc.pdf.Rect((x1 + c.x).Pt(), (y1 + c.y).Pt(), w.Pt(), h.Pt(), "F")
}

func (c Canvas) DrawImage(image *ImageInfo, x, y Unit) {
	c.page.doc.pdf.Image(image.path, (x + c.x).Pt(), (y + c.y).Pt(), image.Width().Pt(), image.Height().Pt(), false, "", 0, "")
}

// ----------------------------------------------------------------------------
// TEXT BASICS
// ----------------------------------------------------------------------------

// SetFont chooses the font family, style and size (in points).
func (c Canvas) SetFont(family, style string, size Unit) {
	c.page.doc.pdf.SetFont(family, style, size.Pt())
}

// TextWidth returns the width of a single line of text.
func (c Canvas) TextWidth(text string) Unit {
	return Unit(c.page.doc.pdf.GetStringWidth(text))
}

// DrawText draws a single line of text at (x, y) relative to this Canvas origin.
func (c Canvas) DrawText(x, y Unit, text string) {
	c.page.doc.pdf.Text((c.x + x).Pt(), (c.y + y).Pt(), text)
}
