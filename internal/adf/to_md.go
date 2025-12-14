package adf

import (
	"fmt"
	"strings"
)

// ToMarkdown converts an Atlassian Document Format document to extended markdown.
// This is the main entry point for ADF to Markdown conversion.
//
// Extended markdown format includes:
//   - ~~~panel type=info for panels
//   - ~~~expand title="..." for expandable sections
//   - @[Name](accountId:xxx) for mentions
//   - {date:YYYY-MM-DD} for dates
//   - {status:TEXT|color=blue} for status badges
//   - {card:url} for inline cards
//   - :shortcode: for emojis
//   - - [x] / - [ ] for task lists
func ToMarkdown(doc map[string]any) string {
	return renderADFDocument(doc)
}

// renderADFDocument converts an ADF document to extended markdown.
func renderADFDocument(doc map[string]any) string {
	content, ok := doc["content"].([]any)
	if !ok {
		return ""
	}

	var parts []string
	for _, node := range content {
		nodeMap, ok := node.(map[string]any)
		if !ok {
			continue
		}
		rendered := renderADFNode(nodeMap, 0)
		if rendered != "" {
			parts = append(parts, rendered)
		}
	}

	return NormalizeWhitespace(strings.Join(parts, "\n\n"))
}

// renderADFNode converts a single ADF node to markdown.
func renderADFNode(node map[string]any, depth int) string {
	nodeType, _ := node["type"].(string)

	switch nodeType {
	case "paragraph":
		return renderParagraph(node)
	case "text":
		return renderText(node)
	case "hardBreak":
		return "  \n" // Two spaces + newline for hard break
	case "heading":
		return renderHeading(node)
	case "bulletList":
		return renderBulletList(node, depth)
	case "orderedList":
		return renderOrderedList(node, depth)
	case "taskList":
		return renderTaskList(node, depth)
	case "listItem":
		return renderListItemContent(node, depth)
	case "taskItem":
		return renderTaskItem(node, depth)
	case "codeBlock":
		return renderCodeBlock(node)
	case "blockquote":
		return renderBlockquote(node)
	case "rule":
		return "---"
	case "panel":
		return renderPanel(node)
	case "expand", "nestedExpand":
		return renderExpand(node)
	case "table":
		return renderTable(node)
	case "mediaSingle":
		return renderMediaSingle(node)
	case "mediaGroup":
		return renderMediaGroup(node)
	case "media":
		return renderMedia(node)
	case "emoji":
		return renderEmoji(node)
	case "mention":
		return renderMention(node)
	case "status":
		return renderStatus(node)
	case "date":
		return renderDate(node)
	case "inlineCard":
		return renderInlineCard(node)
	default:
		// Fallback: try to render content
		return renderContent(node)
	}
}

// renderParagraph renders a paragraph node.
func renderParagraph(node map[string]any) string {
	text := renderContent(node)

	// Check for custom attributes
	if attrs, ok := node["attrs"].(map[string]any); ok && len(attrs) > 0 {
		attrStr := FormatAttrsForFence(attrs, "textAlign")
		if attrStr != "" {
			return fmt.Sprintf("<!-- adf:paragraph%s -->\n%s", attrStr, text)
		}
	}

	return text
}

// renderHeading renders a heading node.
func renderHeading(node map[string]any) string {
	level := 1
	if attrs, ok := node["attrs"].(map[string]any); ok {
		if l, ok := attrs["level"].(float64); ok {
			level = int(l)
		}
	}

	// Clamp level to 1-6
	if level < 1 {
		level = 1
	} else if level > 6 {
		level = 6
	}

	prefix := strings.Repeat("#", level)
	text := renderContent(node)

	// Check for custom attributes (id, textAlign)
	if attrs, ok := node["attrs"].(map[string]any); ok {
		customAttrs := make(map[string]any)
		for k, v := range attrs {
			if k != "level" {
				customAttrs[k] = v
			}
		}
		if len(customAttrs) > 0 {
			attrStr := FormatAttrsForFence(customAttrs, "id", "textAlign")
			if attrStr != "" {
				return fmt.Sprintf("<!-- adf:heading%s -->\n%s %s", attrStr, prefix, text)
			}
		}
	}

	return fmt.Sprintf("%s %s", prefix, text)
}

