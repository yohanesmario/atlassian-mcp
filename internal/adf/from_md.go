package adf

import (
	"fmt"
	"regexp"
	"strings"
)

// FromMarkdown converts extended markdown text to Atlassian Document Format.
// This is the main entry point for Markdown to ADF conversion.
//
// Supports parsing of:
//   - Standard markdown (headings, lists, code blocks, tables, etc.)
//   - Extended fence blocks (~~~panel, ~~~expand, ~~~mediaSingle)
//   - Extended inline syntax ({user:}, {date:}, {status:}, {card:})
//   - Task lists (- [x], - [ ])
//   - Nested lists
//   - Emoji shortcodes (:smile:)
func FromMarkdown(text string) map[string]any {
	return parseMarkdownDocument(text)
}

// parseMarkdownDocument converts extended markdown to an ADF document.
func parseMarkdownDocument(text string) map[string]any {
	lines := strings.Split(text, "\n")
	content := parseBlocks(lines)

	return map[string]any{
		"type":    "doc",
		"version": 1,
		"content": content,
	}
}

// parseBlocks parses markdown lines into ADF block nodes.
func parseBlocks(lines []string) []any {
	var content []any
	i := 0

	// Pending metadata comment for next block
	var pendingMetadata map[string]string

	for i < len(lines) {
		line := lines[i]

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			i++
			continue
		}

		// Check for metadata comment
		if match := MetadataCommentRe.FindStringSubmatch(line); match != nil {
			pendingMetadata = ParseAttrs(match[2])
			i++
			continue
		}

		// Extended fence blocks: ~~~panel, ~~~expand, ~~~mediaSingle, ~~~mediaGroup
		if match := FenceBlockRe.FindStringSubmatch(line); match != nil {
			blockType := match[1]
			attrStr := match[2]
			node, endIdx := parseFenceBlock(lines, i, blockType, attrStr)
			if node != nil {
				content = append(content, node)
			}
			i = endIdx
			pendingMetadata = nil
			continue
		}

		// Code block (``` syntax)
		if strings.HasPrefix(line, "```") {
			node, endIdx := parseCodeBlock(lines, i)
			if node != nil {
				content = append(content, node)
			}
			i = endIdx
			pendingMetadata = nil
			continue
		}

		// Heading
		if strings.HasPrefix(line, "#") {
			node := parseHeading(line, pendingMetadata)
			if node != nil {
				content = append(content, node)
			}
			i++
			pendingMetadata = nil
			continue
		}

		// Horizontal rule
		if line == "---" || line == "***" || line == "___" {
			content = append(content, map[string]any{"type": "rule"})
			i++
			pendingMetadata = nil
			continue
		}

		// Blockquote
		if strings.HasPrefix(line, "> ") || line == ">" {
			node, endIdx := parseBlockquote(lines, i)
			if node != nil {
				content = append(content, node)
			}
			i = endIdx
			pendingMetadata = nil
			continue
		}

		// Table
		if strings.HasPrefix(line, "|") && strings.Contains(line, "|") {
			node, endIdx := parseTable(lines, i)
			if node != nil {
				content = append(content, node)
			}
			i = endIdx
			pendingMetadata = nil
			continue
		}

		// Task list: - [x] or - [ ]
		if TaskItemRe.MatchString(line) {
			node, endIdx := parseTaskList(lines, i)
			if node != nil {
				content = append(content, node)
			}
			i = endIdx
			pendingMetadata = nil
			continue
		}

		// Bullet list
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") || strings.HasPrefix(line, "+ ") {
			node, endIdx := parseBulletList(lines, i, 0)
			if node != nil {
				content = append(content, node)
			}
			i = endIdx
			pendingMetadata = nil
			continue
		}

		// Ordered list
		if matched, _ := regexp.MatchString(`^\d+\.\s`, line); matched {
			node, endIdx := parseOrderedList(lines, i, 0)
			if node != nil {
				content = append(content, node)
			}
			i = endIdx
			pendingMetadata = nil
			continue
		}

		// Standalone image: ![alt](src)
		imageRe := regexp.MustCompile(`^!\[([^\]]*)\]\(([^)]+)\)$`)
		if matches := imageRe.FindStringSubmatch(strings.TrimSpace(line)); matches != nil {
			node := parseStandaloneImage(matches[1], matches[2])
			if node != nil {
				content = append(content, node)
			}
			i++
			pendingMetadata = nil
			continue
		}

		// Regular paragraph
		node, endIdx := parseParagraph(lines, i, pendingMetadata)
		if node != nil {
			content = append(content, node)
		}
		i = endIdx
		pendingMetadata = nil
	}

	return content
}

