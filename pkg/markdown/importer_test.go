package markdown_test

import (
	"testing"

	"github.com/knomantem/knomantem/pkg/markdown"
)

func TestImportMarkdown_Heading(t *testing.T) {
	input := "# Hello World\n"
	doc, err := markdown.ImportMarkdown(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc["type"] != "doc" {
		t.Errorf("root type: got %q, want %q", doc["type"], "doc")
	}
	content, ok := doc["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatal("expected non-empty content")
	}
	heading, ok := content[0].(map[string]any)
	if !ok {
		t.Fatal("expected first content item to be a map")
	}
	if heading["type"] != "heading" {
		t.Errorf("node type: got %q, want %q", heading["type"], "heading")
	}
	attrs, ok := heading["attrs"].(map[string]any)
	if !ok {
		t.Fatal("expected attrs to be a map")
	}
	if attrs["level"] != 1 {
		t.Errorf("heading level: got %v, want 1", attrs["level"])
	}
	hContent, ok := heading["content"].([]any)
	if !ok || len(hContent) == 0 {
		t.Fatal("expected heading content")
	}
	textNode, ok := hContent[0].(map[string]any)
	if !ok {
		t.Fatal("expected text node map")
	}
	if textNode["text"] != "Hello World" {
		t.Errorf("text: got %q, want %q", textNode["text"], "Hello World")
	}
}

func TestImportMarkdown_MultipleHeadingLevels(t *testing.T) {
	input := "# H1\n## H2\n### H3\n"
	doc, err := markdown.ImportMarkdown(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content := doc["content"].([]any)
	if len(content) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(content))
	}
	for i, level := range []int{1, 2, 3} {
		node := content[i].(map[string]any)
		if node["type"] != "heading" {
			t.Errorf("block %d: got type %q, want heading", i, node["type"])
		}
		attrs := node["attrs"].(map[string]any)
		if attrs["level"] != level {
			t.Errorf("block %d: got level %v, want %d", i, attrs["level"], level)
		}
	}
}