// renderBulletList renders a bullet list.
func renderBulletList(node map[string]any, depth int) string {
	content, ok := node["content"].([]any)
	if !ok {
		return ""
	}

	var lines []string
	indent := strings.Repeat("  ", depth)

	for _, item := range content {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		itemContent := renderListItemContent(itemMap, depth)
		lines = append(lines, fmt.Sprintf("%s- %s", indent, itemContent))
	}

	return strings.Join(lines, "\n")
}

// renderOrderedList renders an ordered list.
func renderOrderedList(node map[string]any, depth int) string {
	content, ok := node["content"].([]any)
	if !ok {
		return ""
	}

	startOrder := 1
	if attrs, ok := node["attrs"].(map[string]any); ok {
		if order, ok := attrs["order"].(float64); ok {
			startOrder = int(order)
		}
	}

	var lines []string
	indent := strings.Repeat("  ", depth)

	for i, item := range content {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		itemContent := renderListItemContent(itemMap, depth)
		lines = append(lines, fmt.Sprintf("%s%d. %s", indent, startOrder+i, itemContent))
	}

	return strings.Join(lines, "\n")
}

// renderTaskList renders a task list.
func renderTaskList(node map[string]any, depth int) string {
	content, ok := node["content"].([]any)
	if !ok {
		return ""
	}

	var lines []string
	indent := strings.Repeat("  ", depth)

	for _, item := range content {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}

		// Get task state
		state := "TODO"
		if attrs, ok := itemMap["attrs"].(map[string]any); ok {
			if s, ok := attrs["state"].(string); ok {
				state = s
			}
		}

		checkbox := "[ ]"
		if state == "DONE" {
			checkbox = "[x]"
		}

		itemContent := renderContent(itemMap)
		lines = append(lines, fmt.Sprintf("%s- %s %s", indent, checkbox, itemContent))
	}

	return strings.Join(lines, "\n")
}

// renderTaskItem renders a single task item (used when iterating).
func renderTaskItem(node map[string]any, depth int) string {
	state := "TODO"
	if attrs, ok := node["attrs"].(map[string]any); ok {
		if s, ok := attrs["state"].(string); ok {
			state = s
		}
	}

	checkbox := "[ ]"
	if state == "DONE" {
		checkbox = "[x]"
	}

	return fmt.Sprintf("%s %s", checkbox, renderContent(node))
}

// renderListItemContent renders the content of a list item, handling nested lists.
func renderListItemContent(node map[string]any, depth int) string {
	content, ok := node["content"].([]any)
	if !ok {
		return ""
	}

	var parts []string
	var nestedLists []string

	for i, child := range content {
		childMap, ok := child.(map[string]any)
		if !ok {
			continue
		}

		childType, _ := childMap["type"].(string)

		switch childType {
		case "paragraph":
			text := renderContent(childMap)
			if i == 0 {
				parts = append(parts, text)
			} else {
				// Additional paragraphs in list item
				parts = append(parts, "\n"+strings.Repeat("  ", depth+1)+text)
			}
		case "bulletList":
			nestedLists = append(nestedLists, renderBulletList(childMap, depth+1))
		case "orderedList":
			nestedLists = append(nestedLists, renderOrderedList(childMap, depth+1))
		case "taskList":
			nestedLists = append(nestedLists, renderTaskList(childMap, depth+1))
		default:
			parts = append(parts, renderADFNode(childMap, depth+1))
		}
	}

	result := strings.Join(parts, "")
	if len(nestedLists) > 0 {
		result += "\n" + strings.Join(nestedLists, "\n")
	}

	return strings.TrimRight(result, "\n")
}

