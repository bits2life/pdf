package pdf

import (
	"image/color"
)

// -----------------------------------------------------------------------------
// Bit-mask definitions
// -----------------------------------------------------------------------------

type styleField uint16

const (
	fieldBlock styleField = 1 << iota
	fieldFont
	fieldSize
	fieldLineHeight
	fieldColor
	fieldBold
	fieldItalic
	fieldUnderline
	fieldAlign
	fieldBreak
)

// -----------------------------------------------------------------------------
// StyleDefinition - used for defining styles with masks
// -----------------------------------------------------------------------------

type StyleDefinition struct {
	mask       styleField // which fields are active
	Block      bool
	Font       string
	Size       Unit
	LineHeight Unit // ignore for inline styles
	Color      color.Color
	Bold       bool
	Italic     bool
	Underline  bool
	Align      Alignment // Left, Right, Center; ignore for inline styles
	Break      BreakMode // Wrap, NoWrap, Ellipsis, Clip; ignore for inline styles
}

// -----------------------------------------------------------------------------
// Style - merged/resolved style without mask (for actual rendering)
// -----------------------------------------------------------------------------

type Style struct {
	Block      bool
	Font       string
	Size       Unit
	LineHeight Unit
	Color      color.Color
	Bold       bool
	Italic     bool
	Underline  bool
	Align      Alignment
	Break      BreakMode

	// To render beneath y-coordinate, add offset before text, advance by lineHeightPt
	// for each line, and end by advancing by lineHeightPt - offset to set cursor just
	// below the last line.
	BaseOffset   Unit // Given top-of-line y-coordinate, add this for baseline
	LineHeightPt Unit // Given top-of-line y-coordinate, add this for height
}

func (s *Style) getVariant() string {
	styleString := ""
	if s.Bold {
		styleString += "B"
	}
	if s.Italic {
		styleString += "I"
	}
	return styleString
}

// -----------------------------------------------------------------------------
// StyleBuilder
// -----------------------------------------------------------------------------

type StyleBuilder struct {
	mask styleField

	Font       string
	Size       Unit
	LineHeight Unit
	Color      color.Color
	Bold       bool
	Italic     bool
	Underline  bool
	Align      Alignment
	Break      BreakMode
}

// Entry-point
func NewStyle() *StyleBuilder { return &StyleBuilder{} }

// ── chainable setters ──
func (b *StyleBuilder) WithFont(f string) *StyleBuilder { b.mask |= fieldFont; b.Font = f; return b }
func (b *StyleBuilder) WithSize(sz Unit) *StyleBuilder  { b.mask |= fieldSize; b.Size = sz; return b }
func (b *StyleBuilder) WithLineHeight(lh Unit) *StyleBuilder {
	b.mask |= fieldLineHeight
	b.LineHeight = lh
	return b
}
func (b *StyleBuilder) WithColor(c color.Color) *StyleBuilder {
	b.mask |= fieldColor
	b.Color = c
	return b
}
func (b *StyleBuilder) WithBold(v bool) *StyleBuilder { b.mask |= fieldBold; b.Bold = v; return b }
func (b *StyleBuilder) WithItalic(v bool) *StyleBuilder {
	b.mask |= fieldItalic
	b.Italic = v
	return b
}
func (b *StyleBuilder) WithUnderline(v bool) *StyleBuilder {
	b.mask |= fieldUnderline
	b.Underline = v
	return b
}
func (b *StyleBuilder) WithAlign(a Alignment) *StyleBuilder {
	b.mask |= fieldAlign
	b.Align = a
	return b
}
func (b *StyleBuilder) WithBreak(br BreakMode) *StyleBuilder {
	b.mask |= fieldBreak
	b.Break = br
	return b
}
func (b *StyleBuilder) AsInline() *StyleDefinition { return b.build(false) }
func (b *StyleBuilder) AsBlock() *StyleDefinition  { return b.build(true) }

