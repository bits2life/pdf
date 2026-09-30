package pdf

import (
	"github.com/jung-kurt/gofpdf"
)

type FontMetrics struct {
	Ascent  int
	Descent int
}

var coreFontMetrics = map[string]FontMetrics{
	"Helvetica":    {Ascent: 718, Descent: -207},
	"Helvetica-B":  {Ascent: 718, Descent: -207},
	"Helvetica-I":  {Ascent: 718, Descent: -207},
	"Helvetica-BI": {Ascent: 718, Descent: -207},
	"Times":        {Ascent: 683, Descent: -217},
	"Times-B":      {Ascent: 676, Descent: -205},
	"Times-I":      {Ascent: 683, Descent: -205},
	"Times-BI":     {Ascent: 669, Descent: -205},
	"Courier":      {Ascent: 629, Descent: -157},
	"Courier-B":    {Ascent: 626, Descent: -142},
	"Courier-I":    {Ascent: 629, Descent: -157},
	"Courier-BI":   {Ascent: 626, Descent: -142},
	"Symbol":       {Ascent: 1010, Descent: -293},
	"ZapfDingbats": {Ascent: 820, Descent: -143},
}

func getFontMetrics(pdf *gofpdf.Fpdf, fontName, variant string) FontMetrics {
	desc := pdf.GetFontDesc(fontName, variant)

	if desc.Ascent != 0 || desc.Descent != 0 {
		return FontMetrics{Ascent: desc.Ascent, Descent: desc.Descent}
	}

	fullFontName := fontName
	if variant != "" {
		fullFontName = fontName + "-" + variant
	}

	if metrics, exists := coreFontMetrics[fullFontName]; exists {
		return metrics
	}

	if metrics, exists := coreFontMetrics[fontName]; exists {
		return metrics
	}

	// Default to Helvetica;
	return FontMetrics{Ascent: 718, Descent: -207}
}

func getOffset(pdf *gofpdf.Fpdf, fontName, variant string, size Unit, lineHeight Unit) (Unit, Unit) {
	metrics := getFontMetrics(pdf, fontName, variant)
	baseOffset := ((Unit(metrics.Ascent) / (Unit(metrics.Ascent - metrics.Descent))) + (lineHeight-1.0)/2) * size
	lineHeightPt := size * lineHeight
	return baseOffset, lineHeightPt
}