// parseFenceBlock parses extended fence blocks like ~~~panel, ~~~expand.
func parseFenceBlock(lines []string, startIdx int, blockType, attrStr string) (map[string]any, int) {
	i := startIdx + 1
	var contentLines []string

	// Find closing fence
	for i < len(lines) {
		if FenceCloseRe.MatchString(lines[i]) {
			i++
			break
		}
		contentLines = append(contentLines, lines[i])
		i++
	}

	innerContent := strings.Join(contentLines, "\n")
	attrs := ParseAttrs(attrStr)

	switch blockType {
	case "panel":
		return parsePanelBlock(attrs, innerContent), i
	case "expand":
		return parseExpandBlock(attrs, innerContent), i
	case "mediaSingle":
		return parseMediaSingleBlock(attrs, innerContent), i
	case "mediaGroup":
		return parseMediaGroupBlock(innerContent), i
	default:
		// Unknown fence block, treat as code block
		return map[string]any{
			"type": "codeBlock",
			"attrs": map[string]any{
				"language": blockType,
			},
			"content": []any{
				map[string]any{"type": "text", "text": innerContent},
			},
		}, i
	}
}

// parsePanelBlock parses a panel fence block.
func parsePanelBlock(attrs map[string]string, content string) map[string]any {
	panelType := attrs["type"]
	if panelType == "" {
		panelType = "info"
	}

	// Parse inner content as blocks
	innerBlocks := parseBlocks(strings.Split(content, "\n"))

	return map[string]any{
		"type": "panel",
		"attrs": map[string]any{
			"panelType": panelType,
		},
		"content": innerBlocks,
	}
}

// parseExpandBlock parses an expand fence block.
func parseExpandBlock(attrs map[string]string, content string) map[string]any {
	title := attrs["title"]

	// Parse inner content as blocks
	innerBlocks := parseBlocks(strings.Split(content, "\n"))

	return map[string]any{
		"type": "expand",
		"attrs": map[string]any{
			"title": title,
		},
		"content": innerBlocks,
	}
}

// parseMediaSingleBlock parses a mediaSingle fence block.
func parseMediaSingleBlock(attrs map[string]string, content string) map[string]any {
	layout := attrs["layout"]
	if layout == "" {
		layout = "align-start"
	}

	nodeAttrs := map[string]any{
		"layout": layout,
	}

	if width := attrs["width"]; width != "" {
		var w float64
		fmt.Sscanf(width, "%f", &w)
		if w > 0 {
			nodeAttrs["width"] = w
			// widthType is required for width to be respected
			widthType := attrs["widthType"]
			if widthType == "" {
				widthType = "pixel"
			}
			nodeAttrs["widthType"] = widthType
		}
	}

	// Parse the image from content
	imageRe := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	matches := imageRe.FindStringSubmatch(content)

	var mediaNode map[string]any
	if matches != nil {
		mediaNode = parseMediaFromImage(matches[1], matches[2])
	} else {
		mediaNode = map[string]any{
			"type": "media",
			"attrs": map[string]any{
				"type": "file",
			},
		}
	}

	return map[string]any{
		"type":    "mediaSingle",
		"attrs":   nodeAttrs,
		"content": []any{mediaNode},
	}
}

