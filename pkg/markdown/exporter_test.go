package markdown_test

import (
	"strings"
	"testing"

	"github.com/knomantem/knomantem/pkg/markdown"
)

// helper to build a simple doc AST for export tests.
func makeDoc(content ...any) map[string]any {
	return map[string]any{"type": "doc", "content": content}
}

func heading(level int, text string) map[string]any {
	return map[string]any{
		"type":    "heading",
		"attrs":   map[string]any{"level": level},
		"content": []any{textNode(text, nil)},
	}
}

func paragraph(nodes ...any) map[string]any {
	return map[string]any{
		"type":    "paragraph",
		"content": nodes,
	}
}

func textNode(text string, marks []any) map[string]any {
	n := map[string]any{"type": "text", "text": text}
	if len(marks) > 0 {
		n["marks"] = marks
	}
	return n
}

func boldMark() map[string]any { return map[string]any{"type": "bold"} }
func italicMark() map[string]any { return map[string]any{"type": "italic"} }
func codeMark() map[string]any { return map[string]any{"type": "code"} }
func strikeMark() map[string]any { return map[string]any{"type": "strike"} }
func linkMark(href string) map[string]any {
	return map[string]any{
		"type":  "link",
		"attrs": map[string]any{"href": href},
	}
}

func codeBlock(lang, code string) map[string]any {
	attrs := map[string]any{}
	if lang != "" {
		attrs["language"] = lang
	}
	return map[string]any{
		"type":    "code_block",
		"attrs":   attrs,
		"content": []any{textNode(code, nil)},
	}
}

func blockquote(children ...any) map[string]any {
	return map[string]any{"type": "blockquote", "content": children}
}

func bulletList(items ...any) map[string]any {
	return map[string]any{"type": "bullet_list", "content": items}
}

func orderedList(start int, items ...any) map[string]any {
	return map[string]any{
		"type":    "ordered_list",
		"attrs":   map[string]any{"start": start},
		"content": items,
	}
}

func listItem(children ...any) map[string]any {
	return map[string]any{"type": "list_item", "content": children}
}

// ---- Tests ----

func TestExportMarkdown_RequiresDocRoot(t *testing.T) {
	_, err := markdown.ExportMarkdown(map[string]any{"type": "paragraph"})
	if err == nil {
		t.Error("expected error for non-doc root")
	}
}

func TestExportMarkdown_EmptyDoc(t *testing.T) {
	out, err := markdown.ExportMarkdown(makeDoc())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "" {
		t.Errorf("expected empty output for empty doc, got %q", out)
	}
}

