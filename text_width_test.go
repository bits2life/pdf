package pdf

import (
	"testing"
)

// Helvetica's glyphs for o/ö, a/å/ä and A/Å/Ä share widths, so a Swedish word
// must measure the same as its ASCII spelling. Measuring the UTF-8 bytes made
// each of these letters count as two glyphs.
func TestStringWidthNonASCII(t *testing.T) {
	doc := NewDocument(A4, Portrait)
	doc.pdf.SetFont("Helvetica", "", 10)

	cases := []struct{ text, ascii string }{
		{"Mötets", "Motets"},
		{"öppnande", "oppnande"},
		{"Mötets öppnande", "Motets oppnande"},
		{"Årsmöte i Älvsjö", "Arsmote i Alvsjo"},
	}
	for _, c := range cases {
		got := stringWidth(doc.pdf, c.text)
		if want := Unit(doc.pdf.GetStringWidth(c.ascii)); got != want {
			t.Errorf("stringWidth(%q) = %v, want %v (width of %q)", c.text, got, want, c.ascii)
		}
		if want := Unit(doc.pdf.GetStringWidth(ConvertToCP1252(c.text))); got != want {
			t.Errorf("stringWidth(%q) = %v, want CP1252 width %v", c.text, got, want)
		}
	}

	// § has no ASCII twin; it must measure as one glyph (556 in Helvetica).
	if got, want := stringWidth(doc.pdf, "§"), Unit(doc.pdf.GetStringWidth("\xa7")); got != want {
		t.Errorf("stringWidth(§) = %v, want %v", got, want)
	}
}

func TestRichTextMeasuresNonASCII(t *testing.T) {
	doc := NewDocument(A4, Portrait)
	page, err := doc.NewPage()
	if err != nil {
		t.Fatal(err)
	}

	measure := func(text string) Unit {
		w, _, _ := doc.NewRichText(text).Measure(Mm(500), Mm(500))
		return w
	}
	if got, want := measure("<b>Mötets</b> öppnande"), measure("<b>Motets</b> oppnande"); got != want {
		t.Errorf("rich text width = %v, want %v", got, want)
	}

	canvas := page.Canvas()
	canvas.SetFont("Helvetica", "B", 12)
	if got, want := canvas.TextWidth("Mötets öppnande"), canvas.TextWidth("Motets oppnande"); got != want {
		t.Errorf("Canvas.TextWidth = %v, want %v", got, want)
	}
}

func TestConvertToCP1252(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Mötets öppnande", "M\xf6tets \xf6ppnande"},
		{"ÅÄÖåäö§€", "\xc5\xc4\xd6\xe5\xe4\xf6\xa7\x80"},
		// Runes outside Windows-1252 are replaced rather than panicking.
		{"Ω → ✓", "? ? ?"},
	}
	for _, c := range cases {
		if got := ConvertToCP1252(c.in); got != c.want {
			t.Errorf("ConvertToCP1252(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