// parseMediaGroupBlock parses a mediaGroup fence block.
func parseMediaGroupBlock(content string) map[string]any {
	imageRe := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	matches := imageRe.FindAllStringSubmatch(content, -1)

	var mediaNodes []any
	for _, match := range matches {
		mediaNodes = append(mediaNodes, parseMediaFromImage(match[1], match[2]))
	}

	return map[string]any{
		"type":    "mediaGroup",
		"content": mediaNodes,
	}
}

// parseMediaFromImage creates a media node from image alt and src.
func parseMediaFromImage(alt, src string) map[string]any {
	// Check for jira-media format: jira-media:id:collection:type
	if strings.HasPrefix(src, "jira-media:") {
		parts := strings.SplitN(strings.TrimPrefix(src, "jira-media:"), ":", 3)
		if len(parts) >= 3 {
			return map[string]any{
				"type": "media",
				"attrs": map[string]any{
					"id":         parts[0],
					"collection": parts[1],
					"type":       parts[2],
					"alt":        alt,
				},
			}
		}
	}

	// New image (URL or local path) - create placeholder for upload
	return map[string]any{
		"type": "media",
		"attrs": map[string]any{
			"id":      fmt.Sprintf("__PENDING_UPLOAD_%s__", GenerateLocalID()),
			"type":    "file",
			"alt":     alt,
			"_source": src,
		},
	}
}

// parseStandaloneImage parses a standalone image line into mediaSingle.
func parseStandaloneImage(alt, src string) map[string]any {
	mediaNode := parseMediaFromImage(alt, src)

	return map[string]any{
		"type": "mediaSingle",
		"attrs": map[string]any{
			"layout": "align-start",
		},
		"content": []any{mediaNode},
	}
}

// parseCodeBlock parses a ``` code block.
func parseCodeBlock(lines []string, startIdx int) (map[string]any, int) {
	line := lines[startIdx]
	lang := strings.TrimPrefix(line, "```")
	lang = strings.TrimSpace(lang)

	i := startIdx + 1
	var codeLines []string

	for i < len(lines) && !strings.HasPrefix(lines[i], "```") {
		codeLines = append(codeLines, lines[i])
		i++
	}
	i++ // Skip closing ```

	return map[string]any{
		"type": "codeBlock",
		"attrs": map[string]any{
			"language": lang,
		},
		"content": []any{
			map[string]any{"type": "text", "text": strings.Join(codeLines, "\n")},
		},
	}, i
}

// parseHeading parses a heading line.
func parseHeading(line string, metadata map[string]string) map[string]any {
	level := 0
	for _, c := range line {
		if c == '#' {
			level++
		} else {
			break
		}
	}

	if level == 0 || level > 6 || len(line) <= level || line[level] != ' ' {
		return nil
	}

	headingText := strings.TrimSpace(line[level+1:])
	attrs := map[string]any{
		"level": level,
	}

	// Apply metadata attributes
	for k, v := range metadata {
		attrs[k] = v
	}

	return map[string]any{
		"type":    "heading",
		"attrs":   attrs,
		"content": parseInlineContent(headingText),
	}
}

// parseBlockquote parses a blockquote.
func parseBlockquote(lines []string, startIdx int) (map[string]any, int) {
	i := startIdx
	var quoteLines []string

	for i < len(lines) {
		line := lines[i]
		if strings.HasPrefix(line, "> ") {
			quoteLines = append(quoteLines, strings.TrimPrefix(line, "> "))
			i++
		} else if line == ">" {
			quoteLines = append(quoteLines, "")
			i++
		} else {
			break
		}
	}

	// Recursively parse the quoted content
	innerContent := parseBlocks(quoteLines)

	return map[string]any{
		"type":    "blockquote",
		"content": innerContent,
	}, i
}

