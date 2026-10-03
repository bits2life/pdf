package pdf

import (
	"bytes"
	"fmt"
	"image/color"
	"strconv"

	"github.com/jung-kurt/gofpdf"
)

// parseHexColor converts a hex color string (e.g., "#FF0000" or "FF0000") to RGB values
func parseHexColor(hex string) (r, g, b int, err error) {
	// Remove # prefix if present
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}

	// Handle 3-character hex codes by expanding them
	if len(hex) == 3 {
		hex = string(hex[0]) + string(hex[0]) + string(hex[1]) + string(hex[1]) + string(hex[2]) + string(hex[2])
	}

	if len(hex) != 6 {
		return 0, 0, 0, fmt.Errorf("invalid hex color format: %s", hex)
	}

	// Parse each component
	rVal, err := strconv.ParseUint(hex[0:2], 16, 8)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid red component: %s", hex[0:2])
	}

	gVal, err := strconv.ParseUint(hex[2:4], 16, 8)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid green component: %s", hex[2:4])
	}

	bVal, err := strconv.ParseUint(hex[4:6], 16, 8)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid blue component: %s", hex[4:6])
	}

	return int(rVal), int(gVal), int(bVal), nil
}

// ColorStackEntry represents a color entry in the stack
type ColorStackEntry struct {
	R, G, B int
}

// Document wraps an internal gofpdf.Fpdf instance
type Document struct {
	filename string

	pdf *gofpdf.Fpdf

	format      PageFormat
	orientation PageOrientation
	styles      *StyleRegistry
	colors      map[string]color.Color
	fonts       map[string]string
	images      map[string]*ImageInfo

	// Color stack for push/pop operations
	colorStack []ColorStackEntry

	pageNum int
	tail    *Page
}

type PageOption func(*Page) error

func NewDocument(format PageFormat, orient PageOrientation) *Document {
	doc := &Document{
		filename:    "document.pdf",
		format:      format,
		orientation: orient,
		colors:      make(map[string]color.Color),
		fonts:       make(map[string]string),
		images:      make(map[string]*ImageInfo),
		pageNum:     0,
	}

	doc.pdf = gofpdf.New(doc.orientationType(), "pt", "", "")
	doc.styles = DefaultStyles(doc)

	return doc
}

func (d *Document) WithFilename(filename string) *Document {
	d.filename = filename
	return d
}

func (d *Document) Styles(fn func(r *Registrar)) {
	d.styles.Define(d, fn)
}

func (d *Document) Color(name string, color color.Color) {
	d.colors[name] = color
}

func (d *Document) Font(name string, font string) {
	d.fonts[name] = font
}

type Page struct {
	doc    *Document
	Number int
	Active bool
}

func (d *Document) NewPage(opts ...PageOption) (*Page, error) {
	d.pageNum++
	if d.tail != nil {
		d.tail.Active = false
	}

	if d.pageNum == 1 {
		d.pdf.AddPageFormat(d.orientationType(), d.sizeType())
	} else {
		d.pdf.AddPage()
	}

	page := &Page{
		doc:    d,
		Number: d.pageNum,
		Active: true,
	}

	d.tail = page
	return page, nil
}

func (d *Document) sizeType() gofpdf.SizeType {
	switch d.format {
	case A4:
		return gofpdf.SizeType{Wd: Mm(210).Pt(), Ht: Mm(297).Pt()}
	case Letter:
		return gofpdf.SizeType{Wd: Mm(215.9).Pt(), Ht: Mm(279.4).Pt()}
	default:
		return gofpdf.SizeType{Wd: Mm(210).Pt(), Ht: Mm(297).Pt()}
	}
}

func (d *Document) PageSize() (w, h Unit) {
	x := d.sizeType()
	return Unit(x.Wd), Unit(x.Ht)
}

func (d *Document) orientationType() string {
	switch d.orientation {
	case Portrait:
		return "P"
	case Landscape:
		return "L"
	}
	return "P"
}

func (p *Page) Canvas() Canvas {
	w, h := p.doc.pdf.GetPageSize()

	return Canvas{
		pdf:  p.doc.pdf,
		page: p,
		x:    0,
		y:    0,
		w:    Unit(w),
		h:    Unit(h),
	}
}

func (d *Document) Close() {
	d.pdf.OutputFileAndClose(d.filename)
}

// Stream returns the PDF content as a byte slice
func (d *Document) Bytes() ([]byte, error) {
	var buf bytes.Buffer
	err := d.pdf.Output(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// PushColor pushes a hex color onto the stack and sets it as current draw and fill color
func (d *Document) PushColor(hex string) {
	r, g, b, err := parseHexColor(hex)
	if err != nil {
		// Default to black if parsing fails
		r, g, b = 0, 0, 0
	}

	entry := ColorStackEntry{R: r, G: g, B: b}
	d.colorStack = append(d.colorStack, entry)
	d.pdf.SetDrawColor(r, g, b)
	d.pdf.SetFillColor(r, g, b)
}

// PopColor pops the most recent color from the stack and sets the previous color
// If the stack becomes empty, it sets color to black (0, 0, 0)
func (d *Document) PopColor() {
	if len(d.colorStack) == 0 {
		return // Nothing to pop
	}

	// Remove the current color
	d.colorStack = d.colorStack[:len(d.colorStack)-1]

	// Set to previous color or default black if stack is empty
	if len(d.colorStack) > 0 {
		prev := d.colorStack[len(d.colorStack)-1]
		d.pdf.SetDrawColor(prev.R, prev.G, prev.B)
		d.pdf.SetFillColor(prev.R, prev.G, prev.B)
	} else {
		d.pdf.SetDrawColor(0, 0, 0)
		d.pdf.SetFillColor(0, 0, 0)
	}
}

// PushNamedColor pushes a named color from the colors map onto the stack
func (d *Document) PushNamedColor(name string) {
	namedColor, exists := d.colors[name]
	if !exists {
		return // Color doesn't exist, do nothing
	}

	// Convert color.Color to RGB values
	r, g, b, _ := namedColor.RGBA()
	// RGBA returns 16-bit values, convert to 8-bit
	r8 := int(r >> 8)
	g8 := int(g >> 8)
	b8 := int(b >> 8)

	// Convert RGB to hex string
	hex := fmt.Sprintf("%02X%02X%02X", r8, g8, b8)
	d.PushColor(hex)
}