func TestExportMarkdown_Heading(t *testing.T) {
	doc := makeDoc(heading(1, "Title"))
	out, err := markdown.ExportMarkdown(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(out, "# Title") {
		t.Errorf("expected output to start with '# Title', got %q", out)
	}
}

func TestExportMarkdown_AllHeadingLevels(t *testing.T) {
	levels := []int{1, 2, 3, 4, 5, 6}
	for _, level := range levels {
		doc := makeDoc(heading(level, "Text"))
		out, err := markdown.ExportMarkdown(doc)
		if err != nil {
			t.Fatalf("level %d: unexpected error: %v", level, err)
		}
		expected := strings.Repeat("#", level) + " Text"
		if !strings.Contains(out, expected) {
			t.Errorf("level %d: expected %q in output, got %q", level, expected, out)
		}
	}
}

func TestExportMarkdown_Paragraph(t *testing.T) {
	doc := makeDoc(paragraph(textNode("Hello, world.", nil)))
	out, err := markdown.ExportMarkdown(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Hello, world.") {
		t.Errorf("expected 'Hello, world.' in output, got %q", out)
	}
}

func TestExportMarkdown_BoldMark(t *testing.T) {
	doc := makeDoc(paragraph(textNode("important", []any{boldMark()})))
	out, err := markdown.ExportMarkdown(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "**important**") {
		t.Errorf("expected bold markup in output, got %q", out)
	}
}

func TestExportMarkdown_ItalicMark(t *testing.T) {
	doc := makeDoc(paragraph(textNode("emphasis", []any{italicMark()})))
	out, err := markdown.ExportMarkdown(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "*emphasis*") {
		t.Errorf("expected italic markup in output, got %q", out)
	}
}

func TestExportMarkdown_CodeMark(t *testing.T) {
	doc := makeDoc(paragraph(textNode("fmt.Println", []any{codeMark()})))
	out, err := markdown.ExportMarkdown(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "`fmt.Println`") {
		t.Errorf("expected code markup in output, got %q", out)
	}
}

func TestExportMarkdown_StrikeMark(t *testing.T) {
	doc := makeDoc(paragraph(textNode("deleted", []any{strikeMark()})))
	out, err := markdown.ExportMarkdown(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "~~deleted~~") {
		t.Errorf("expected strikethrough markup in output, got %q", out)
	}
}

func TestExportMarkdown_LinkMark(t *testing.T) {
	doc := makeDoc(paragraph(textNode("Click here", []any{linkMark("https://example.com")})))
	out, err := markdown.ExportMarkdown(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "[Click here](https://example.com)") {
		t.Errorf("expected link markup in output, got %q", out)
	}
}

func TestExportMarkdown_CodeBlock(t *testing.T) {
	doc := makeDoc(codeBlock("go", "fmt.Println(\"hello\")"))
	out, err := markdown.ExportMarkdown(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "```go") {
		t.Errorf("expected opening code fence with language, got %q", out)
	}
	if !strings.Contains(out, "fmt.Println") {
		t.Errorf("expected code content in output, got %q", out)
	}
	if !strings.Contains(out, "```") {
		t.Errorf("expected closing code fence, got %q", out)
	}
}

func TestExportMarkdown_CodeBlockNoLanguage(t *testing.T) {
	doc := makeDoc(codeBlock("", "plain code"))
	out, err := markdown.ExportMarkdown(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "```\n") {
		t.Errorf("expected code fence without language, got %q", out)
	}
}

func TestExportMarkdown_Blockquote(t *testing.T) {
	doc := makeDoc(blockquote(paragraph(textNode("A quoted line.", nil))))
	out, err := markdown.ExportMarkdown(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "> A quoted line.") {
		t.Errorf("expected blockquote prefix, got %q", out)
	}
}

func TestExportMarkdown_BulletList(t *testing.T) {
	doc := makeDoc(bulletList(
		listItem(paragraph(textNode("item one", nil))),
		listItem(paragraph(textNode("item two", nil))),
	))
	out, err := markdown.ExportMarkdown(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "- item one") {
		t.Errorf("expected bullet item, got %q", out)
	}
	if !strings.Contains(out, "- item two") {
		t.Errorf("expected second bullet item, got %q", out)
	}
}

func TestExportMarkdown_OrderedList(t *testing.T) {
	doc := makeDoc(orderedList(1,
		listItem(paragraph(textNode("first", nil))),
		listItem(paragraph(textNode("second", nil))),
	))
	out, err := markdown.ExportMarkdown(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "1. first") {
		t.Errorf("expected ordered list item, got %q", out)
	}
	if !strings.Contains(out, "2. second") {
		t.Errorf("expected second ordered item, got %q", out)
	}
}

func TestExportMarkdown_RoundTrip(t *testing.T) {
	// Import markdown then export it and check key content is preserved
	input := "# Hello\n\nA paragraph with **bold** text.\n\n- item\n- another\n"
	doc, err := markdown.ImportMarkdown(input)
	if err != nil {
		t.Fatalf("import error: %v", err)
	}
	out, err := markdown.ExportMarkdown(doc)
	if err != nil {
		t.Fatalf("export error: %v", err)
	}
	if !strings.Contains(out, "# Hello") {
		t.Errorf("expected heading in round-trip output, got %q", out)
	}
	if !strings.Contains(out, "**bold**") {
		t.Errorf("expected bold in round-trip output, got %q", out)
	}
	if !strings.Contains(out, "- item") {
		t.Errorf("expected bullet item in round-trip output, got %q", out)
	}
}

func TestExportMarkdown_MultipleBlocks(t *testing.T) {
	doc := makeDoc(
		heading(1, "Title"),
		paragraph(textNode("First paragraph.", nil)),
		heading(2, "Section"),
		paragraph(textNode("Second paragraph.", nil)),
	)
	out, err := markdown.ExportMarkdown(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "# Title") {
		t.Errorf("missing h1")
	}
	if !strings.Contains(out, "## Section") {
		t.Errorf("missing h2")
	}
	if !strings.Contains(out, "First paragraph.") {
		t.Errorf("missing first paragraph")
	}
	if !strings.Contains(out, "Second paragraph.") {
		t.Errorf("missing second paragraph")
	}
}