func TestImportMarkdown_Paragraph(t *testing.T) {
	input := "This is a simple paragraph.\n"
	doc, err := markdown.ImportMarkdown(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content := doc["content"].([]any)
	if len(content) != 1 {
		t.Fatalf("expected 1 block, got %d", len(content))
	}
	para := content[0].(map[string]any)
	if para["type"] != "paragraph" {
		t.Errorf("type: got %q, want paragraph", para["type"])
	}
}

func TestImportMarkdown_BoldAndItalic(t *testing.T) {
	input := "**bold** and *italic*\n"
	doc, err := markdown.ImportMarkdown(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content := doc["content"].([]any)
	para := content[0].(map[string]any)
	pContent := para["content"].([]any)

	// Find the bold node
	var foundBold, foundItalic bool
	for _, item := range pContent {
		node, ok := item.(map[string]any)
		if !ok {
			continue
		}
		marks, ok := node["marks"].([]any)
		if !ok {
			continue
		}
		for _, m := range marks {
			mark := m.(map[string]any)
			if mark["type"] == "bold" {
				foundBold = true
			}
			if mark["type"] == "italic" {
				foundItalic = true
			}
		}
	}
	if !foundBold {
		t.Error("expected a bold mark")
	}
	if !foundItalic {
		t.Error("expected an italic mark")
	}
}

func TestImportMarkdown_InlineCode(t *testing.T) {
	input := "Use `fmt.Println` here.\n"
	doc, err := markdown.ImportMarkdown(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content := doc["content"].([]any)
	para := content[0].(map[string]any)
	pContent := para["content"].([]any)

	var foundCode bool
	for _, item := range pContent {
		node := item.(map[string]any)
		marks, ok := node["marks"].([]any)
		if !ok {
			continue
		}
		for _, m := range marks {
			mark := m.(map[string]any)
			if mark["type"] == "code" {
				foundCode = true
				if node["text"] != "fmt.Println" {
					t.Errorf("inline code text: got %q, want %q", node["text"], "fmt.Println")
				}
			}
		}
	}
	if !foundCode {
		t.Error("expected an inline code mark")
	}
}

func TestImportMarkdown_Link(t *testing.T) {
	input := "[Google](https://google.com)\n"
	doc, err := markdown.ImportMarkdown(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content := doc["content"].([]any)
	para := content[0].(map[string]any)
	pContent := para["content"].([]any)

	var foundLink bool
	for _, item := range pContent {
		node := item.(map[string]any)
		marks, ok := node["marks"].([]any)
		if !ok {
			continue
		}
		for _, m := range marks {
			mark := m.(map[string]any)
			if mark["type"] == "link" {
				foundLink = true
				attrs := mark["attrs"].(map[string]any)
				if attrs["href"] != "https://google.com" {
					t.Errorf("link href: got %q, want %q", attrs["href"], "https://google.com")
				}
			}
		}
	}
	if !foundLink {
		t.Error("expected a link mark")
	}
}

func TestImportMarkdown_FencedCodeBlock(t *testing.T) {
	input := "```go\nfmt.Println(\"hello\")\n```\n"
	doc, err := markdown.ImportMarkdown(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content := doc["content"].([]any)
	if len(content) != 1 {
		t.Fatalf("expected 1 block, got %d", len(content))
	}
	block := content[0].(map[string]any)
	if block["type"] != "code_block" {
		t.Errorf("type: got %q, want code_block", block["type"])
	}
	attrs := block["attrs"].(map[string]any)
	if attrs["language"] != "go" {
		t.Errorf("language: got %q, want go", attrs["language"])
	}
	blockContent := block["content"].([]any)
	textNode := blockContent[0].(map[string]any)
	if textNode["text"] != "fmt.Println(\"hello\")" {
		t.Errorf("code text: got %q, want fmt.Println(\"hello\")", textNode["text"])
	}
}

func TestImportMarkdown_Blockquote(t *testing.T) {
	input := "> This is a quote\n"
	doc, err := markdown.ImportMarkdown(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content := doc["content"].([]any)
	block := content[0].(map[string]any)
	if block["type"] != "blockquote" {
		t.Errorf("type: got %q, want blockquote", block["type"])
	}
	bContent := block["content"].([]any)
	if len(bContent) == 0 {
		t.Fatal("expected blockquote to have content")
	}
}

func TestImportMarkdown_BulletList(t *testing.T) {
	input := "- item one\n- item two\n- item three\n"
	doc, err := markdown.ImportMarkdown(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content := doc["content"].([]any)
	list := content[0].(map[string]any)
	if list["type"] != "bullet_list" {
		t.Errorf("type: got %q, want bullet_list", list["type"])
	}
	items := list["content"].([]any)
	if len(items) != 3 {
		t.Errorf("expected 3 list items, got %d", len(items))
	}
	for i, item := range items {
		li := item.(map[string]any)
		if li["type"] != "list_item" {
			t.Errorf("item %d: got type %q, want list_item", i, li["type"])
		}
	}
}

func TestImportMarkdown_OrderedList(t *testing.T) {
	input := "1. first\n2. second\n3. third\n"
	doc, err := markdown.ImportMarkdown(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content := doc["content"].([]any)
	list := content[0].(map[string]any)
	if list["type"] != "ordered_list" {
		t.Errorf("type: got %q, want ordered_list", list["type"])
	}
	items := list["content"].([]any)
	if len(items) != 3 {
		t.Errorf("expected 3 list items, got %d", len(items))
	}
}

func TestImportMarkdown_Strikethrough(t *testing.T) {
	input := "~~deleted~~\n"
	doc, err := markdown.ImportMarkdown(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content := doc["content"].([]any)
	para := content[0].(map[string]any)
	pContent := para["content"].([]any)

	var foundStrike bool
	for _, item := range pContent {
		node := item.(map[string]any)
		marks, ok := node["marks"].([]any)
		if !ok {
			continue
		}
		for _, m := range marks {
			mark := m.(map[string]any)
			if mark["type"] == "strike" {
				foundStrike = true
			}
		}
	}
	if !foundStrike {
		t.Error("expected a strike mark")
	}
}

func TestImportMarkdown_EmptyInput(t *testing.T) {
	doc, err := markdown.ImportMarkdown("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc["type"] != "doc" {
		t.Errorf("root type: got %q, want doc", doc["type"])
	}
	content, ok := doc["content"].([]any)
	if !ok {
		t.Fatal("content should be a slice")
	}
	if len(content) != 0 {
		t.Errorf("expected empty content for empty input, got %d blocks", len(content))
	}
}

func TestImportMarkdown_ComplexDocument(t *testing.T) {
	input := `# Document Title

This is a paragraph with **bold** and *italic* and ` + "`code`" + ` text.

## Section

> A blockquote here.

- Bullet one
- Bullet two

1. Ordered one
2. Ordered two

` + "```python\nprint('hello')\n```\n"

	doc, err := markdown.ImportMarkdown(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content := doc["content"].([]any)
	if len(content) == 0 {
		t.Fatal("expected non-empty content for complex document")
	}

	// First block should be heading level 1
	h1 := content[0].(map[string]any)
	if h1["type"] != "heading" {
		t.Errorf("first block: got %q, want heading", h1["type"])
	}
}
