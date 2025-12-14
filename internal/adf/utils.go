package adf

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Extended markdown regex patterns for parsing
var (
	// FenceBlockRe matches extended fence blocks like ~~~panel type=info
	FenceBlockRe = regexp.MustCompile(`^~~~(\w+)(?:\s+(.*))?$`)

	// FenceCloseRe matches fence block closing
	FenceCloseRe = regexp.MustCompile(`^~~~\s*$`)

	// MetadataCommentRe matches ADF metadata comments like <!-- adf:paragraph textAlign="center" -->
	MetadataCommentRe = regexp.MustCompile(`<!--\s*adf:(\w+)\s+(.+?)\s*-->`)

	// AttrPairRe matches key="value" or key=value pairs in attribute strings
	AttrPairRe = regexp.MustCompile(`(\w+)=(?:"([^"]*)"|([^\s"]+))`)

	// Extended inline syntax patterns
	ExtMentionRe = regexp.MustCompile(`\{user:([^}]+)\}`)
	ExtDateRe    = regexp.MustCompile(`\{date:([^}]+)\}`)
	ExtStatusRe  = regexp.MustCompile(`\{status:([^|}]+)(?:\|([^}]+))?\}`)
	ExtCardRe    = regexp.MustCompile(`\{card:([^}]+)\}`)
	ExtColorRe   = regexp.MustCompile(`\{color:([^}]+)\}(.+?)\{color\}`)
	EmojiCodeRe  = regexp.MustCompile(`:([a-z0-9_+-]+):`)

	// Task list pattern: - [x] or - [ ]
	TaskItemRe = regexp.MustCompile(`^(\s*)- \[([ xX])\]\s+(.*)$`)

	// Nested list indentation pattern
	ListIndentRe = regexp.MustCompile(`^(\s*)([*+-]|\d+\.)\s+(.*)$`)
)

// PanelEmoji maps panel types to their emoji representations for lossy conversion fallback.
var PanelEmoji = map[string]string{
	"info":    "info",
	"note":    "note",
	"warning": "warning",
	"success": "success",
	"error":   "error",
}

// StatusColors contains valid status colors for ADF status nodes.
var StatusColors = map[string]bool{
	"neutral": true,
	"purple":  true,
	"blue":    true,
	"green":   true,
	"yellow":  true,
	"red":     true,
}

// ParseAttrs parses a space-separated string of key="value" or key=value pairs into a map.
func ParseAttrs(attrStr string) map[string]string {
	result := make(map[string]string)
	matches := AttrPairRe.FindAllStringSubmatch(attrStr, -1)
	for _, match := range matches {
		if len(match) >= 4 {
			// match[1] = key, match[2] = quoted value, match[3] = unquoted value
			value := match[2]
			if value == "" {
				value = match[3]
			}
			result[match[1]] = value
		}
	}
	return result
}

// FormatAttrsForFence formats attributes for extended fence block output.
func FormatAttrsForFence(attrs map[string]any, keys ...string) string {
	var parts []string
	for _, key := range keys {
		if val, ok := attrs[key]; ok {
			switch v := val.(type) {
			case string:
				if v != "" {
					parts = append(parts, fmt.Sprintf(`%s="%s"`, key, v))
				}
			case float64:
				parts = append(parts, fmt.Sprintf(`%s="%v"`, key, v))
			case int:
				parts = append(parts, fmt.Sprintf(`%s="%d"`, key, v))
			case bool:
				parts = append(parts, fmt.Sprintf(`%s="%t"`, key, v))
			}
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return " " + strings.Join(parts, " ")
}

// EscapeMarkdown escapes special markdown characters in text.
func EscapeMarkdown(text string) string {
	// Characters that need escaping in markdown contexts
	replacer := strings.NewReplacer(
		`\`, `\\`,
		"`", "\\`",
		"*", "\\*",
		"_", "\\_",
		"{", "\\{",
		"}", "\\}",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		".", "\\.",
		"!", "\\!",
		"|", "\\|",
	)
	return replacer.Replace(text)
}

// UnescapeMarkdown removes backslash escapes from markdown text.
func UnescapeMarkdown(text string) string {
	// Process escaped characters
	var result strings.Builder
	i := 0
	for i < len(text) {
		if i+1 < len(text) && text[i] == '\\' {
			next := text[i+1]
			if strings.ContainsRune(`\`+"*_{}[]()#+-.!|", rune(next)) {
				result.WriteByte(next)
				i += 2
				continue
			}
		}
		result.WriteByte(text[i])
		i++
	}
	return result.String()
}

// GenerateLocalID generates a unique local ID for ADF nodes.
func GenerateLocalID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x-%x", time.Now().UnixNano()&0xFFFFFFFF, b)
}

// ParseTimestamp converts date input (ISO date or milliseconds) to ADF timestamp format.
func ParseTimestamp(input string) string {
	input = strings.TrimSpace(input)

	// If it looks like milliseconds already (all digits, long enough)
	if len(input) >= 10 && IsAllDigits(input) {
		return input
	}

	// Try parsing as ISO date (YYYY-MM-DD)
	if t, err := time.Parse("2006-01-02", input); err == nil {
		return fmt.Sprintf("%d", t.UnixMilli())
	}

	// Try parsing as ISO datetime
	if t, err := time.Parse(time.RFC3339, input); err == nil {
		return fmt.Sprintf("%d", t.UnixMilli())
	}

	// Return as-is if can't parse
	return input
}

// FormatTimestamp converts ADF timestamp (milliseconds) to ISO date string.
func FormatTimestamp(timestamp string) string {
	timestamp = strings.TrimSpace(timestamp)

	// Parse as milliseconds
	var ms int64
	if _, err := fmt.Sscanf(timestamp, "%d", &ms); err == nil {
		t := time.UnixMilli(ms)
		return t.Format("2006-01-02")
	}

	// Return as-is if can't parse
	return timestamp
}

// IsAllDigits checks if a string contains only digit characters.
func IsAllDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// GetIndentLevel calculates the indentation level from leading whitespace.
func GetIndentLevel(line string) int {
	spaces := 0
	for _, c := range line {
		if c == ' ' {
			spaces++
		} else if c == '\t' {
			spaces += 2 // Treat tab as 2 spaces
		} else {
			break
		}
	}
	return spaces / 2 // Each level is 2 spaces
}

// TrimIndent removes a specific number of indent levels from a line.
func TrimIndent(line string, levels int) string {
	toRemove := levels * 2
	removed := 0
	for i, c := range line {
		if removed >= toRemove {
			return line[i:]
		}
		if c == ' ' {
			removed++
		} else if c == '\t' {
			removed += 2
		} else {
			return line[i:]
		}
	}
	return ""
}

// SplitStatusAttrs parses status attributes from "color=blue" format.
func SplitStatusAttrs(attrStr string) map[string]string {
	result := make(map[string]string)
	if attrStr == "" {
		return result
	}
	pairs := strings.Split(attrStr, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return result
}

// NormalizeWhitespace cleans up excessive whitespace in output.
func NormalizeWhitespace(text string) string {
	// Replace multiple consecutive blank lines with double newline
	re := regexp.MustCompile(`\n{3,}`)
	text = re.ReplaceAllString(text, "\n\n")

	// Trim trailing whitespace from each line
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}
