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

func TestKeepTogetherFits(t *testing.T) {
	r := NewReport(pdf.A4, pdf.Portrait)
	a := &probeBlock{height: Mm(20)}
	b := &probeBlock{height: Mm(20)}

	if err := layoutWithin(t, r, r.KeepTogether(a, b)); err != nil {
		t.Fatal(err)
	}
	r.Render()

	if r.PageCount() != 1 {
		t.Errorf("PageCount() = %d, want 1", r.PageCount())
	}
	if got := pageContentCounts(r); len(got) != 1 || got[0] != 1 {
		t.Errorf("page content counts = %v, want [1]", got)
	}
	if a.draws != 1 || b.draws != 1 {
		t.Errorf("draws a=%d b=%d, want 1 each", a.draws, b.draws)
	}
}

func TestKeepTogetherMovesToNextPage(t *testing.T) {
	r := NewReport(pdf.A4, pdf.Portrait)
	_, pageHeight := r.doc.PageSize()
	// 30mm remains; the 40mm group fits on a fresh page but not here.
	a := &probeBlock{height: Mm(20)}
	b := &probeBlock{height: Mm(20)}

	if err := layoutWithin(t, r,
		r.Spacer(pageHeight-Mm(30)),
		r.KeepTogether(a, b),
	); err != nil {
		t.Fatal(err)
	}
	r.Render()

	if r.PageCount() != 2 {
		t.Errorf("PageCount() = %d, want 2", r.PageCount())
	}
	if got := pageContentCounts(r); len(got) != 2 || got[0] != 1 || got[1] != 1 {
		t.Errorf("page content counts = %v, want [1 1] (spacer, then the group)", got)
	}
	if a.draws != 1 || b.draws != 1 {
		t.Errorf("draws a=%d b=%d, want 1 each", a.draws, b.draws)
	}
}

func TestKeepTogetherTallerThanPageSplits(t *testing.T) {
	r := NewReport(pdf.A4, pdf.Portrait)
	_, pageHeight := r.doc.PageSize()
	const n, h = 20, 20
	if Unit(n)*Mm(h) <= pageHeight {
		t.Fatalf("test group height %v does not exceed page height %v", Unit(n)*Mm(h), pageHeight)
	}
	probes := make([]ContentBlock, n)
	drawn := make([]*probeBlock, n)
	for i := range probes {
		p := &probeBlock{height: Mm(h)}
		probes[i] = p
		drawn[i] = p
	}

	if err := layoutWithin(t, r, r.KeepTogether(probes...)); err != nil {
		t.Fatal(err)
	}
	r.Render()

	if r.PageCount() < 2 {
		t.Errorf("PageCount() = %d, want at least 2", r.PageCount())
	}
	for i, p := range drawn {
		if p.draws != 1 {
			t.Errorf("probe %d drawn %d times, want 1", i, p.draws)
		}
	}
}

func TestKeepTogetherAdjacent(t *testing.T) {
	r := NewReport(pdf.A4, pdf.Portrait)
	_, pageHeight := r.doc.PageSize()
	firstA, firstB := &probeBlock{height: Mm(20)}, &probeBlock{height: Mm(20)}
	secondA, secondB := &probeBlock{height: Mm(20)}, &probeBlock{height: Mm(20)}

	// 50mm remains: the first 40mm group fits, the second must move.
	if err := layoutWithin(t, r,
		r.Spacer(pageHeight-Mm(50)),
		r.KeepTogether(firstA, firstB),
		r.KeepTogether(secondA, secondB),
	); err != nil {
		t.Fatal(err)
	}
	r.Render()

	if r.PageCount() != 2 {
		t.Errorf("PageCount() = %d, want 2", r.PageCount())
	}
	if got := pageContentCounts(r); len(got) != 2 || got[0] != 2 || got[1] != 1 {
		t.Errorf("page content counts = %v, want [2 1] (spacer+first group, then second group)", got)
	}
	if firstA.draws != 1 || firstB.draws != 1 || secondA.draws != 1 || secondB.draws != 1 {
		t.Errorf("draws first=%d/%d second=%d/%d, want 1 each", firstA.draws, firstB.draws, secondA.draws, secondB.draws)
	}
}

func TestKeepTogetherNested(t *testing.T) {
	r := NewReport(pdf.A4, pdf.Portrait)
	_, pageHeight := r.doc.PageSize()
	a := &probeBlock{height: Mm(20)}
	b := &probeBlock{height: Mm(20)}
	c := &probeBlock{height: Mm(10)}

	if err := layoutWithin(t, r,
		r.Spacer(pageHeight-Mm(30)),
		r.KeepTogether(r.KeepTogether(a, b), c),
	); err != nil {
		t.Fatal(err)
	}
	r.Render()

	if r.PageCount() != 2 {
		t.Errorf("PageCount() = %d, want 2", r.PageCount())
	}
	if got := pageContentCounts(r); len(got) != 2 || got[0] != 1 || got[1] != 1 {
		t.Errorf("page content counts = %v, want [1 1] (spacer, then the nested group)", got)
	}
	if a.draws != 1 || b.draws != 1 || c.draws != 1 {
		t.Errorf("draws a=%d b=%d c=%d, want 1 each", a.draws, b.draws, c.draws)
	}
}

func TestKeepTogetherInsideComposite(t *testing.T) {
	r := NewReport(pdf.A4, pdf.Portrait)
	_, pageHeight := r.doc.PageSize()
	heading := &probeBlock{height: Mm(20)}
	first := &probeBlock{height: Mm(20)}
	after := &probeBlock{height: Mm(5)}

	if err := layoutWithin(t, r,
		r.Spacer(pageHeight-Mm(30)),
		r.Composite(r.KeepTogether(heading, first), after),
	); err != nil {
		t.Fatal(err)
	}
	r.Render()

	if r.PageCount() != 2 {
		t.Errorf("PageCount() = %d, want 2", r.PageCount())
	}
	if got := pageContentCounts(r); len(got) != 2 || got[0] != 1 || got[1] != 1 {
		t.Errorf("page content counts = %v, want [1 1] (spacer, then composite remainder)", got)
	}
	if heading.draws != 1 || first.draws != 1 || after.draws != 1 {
		t.Errorf("draws heading=%d first=%d after=%d, want 1 each", heading.draws, first.draws, after.draws)
	}
}

type hintedGroup struct {
	CompositeBlock
}

func (h *hintedGroup) KeepTogetherHint() bool { return true }

func TestKeepTogetherHintMovesToNextPage(t *testing.T) {
	r := NewReport(pdf.A4, pdf.Portrait)
	_, pageHeight := r.doc.PageSize()
	a := &probeBlock{height: Mm(20)}
	b := &probeBlock{height: Mm(20)}

	if err := layoutWithin(t, r,
		r.Spacer(pageHeight-Mm(30)),
		&hintedGroup{CompositeBlock: CompositeBlock{Blocks: []ContentBlock{a, b}}},
	); err != nil {
		t.Fatal(err)
	}
	r.Render()

	if r.PageCount() != 2 {
		t.Errorf("PageCount() = %d, want 2", r.PageCount())
	}
	if got := pageContentCounts(r); len(got) != 2 || got[0] != 1 || got[1] != 1 {
		t.Errorf("page content counts = %v, want [1 1]", got)
	}
	if a.draws != 1 || b.draws != 1 {
		t.Errorf("draws a=%d b=%d, want 1 each", a.draws, b.draws)
	}
}

func pageContentCounts(r *Report) []int {
	counts := make([]int, len(r.layout.pages))
	for i, p := range r.layout.pages {
		counts[i] = len(p.content)
	}
	return counts
}
