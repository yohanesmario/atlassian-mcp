package adf

import (
	"testing"
)

func TestParseMarkdownDocument(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     string
		wantType  string
		wantLen   int
		checkFunc func(t *testing.T, doc map[string]any)
	}{
		{
			name:     "Empty_Document",
			input:    "",
			wantType: "doc",
			wantLen:  0,
		},
		{
			name:     "Single_Paragraph",
			input:    "Hello World",
			wantType: "doc",
			wantLen:  1,
			checkFunc: func(t *testing.T, doc map[string]any) {
				content := doc["content"].([]any)
				para := content[0].(map[string]any)
				if para["type"] != "paragraph" {
					t.Errorf("expected paragraph, got %v", para["type"])
				}
			},
		},
		{
			name:     "Multiple_Paragraphs",
			input:    "First\n\nSecond",
			wantType: "doc",
			wantLen:  2,
		},
		{
			name:     "Heading_And_Paragraph",
			input:    "# Title\n\nContent here",
			wantType: "doc",
			wantLen:  2,
			checkFunc: func(t *testing.T, doc map[string]any) {
				content := doc["content"].([]any)
				heading := content[0].(map[string]any)
				if heading["type"] != "heading" {
					t.Errorf("expected heading, got %v", heading["type"])
				}
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseMarkdownDocument(tt.input)
			if got["type"] != tt.wantType {
				t.Errorf("type = %v, want %v", got["type"], tt.wantType)
			}
			if got["version"] != 1 {
				t.Errorf("version = %v, want 1", got["version"])
			}
			content, ok := got["content"].([]any)
			if !ok {
				t.Fatal("content is not []any")
			}
			if len(content) != tt.wantLen {
				t.Errorf("content length = %d, want %d", len(content), tt.wantLen)
			}
			if tt.checkFunc != nil {
				tt.checkFunc(t, got)
			}
		})
	}
}

func TestParseHeading(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		line      string
		metadata  map[string]string
		wantLevel int
		wantText  string
	}{
		{
			name:      "Level_1",
			line:      "# Title",
			wantLevel: 1,
			wantText:  "Title",
		},
		{
			name:      "Level_2",
			line:      "## Subtitle",
			wantLevel: 2,
			wantText:  "Subtitle",
		},
		{
			name:      "Level_6",
			line:      "###### Small",
			wantLevel: 6,
			wantText:  "Small",
		},
		{
			name:      "With_Inline_Formatting",
			line:      "# **Bold** Title",
			wantLevel: 1,
			wantText:  "Bold Title", // Text content extracted
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseHeading(tt.line, tt.metadata)
			if got == nil {
				t.Fatal("parseHeading returned nil")
			}
			if got["type"] != "heading" {
				t.Errorf("type = %v, want heading", got["type"])
			}
			attrs := got["attrs"].(map[string]any)
			if attrs["level"] != tt.wantLevel {
				t.Errorf("level = %v, want %d", attrs["level"], tt.wantLevel)
			}
		})
	}
}

func TestParseCodeBlock(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		lines    []string
		startIdx int
		wantLang string
		wantText string
		wantEnd  int
	}{
		{
			name:     "Simple_Code",
			lines:    []string{"```", "code here", "```"},
			startIdx: 0,
			wantLang: "",
			wantText: "code here",
			wantEnd:  3,
		},
		{
			name:     "With_Language",
			lines:    []string{"```go", "func main() {}", "```"},
			startIdx: 0,
			wantLang: "go",
			wantText: "func main() {}",
			wantEnd:  3,
		},
		{
			name:     "Multi_Line",
			lines:    []string{"```python", "def foo():", "    pass", "```"},
			startIdx: 0,
			wantLang: "python",
			wantText: "def foo():\n    pass",
			wantEnd:  4,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, endIdx := parseCodeBlock(tt.lines, tt.startIdx)
			if got == nil {
				t.Fatal("parseCodeBlock returned nil")
			}
			if got["type"] != "codeBlock" {
				t.Errorf("type = %v, want codeBlock", got["type"])
			}
			if endIdx != tt.wantEnd {
				t.Errorf("endIdx = %d, want %d", endIdx, tt.wantEnd)
			}
			attrs, ok := got["attrs"].(map[string]any)
			if ok {
				lang, _ := attrs["language"].(string)
				if lang != tt.wantLang {
					t.Errorf("language = %q, want %q", lang, tt.wantLang)
				}
			}
		})
	}
}

