package report

// ------------------------------------------------------------------------------------------------
// Band
//
// A div with negative horizontal margins matching the report's content padding, so its
// background extends to the paper edges beneath the content.
// ------------------------------------------------------------------------------------------------
func (r *Report) Band(content ContentBlock, background string) *DivBlock {
	p := r.contentPadding
	return r.Div(content).
		WithMargin(Mm(0), -p.Right, Mm(0), -p.Left).
		WithPadding(Mm(5), p.Right, Mm(5), p.Left).
		WithBackground(background)
}
