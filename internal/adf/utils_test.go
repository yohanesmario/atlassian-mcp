package adf

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseAttrs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  map[string]string
	}{
		{name: "Empty_String", input: "", want: map[string]string{}},
		{name: "Single_Attr", input: `type="info"`, want: map[string]string{"type": "info"}},
		{name: "Multiple_Attrs", input: `type="info" title="Test Title"`, want: map[string]string{"type": "info", "title": "Test Title"}},
		{name: "With_Spaces_In_Value", input: `title="Hello World" type="note"`, want: map[string]string{"title": "Hello World", "type": "note"}},
		{name: "Empty_Value", input: `title=""`, want: map[string]string{"title": ""}},
		{name: "Mixed_Valid_Invalid", input: `valid="yes" invalid`, want: map[string]string{"valid": "yes"}},
		{name: "Numbers_In_Value", input: `width="100" height="200"`, want: map[string]string{"width": "100", "height": "200"}},
		{name: "Special_Chars_In_Value", input: `url="https://example.com/path?q=1"`, want: map[string]string{"url": "https://example.com/path?q=1"}},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ParseAttrs(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseAttrs(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatAttrsForFence(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		attrs map[string]any
		keys  []string
		want  string
	}{
		{name: "Empty_Attrs", attrs: map[string]any{}, keys: []string{"type"}, want: ""},
		{name: "Single_String", attrs: map[string]any{"type": "info"}, keys: []string{"type"}, want: ` type="info"`},
		{name: "Multiple_Keys", attrs: map[string]any{"type": "info", "title": "Test"}, keys: []string{"type", "title"}, want: ` type="info" title="Test"`},
		{name: "Missing_Key", attrs: map[string]any{"type": "info"}, keys: []string{"type", "missing"}, want: ` type="info"`},
		{name: "Float_Value", attrs: map[string]any{"width": 100.5}, keys: []string{"width"}, want: ` width="100.5"`},
		{name: "Int_Value", attrs: map[string]any{"count": 42}, keys: []string{"count"}, want: ` count="42"`},
		{name: "Bool_Value", attrs: map[string]any{"enabled": true}, keys: []string{"enabled"}, want: ` enabled="true"`},
		{name: "Empty_String_Value", attrs: map[string]any{"title": ""}, keys: []string{"title"}, want: ""},
		{name: "Order_Preserved", attrs: map[string]any{"a": "1", "b": "2", "c": "3"}, keys: []string{"c", "a", "b"}, want: ` c="3" a="1" b="2"`},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := FormatAttrsForFence(tt.attrs, tt.keys...)
			if got != tt.want {
				t.Errorf("FormatAttrsForFence() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEscapeMarkdown(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "No_Special_Chars", input: "Hello World", want: "Hello World"},
		{name: "Backslash", input: `a\b`, want: `a\\b`},
		{name: "Backtick", input: "code `here`", want: "code \\`here\\`"},
		{name: "Asterisk", input: "*bold*", want: "\\*bold\\*"},
		{name: "Underscore", input: "_italic_", want: "\\_italic\\_"},
		{name: "Curly_Braces", input: "{var}", want: "\\{var\\}"},
		{name: "Square_Brackets", input: "[link]", want: "\\[link\\]"},
		{name: "Parentheses", input: "(url)", want: "\\(url\\)"},
		{name: "Hash", input: "# heading", want: "\\# heading"},
		{name: "Plus", input: "+ item", want: "\\+ item"},
		{name: "Dash", input: "- list", want: "\\- list"},
		{name: "Dot", input: "1. item", want: "1\\. item"},
		{name: "Exclamation", input: "![img]", want: "\\!\\[img\\]"},
		{name: "Pipe", input: "a|b|c", want: "a\\|b\\|c"},
		{name: "All_Special", input: "\\`*_{}[]()#+-.!|", want: "\\\\\\`\\*\\_\\{\\}\\[\\]\\(\\)\\#\\+\\-\\.\\!\\|"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := EscapeMarkdown(tt.input)
			if got != tt.want {
				t.Errorf("EscapeMarkdown(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestUnescapeMarkdown(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "No_Escapes", input: "Hello World", want: "Hello World"},
		{name: "Escaped_Backslash", input: `a\\b`, want: `a\b`},
		{name: "Escaped_Backtick", input: "code \\`here\\`", want: "code \\`here\\`"},
		{name: "Escaped_Asterisk", input: "\\*bold\\*", want: "*bold*"},
		{name: "Escaped_Underscore", input: "\\_italic\\_", want: "_italic_"},
		{name: "Escaped_Braces", input: "\\{var\\}", want: "{var}"},
		{name: "Escaped_Brackets", input: "\\[link\\]", want: "[link]"},
		{name: "Escaped_Parens", input: "\\(url\\)", want: "(url)"},
		{name: "Escaped_Hash", input: "\\# heading", want: "# heading"},
		{name: "Invalid_Escape", input: "\\n newline", want: "\\n newline"},
		{name: "Trailing_Backslash", input: "end\\", want: "end\\"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := UnescapeMarkdown(tt.input)
			if got != tt.want {
				t.Errorf("UnescapeMarkdown(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEscapeUnescapeRoundtrip(t *testing.T) {
	t.Parallel()
	// Note: backticks don't roundtrip because UnescapeMarkdown doesn't handle them
	tests := []string{
		"Hello World",
		"*bold* _italic_",
		"[link](url)",
		"# heading",
		"a|b|c",
		`path\to\file`,
	}
	for _, input := range tests {
		input := input
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			escaped := EscapeMarkdown(input)
			unescaped := UnescapeMarkdown(escaped)
			if unescaped != input {
				t.Errorf("roundtrip(%q) = %q (escaped: %q)", input, unescaped, escaped)
			}
		})
	}
}

func TestGenerateLocalID(t *testing.T) {
	t.Parallel()

	t.Run("Format_Contains_Hyphen", func(t *testing.T) {
		t.Parallel()
		id := GenerateLocalID()
		if !strings.Contains(id, "-") {
			t.Errorf("GenerateLocalID() = %q, expected hyphen separator", id)
		}
	})

	t.Run("Unique_IDs", func(t *testing.T) {
		t.Parallel()
		ids := make(map[string]bool)
		for i := 0; i < 100; i++ {
			id := GenerateLocalID()
			if ids[id] {
				t.Errorf("GenerateLocalID() produced duplicate: %q", id)
			}
			ids[id] = true
		}
	})

	t.Run("Non_Empty", func(t *testing.T) {
		t.Parallel()
		id := GenerateLocalID()
		if id == "" {
			t.Error("GenerateLocalID() returned empty string")
		}
	})
}

func TestParseTimestamp(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		wantFunc func(string) bool
	}{
		{
			name:  "Milliseconds_Passthrough",
			input: "1704067200000",
			wantFunc: func(got string) bool {
				return got == "1704067200000"
			},
		},
		{
			name:  "ISO_Date_Converted",
			input: "2024-01-01",
			wantFunc: func(got string) bool {
				// Should be milliseconds (13+ digits)
				return len(got) >= 10 && IsAllDigits(got)
			},
		},
		{
			name:  "RFC3339_Converted",
			input: "2024-01-01T00:00:00Z",
			wantFunc: func(got string) bool {
				return len(got) >= 10 && IsAllDigits(got)
			},
		},
		{
			name:  "Invalid_Passthrough",
			input: "not-a-date",
			wantFunc: func(got string) bool {
				return got == "not-a-date"
			},
		},
		{
			name:  "Empty_String",
			input: "",
			wantFunc: func(got string) bool {
				return got == ""
			},
		},
		{
			name:  "With_Whitespace",
			input: "  2024-01-01  ",
			wantFunc: func(got string) bool {
				return len(got) >= 10 && IsAllDigits(got)
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ParseTimestamp(tt.input)
			if !tt.wantFunc(got) {
				t.Errorf("ParseTimestamp(%q) = %q, validation failed", tt.input, got)
			}
		})
	}
}

func TestFormatTimestamp(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "Valid_Milliseconds", input: "1704067200000", want: "2024-01-01"},
		{name: "Zero_Timestamp", input: "0", want: "1970-01-01"},
		{name: "Invalid_String", input: "not-a-number", want: "not-a-number"},
		{name: "Empty_String", input: "", want: ""},
		{name: "With_Whitespace", input: "  1704067200000  ", want: "2024-01-01"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := FormatTimestamp(tt.input)
			if got != tt.want {
				t.Errorf("FormatTimestamp(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsAllDigits(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "All_Digits", input: "12345", want: true},
		{name: "Single_Digit", input: "0", want: true},
		{name: "With_Letters", input: "123abc", want: false},
		{name: "With_Space", input: "123 456", want: false},
		{name: "Empty_String", input: "", want: false},
		{name: "With_Dash", input: "123-456", want: false},
		{name: "With_Dot", input: "123.456", want: false},
		{name: "Long_Number", input: "1234567890123456789", want: true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := IsAllDigits(tt.input)
			if got != tt.want {
				t.Errorf("IsAllDigits(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetIndentLevel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{name: "No_Indent", input: "text", want: 0},
		{name: "Two_Spaces", input: "  text", want: 1},
		{name: "Four_Spaces", input: "    text", want: 2},
		{name: "Six_Spaces", input: "      text", want: 3},
		{name: "One_Tab", input: "\ttext", want: 1},
		{name: "Two_Tabs", input: "\t\ttext", want: 2},
		{name: "Mixed_Spaces_Tab", input: "  \ttext", want: 2},
		{name: "Empty_String", input: "", want: 0},
		{name: "Only_Spaces", input: "    ", want: 2},
		{name: "Odd_Spaces", input: "   text", want: 1},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := GetIndentLevel(tt.input)
			if got != tt.want {
				t.Errorf("GetIndentLevel(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestTrimIndent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		line   string
		levels int
		want   string
	}{
		{name: "No_Trim", line: "text", levels: 0, want: "text"},
		{name: "Trim_One_Level", line: "  text", levels: 1, want: "text"},
		{name: "Trim_Two_Levels", line: "    text", levels: 2, want: "text"},
		{name: "Trim_More_Than_Exists", line: "  text", levels: 3, want: "text"},
		{name: "Trim_Tab", line: "\ttext", levels: 1, want: "text"},
		{name: "Partial_Trim", line: "    text", levels: 1, want: "  text"},
		{name: "Empty_Result", line: "    ", levels: 3, want: ""},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := TrimIndent(tt.line, tt.levels)
			if got != tt.want {
				t.Errorf("TrimIndent(%q, %d) = %q, want %q", tt.line, tt.levels, got, tt.want)
			}
		})
	}
}

func TestSplitStatusAttrs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  map[string]string
	}{
		{name: "Empty_String", input: "", want: map[string]string{}},
		{name: "Single_Attr", input: "color=blue", want: map[string]string{"color": "blue"}},
		{name: "Multiple_Attrs", input: "color=blue,style=bold", want: map[string]string{"color": "blue", "style": "bold"}},
		{name: "With_Spaces", input: " color = blue , style = bold ", want: map[string]string{"color": "blue", "style": "bold"}},
		{name: "No_Value", input: "color", want: map[string]string{}},
		{name: "Empty_Value", input: "color=", want: map[string]string{"color": ""}},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := SplitStatusAttrs(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SplitStatusAttrs(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeWhitespace(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "No_Changes", input: "Hello\n\nWorld", want: "Hello\n\nWorld"},
		{name: "Triple_Newlines", input: "Hello\n\n\nWorld", want: "Hello\n\nWorld"},
		{name: "Many_Newlines", input: "Hello\n\n\n\n\nWorld", want: "Hello\n\nWorld"},
		{name: "Trailing_Spaces", input: "Hello   \nWorld   ", want: "Hello\nWorld"},
		{name: "Trailing_Tabs", input: "Hello\t\t\nWorld", want: "Hello\nWorld"},
		{name: "Leading_Trailing", input: "  \n\nHello\n\n  ", want: "Hello"},
		{name: "Empty_String", input: "", want: ""},
		{name: "Only_Whitespace", input: "   \n\n\n   ", want: ""},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NormalizeWhitespace(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeWhitespace(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