func TestParseBlockquote(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		lines    []string
		startIdx int
		wantEnd  int
	}{
		{
			name:     "Single_Line",
			lines:    []string{"> Quote", ""},
			startIdx: 0,
			wantEnd:  1,
		},
		{
			name:     "Multi_Line",
			lines:    []string{"> Line 1", "> Line 2", ""},
			startIdx: 0,
			wantEnd:  2,
		},
		{
			name:     "Empty_Quote_Line",
			lines:    []string{"> Line 1", ">", "> Line 2", ""},
			startIdx: 0,
			wantEnd:  3,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, endIdx := parseBlockquote(tt.lines, tt.startIdx)
			if got == nil {
				t.Fatal("parseBlockquote returned nil")
			}
			if got["type"] != "blockquote" {
				t.Errorf("type = %v, want blockquote", got["type"])
			}
			if endIdx != tt.wantEnd {
				t.Errorf("endIdx = %d, want %d", endIdx, tt.wantEnd)
			}
		})
	}
}

func TestParseTable(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		lines    []string
		startIdx int
		wantEnd  int
		wantRows int
	}{
		{
			name: "Simple_Table",
			lines: []string{
				"| A | B |",
				"|---|---|",
				"| 1 | 2 |",
			},
			startIdx: 0,
			wantEnd:  3,
			wantRows: 2, // header + 1 data row
		},
		{
			name: "Multi_Row_Table",
			lines: []string{
				"| Header 1 | Header 2 |",
				"|----------|----------|",
				"| Row 1    | Data 1   |",
				"| Row 2    | Data 2   |",
			},
			startIdx: 0,
			wantEnd:  4,
			wantRows: 3,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, endIdx := parseTable(tt.lines, tt.startIdx)
			if got == nil {
				t.Fatal("parseTable returned nil")
			}
			if got["type"] != "table" {
				t.Errorf("type = %v, want table", got["type"])
			}
			if endIdx != tt.wantEnd {
				t.Errorf("endIdx = %d, want %d", endIdx, tt.wantEnd)
			}
			content, ok := got["content"].([]any)
			if !ok {
				t.Fatal("content is not []any")
			}
			if len(content) != tt.wantRows {
				t.Errorf("rows = %d, want %d", len(content), tt.wantRows)
			}
		})
	}
}

func TestParseBulletList(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		lines     []string
		startIdx  int
		depth     int
		wantItems int
	}{
		{
			name:      "Simple_List",
			lines:     []string{"- Item 1", "- Item 2", ""},
			startIdx:  0,
			depth:     0,
			wantItems: 2,
		},
		{
			name:      "Single_Item",
			lines:     []string{"- Only item", ""},
			startIdx:  0,
			depth:     0,
			wantItems: 1,
		},
		{
			name:      "With_Asterisk",
			lines:     []string{"* Item 1", "* Item 2", ""},
			startIdx:  0,
			depth:     0,
			wantItems: 2,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, _ := parseBulletList(tt.lines, tt.startIdx, tt.depth)
			if got == nil {
				t.Fatal("parseBulletList returned nil")
			}
			if got["type"] != "bulletList" {
				t.Errorf("type = %v, want bulletList", got["type"])
			}
			content, ok := got["content"].([]any)
			if !ok {
				t.Fatal("content is not []any")
			}
			if len(content) != tt.wantItems {
				t.Errorf("items = %d, want %d", len(content), tt.wantItems)
			}
		})
	}
}

func TestParseOrderedList(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		lines     []string
		startIdx  int
		depth     int
		wantItems int
	}{
		{
			name:      "Simple_List",
			lines:     []string{"1. First", "2. Second", ""},
			startIdx:  0,
			depth:     0,
			wantItems: 2,
		},
		{
			name:      "Non_Sequential",
			lines:     []string{"1. First", "5. Fifth", ""},
			startIdx:  0,
			depth:     0,
			wantItems: 2,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, _ := parseOrderedList(tt.lines, tt.startIdx, tt.depth)
			if got == nil {
				t.Fatal("parseOrderedList returned nil")
			}
			if got["type"] != "orderedList" {
				t.Errorf("type = %v, want orderedList", got["type"])
			}
			content, ok := got["content"].([]any)
			if !ok {
				t.Fatal("content is not []any")
			}
			if len(content) != tt.wantItems {
				t.Errorf("items = %d, want %d", len(content), tt.wantItems)
			}
		})
	}
}