// parseTable parses a markdown table.
func parseTable(lines []string, startIdx int) (map[string]any, int) {
	i := startIdx
	var tableRows []any
	isFirstRow := true

	for i < len(lines) && strings.HasPrefix(lines[i], "|") {
		rowLine := strings.TrimSpace(lines[i])

		// Skip separator row
		if strings.Contains(rowLine, "---") {
			i++
			continue
		}

		cells := parseTableRow(rowLine)
		var rowContent []any

		cellType := "tableCell"
		if isFirstRow {
			cellType = "tableHeader"
			isFirstRow = false
		}

		for _, cell := range cells {
			rowContent = append(rowContent, map[string]any{
				"type":  cellType,
				"attrs": map[string]any{},
				"content": []any{
					map[string]any{
						"type":    "paragraph",
						"content": parseInlineContent(cell),
					},
				},
			})
		}

		tableRows = append(tableRows, map[string]any{
			"type":    "tableRow",
			"content": rowContent,
		})
		i++
	}

	if len(tableRows) == 0 {
		return nil, i
	}

	return map[string]any{
		"type": "table",
		"attrs": map[string]any{
			"isNumberColumnEnabled": false,
			"layout":                "default",
		},
		"content": tableRows,
	}, i
}

// parseTableRow parses a markdown table row into cells.
func parseTableRow(line string) []string {
	line = strings.Trim(line, "|")
	parts := strings.Split(line, "|")
	var cells []string
	for _, p := range parts {
		cells = append(cells, strings.TrimSpace(p))
	}
	return cells
}

// parseTaskList parses a task list.
func parseTaskList(lines []string, startIdx int) (map[string]any, int) {
	i := startIdx
	var items []any

	for i < len(lines) {
		match := TaskItemRe.FindStringSubmatch(lines[i])
		if match == nil {
			break
		}

		state := "TODO"
		if match[2] == "x" || match[2] == "X" {
			state = "DONE"
		}

		itemText := match[3]
		items = append(items, map[string]any{
			"type": "taskItem",
			"attrs": map[string]any{
				"localId": GenerateLocalID(),
				"state":   state,
			},
			"content": parseInlineContent(itemText),
		})
		i++
	}

	return map[string]any{
		"type": "taskList",
		"attrs": map[string]any{
			"localId": GenerateLocalID(),
		},
		"content": items,
	}, i
}

// parseBulletList parses a bullet list with nested list support.
func parseBulletList(lines []string, startIdx, depth int) (map[string]any, int) {
	i := startIdx
	var items []any
	expectedIndent := depth

	for i < len(lines) {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			i++
			break
		}

		// Check indentation level
		currentIndent := GetIndentLevel(line)
		trimmedLine := strings.TrimLeft(line, " \t")

		// If less indented, this list is done
		if currentIndent < expectedIndent {
			break
		}

		// If more indented, this is a nested list (handled in list item)
		if currentIndent > expectedIndent {
			break
		}

		// Check for bullet marker at expected indent
		if !strings.HasPrefix(trimmedLine, "- ") && !strings.HasPrefix(trimmedLine, "* ") && !strings.HasPrefix(trimmedLine, "+ ") {
			break
		}

		// Parse this item and any nested content
		itemContent, endIdx := parseListItem(lines, i, depth)
		items = append(items, map[string]any{
			"type":    "listItem",
			"content": itemContent,
		})
		i = endIdx
	}

	if len(items) == 0 {
		return nil, startIdx
	}

	return map[string]any{
		"type":    "bulletList",
		"content": items,
	}, i
}

