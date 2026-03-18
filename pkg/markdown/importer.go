// Package markdown provides bi-directional conversion between CommonMark text
// and the ProseMirror-compatible JSON AST used by Knomantem.
package markdown

import (
	"bytes"
	"fmt"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// ImportMarkdown parses a CommonMark string and returns a ProseMirror JSON AST
// as a map[string]any with type "doc" at the root.
//
// Supported block types: heading, paragraph, bullet_list, ordered_list,
// code_block, blockquote.
// Supported inline marks: bold, italic, code, link, strike.
func ImportMarkdown(markdown string) (map[string]any, error) {
	md := goldmark.New(
		goldmark.WithExtensions(extension.Strikethrough),
	)

	src := []byte(markdown)
	reader := text.NewReader(src)
	parser := md.Parser()
	doc := parser.Parse(reader)

	conv := &converter{src: src}
	docNode, err := conv.convertDocument(doc)
	if err != nil {
		return nil, fmt.Errorf("markdown import: %w", err)
	}
	return docNode, nil
}

// converter holds source bytes and walks the goldmark AST.
type converter struct {
	src []byte
}

func (c *converter) convertDocument(node ast.Node) (map[string]any, error) {
	children := []any{}
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		block, err := c.convertBlock(child)
		if err != nil {
			return nil, err
		}
		if block != nil {
			children = append(children, block)
		}
	}
	return map[string]any{
		"type":    "doc",
		"content": children,
	}, nil
}

func (c *converter) convertBlock(node ast.Node) (map[string]any, error) {
	switch node.Kind() {
	case ast.KindHeading:
		return c.convertHeading(node.(*ast.Heading))
	case ast.KindParagraph:
		return c.convertParagraph(node.(*ast.Paragraph))
	case ast.KindFencedCodeBlock:
		return c.convertFencedCodeBlock(node.(*ast.FencedCodeBlock))
	case ast.KindCodeBlock:
		return c.convertCodeBlock(node.(*ast.CodeBlock))
	case ast.KindBlockquote:
		return c.convertBlockquote(node.(*ast.Blockquote))
	case ast.KindList:
		return c.convertList(node.(*ast.List))
	case ast.KindHTMLBlock, ast.KindThematicBreak:
		// skip HTML blocks and thematic breaks for now
		return nil, nil
	default:
		// fall back to paragraph for unknown block types
		return c.convertGenericBlock(node)
	}
}

func (c *converter) convertHeading(node *ast.Heading) (map[string]any, error) {
	inline, err := c.convertInlineChildren(node)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"type":    "heading",
		"attrs":   map[string]any{"level": node.Level},
		"content": inline,
	}, nil
}

func (c *converter) convertParagraph(node *ast.Paragraph) (map[string]any, error) {
	inline, err := c.convertInlineChildren(node)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"type":    "paragraph",
		"content": inline,
	}, nil
}

func (c *converter) convertFencedCodeBlock(node *ast.FencedCodeBlock) (map[string]any, error) {
	var buf bytes.Buffer
	for i := 0; i < node.Lines().Len(); i++ {
		line := node.Lines().At(i)
		buf.Write(line.Value(c.src))
	}
	lang := ""
	if node.Info != nil {
		info := string(node.Info.Segment.Value(c.src))
		// Extract just the language identifier (first word)
		for i, ch := range info {
			if ch == ' ' || ch == '\t' || ch == '\n' {
				lang = info[:i]
				break
			}
			lang = info[:i+1]
		}
	}
	code := buf.String()
	// Remove trailing newline added by goldmark
	if len(code) > 0 && code[len(code)-1] == '\n' {
		code = code[:len(code)-1]
	}
	attrs := map[string]any{}
	if lang != "" {
		attrs["language"] = lang
	}
	return map[string]any{
		"type":  "code_block",
		"attrs": attrs,
		"content": []any{
			map[string]any{"type": "text", "text": code},
		},
	}, nil
}

func (c *converter) convertCodeBlock(node *ast.CodeBlock) (map[string]any, error) {
	var buf bytes.Buffer
	for i := 0; i < node.Lines().Len(); i++ {
		line := node.Lines().At(i)
		buf.Write(line.Value(c.src))
	}
	code := buf.String()
	if len(code) > 0 && code[len(code)-1] == '\n' {
		code = code[:len(code)-1]
	}
	return map[string]any{
		"type":  "code_block",
		"attrs": map[string]any{},
		"content": []any{
			map[string]any{"type": "text", "text": code},
		},
	}, nil
}

func (c *converter) convertBlockquote(node *ast.Blockquote) (map[string]any, error) {
	children := []any{}
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		block, err := c.convertBlock(child)
		if err != nil {
			return nil, err
		}
		if block != nil {
			children = append(children, block)
		}
	}
	return map[string]any{
		"type":    "blockquote",
		"content": children,
	}, nil
}

