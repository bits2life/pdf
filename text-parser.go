package pdf

// TextParser is a parser for text content with simple markup.
//
// The basic markup is simple:
// Text can contain simple tags <xxx>... <yy>foo</yy></xxx>, <br/> and
// textual content. The semantics are almost as simple, each tag
// corresponds to a style, which can be either a block or inline style.
//
// For details on the styles, see @styles.go.
//
// Blocks cannot appear inside other blocks.
//
// When parsing, first tokenize the text into a stream of tokens.
// Then split the stream into blocks, where each block is a sequence
// of tokens that can be rendered together. Text not inside a block
// (including text with inline styles) is assumed to belong to a
// block of the default style (the default value of a style stack with
// nothing push to it).
//
// The content of a block can be trimmed (removing leading and trailing
// whitespace). (Note that this is NOT true for inline styles.)
//'
// A block cannot be empty. If an empty block is encountered, it is
// ignored.
//
// Whitespace in the source text is not considered relevant (any run
// of whitespace, including line feeds, are collapsed into a single
// whitespace token or space).

type TextElement struct {
	Type    ElementType
	Content string
	Tag     string // for open/close tags
}

type ElementType int

const (
	TextNode ElementType = iota
	OpenTag
	CloseTag
	LineBreak
)

type Block struct {
	Tokens []TextElement
}

func ParseText(text string) []Block {
	if text == "" {
		return []Block{}
	}

	tokens := tokenize(text)
	if len(tokens) == 0 {
		return []Block{}
	}

	blocks := splitIntoBlocks(tokens)
	return filterEmptyBlocks(blocks)
}

func tokenize(text string) []TextElement {
	var tokens []TextElement
	i := 0

	for i < len(text) {
		if text[i] == '<' {
			// Parse tag
			tagStart := i
			i++

			// Find end of tag
			for i < len(text) && text[i] != '>' {
				i++
			}

			if i >= len(text) {
				// Malformed tag, treat as text
				tokens = append(tokens, TextElement{
					Type:    TextNode,
					Content: string(text[tagStart]),
				})
				i = tagStart + 1
				continue
			}

			tagContent := text[tagStart+1 : i]
			i++ // skip '>'

			// Check for self-closing tag
			if tagContent == "br/" {
				tokens = append(tokens, TextElement{
					Type: LineBreak,
				})
				continue
			}

			// Check for closing tag
			if len(tagContent) > 0 && tagContent[0] == '/' {
				tokens = append(tokens, TextElement{
					Type: CloseTag,
					Tag:  tagContent[1:],
				})
				continue
			}

			// Opening tag
			tokens = append(tokens, TextElement{
				Type: OpenTag,
				Tag:  tagContent,
			})
		} else {
			// Parse text content
			textStart := i
			for i < len(text) && text[i] != '<' {
				i++
			}

			content := text[textStart:i]
			if content != "" {
				// Collapse whitespace
				content = collapseWhitespace(content)
				if content != "" {
					tokens = append(tokens, TextElement{
						Type:    TextNode,
						Content: content,
					})
				}
			}
		}
	}

	return tokens
}

func collapseWhitespace(text string) string {
	var result []rune
	inWhitespace := false

	for _, r := range text {
		if isWhitespace(r) {
			if !inWhitespace {
				result = append(result, ' ')
				inWhitespace = true
			}
		} else {
			result = append(result, r)
			inWhitespace = false
		}
	}

	return string(result)
}

func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

func splitIntoBlocks(tokens []TextElement) []Block {
	var blocks []Block
	var currentBlock []TextElement

	// TODO: We need the actual document style registry to determine
	// wether style is block or inline.
	reg := DefaultStyles(nil)

	for _, token := range tokens {
		if token.Type == OpenTag {
			style := reg.Get(token.Tag)
			if style != nil && style.Block {
				// This is a block tag, start a new block
				if len(currentBlock) > 0 {
					blocks = append(blocks, Block{Tokens: currentBlock})
					currentBlock = nil
				}
			}
		} else if token.Type == CloseTag {
			style := reg.Get(token.Tag)
			if style != nil && style.Block {
				// End of block tag, finish current block
				currentBlock = append(currentBlock, token)
				if len(currentBlock) > 0 {
					blocks = append(blocks, Block{Tokens: currentBlock})
					currentBlock = nil
				}
				continue
			}
		}

		currentBlock = append(currentBlock, token)
	}

	// Add remaining tokens as final block
	if len(currentBlock) > 0 {
		blocks = append(blocks, Block{Tokens: currentBlock})
	}

	return blocks
}

func filterEmptyBlocks(blocks []Block) []Block {
	var result []Block

	for _, block := range blocks {
		if !isEmptyBlock(block) {
			result = append(result, block)
		}
	}

	return result
}

func isEmptyBlock(block Block) bool {
	hasContent := false

	for _, token := range block.Tokens {
		if token.Type == TextNode {
			// Check if content has non-whitespace characters
			trimmed := collapseWhitespace(token.Content)
			if trimmed != "" && trimmed != " " {
				hasContent = true
				break
			}
		}
	}

	return !hasContent
}