func (b *StyleBuilder) build(block bool) *StyleDefinition {
	b.mask |= fieldBlock
	return &StyleDefinition{
		mask:       b.mask,
		Block:      block,
		Font:       b.Font,
		Size:       b.Size,
		LineHeight: b.LineHeight,
		Color:      b.Color,
		Bold:       b.Bold,
		Italic:     b.Italic,
		Underline:  b.Underline,
		Align:      b.Align,
		Break:      b.Break,
	}
}

// ----------------------------------------------------------------------------
// Style registry
// ----------------------------------------------------------------------------

type StyleRegistry struct {
	base   *Style
	styles map[string]*StyleDefinition
	root   *styleNode
	doc    *Document
}

// entry-point
func NewStyleRegistry(base *Style) *StyleRegistry {
	reg := &StyleRegistry{
		base:   base,
		styles: map[string]*StyleDefinition{},
		root: &styleNode{
			parent:   nil,
			merged:   base,
			children: make(map[*StyleDefinition]*styleNode),
		},
	}
	return reg
}

func (reg *StyleRegistry) Define(doc *Document, fn func(r *Registrar)) {
	reg.doc = doc
	tmp := &Registrar{reg: reg}
	fn(tmp)

	for _, e := range tmp.pending {
		if e.isBlock {
			reg.styles[e.name] = e.sb.AsBlock()
		} else {
			reg.styles[e.name] = e.sb.AsInline()
		}

		// Reset base style if "base" is defined.
		if e.name == "base" {
			s := reg.NewStack()
			s.Push("base")
			reg.base = s.Current()
			reg.root.merged = reg.base
			reg.root.children = make(map[*StyleDefinition]*styleNode) // drop old cache
		}
	}
}

func (r *StyleRegistry) Get(name string) *StyleDefinition {
	return r.styles[name]
}

func (r *StyleRegistry) NewStack() *StyleStack {
	return &StyleStack{
		reg:     r,
		current: r.root,
	}
}

/* ---------- style tree / cache ---------- */

type styleNode struct {
	parent   *styleNode // nil for root
	merged   *Style
	children map[*StyleDefinition]*styleNode // nil for inline styles
}

/* ---------- definition helper ---------- */

type Registrar struct {
	reg     *StyleRegistry
	pending []entry
}

type entry struct {
	name    string
	isBlock bool
	sb      *StyleBuilder
}

func (r *Registrar) Inline(name string) *StyleBuilder {
	sb := &StyleBuilder{}
	r.pending = append(r.pending, entry{name, false, sb})
	return sb
}

func (r *Registrar) Block(name string) *StyleBuilder {
	sb := &StyleBuilder{}
	r.pending = append(r.pending, entry{name, true, sb})
	return sb
}

// -----------------------------------------------------------------------------
// StyleStack
// -----------------------------------------------------------------------------

type StyleStack struct {
	reg     *StyleRegistry
	current *styleNode
	setRoot *styleNode // tracks the "root" set by SetRoot(), nil means use original root
}

func (s *StyleStack) Push(name string) {
	style, ok := s.reg.styles[name]
	if !ok {
		panic("unknown style tag: " + name)
	}

	// if we've already merged this style here, follow the edge
	if next, ok := s.current.children[style]; ok {
		s.current = next
		return
	}

	// otherwise do one merge and record it
	merged := s.mergeOnce(s.current.merged, style)
	node := &styleNode{
		parent:   s.current,
		merged:   merged,
		children: make(map[*StyleDefinition]*styleNode),
	}
	s.current.children[style] = node
	s.current = node
}

func (s *StyleStack) Pop() bool {
	// Check if we're at the set root (if any) or the original root
	if s.current.parent == nil || s.current == s.setRoot {
		// already at base or at set root
		return false
	}
	s.current = s.current.parent
	return true
}

func (s *StyleStack) SetRoot() {
	s.setRoot = s.current
}

func (s *StyleStack) Reset() {
	if s.setRoot != nil {
		s.current = s.setRoot
	} else {
		s.current = s.reg.root
	}
}