func TestParseTaskList(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		lines     []string
		startIdx  int
		wantEnd   int
		wantItems int
	}{
		{
			name:      "Mixed_States",
			lines:     []string{"- [x] Done", "- [ ] Todo", ""},
			startIdx:  0,
			wantEnd:   2,
			wantItems: 2,
		},
		{
			name:      "All_Done",
			lines:     []string{"- [x] Task 1", "- [X] Task 2", ""},
			startIdx:  0,
			wantEnd:   2,
			wantItems: 2,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, endIdx := parseTaskList(tt.lines, tt.startIdx)
			if got == nil {
				t.Fatal("parseTaskList returned nil")
			}
			if got["type"] != "taskList" {
				t.Errorf("type = %v, want taskList", got["type"])
			}
			if endIdx != tt.wantEnd {
				t.Errorf("endIdx = %d, want %d", endIdx, tt.wantEnd)
			}
			content, ok := got["content"].([]any)
			if !ok {
				t.Fatal("content is not []any")
			}
			if len(content) != tt.wantItems {
				t.Errorf("items = %d, want %d", len(content), tt.wantItems)
			}
		})
	}
}

func TestParseInlineContent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     string
		wantLen   int
		checkFunc func(t *testing.T, nodes []any)
	}{
		{
			name:    "Plain_Text",
			input:   "Hello World",
			wantLen: 1,
			checkFunc: func(t *testing.T, nodes []any) {
				node := nodes[0].(map[string]any)
				if node["type"] != "text" {
					t.Errorf("expected text node, got %v", node["type"])
				}
				if node["text"] != "Hello World" {
					t.Errorf("text = %q, want %q", node["text"], "Hello World")
				}
			},
		},
		{
			name:    "Bold_Text",
			input:   "**bold**",
			wantLen: 1,
			checkFunc: func(t *testing.T, nodes []any) {
				node := nodes[0].(map[string]any)
				if node["type"] != "text" {
					t.Errorf("expected text node, got %v", node["type"])
				}
				marks, ok := node["marks"].([]any)
				if !ok || len(marks) == 0 {
					t.Error("expected marks on bold text")
					return
				}
				mark := marks[0].(map[string]any)
				if mark["type"] != "strong" {
					t.Errorf("expected strong mark, got %v", mark["type"])
				}
			},
		},
		{
			name:    "Italic_Text",
			input:   "*italic*",
			wantLen: 1,
			checkFunc: func(t *testing.T, nodes []any) {
				node := nodes[0].(map[string]any)
				marks, ok := node["marks"].([]any)
				if !ok || len(marks) == 0 {
					t.Error("expected marks on italic text")
					return
				}
				mark := marks[0].(map[string]any)
				if mark["type"] != "em" {
					t.Errorf("expected em mark, got %v", mark["type"])
				}
			},
		},
		{
			name:    "Inline_Code",
			input:   "`code`",
			wantLen: 1,
			checkFunc: func(t *testing.T, nodes []any) {
				node := nodes[0].(map[string]any)
				marks, ok := node["marks"].([]any)
				if !ok || len(marks) == 0 {
					t.Error("expected marks on code text")
					return
				}
				mark := marks[0].(map[string]any)
				if mark["type"] != "code" {
					t.Errorf("expected code mark, got %v", mark["type"])
				}
			},
		},
		{
			name:    "Link",
			input:   "[Click](https://example.com)",
			wantLen: 1,
			checkFunc: func(t *testing.T, nodes []any) {
				node := nodes[0].(map[string]any)
				marks, ok := node["marks"].([]any)
				if !ok || len(marks) == 0 {
					t.Error("expected marks on link text")
					return
				}
				mark := marks[0].(map[string]any)
				if mark["type"] != "link" {
					t.Errorf("expected link mark, got %v", mark["type"])
				}
				attrs := mark["attrs"].(map[string]any)
				if attrs["href"] != "https://example.com" {
					t.Errorf("href = %q, want %q", attrs["href"], "https://example.com")
				}
			},
		},
		{
			name:  "User_Mention",
			input: "{user:abc123}",
			checkFunc: func(t *testing.T, nodes []any) {
				found := false
				for _, n := range nodes {
					node := n.(map[string]any)
					if node["type"] == "mention" {
						found = true
						attrs := node["attrs"].(map[string]any)
						if attrs["id"] != "abc123" {
							t.Errorf("mention id = %q, want %q", attrs["id"], "abc123")
						}
					}
				}
				if !found {
					t.Error("expected mention node")
				}
			},
		},
		{
			name:  "Date",
			input: "{date:2024-01-01}",
			checkFunc: func(t *testing.T, nodes []any) {
				found := false
				for _, n := range nodes {
					node := n.(map[string]any)
					if node["type"] == "date" {
						found = true
					}
				}
				if !found {
					t.Error("expected date node")
				}
			},
		},
		{
			name:  "Status",
			input: "{status:In Progress|color=blue}",
			checkFunc: func(t *testing.T, nodes []any) {
				found := false
				for _, n := range nodes {
					node := n.(map[string]any)
					if node["type"] == "status" {
						found = true
						attrs := node["attrs"].(map[string]any)
						if attrs["text"] != "In Progress" {
							t.Errorf("status text = %q, want %q", attrs["text"], "In Progress")
						}
						if attrs["color"] != "blue" {
							t.Errorf("status color = %q, want %q", attrs["color"], "blue")
						}
					}
				}
				if !found {
					t.Error("expected status node")
				}
			},
		},
		{
			name:  "Emoji",
			input: ":smile:",
			checkFunc: func(t *testing.T, nodes []any) {
				found := false
				for _, n := range nodes {
					node := n.(map[string]any)
					if node["type"] == "emoji" {
						found = true
						attrs := node["attrs"].(map[string]any)
						if attrs["shortName"] != ":smile:" {
							t.Errorf("emoji shortName = %q, want %q", attrs["shortName"], ":smile:")
						}
					}
				}
				if !found {
					t.Error("expected emoji node")
				}
			},
		},
		{
			name:  "Inline_Card",
			input: "{card:https://example.com}",
			checkFunc: func(t *testing.T, nodes []any) {
				found := false
				for _, n := range nodes {
					node := n.(map[string]any)
					if node["type"] == "inlineCard" {
						found = true
						attrs := node["attrs"].(map[string]any)
						if attrs["url"] != "https://example.com" {
							t.Errorf("card url = %q, want %q", attrs["url"], "https://example.com")
						}
					}
				}
				if !found {
					t.Error("expected inlineCard node")
				}
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseInlineContent(tt.input)
			if tt.wantLen > 0 && len(got) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(got), tt.wantLen)
			}
			if tt.checkFunc != nil {
				tt.checkFunc(t, got)
			}
		})
	}
}