// renderCodeBlock renders a code block.
func renderCodeBlock(node map[string]any) string {
	lang := ""
	if attrs, ok := node["attrs"].(map[string]any); ok {
		lang, _ = attrs["language"].(string)
	}

	content := renderContent(node)
	return fmt.Sprintf("```%s\n%s\n```", lang, content)
}

// renderBlockquote renders a blockquote.
func renderBlockquote(node map[string]any) string {
	content := renderContent(node)
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")

	var quoted []string
	for _, line := range lines {
		quoted = append(quoted, "> "+line)
	}

	return strings.Join(quoted, "\n")
}

// renderPanel renders a panel using extended fence syntax.
func renderPanel(node map[string]any) string {
	panelType := "info"
	if attrs, ok := node["attrs"].(map[string]any); ok {
		if pt, ok := attrs["panelType"].(string); ok {
			panelType = pt
		}
	}

	content := strings.TrimSpace(renderBlockContent(node))

	return fmt.Sprintf("~~~panel type=%s\n%s\n~~~", panelType, content)
}

// renderExpand renders an expand/nestedExpand using extended fence syntax.
func renderExpand(node map[string]any) string {
	title := ""
	if attrs, ok := node["attrs"].(map[string]any); ok {
		title, _ = attrs["title"].(string)
	}

	content := strings.TrimSpace(renderBlockContent(node))

	if title != "" {
		return fmt.Sprintf("~~~expand title=\"%s\"\n%s\n~~~", title, content)
	}
	return fmt.Sprintf("~~~expand\n%s\n~~~", content)
}

