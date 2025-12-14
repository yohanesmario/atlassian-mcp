package adf

import (
	"strings"
	"testing"
)

func TestRenderADFDocument(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		doc  map[string]any
		want string
	}{
		{
			name: "Empty_Document",
			doc:  map[string]any{},
			want: "",
		},
		{
			name: "Empty_Content",
			doc:  map[string]any{"content": []any{}},
			want: "",
		},
		{
			name: "Single_Paragraph",
			doc: map[string]any{
				"type":    "doc",
				"version": 1,
				"content": []any{
					map[string]any{
						"type": "paragraph",
						"content": []any{
							map[string]any{"type": "text", "text": "Hello World"},
						},
					},
				},
			},
			want: "Hello World",
		},
		{
			name: "Multiple_Paragraphs",
			doc: map[string]any{
				"content": []any{
					map[string]any{
						"type":    "paragraph",
						"content": []any{map[string]any{"type": "text", "text": "First"}},
					},
					map[string]any{
						"type":    "paragraph",
						"content": []any{map[string]any{"type": "text", "text": "Second"}},
					},
				},
			},
			want: "First\n\nSecond",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := renderADFDocument(tt.doc)
			if got != tt.want {
				t.Errorf("renderADFDocument() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderParagraph(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		node map[string]any
		want string
	}{
		{
			name: "Simple_Text",
			node: map[string]any{
				"type":    "paragraph",
				"content": []any{map[string]any{"type": "text", "text": "Hello"}},
			},
			want: "Hello",
		},
		{
			name: "With_TextAlign",
			node: map[string]any{
				"type":    "paragraph",
				"attrs":   map[string]any{"textAlign": "center"},
				"content": []any{map[string]any{"type": "text", "text": "Centered"}},
			},
			want: "<!-- adf:paragraph textAlign=\"center\" -->\nCentered",
		},
		{
			name: "Empty_Content",
			node: map[string]any{"type": "paragraph"},
			want: "",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := renderParagraph(tt.node)
			if got != tt.want {
				t.Errorf("renderParagraph() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderHeading(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		node map[string]any
		want string
	}{
		{
			name: "Level_1",
			node: map[string]any{
				"type":    "heading",
				"attrs":   map[string]any{"level": float64(1)},
				"content": []any{map[string]any{"type": "text", "text": "Title"}},
			},
			want: "# Title",
		},
		{
			name: "Level_3",
			node: map[string]any{
				"type":    "heading",
				"attrs":   map[string]any{"level": float64(3)},
				"content": []any{map[string]any{"type": "text", "text": "Subtitle"}},
			},
			want: "### Subtitle",
		},
		{
			name: "Level_6",
			node: map[string]any{
				"type":    "heading",
				"attrs":   map[string]any{"level": float64(6)},
				"content": []any{map[string]any{"type": "text", "text": "Small"}},
			},
			want: "###### Small",
		},
		{
			name: "Level_Clamped_High",
			node: map[string]any{
				"type":    "heading",
				"attrs":   map[string]any{"level": float64(10)},
				"content": []any{map[string]any{"type": "text", "text": "Clamped"}},
			},
			want: "###### Clamped",
		},
		{
			name: "Level_Clamped_Low",
			node: map[string]any{
				"type":    "heading",
				"attrs":   map[string]any{"level": float64(0)},
				"content": []any{map[string]any{"type": "text", "text": "Clamped"}},
			},
			want: "# Clamped",
		},
		{
			name: "Default_Level",
			node: map[string]any{
				"type":    "heading",
				"content": []any{map[string]any{"type": "text", "text": "Default"}},
			},
			want: "# Default",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := renderHeading(tt.node)
			if got != tt.want {
				t.Errorf("renderHeading() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderBulletList(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		node  map[string]any
		depth int
		want  string
	}{
		{
			name: "Simple_List",
			node: map[string]any{
				"type": "bulletList",
				"content": []any{
					map[string]any{
						"type": "listItem",
						"content": []any{
							map[string]any{
								"type":    "paragraph",
								"content": []any{map[string]any{"type": "text", "text": "Item 1"}},
							},
						},
					},
					map[string]any{
						"type": "listItem",
						"content": []any{
							map[string]any{
								"type":    "paragraph",
								"content": []any{map[string]any{"type": "text", "text": "Item 2"}},
							},
						},
					},
				},
			},
			depth: 0,
			want:  "- Item 1\n- Item 2",
		},
		{
			name: "Nested_List",
			node: map[string]any{
				"type": "bulletList",
				"content": []any{
					map[string]any{
						"type": "listItem",
						"content": []any{
							map[string]any{
								"type":    "paragraph",
								"content": []any{map[string]any{"type": "text", "text": "Nested"}},
							},
						},
					},
				},
			},
			depth: 1,
			want:  "  - Nested",
		},
		{
			name:  "Empty_List",
			node:  map[string]any{"type": "bulletList"},
			depth: 0,
			want:  "",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := renderBulletList(tt.node, tt.depth)
			if got != tt.want {
				t.Errorf("renderBulletList() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderOrderedList(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		node  map[string]any
		depth int
		want  string
	}{
		{
			name: "Simple_List",
			node: map[string]any{
				"type": "orderedList",
				"content": []any{
					map[string]any{
						"type": "listItem",
						"content": []any{
							map[string]any{
								"type":    "paragraph",
								"content": []any{map[string]any{"type": "text", "text": "First"}},
							},
						},
					},
					map[string]any{
						"type": "listItem",
						"content": []any{
							map[string]any{
								"type":    "paragraph",
								"content": []any{map[string]any{"type": "text", "text": "Second"}},
							},
						},
					},
				},
			},
			depth: 0,
			want:  "1. First\n2. Second",
		},
		{
			name: "Custom_Start",
			node: map[string]any{
				"type":  "orderedList",
				"attrs": map[string]any{"order": float64(5)},
				"content": []any{
					map[string]any{
						"type": "listItem",
						"content": []any{
							map[string]any{
								"type":    "paragraph",
								"content": []any{map[string]any{"type": "text", "text": "Item"}},
							},
						},
					},
				},
			},
			depth: 0,
			want:  "5. Item",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := renderOrderedList(tt.node, tt.depth)
			if got != tt.want {
				t.Errorf("renderOrderedList() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderTaskList(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		node  map[string]any
		depth int
		want  string
	}{
		{
			name: "Mixed_States",
			node: map[string]any{
				"type": "taskList",
				"content": []any{
					map[string]any{
						"type":    "taskItem",
						"attrs":   map[string]any{"state": "DONE"},
						"content": []any{map[string]any{"type": "text", "text": "Done task"}},
					},
					map[string]any{
						"type":    "taskItem",
						"attrs":   map[string]any{"state": "TODO"},
						"content": []any{map[string]any{"type": "text", "text": "Todo task"}},
					},
				},
			},
			depth: 0,
			want:  "- [x] Done task\n- [ ] Todo task",
		},
		{
			name: "Default_State",
			node: map[string]any{
				"type": "taskList",
				"content": []any{
					map[string]any{
						"type":    "taskItem",
						"content": []any{map[string]any{"type": "text", "text": "No state"}},
					},
				},
			},
			depth: 0,
			want:  "- [ ] No state",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := renderTaskList(tt.node, tt.depth)
			if got != tt.want {
				t.Errorf("renderTaskList() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderCodeBlock(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		node map[string]any
		want string
	}{
		{
			name: "With_Language",
			node: map[string]any{
				"type":    "codeBlock",
				"attrs":   map[string]any{"language": "go"},
				"content": []any{map[string]any{"type": "text", "text": "func main() {}"}},
			},
			want: "```go\nfunc main() {}\n```",
		},
		{
			name: "No_Language",
			node: map[string]any{
				"type":    "codeBlock",
				"content": []any{map[string]any{"type": "text", "text": "plain code"}},
			},
			want: "```\nplain code\n```",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := renderCodeBlock(tt.node)
			if got != tt.want {
				t.Errorf("renderCodeBlock() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderBlockquote(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		node map[string]any
		want string
	}{
		{
			name: "Single_Line",
			node: map[string]any{
				"type": "blockquote",
				"content": []any{
					map[string]any{
						"type":    "paragraph",
						"content": []any{map[string]any{"type": "text", "text": "Quote"}},
					},
				},
			},
			want: "> Quote",
		},
		{
			name: "Multi_Line",
			node: map[string]any{
				"type": "blockquote",
				"content": []any{
					map[string]any{
						"type":    "paragraph",
						"content": []any{map[string]any{"type": "text", "text": "Line 1\nLine 2"}},
					},
				},
			},
			want: "> Line 1\n> Line 2",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := renderBlockquote(tt.node)
			if got != tt.want {
				t.Errorf("renderBlockquote() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderPanel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		node map[string]any
		want string
	}{
		{
			name: "Info_Panel",
			node: map[string]any{
				"type":  "panel",
				"attrs": map[string]any{"panelType": "info"},
				"content": []any{
					map[string]any{
						"type":    "paragraph",
						"content": []any{map[string]any{"type": "text", "text": "Info message"}},
					},
				},
			},
			want: "~~~panel type=info\nInfo message\n~~~",
		},
		{
			name: "Warning_Panel",
			node: map[string]any{
				"type":  "panel",
				"attrs": map[string]any{"panelType": "warning"},
				"content": []any{
					map[string]any{
						"type":    "paragraph",
						"content": []any{map[string]any{"type": "text", "text": "Warning!"}},
					},
				},
			},
			want: "~~~panel type=warning\nWarning!\n~~~",
		},
		{
			name: "Default_Type",
			node: map[string]any{
				"type": "panel",
				"content": []any{
					map[string]any{
						"type":    "paragraph",
						"content": []any{map[string]any{"type": "text", "text": "Default"}},
					},
				},
			},
			want: "~~~panel type=info\nDefault\n~~~",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := renderPanel(tt.node)
			if got != tt.want {
				t.Errorf("renderPanel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderExpand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		node map[string]any
		want string
	}{
		{
			name: "With_Title",
			node: map[string]any{
				"type":  "expand",
				"attrs": map[string]any{"title": "Click to expand"},
				"content": []any{
					map[string]any{
						"type":    "paragraph",
						"content": []any{map[string]any{"type": "text", "text": "Hidden content"}},
					},
				},
			},
			want: "~~~expand title=\"Click to expand\"\nHidden content\n~~~",
		},
		{
			name: "No_Title",
			node: map[string]any{
				"type": "expand",
				"content": []any{
					map[string]any{
						"type":    "paragraph",
						"content": []any{map[string]any{"type": "text", "text": "Expandable"}},
					},
				},
			},
			want: "~~~expand\nExpandable\n~~~",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := renderExpand(tt.node)
			if got != tt.want {
				t.Errorf("renderExpand() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderText(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		node map[string]any
		want string
	}{
		{
			name: "Plain_Text",
			node: map[string]any{"type": "text", "text": "Hello"},
			want: "Hello",
		},
		{
			name: "Bold",
			node: map[string]any{
				"type":  "text",
				"text":  "Bold",
				"marks": []any{map[string]any{"type": "strong"}},
			},
			want: "**Bold**",
		},
		{
			name: "Italic",
			node: map[string]any{
				"type":  "text",
				"text":  "Italic",
				"marks": []any{map[string]any{"type": "em"}},
			},
			want: "*Italic*",
		},
		{
			name: "Code",
			node: map[string]any{
				"type":  "text",
				"text":  "code",
				"marks": []any{map[string]any{"type": "code"}},
			},
			want: "`code`",
		},
		{
			name: "Strike",
			node: map[string]any{
				"type":  "text",
				"text":  "deleted",
				"marks": []any{map[string]any{"type": "strike"}},
			},
			want: "~~deleted~~",
		},
		{
			name: "Underline",
			node: map[string]any{
				"type":  "text",
				"text":  "underlined",
				"marks": []any{map[string]any{"type": "underline"}},
			},
			want: "<u>underlined</u>",
		},
		{
			name: "Link",
			node: map[string]any{
				"type": "text",
				"text": "Click here",
				"marks": []any{
					map[string]any{
						"type":  "link",
						"attrs": map[string]any{"href": "https://example.com"},
					},
				},
			},
			want: "[Click here](https://example.com)",
		},
		{
			name: "Bold_And_Italic",
			node: map[string]any{
				"type": "text",
				"text": "Both",
				"marks": []any{
					map[string]any{"type": "strong"},
					map[string]any{"type": "em"},
				},
			},
			want: "***Both***",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := renderText(tt.node)
			if got != tt.want {
				t.Errorf("renderText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderEmoji(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		node map[string]any
		want string
	}{
		{
			name: "With_ShortCode",
			node: map[string]any{
				"type":  "emoji",
				"attrs": map[string]any{"shortName": ":smile:"},
			},
			want: ":smile:",
		},
		{
			name: "With_Text_Fallback",
			node: map[string]any{
				"type":  "emoji",
				"attrs": map[string]any{"text": "\U0001F600"},
			},
			want: "\U0001F600",
		},
		{
			name: "Prefer_ShortCode",
			node: map[string]any{
				"type":  "emoji",
				"attrs": map[string]any{"shortName": ":grin:", "text": "\U0001F601"},
			},
			want: ":grin:",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := renderEmoji(tt.node)
			if got != tt.want {
				t.Errorf("renderEmoji() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderMention(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		node map[string]any
		want string
	}{
		{
			name: "User_Mention",
			node: map[string]any{
				"type":  "mention",
				"attrs": map[string]any{"id": "user123", "text": "@John"},
			},
			want: "@[John](accountId:user123)",
		},
		{
			name: "No_ID",
			node: map[string]any{
				"type":  "mention",
				"attrs": map[string]any{"text": "@Unknown"},
			},
			want: "@Unknown",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := renderMention(tt.node)
			if got != tt.want {
				t.Errorf("renderMention() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderStatus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		node map[string]any
		want string
	}{
		{
			name: "With_Color",
			node: map[string]any{
				"type":  "status",
				"attrs": map[string]any{"text": "In Progress", "color": "blue"},
			},
			want: "{status:In Progress|color=blue}",
		},
		{
			name: "No_Color",
			node: map[string]any{
				"type":  "status",
				"attrs": map[string]any{"text": "Done"},
			},
			want: "{status:Done}",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := renderStatus(tt.node)
			if got != tt.want {
				t.Errorf("renderStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderDate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		node map[string]any
		want string
	}{
		{
			name: "Valid_Timestamp",
			node: map[string]any{
				"type":  "date",
				"attrs": map[string]any{"timestamp": "1704067200000"},
			},
			want: "{date:2024-01-01}",
		},
		{
			name: "No_Timestamp",
			node: map[string]any{
				"type":  "date",
				"attrs": map[string]any{},
			},
			want: "",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := renderDate(tt.node)
			if got != tt.want {
				t.Errorf("renderDate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderInlineCard(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		node map[string]any
		want string
	}{
		{
			name: "With_URL",
			node: map[string]any{
				"type":  "inlineCard",
				"attrs": map[string]any{"url": "https://example.com/page"},
			},
			want: "{card:https://example.com/page}",
		},
		{
			name: "No_URL",
			node: map[string]any{
				"type":  "inlineCard",
				"attrs": map[string]any{},
			},
			want: "",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := renderInlineCard(tt.node)
			if got != tt.want {
				t.Errorf("renderInlineCard() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderADFNode_Rule(t *testing.T) {
	t.Parallel()
	node := map[string]any{"type": "rule"}
	got := renderADFNode(node, 0)
	if got != "---" {
		t.Errorf("renderADFNode(rule) = %q, want %q", got, "---")
	}
}

func TestRenderADFNode_HardBreak(t *testing.T) {
	t.Parallel()
	node := map[string]any{"type": "hardBreak"}
	got := renderADFNode(node, 0)
	if got != "  \n" {
		t.Errorf("renderADFNode(hardBreak) = %q, want %q", got, "  \\n")
	}
}

func TestRenderTable(t *testing.T) {
	t.Parallel()
	node := map[string]any{
		"type": "table",
		"content": []any{
			map[string]any{
				"type": "tableRow",
				"content": []any{
					map[string]any{
						"type": "tableHeader",
						"content": []any{
							map[string]any{
								"type":    "paragraph",
								"content": []any{map[string]any{"type": "text", "text": "Header 1"}},
							},
						},
					},
					map[string]any{
						"type": "tableHeader",
						"content": []any{
							map[string]any{
								"type":    "paragraph",
								"content": []any{map[string]any{"type": "text", "text": "Header 2"}},
							},
						},
					},
				},
			},
			map[string]any{
				"type": "tableRow",
				"content": []any{
					map[string]any{
						"type": "tableCell",
						"content": []any{
							map[string]any{
								"type":    "paragraph",
								"content": []any{map[string]any{"type": "text", "text": "Cell 1"}},
							},
						},
					},
					map[string]any{
						"type": "tableCell",
						"content": []any{
							map[string]any{
								"type":    "paragraph",
								"content": []any{map[string]any{"type": "text", "text": "Cell 2"}},
							},
						},
					},
				},
			},
		},
	}

	got := renderTable(node)
	if !strings.Contains(got, "Header 1") || !strings.Contains(got, "Header 2") {
		t.Errorf("renderTable() missing headers, got: %q", got)
	}
	if !strings.Contains(got, "Cell 1") || !strings.Contains(got, "Cell 2") {
		t.Errorf("renderTable() missing cells, got: %q", got)
	}
	if !strings.Contains(got, "---") {
		t.Errorf("renderTable() missing separator, got: %q", got)
	}
}