func TestParseFenceBlock_Panel(t *testing.T) {
	t.Parallel()
	lines := []string{
		"~~~panel type=info",
		"Panel content here",
		"~~~",
	}
	got, endIdx := parseFenceBlock(lines, 0, "panel", "type=info")
	if got == nil {
		t.Fatal("parseFenceBlock returned nil")
	}
	if got["type"] != "panel" {
		t.Errorf("type = %v, want panel", got["type"])
	}
	if endIdx != 3 {
		t.Errorf("endIdx = %d, want 3", endIdx)
	}
	attrs := got["attrs"].(map[string]any)
	if attrs["panelType"] != "info" {
		t.Errorf("panelType = %v, want info", attrs["panelType"])
	}
}

func TestParseFenceBlock_Expand(t *testing.T) {
	t.Parallel()
	lines := []string{
		`~~~expand title="Click me"`,
		"Expandable content",
		"~~~",
	}
	got, endIdx := parseFenceBlock(lines, 0, "expand", `title="Click me"`)
	if got == nil {
		t.Fatal("parseFenceBlock returned nil")
	}
	if got["type"] != "expand" {
		t.Errorf("type = %v, want expand", got["type"])
	}
	if endIdx != 3 {
		t.Errorf("endIdx = %d, want 3", endIdx)
	}
	attrs := got["attrs"].(map[string]any)
	if attrs["title"] != "Click me" {
		t.Errorf("title = %v, want 'Click me'", attrs["title"])
	}
}

func TestHorizontalRule(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
	}{
		{name: "Dashes", input: "---"},
		{name: "Asterisks", input: "***"},
		{name: "Underscores", input: "___"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc := parseMarkdownDocument(tt.input)
			content := doc["content"].([]any)
			if len(content) != 1 {
				t.Fatalf("expected 1 node, got %d", len(content))
			}
			node := content[0].(map[string]any)
			if node["type"] != "rule" {
				t.Errorf("type = %v, want rule", node["type"])
			}
		})
	}
}
