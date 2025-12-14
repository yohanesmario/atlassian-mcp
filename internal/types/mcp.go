package types

import "encoding/json"

// MCP JSON-RPC types

// Request represents a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

// Error represents a JSON-RPC 2.0 error.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Tool represents an MCP tool definition.
type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

// ToolCallParams represents parameters for a tool call.
type ToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// TextContent represents text content in a tool response.
type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// VerbArgs represents verb-based dispatching arguments.
type VerbArgs struct {
	Verb  string `json:"verb"`
	Param string `json:"param"`
}

// SearchUsersHelp contains help text for the search_users verb.
const SearchUsersHelp = `Search for users by name or email. Param: search query

Example: search_users with param "John" or "john@example.com"

Returns matching users with:
- Display name
- Account ID
- Ready-to-use mention format: @[Name](accountId:xxx)

Use the mention format in comments, descriptions, or page content.`

// FormatDocumentation contains the unified extended markdown syntax reference.
const FormatDocumentation = `# Extended Markdown Format Reference

This MCP uses an extended markdown format that supports Atlassian-specific features.
All standard markdown is supported, plus the extensions below.

## Standard Markdown

### Headings
    # Heading 1
    ## Heading 2
    ### Heading 3 (up to 6 levels)

### Text Formatting
    **bold text** or __bold__
    *italic text* or _italic_
    ~~strikethrough~~
    ` + "`" + `inline code` + "`" + `
    <u>underline</u>
    <sub>subscript</sub>
    <sup>superscript</sup>

### Links
    [Link text](https://example.com)
    [Link with title](https://example.com "Title")

### Lists
    - Unordered item 1
    - Unordered item 2
      - Nested item (2 spaces indent)
        - Deeper nested (4 spaces)

    1. Ordered item 1
    2. Ordered item 2
       1. Nested ordered (3 spaces indent)

### Task Lists
    - [x] Completed task
    - [ ] Incomplete task

### Code Blocks
    ` + "```" + `python
    def hello():
        print("Hello!")
    ` + "```" + `

### Blockquotes
    > This is a quote
    > spanning multiple lines

### Tables
    | Header 1 | Header 2 |
    |----------|----------|
    | Cell 1   | Cell 2   |

### Horizontal Rule
    ---

---

## Atlassian Extensions

### Panels
    ~~~panel type=info
    Info panel content
    ~~~

Types: info, note, warning, error, success

### Expand/Collapse Sections
    ~~~expand title="Click to expand"
    Hidden content here.
    Supports any markdown inside.
    ~~~

### Media/Images

**Existing attachment (roundtrip format):**
    ~~~mediaSingle layout=center width=480 widthType=pixel
    ![alt text](jira-media:file-id:collection:file)
    ~~~

**New image from URL (auto-uploaded):**
    ~~~mediaSingle layout=center width=50
    ![description](https://example.com/image.png)
    ~~~

**New image from local file (auto-uploaded):**
    ~~~mediaSingle width=75
    ![description](/path/to/local/image.png)
    ~~~

**Standalone image (simple format):**
    ![alt text](https://example.com/image.png)

Layout options: align-start, align-end, center, wide, full-width, wrap-left, wrap-right

Note: In Jira, image uploads work in issue descriptions only, not in comments.

### Mentions
    @[Name](accountId:xxx)

Example: @[John Doe](accountId:123456:xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)

### Dates
    {date:2024-01-15}

### Status Lozenges
    {status:TODO|color=blue}
    {status:IN PROGRESS|color=yellow}
    {status:DONE|color=green}

Colors: neutral, purple, blue, green, yellow, red

### Inline Cards (Smart Links) - Confluence only
    {card:https://example.com/page}

### Text Color
    {color:#ff0000}Red text{color}

### Emoji
    :smile: :thumbsup: :warning: :check_mark: :cross_mark:

---

## Tips

1. **Roundtrip preservation**: Content from get_issue/get_page uses this format.
   Copy as-is into update_issue/update_page to preserve formatting.

2. **Nested content**: Panels and expand blocks support full markdown inside,
   including lists, code blocks, and other blocks.

3. **Media uploads**: URLs and local paths are automatically uploaded
   as attachments when you update/create content. Max 10MB (Jira) / 25MB (Confluence).
`
