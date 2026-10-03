package pdf

import (
	"reflect"
	"testing"
)

func TestParseText_PlainText(t *testing.T) {
	text := "Hello world"
	blocks := ParseText(text)

	expected := []Block{
		{
			Tokens: []TextElement{
				{Type: TextNode, Content: "Hello world"},
			},
		},
	}

	if !reflect.DeepEqual(blocks, expected) {
		t.Errorf("ParseText(%q) = %v, want %v", text, blocks, expected)
	}
}

func TestParseText_SimpleInlineTag(t *testing.T) {
	text := "Hello <b>bold</b> world"
	blocks := ParseText(text)

	expected := []Block{
		{
			Tokens: []TextElement{
				{Type: TextNode, Content: "Hello "},
				{Type: OpenTag, Tag: "b"},
				{Type: TextNode, Content: "bold"},
				{Type: CloseTag, Tag: "b"},
				{Type: TextNode, Content: " world"},
			},
		},
	}

	if !reflect.DeepEqual(blocks, expected) {
		t.Errorf("ParseText(%q) = %v, want %v", text, blocks, expected)
	}
}

func TestParseText_BlockTag(t *testing.T) {
	text := "<h1>Title</h1><p>Paragraph text</p>"
	blocks := ParseText(text)

	expected := []Block{
		{
			Tokens: []TextElement{
				{Type: OpenTag, Tag: "h1"},
				{Type: TextNode, Content: "Title"},
				{Type: CloseTag, Tag: "h1"},
			},
		},
		{
			Tokens: []TextElement{
				{Type: OpenTag, Tag: "p"},
				{Type: TextNode, Content: "Paragraph text"},
				{Type: CloseTag, Tag: "p"},
			},
		},
	}

	if !reflect.DeepEqual(blocks, expected) {
		t.Errorf("ParseText(%q) = %v, want %v", text, blocks, expected)
	}
}

func TestParseText_LineBreak(t *testing.T) {
	text := "Line 1<br/>Line 2"
	blocks := ParseText(text)

	expected := []Block{
		{
			Tokens: []TextElement{
				{Type: TextNode, Content: "Line 1"},
				{Type: LineBreak},
				{Type: TextNode, Content: "Line 2"},
			},
		},
	}

	if !reflect.DeepEqual(blocks, expected) {
		t.Errorf("ParseText(%q) = %v, want %v", text, blocks, expected)
	}
}

func TestParseText_NestedInlineTags(t *testing.T) {
	text := "Normal <b>bold <em>bold italic</em> bold</b> normal"
	blocks := ParseText(text)

	expected := []Block{
		{
			Tokens: []TextElement{
				{Type: TextNode, Content: "Normal "},
				{Type: OpenTag, Tag: "b"},
				{Type: TextNode, Content: "bold "},
				{Type: OpenTag, Tag: "em"},
				{Type: TextNode, Content: "bold italic"},
				{Type: CloseTag, Tag: "em"},
				{Type: TextNode, Content: " bold"},
				{Type: CloseTag, Tag: "b"},
				{Type: TextNode, Content: " normal"},
			},
		},
	}

	if !reflect.DeepEqual(blocks, expected) {
		t.Errorf("ParseText(%q) = %v, want %v", text, blocks, expected)
	}
}

func TestParseText_WhitespaceCollapsing(t *testing.T) {
	text := "Hello    \n\t   world"
	blocks := ParseText(text)

	expected := []Block{
		{
			Tokens: []TextElement{
				{Type: TextNode, Content: "Hello world"},
			},
		},
	}

	if !reflect.DeepEqual(blocks, expected) {
		t.Errorf("ParseText(%q) = %v, want %v", text, blocks, expected)
	}
}

func TestParseText_EmptyBlocksIgnored(t *testing.T) {
	text := "<h1></h1><p>Content</p><h2>   </h2>"
	blocks := ParseText(text)

	// Empty blocks should be ignored
	expected := []Block{
		{
			Tokens: []TextElement{
				{Type: OpenTag, Tag: "p"},
				{Type: TextNode, Content: "Content"},
				{Type: CloseTag, Tag: "p"},
			},
		},
	}

	if !reflect.DeepEqual(blocks, expected) {
		t.Errorf("ParseText(%q) = %v, want %v", text, blocks, expected)
	}
}

func TestParseText_MixedBlockAndInline(t *testing.T) {
	text := "<h1>Title with <em>emphasis</em></h1><p>Paragraph with <b>bold</b> text</p>"
	blocks := ParseText(text)

	expected := []Block{
		{
			Tokens: []TextElement{
				{Type: OpenTag, Tag: "h1"},
				{Type: TextNode, Content: "Title with "},
				{Type: OpenTag, Tag: "em"},
				{Type: TextNode, Content: "emphasis"},
				{Type: CloseTag, Tag: "em"},
				{Type: CloseTag, Tag: "h1"},
			},
		},
		{
			Tokens: []TextElement{
				{Type: OpenTag, Tag: "p"},
				{Type: TextNode, Content: "Paragraph with "},
				{Type: OpenTag, Tag: "b"},
				{Type: TextNode, Content: "bold"},
				{Type: CloseTag, Tag: "b"},
				{Type: TextNode, Content: " text"},
				{Type: CloseTag, Tag: "p"},
			},
		},
	}

	if !reflect.DeepEqual(blocks, expected) {
		t.Errorf("ParseText(%q) = %v, want %v", text, blocks, expected)
	}
}

func TestParseText_EmptyString(t *testing.T) {
	text := ""
	blocks := ParseText(text)

	expected := []Block{}

	if !reflect.DeepEqual(blocks, expected) {
		t.Errorf("ParseText(%q) = %v, want %v", text, blocks, expected)
	}
}

func TestParseText_OnlyWhitespace(t *testing.T) {
	text := "   \n\t  "
	blocks := ParseText(text)

	// Should result in empty blocks since whitespace-only content should be ignored
	expected := []Block{}

	if len(blocks) != 0 {
		t.Errorf("ParseText(%q) = %v, want %v", text, blocks, expected)
	}
}
