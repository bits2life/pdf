package report

import (
	"strings"
	"testing"
	"time"

	"go.bits2life.com/pdf"
)

// layoutWithin lays out the given blocks, one per source call, and fails the test binary
// outright if layout does not finish in time (a hung layout keeps allocating pages).
func layoutWithin(t *testing.T, r *Report, blocks ...ContentBlock) error {
	t.Helper()
	done := make(chan error, 1)
	go func() {
		done <- r.Layout(func() ContentBlock {
			if len(blocks) == 0 {
				return nil
			}
			blk := blocks[0]
			blocks = blocks[1:]
			return blk
		})
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(10 * time.Second):
		panic("report layout did not finish")
	}
}

// probeBlock is a fixed-height block that counts how often it is drawn.
type probeBlock struct {
	height Unit
	draws  int
}

func (b *probeBlock) Measure(maxWidth, maxHeight Unit) (w, h Unit, remain ContentBlock) {
	return maxWidth, b.height, nil
}

func (b *probeBlock) Draw(canvas pdf.Canvas, x, y, maxWidth, maxHeight Unit) error {
	b.draws++
	return nil
}

func TestLayoutNoWrapTextWiderThanPage(t *testing.T) {
	r := NewReport(pdf.A4, pdf.Portrait)
	r.Styles(func(s *pdf.Registrar) {
		s.Block("base").WithSize(14)
	})
	after := &probeBlock{height: Mm(5)}
	sentence := "This sentence uses the default no-wrap style and is much, much too long to ever fit on a single line of an A4 page in portrait."

	if err := layoutWithin(t, r, r.Composite(r.Text(sentence), after)); err != nil {
		t.Fatal(err)
	}
	r.Render()

	if r.PageCount() != 1 {
		t.Errorf("PageCount() = %d, want 1", r.PageCount())
	}
	if after.draws != 1 {
		t.Errorf("block after the wide text drawn %d times, want 1", after.draws)
	}
}

func TestLayoutParagraphTallerThanRemainingSpace(t *testing.T) {
	r := NewReport(pdf.A4, pdf.Portrait)
	_, pageHeight := r.doc.PageSize()
	paragraph := "<p>" + strings.Repeat("A paragraph that wraps over several lines. ", 20) + "</p>"

	if err := layoutWithin(t, r, r.Spacer(pageHeight-Mm(5)), r.Text(paragraph)); err != nil {
		t.Fatal(err)
	}
	if r.PageCount() != 2 {
		t.Errorf("PageCount() = %d, want 2", r.PageCount())
	}
}

func TestLayoutCompositeKeepsBlocksAfterPageBreak(t *testing.T) {
	r := NewReport(pdf.A4, pdf.Portrait)
	paragraphs := strings.Repeat("<p>"+strings.Repeat("Filler text for the page. ", 30)+"</p>", 40)
	after := &probeBlock{height: Mm(5)}

	if err := layoutWithin(t, r, r.Composite(r.Text(paragraphs), after)); err != nil {
		t.Fatal(err)
	}
	r.Render()

	if r.PageCount() < 2 {
		t.Errorf("PageCount() = %d, want at least 2", r.PageCount())
	}
	if after.draws != 1 {
		t.Errorf("block after the split text drawn %d times, want 1", after.draws)
	}
}

func TestLayoutParagraphTallerThanPageFails(t *testing.T) {
	r := NewReport(pdf.A4, pdf.Portrait).WithPadding(Mm(10))
	paragraph := "<p>" + strings.Repeat("One paragraph that is taller than a whole page. ", 600) + "</p>"

	if err := layoutWithin(t, r, r.Text(paragraph)); err == nil {
		t.Error("Layout() succeeded, want an error for a paragraph taller than a page")
	}
}