// renderTable renders a table.
func renderTable(node map[string]any) string {
	content, ok := node["content"].([]any)
	if !ok || len(content) == 0 {
		return ""
	}

	var rows [][]string
	var isHeaderRow []bool

	// Extract all rows and cells
	for _, row := range content {
		rowMap, ok := row.(map[string]any)
		if !ok {
			continue
		}
		rowContent, ok := rowMap["content"].([]any)
		if !ok {
			continue
		}

		var cells []string
		hasHeader := false
		for _, cell := range rowContent {
			cellMap, ok := cell.(map[string]any)
			if !ok {
				continue
			}
			cellType, _ := cellMap["type"].(string)
			if cellType == "tableHeader" {
				hasHeader = true
			}
			cellText := strings.TrimSpace(renderContent(cellMap))
			cellText = strings.ReplaceAll(cellText, "\n", " ")
			cellText = strings.ReplaceAll(cellText, "|", "\\|")
			cells = append(cells, cellText)
		}
		rows = append(rows, cells)
		isHeaderRow = append(isHeaderRow, hasHeader)
	}

	if len(rows) == 0 {
		return ""
	}

	// Calculate column count
	colCount := 0
	for _, row := range rows {
		if len(row) > colCount {
			colCount = len(row)
		}
	}

	var sb strings.Builder

	for i, row := range rows {
		// Pad row to have consistent column count
		for len(row) < colCount {
			row = append(row, "")
		}

		sb.WriteString("| " + strings.Join(row, " | ") + " |\n")

		// Add separator after header row
		if isHeaderRow[i] {
			var sep []string
			for range row {
				sep = append(sep, "---")
			}
			sb.WriteString("| " + strings.Join(sep, " | ") + " |\n")
		}
	}

	// If first row wasn't a header, add separator after it
	if len(isHeaderRow) > 0 && !isHeaderRow[0] {
		lines := strings.Split(strings.TrimRight(sb.String(), "\n"), "\n")
		if len(lines) > 0 {
			var sep []string
			for i := 0; i < colCount; i++ {
				sep = append(sep, "---")
			}
			separator := "| " + strings.Join(sep, " | ") + " |"

			sb.Reset()
			sb.WriteString(lines[0] + "\n")
			sb.WriteString(separator + "\n")
			for _, line := range lines[1:] {
				sb.WriteString(line + "\n")
			}
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}

// renderMediaSingle renders a mediaSingle with extended syntax.
func renderMediaSingle(node map[string]any) string {
	layout := ""
	width := ""
	widthType := ""
	if attrs, ok := node["attrs"].(map[string]any); ok {
		layout, _ = attrs["layout"].(string)
		if w, ok := attrs["width"].(float64); ok {
			width = fmt.Sprintf("%.0f", w)
		}
		widthType, _ = attrs["widthType"].(string)
	}

	mediaContent := renderContent(node)

	// Build attributes string
	var attrParts []string
	if layout != "" {
		attrParts = append(attrParts, fmt.Sprintf("layout=%s", layout))
	}
	if width != "" {
		attrParts = append(attrParts, fmt.Sprintf("width=%s", width))
		if widthType != "" {
			attrParts = append(attrParts, fmt.Sprintf("widthType=%s", widthType))
		}
	}

	attrStr := ""
	if len(attrParts) > 0 {
		attrStr = " " + strings.Join(attrParts, " ")
	}

	return fmt.Sprintf("~~~mediaSingle%s\n%s\n~~~", attrStr, strings.TrimSpace(mediaContent))
}

// renderMediaGroup renders a mediaGroup with extended syntax.
func renderMediaGroup(node map[string]any) string {
	mediaContent := renderContent(node)
	return fmt.Sprintf("~~~mediaGroup\n%s\n~~~", strings.TrimSpace(mediaContent))
}

// renderMedia renders a media node as markdown image.
func renderMedia(node map[string]any) string {
	attrs, ok := node["attrs"].(map[string]any)
	if !ok {
		return "[attachment]"
	}

	id, _ := attrs["id"].(string)
	collection, _ := attrs["collection"].(string)
	mediaType, _ := attrs["type"].(string)
	alt, _ := attrs["alt"].(string)

	if alt == "" {
		alt = "attachment"
	}

	if id != "" {
		// Full jira-media format for roundtrip support
		return fmt.Sprintf("![%s](jira-media:%s:%s:%s)", alt, id, collection, mediaType)
	}

	return fmt.Sprintf("[%s]", alt)
}

// renderEmoji renders an emoji node.
func renderEmoji(node map[string]any) string {
	attrs, ok := node["attrs"].(map[string]any)
	if !ok {
		return ""
	}

	// Prefer shortcode for roundtrip fidelity
	if shortName, ok := attrs["shortName"].(string); ok && shortName != "" {
		// shortName is already in :name: format from Jira
		if strings.HasPrefix(shortName, ":") && strings.HasSuffix(shortName, ":") {
			return shortName
		}
		return fmt.Sprintf(":%s:", shortName)
	}

	// Fall back to text (unicode)
	if text, ok := attrs["text"].(string); ok && text != "" {
		return text
	}

	return ""
}

// renderMention renders a mention with display name for readability.
// Format: @[DisplayName](accountId:xxx)
func renderMention(node map[string]any) string {
	attrs, ok := node["attrs"].(map[string]any)
	if !ok {
		return "@unknown"
	}

	id, _ := attrs["id"].(string)
	text, _ := attrs["text"].(string)

	// Clean up display name (remove leading @)
	displayName := strings.TrimPrefix(text, "@")
	if displayName == "" {
		displayName = id
	}

	if id != "" {
		return fmt.Sprintf("@[%s](accountId:%s)", displayName, id)
	}

	if text != "" {
		return text
	}

	return "@unknown"
}

// renderStatus renders a status using extended syntax.
func renderStatus(node map[string]any) string {
	attrs, ok := node["attrs"].(map[string]any)
	if !ok {
		return ""
	}

	text, _ := attrs["text"].(string)
	color, _ := attrs["color"].(string)

	if text == "" {
		return ""
	}

	if color != "" {
		return fmt.Sprintf("{status:%s|color=%s}", text, color)
	}

	return fmt.Sprintf("{status:%s}", text)
}

// renderDate renders a date using extended syntax.
func renderDate(node map[string]any) string {
	attrs, ok := node["attrs"].(map[string]any)
	if !ok {
		return ""
	}

	timestamp, _ := attrs["timestamp"].(string)
	if timestamp == "" {
		return ""
	}

	// Convert timestamp to ISO date for readability
	dateStr := FormatTimestamp(timestamp)
	return fmt.Sprintf("{date:%s}", dateStr)
}

// renderInlineCard renders an inline card using extended syntax.
func renderInlineCard(node map[string]any) string {
	attrs, ok := node["attrs"].(map[string]any)
	if !ok {
		return ""
	}

	if url, ok := attrs["url"].(string); ok {
		return fmt.Sprintf("{card:%s}", url)
	}

	return ""
}

// renderText renders a text node with marks applied.
func renderText(node map[string]any) string {
	text, _ := node["text"].(string)
	if text == "" {
		return ""
	}

	marks, ok := node["marks"].([]any)
	if !ok || len(marks) == 0 {
		return text
	}

	return applyMarks(text, marks)
}

// applyMarks applies formatting marks to text.
func applyMarks(text string, marks []any) string {
	// Process marks in order (innermost to outermost)
	// Order: code, link, em, strong, strike, underline, textColor, backgroundColor, subsup

	// First pass: identify mark types
	var hasCode, hasLink, hasEm, hasStrong, hasStrike, hasUnderline, hasSubsup bool
	var linkHref, linkTitle string
	var textColor, bgColor string
	var subType string

	for _, mark := range marks {
		markMap, ok := mark.(map[string]any)
		if !ok {
			continue
		}
		markType, _ := markMap["type"].(string)
		attrs, _ := markMap["attrs"].(map[string]any)

		switch markType {
		case "code":
			hasCode = true
		case "link":
			hasLink = true
			linkHref, _ = attrs["href"].(string)
			linkTitle, _ = attrs["title"].(string)
		case "em":
			hasEm = true
		case "strong":
			hasStrong = true
		case "strike":
			hasStrike = true
		case "underline":
			hasUnderline = true
		case "textColor":
			textColor, _ = attrs["color"].(string)
		case "backgroundColor":
			bgColor, _ = attrs["color"].(string)
		case "subsup":
			hasSubsup = true
			subType, _ = attrs["type"].(string)
		}
	}

	result := text

	// Apply marks from innermost to outermost
	if hasCode {
		result = "`" + result + "`"
	}

	if hasLink {
		if linkTitle != "" {
			result = fmt.Sprintf("[%s](%s \"%s\")", result, linkHref, linkTitle)
		} else {
			result = fmt.Sprintf("[%s](%s)", result, linkHref)
		}
	}

	if hasEm {
		result = "*" + result + "*"
	}

	if hasStrong {
		result = "**" + result + "**"
	}

	if hasStrike {
		result = "~~" + result + "~~"
	}

	if hasUnderline {
		result = "<u>" + result + "</u>"
	}

	if textColor != "" {
		result = fmt.Sprintf("{color:%s}%s{color}", textColor, result)
	}

	if bgColor != "" {
		result = fmt.Sprintf(`<mark style="background:%s">%s</mark>`, bgColor, result)
	}

	if hasSubsup {
		if subType == "sub" {
			result = "<sub>" + result + "</sub>"
		} else if subType == "sup" {
			result = "<sup>" + result + "</sup>"
		}
	}

	return result
}

// renderContent renders the content array of a node (inline, no separators).
func renderContent(node map[string]any) string {
	content, ok := node["content"].([]any)
	if !ok {
		return ""
	}

	var sb strings.Builder
	for _, child := range content {
		childMap, ok := child.(map[string]any)
		if !ok {
			continue
		}
		sb.WriteString(renderADFNode(childMap, 0))
	}
	return sb.String()
}

// renderBlockContent renders block-level content with newlines between children.
func renderBlockContent(node map[string]any) string {
	content, ok := node["content"].([]any)
	if !ok {
		return ""
	}

	var parts []string
	for _, child := range content {
		childMap, ok := child.(map[string]any)
		if !ok {
			continue
		}
		rendered := renderADFNode(childMap, 0)
		if rendered != "" {
			parts = append(parts, rendered)
		}
	}
	return strings.Join(parts, "\n\n")
}