// parseOrderedList parses an ordered list with nested list support.
func parseOrderedList(lines []string, startIdx, depth int) (map[string]any, int) {
	i := startIdx
	var items []any
	expectedIndent := depth
	startOrder := 1
	firstItem := true

	orderedRe := regexp.MustCompile(`^(\d+)\.\s+(.*)$`)

	for i < len(lines) {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			i++
			break
		}

		currentIndent := GetIndentLevel(line)
		trimmedLine := strings.TrimLeft(line, " \t")

		if currentIndent < expectedIndent {
			break
		}

		if currentIndent > expectedIndent {
			break
		}

		match := orderedRe.FindStringSubmatch(trimmedLine)
		if match == nil {
			break
		}

		if firstItem {
			fmt.Sscanf(match[1], "%d", &startOrder)
			firstItem = false
		}

		itemContent, endIdx := parseListItem(lines, i, depth)
		items = append(items, map[string]any{
			"type":    "listItem",
			"content": itemContent,
		})
		i = endIdx
	}

	if len(items) == 0 {
		return nil, startIdx
	}

	node := map[string]any{
		"type":    "orderedList",
		"content": items,
	}

	if startOrder != 1 {
		node["attrs"] = map[string]any{"order": startOrder}
	}

	return node, i
}

// parseListItem parses a list item including any nested lists.
func parseListItem(lines []string, startIdx, depth int) ([]any, int) {
	line := lines[startIdx]
	trimmedLine := strings.TrimLeft(line, " \t")

	// Extract the item text (after the marker)
	var itemText string
	if strings.HasPrefix(trimmedLine, "- ") || strings.HasPrefix(trimmedLine, "* ") || strings.HasPrefix(trimmedLine, "+ ") {
		itemText = strings.TrimSpace(trimmedLine[2:])
	} else {
		// Ordered list marker
		idx := strings.Index(trimmedLine, ". ")
		if idx >= 0 {
			itemText = strings.TrimSpace(trimmedLine[idx+2:])
		}
	}

	content := []any{
		map[string]any{
			"type":    "paragraph",
			"content": parseInlineContent(itemText),
		},
	}

	i := startIdx + 1
	expectedNestedIndent := depth + 1

	// Check for nested content
	for i < len(lines) {
		nextLine := lines[i]
		if strings.TrimSpace(nextLine) == "" {
			i++
			continue
		}

		nextIndent := GetIndentLevel(nextLine)
		if nextIndent < expectedNestedIndent {
			break
		}

		trimmedNext := strings.TrimLeft(nextLine, " \t")

		// Check for nested bullet list
		if strings.HasPrefix(trimmedNext, "- ") || strings.HasPrefix(trimmedNext, "* ") || strings.HasPrefix(trimmedNext, "+ ") {
			nestedList, endIdx := parseBulletList(lines, i, depth+1)
			if nestedList != nil {
				content = append(content, nestedList)
			}
			i = endIdx
			continue
		}

		// Check for nested ordered list
		if matched, _ := regexp.MatchString(`^\d+\.\s`, trimmedNext); matched {
			nestedList, endIdx := parseOrderedList(lines, i, depth+1)
			if nestedList != nil {
				content = append(content, nestedList)
			}
			i = endIdx
			continue
		}

		// Check for nested task list
		if TaskItemRe.MatchString(trimmedNext) {
			nestedList, endIdx := parseTaskList(lines, i)
			if nestedList != nil {
				content = append(content, nestedList)
			}
			i = endIdx
			continue
		}

		// Otherwise it's continuation text
		break
	}

	return content, i
}

// parseParagraph parses a paragraph.
func parseParagraph(lines []string, startIdx int, metadata map[string]string) (map[string]any, int) {
	i := startIdx
	var paraLines []string

	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			i++
			break
		}

		// Check if this line starts a new block
		if strings.HasPrefix(line, "#") ||
			strings.HasPrefix(line, "```") ||
			strings.HasPrefix(line, "~~~") ||
			strings.HasPrefix(line, "> ") ||
			strings.HasPrefix(line, "- ") ||
			strings.HasPrefix(line, "* ") ||
			strings.HasPrefix(line, "+ ") ||
			strings.HasPrefix(line, "|") ||
			line == "---" || line == "***" || line == "___" {
			if len(paraLines) > 0 {
				break
			}
		}

		if matched, _ := regexp.MatchString(`^\d+\.\s`, line); matched && len(paraLines) > 0 {
			break
		}

		if TaskItemRe.MatchString(line) && len(paraLines) > 0 {
			break
		}

		paraLines = append(paraLines, line)
		i++
	}

	if len(paraLines) == 0 {
		return nil, i
	}

	paraText := strings.Join(paraLines, "\n")
	node := map[string]any{
		"type":    "paragraph",
		"content": parseInlineContent(paraText),
	}

	// Apply metadata attributes
	if len(metadata) > 0 {
		attrs := make(map[string]any)
		for k, v := range metadata {
			attrs[k] = v
		}
		node["attrs"] = attrs
	}

	return node, i
}