func (s *StyleStack) Current() *Style {
	style := s.current.merged

	// If BaseOffset is 0 and LineHeightPt is 0, it means this style was never properly calculated
	// This can happen with the base style. Calculate it now if we have a document.
	if style.BaseOffset == 0 && style.LineHeightPt == 0 && s.reg.doc != nil && s.reg.doc.pdf != nil {
		style = &Style{
			Block:      style.Block,
			Font:       style.Font,
			Size:       style.Size,
			LineHeight: style.LineHeight,
			Color:      style.Color,
			Bold:       style.Bold,
			Italic:     style.Italic,
			Underline:  style.Underline,
			Align:      style.Align,
			Break:      style.Break,
		}
		style.BaseOffset, style.LineHeightPt = getOffset(s.reg.doc.pdf, style.Font, style.getVariant(), style.Size, style.LineHeight)

		// Update the cached version
		s.current.merged = style
	}

	return style
}

// -----------------------------------------------------------------------------
// mergeOnce: apply only b's masked fields on top of a
// -----------------------------------------------------------------------------

func (s *StyleStack) mergeOnce(a *Style, b *StyleDefinition) *Style {
	out := *a // shallow‐copy all of a

	if b.mask&fieldBlock != 0 {
		out.Block = b.Block
	}
	if b.mask&fieldFont != 0 {
		out.Font = b.Font
	}
	if b.mask&fieldSize != 0 {
		out.Size = b.Size
	}
	if b.mask&fieldLineHeight != 0 && b.Block { // ignored for inline styles
		out.LineHeight = b.LineHeight
	}
	if b.mask&fieldColor != 0 {
		out.Color = b.Color
	}
	if b.mask&fieldBold != 0 {
		out.Bold = b.Bold
	}
	if b.mask&fieldItalic != 0 {
		out.Italic = b.Italic
	}
	if b.mask&fieldUnderline != 0 {
		out.Underline = b.Underline
	}
	if b.mask&fieldAlign != 0 && b.Block { // ignored for inline styles
		out.Align = b.Align
	}
	if b.mask&fieldBreak != 0 && b.Block { // ignored for inline styles
		out.Break = b.Break
	}

	out.BaseOffset, out.LineHeightPt = getOffset(s.reg.doc.pdf, out.Font, out.getVariant(), out.Size, out.LineHeight)
	return &out
}

// ----------------------------------------------------------------------------
// 6. Default styles
// ----------------------------------------------------------------------------

func DefaultStyle() *Style {
	def := NewStyle().WithFont("Helvetica").WithSize(10).WithLineHeight(1.4).WithColor(color.Black).WithAlign(Left).WithBreak(NoWrap).AsBlock()
	// Convert StyleDefinition to Style by creating a merged style without mask
	return &Style{
		Block:      def.Block,
		Font:       def.Font,
		Size:       def.Size,
		LineHeight: def.LineHeight,
		Color:      def.Color,
		Bold:       def.Bold,
		Italic:     def.Italic,
		Underline:  def.Underline,
		Align:      def.Align,
		Break:      def.Break,
	}
}

func DefaultStyles(doc *Document) *StyleRegistry {
	reg := NewStyleRegistry(DefaultStyle())
	reg.Define(doc, func(r *Registrar) {
		// block styles
		r.Block("p").WithLineHeight(1.4).WithBreak(Wrap)
		r.Block("h1").WithSize(24).WithBold(true).WithLineHeight(1.1)

		// partial block styles
		r.Block("right").WithAlign(Right)
		r.Block("left").WithAlign(Left)
		r.Block("center").WithAlign(Center)
		r.Block("wrap").WithBreak(Wrap)
		r.Block("ellipsis").WithBreak(Ellipsis)
		r.Block("clip").WithBreak(Clip)

		// inline styles
		r.Inline("b").WithBold(true)
		r.Inline("i").WithItalic(true)
		r.Inline("em").WithItalic(true).WithBold(false)
		r.Inline("red").WithColor(color.RGBA{R: 200, A: 255})

		// table styles
		r.Block("td").WithSize(7)
		r.Block("th").WithSize(7).WithBold(true).WithColor(color.White)
		r.Block("footer").WithSize(7).WithAlign(Center)
	})
	return reg
}
