package markdown

import (
	"fmt"
	"strings"
)

// ExportMarkdown converts a ProseMirror JSON AST (map[string]any with type
// "doc" at the root) into a CommonMark string.
//
// Supported node types: doc, heading, paragraph, bullet_list, ordered_list,
// list_item, code_block, blockquote, text (with marks: bold, italic, code,
// link, strike).
func ExportMarkdown(doc map[string]any) (string, error) {
	if doc["type"] != "doc" {
		return "", fmt.Errorf("export: expected root type 'doc', got %q", doc["type"])
	}
	var sb strings.Builder
	if err := exportBlocks(&sb, getContent(doc), 0, ""); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// exportBlocks writes block nodes, handling nesting for blockquote/list.
// prefix is prepended to every line (used for blockquote nesting).
// listCounter is the current ordered list item counter (0 = bullet list).
func exportBlocks(sb *strings.Builder, blocks []any, listDepth int, prefix string) error {
	orderedCounters := make([]int, 0)
	_ = orderedCounters

	for _, raw := range blocks {
		node, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if err := exportBlock(sb, node, listDepth, prefix); err != nil {
			return err
		}
	}
	return nil
}

func exportBlock(sb *strings.Builder, node map[string]any, listDepth int, prefix string) error {
	nodeType, _ := node["type"].(string)
	switch nodeType {
	case "heading":
		return exportHeading(sb, node, prefix)
	case "paragraph":
		return exportParagraph(sb, node, prefix)
	case "code_block":
		return exportCodeBlock(sb, node, prefix)
	case "blockquote":
		return exportBlockquote(sb, node, prefix)
	case "bullet_list":
		return exportBulletList(sb, node, listDepth, prefix)
	case "ordered_list":
		return exportOrderedList(sb, node, listDepth, prefix)
	case "list_item":
		// list_item is handled by list exporters; skip if encountered at top
		return nil
	default:
		// unknown block — try to render as paragraph
		return exportParagraph(sb, node, prefix)
	}
}

func exportHeading(sb *strings.Builder, node map[string]any, prefix string) error {
	level := 1
	if attrs, ok := node["attrs"].(map[string]any); ok {
		if l, ok := attrs["level"].(int); ok {
			level = l
		} else if l, ok := attrs["level"].(float64); ok {
			level = int(l)
		}
	}
	if level < 1 {
		level = 1
	}
	if level > 6 {
		level = 6
	}
	hashes := strings.Repeat("#", level)
	text, err := renderInlineContent(getContent(node))
	if err != nil {
		return err
	}
	sb.WriteString(prefix)
	sb.WriteString(hashes)
	sb.WriteString(" ")
	sb.WriteString(text)
	sb.WriteString("\n\n")
	return nil
}

func exportParagraph(sb *strings.Builder, node map[string]any, prefix string) error {
	text, err := renderInlineContent(getContent(node))
	if err != nil {
		return err
	}
	if text == "" {
		return nil
	}
	// Prefix every line for blockquote support
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		sb.WriteString(prefix)
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	return nil
}

func exportCodeBlock(sb *strings.Builder, node map[string]any, prefix string) error {
	lang := ""
	if attrs, ok := node["attrs"].(map[string]any); ok {
		if l, ok := attrs["language"].(string); ok {
			lang = l
		}
	}
	// Extract raw text — code_block content is typically a single text node
	var codeBuf strings.Builder
	for _, raw := range getContent(node) {
		child, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if child["type"] == "text" {
			if t, ok := child["text"].(string); ok {
				codeBuf.WriteString(t)
			}
		}
	}

	sb.WriteString(prefix)
	sb.WriteString("```")
	sb.WriteString(lang)
	sb.WriteString("\n")
	// Each line of code gets the prefix
	code := codeBuf.String()
	for _, line := range strings.Split(code, "\n") {
		sb.WriteString(prefix)
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	sb.WriteString(prefix)
	sb.WriteString("```\n\n")
	return nil
}

func exportBlockquote(sb *strings.Builder, node map[string]any, prefix string) error {
	// Render child blocks with "> " prefix
	childPrefix := prefix + "> "
	for _, raw := range getContent(node) {
		child, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if err := exportBlock(sb, child, 0, childPrefix); err != nil {
			return err
		}
	}
	return nil
}

func exportBulletList(sb *strings.Builder, node map[string]any, listDepth int, prefix string) error {
	indent := strings.Repeat("  ", listDepth)
	for _, raw := range getContent(node) {
		item, ok := raw.(map[string]any)
		if !ok || item["type"] != "list_item" {
			continue
		}
		if err := exportListItem(sb, item, listDepth, prefix+indent+"- ", prefix+indent+"  "); err != nil {
			return err
		}
	}
	if listDepth == 0 {
		sb.WriteString("\n")
	}
	return nil
}

func exportOrderedList(sb *strings.Builder, node map[string]any, listDepth int, prefix string) error {
	start := 1
	if attrs, ok := node["attrs"].(map[string]any); ok {
		if s, ok := attrs["start"].(int); ok {
			start = s
		} else if s, ok := attrs["start"].(float64); ok {
			start = int(s)
		}
	}
	indent := strings.Repeat("  ", listDepth)
	counter := start
	for _, raw := range getContent(node) {
		item, ok := raw.(map[string]any)
		if !ok || item["type"] != "list_item" {
			continue
		}
		itemPrefix := fmt.Sprintf("%s%s%d. ", prefix, indent, counter)
		contPrefix := prefix + indent + strings.Repeat(" ", len(fmt.Sprintf("%d. ", counter)))
		if err := exportListItem(sb, item, listDepth, itemPrefix, contPrefix); err != nil {
			return err
		}
		counter++
	}
	if listDepth == 0 {
		sb.WriteString("\n")
	}
	return nil
}

// exportListItem renders a list_item node. itemPrefix is used for the first
// paragraph; contPrefix is used for continuation paragraphs and nested lists.
func exportListItem(sb *strings.Builder, item map[string]any, listDepth int, itemPrefix, contPrefix string) error {
	children := getContent(item)
	if len(children) == 0 {
		sb.WriteString(itemPrefix)
		sb.WriteString("\n")
		return nil
	}

	for i, raw := range children {
		child, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		childType, _ := child["type"].(string)
		switch childType {
		case "paragraph":
			text, err := renderInlineContent(getContent(child))
			if err != nil {
				return err
			}
			if i == 0 {
				sb.WriteString(itemPrefix)
				sb.WriteString(text)
				sb.WriteString("\n")
			} else {
				sb.WriteString(contPrefix)
				sb.WriteString(text)
				sb.WriteString("\n")
			}
		case "bullet_list":
			var nested strings.Builder
			if err := exportBulletList(&nested, child, listDepth+1, contPrefix); err != nil {
				return err
			}
			sb.WriteString(nested.String())
		case "ordered_list":
			var nested strings.Builder
			if err := exportOrderedList(&nested, child, listDepth+1, contPrefix); err != nil {
				return err
			}
			sb.WriteString(nested.String())
		default:
			var nested strings.Builder
			if err := exportBlock(&nested, child, listDepth+1, contPrefix); err != nil {
				return err
			}
			sb.WriteString(nested.String())
		}
	}
	return nil
}

// renderInlineContent converts a slice of inline nodes to a plain string with
// CommonMark markup applied for marks (bold, italic, code, link, strike).
func renderInlineContent(nodes []any) (string, error) {
	var sb strings.Builder
	for _, raw := range nodes {
		node, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		nodeType, _ := node["type"].(string)
		switch nodeType {
		case "text":
			text, _ := node["text"].(string)
			marks := getMarks(node)
			rendered, err := applyMarks(text, marks)
			if err != nil {
				return "", err
			}
			sb.WriteString(rendered)
		case "hard_break":
			sb.WriteString("  \n")
		default:
			// Unknown inline node — try to get its text children
			for _, childRaw := range getContent(node) {
				child, ok := childRaw.(map[string]any)
				if !ok {
					continue
				}
				if child["type"] == "text" {
					if t, ok := child["text"].(string); ok {
						sb.WriteString(t)
					}
				}
			}
		}
	}
	return sb.String(), nil
}

// applyMarks wraps text with CommonMark syntax for each mark.
func applyMarks(text string, marks []any) (string, error) {
	result := text
	// Reverse-iterate so innermost marks are applied first,
	// but for ProseMirror flat marks we apply outer-to-inner for correct nesting.
	for _, raw := range marks {
		mark, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		markType, _ := mark["type"].(string)
		switch markType {
		case "bold":
			result = "**" + result + "**"
		case "italic":
			result = "*" + result + "*"
		case "code":
			result = "`" + result + "`"
		case "strike":
			result = "~~" + result + "~~"
		case "link":
			href := ""
			if attrs, ok := mark["attrs"].(map[string]any); ok {
				href, _ = attrs["href"].(string)
			}
			result = "[" + result + "](" + href + ")"
		case "highlight":
			// No CommonMark equivalent; leave as-is
		}
	}
	return result, nil
}

// getContent safely extracts the "content" field as []any.
func getContent(node map[string]any) []any {
	if c, ok := node["content"].([]any); ok {
		return c
	}
	return nil
}

// getMarks safely extracts the "marks" field as []any.
func getMarks(node map[string]any) []any {
	if m, ok := node["marks"].([]any); ok {
		return m
	}
	return nil
}