// parseInlineContent parses inline markdown into ADF inline nodes.
func parseInlineContent(text string) []any {
	if text == "" {
		return []any{}
	}

	var result []any

	// Pattern definitions with handlers
	patterns := []struct {
		name    string
		re      *regexp.Regexp
		handler func(match []string) map[string]any
	}{
		// Extended syntax - must come first
		{
			name: "mention",
			re:   ExtMentionRe,
			handler: func(match []string) map[string]any {
				return map[string]any{
					"type": "mention",
					"attrs": map[string]any{
						"id":   match[1],
						"text": "@" + match[1],
					},
				}
			},
		},
		{
			name: "date",
			re:   ExtDateRe,
			handler: func(match []string) map[string]any {
				return map[string]any{
					"type": "date",
					"attrs": map[string]any{
						"timestamp": ParseTimestamp(match[1]),
					},
				}
			},
		},
		{
			name: "status",
			re:   ExtStatusRe,
			handler: func(match []string) map[string]any {
				attrs := map[string]any{
					"text":    match[1],
					"localId": GenerateLocalID(),
				}
				if match[2] != "" {
					statusAttrs := SplitStatusAttrs(match[2])
					if color := statusAttrs["color"]; color != "" {
						attrs["color"] = color
					}
				}
				return map[string]any{
					"type":  "status",
					"attrs": attrs,
				}
			},
		},
		{
			name: "card",
			re:   ExtCardRe,
			handler: func(match []string) map[string]any {
				return map[string]any{
					"type": "inlineCard",
					"attrs": map[string]any{
						"url": match[1],
					},
				}
			},
		},
		{
			name: "emoji",
			re:   EmojiCodeRe,
			handler: func(match []string) map[string]any {
				return map[string]any{
					"type": "emoji",
					"attrs": map[string]any{
						"shortName": ":" + match[1] + ":",
					},
				}
			},
		},
		// Legacy mention format: @[DisplayName](accountId:xxx)
		{
			name: "legacyMention",
			re:   regexp.MustCompile(`@\[([^\]]+)\]\(accountId:([^)]+)\)`),
			handler: func(match []string) map[string]any {
				return map[string]any{
					"type": "mention",
					"attrs": map[string]any{
						"id":   match[2],
						"text": "@" + match[1],
					},
				}
			},
		},
		// Links: [text](url) or [text](url "title")
		{
			name: "link",
			re:   regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)(?:\s+"([^"]+)")?\)`),
			handler: func(match []string) map[string]any {
				marks := []any{
					map[string]any{
						"type": "link",
						"attrs": map[string]any{
							"href": match[2],
						},
					},
				}
				if match[3] != "" {
					marks[0].(map[string]any)["attrs"].(map[string]any)["title"] = match[3]
				}
				return map[string]any{
					"type":  "text",
					"text":  match[1],
					"marks": marks,
				}
			},
		},
		// Bold: **text** or __text__
		{
			name: "bold",
			re:   regexp.MustCompile(`\*\*([^*]+)\*\*|__([^_]+)__`),
			handler: func(match []string) map[string]any {
				text := match[1]
				if text == "" {
					text = match[2]
				}
				return map[string]any{
					"type": "text",
					"text": text,
					"marks": []any{
						map[string]any{"type": "strong"},
					},
				}
			},
		},
		// Italic: *text* or _text_ (bold ** is processed first, so simple pattern is safe)
		{
			name: "italic",
			re:   regexp.MustCompile(`\*([^*]+)\*|_([^_]+)_`),
			handler: func(match []string) map[string]any {
				text := match[1]
				if text == "" {
					text = match[2]
				}
				return map[string]any{
					"type": "text",
					"text": text,
					"marks": []any{
						map[string]any{"type": "em"},
					},
				}
			},
		},
		// Strikethrough: ~~text~~
		{
			name: "strike",
			re:   regexp.MustCompile(`~~([^~]+)~~`),
			handler: func(match []string) map[string]any {
				return map[string]any{
					"type": "text",
					"text": match[1],
					"marks": []any{
						map[string]any{"type": "strike"},
					},
				}
			},
		},
		// Inline code: `code`
		{
			name: "code",
			re:   regexp.MustCompile("`([^`]+)`"),
			handler: func(match []string) map[string]any {
				return map[string]any{
					"type": "text",
					"text": match[1],
					"marks": []any{
						map[string]any{"type": "code"},
					},
				}
			},
		},
		// Text color: {color:#hex}text{color}
		{
			name: "textColor",
			re:   ExtColorRe,
			handler: func(match []string) map[string]any {
				return map[string]any{
					"type": "text",
					"text": match[2],
					"marks": []any{
						map[string]any{
							"type": "textColor",
							"attrs": map[string]any{
								"color": match[1],
							},
						},
					},
				}
			},
		},
		// Underline: <u>text</u>
		{
			name: "underline",
			re:   regexp.MustCompile(`<u>([^<]+)</u>`),
			handler: func(match []string) map[string]any {
				return map[string]any{
					"type": "text",
					"text": match[1],
					"marks": []any{
						map[string]any{"type": "underline"},
					},
				}
			},
		},
		// Subscript: <sub>text</sub>
		{
			name: "subscript",
			re:   regexp.MustCompile(`<sub>([^<]+)</sub>`),
			handler: func(match []string) map[string]any {
				return map[string]any{
					"type": "text",
					"text": match[1],
					"marks": []any{
						map[string]any{
							"type": "subsup",
							"attrs": map[string]any{
								"type": "sub",
							},
						},
					},
				}
			},
		},
		// Superscript: <sup>text</sup>
		{
			name: "superscript",
			re:   regexp.MustCompile(`<sup>([^<]+)</sup>`),
			handler: func(match []string) map[string]any {
				return map[string]any{
					"type": "text",
					"text": match[1],
					"marks": []any{
						map[string]any{
							"type": "subsup",
							"attrs": map[string]any{
								"type": "sup",
							},
						},
					},
				}
			},
		},
	}

	// Process text with inline patterns
	remaining := text
	for len(remaining) > 0 {
		earliestMatch := -1
		var earliestPattern int
		var earliestResult []int

		// Find the earliest matching pattern
		for pi, p := range patterns {
			loc := p.re.FindStringIndex(remaining)
			if loc != nil && (earliestMatch == -1 || loc[0] < earliestMatch) {
				earliestMatch = loc[0]
				earliestPattern = pi
				earliestResult = loc
			}
		}

		if earliestMatch == -1 {
			// No more matches, add remaining text
			if remaining != "" {
				result = append(result, map[string]any{
					"type": "text",
					"text": remaining,
				})
			}
			break
		}

		// Add text before the match
		if earliestMatch > 0 {
			result = append(result, map[string]any{
				"type": "text",
				"text": remaining[:earliestMatch],
			})
		}

		// Process the match
		matchStr := remaining[earliestResult[0]:earliestResult[1]]
		submatches := patterns[earliestPattern].re.FindStringSubmatch(matchStr)
		result = append(result, patterns[earliestPattern].handler(submatches))

		remaining = remaining[earliestResult[1]:]
	}

	if len(result) == 0 {
		return []any{map[string]any{"type": "text", "text": text}}
	}

	return result
}