func (c *converter) convertList(node *ast.List) (map[string]any, error) {
	listType := "bullet_list"
	attrs := map[string]any{}
	if node.IsOrdered() {
		listType = "ordered_list"
		attrs["start"] = node.Start
	}

	items := []any{}
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Kind() == ast.KindListItem {
			item, err := c.convertListItem(child.(*ast.ListItem))
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
	}

	result := map[string]any{
		"type":    listType,
		"content": items,
	}
	if len(attrs) > 0 {
		result["attrs"] = attrs
	}
	return result, nil
}

func (c *converter) convertListItem(node *ast.ListItem) (map[string]any, error) {
	children := []any{}
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		block, err := c.convertBlock(child)
		if err != nil {
			return nil, err
		}
		if block != nil {
			children = append(children, block)
		}
	}
	return map[string]any{
		"type":    "list_item",
		"content": children,
	}, nil
}

func (c *converter) convertGenericBlock(node ast.Node) (map[string]any, error) {
	inline, err := c.convertInlineChildren(node)
	if err != nil {
		return nil, err
	}
	if len(inline) == 0 {
		return nil, nil
	}
	return map[string]any{
		"type":    "paragraph",
		"content": inline,
	}, nil
}

// convertInlineChildren walks the direct inline children of a block node and
// returns a slice of ProseMirror text/inline nodes.
func (c *converter) convertInlineChildren(node ast.Node) ([]any, error) {
	nodes := []any{}
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		inline, err := c.convertInline(child, nil)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, inline...)
	}
	return nodes, nil
}

// convertInline converts an inline AST node, inheriting parent marks.
func (c *converter) convertInline(node ast.Node, parentMarks []any) ([]any, error) {
	switch node.Kind() {
	case ast.KindText:
		t := node.(*ast.Text)
		val := string(t.Segment.Value(c.src))
		if t.SoftLineBreak() {
			val += " "
		}
		if t.HardLineBreak() {
			val += "\n"
		}
		return []any{textNode(val, parentMarks)}, nil

	case ast.KindString:
		s := node.(*ast.String)
		return []any{textNode(string(s.Value), parentMarks)}, nil

	case ast.KindCodeSpan:
		marks := appendMark(parentMarks, map[string]any{"type": "code"})
		var buf []byte
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			if t, ok := child.(*ast.Text); ok {
				buf = append(buf, t.Segment.Value(c.src)...)
			} else if s, ok := child.(*ast.String); ok {
				buf = append(buf, s.Value...)
			}
		}
		return []any{textNode(string(buf), marks)}, nil

	case ast.KindEmphasis:
		e := node.(*ast.Emphasis)
		var markType string
		if e.Level == 2 {
			markType = "bold"
		} else {
			markType = "italic"
		}
		marks := appendMark(parentMarks, map[string]any{"type": markType})
		return c.collectInlineChildren(node, marks)

	case extast.KindStrikethrough:
		marks := appendMark(parentMarks, map[string]any{"type": "strike"})
		return c.collectInlineChildren(node, marks)

	case ast.KindLink:
		l := node.(*ast.Link)
		marks := appendMark(parentMarks, map[string]any{
			"type":  "link",
			"attrs": map[string]any{"href": string(l.Destination), "title": string(l.Title)},
		})
		return c.collectInlineChildren(node, marks)

	case ast.KindImage:
		// Represent images as a simple text placeholder for now
		img := node.(*ast.Image)
		altText := ""
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			if t, ok := child.(*ast.Text); ok {
				altText += string(t.Segment.Value(c.src))
			}
		}
		_ = img
		return []any{textNode(altText, parentMarks)}, nil

	case ast.KindRawHTML:
		raw := node.(*ast.RawHTML)
		var buf bytes.Buffer
		for i := 0; i < raw.Segments.Len(); i++ {
			seg := raw.Segments.At(i)
			buf.Write(seg.Value(c.src))
		}
		return []any{textNode(buf.String(), parentMarks)}, nil

	default:
		// recurse for unknown inline nodes
		return c.collectInlineChildren(node, parentMarks)
	}
}

func (c *converter) collectInlineChildren(node ast.Node, marks []any) ([]any, error) {
	result := []any{}
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		nodes, err := c.convertInline(child, marks)
		if err != nil {
			return nil, err
		}
		result = append(result, nodes...)
	}
	return result, nil
}

// textNode creates a ProseMirror text node, optionally with marks.
func textNode(text string, marks []any) map[string]any {
	if text == "" {
		text = " "
	}
	n := map[string]any{"type": "text", "text": text}
	if len(marks) > 0 {
		n["marks"] = marks
	}
	return n
}

// appendMark returns a new slice with the given mark appended.
func appendMark(marks []any, mark map[string]any) []any {
	result := make([]any, len(marks)+1)
	copy(result, marks)
	result[len(marks)] = mark
	return result
}
