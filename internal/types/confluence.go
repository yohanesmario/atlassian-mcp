package types

// Confluence-specific types

// ConfluenceCreatePageParams represents parameters for creating a Confluence page.
type ConfluenceCreatePageParams struct {
	SpaceID  string `json:"spaceId"`
	Title    string `json:"title"`
	Body     string `json:"body"`     // Markdown content
	ParentID string `json:"parentId"` // Optional parent page ID
}

// ConfluenceUpdatePageParams represents parameters for updating a Confluence page.
type ConfluenceUpdatePageParams struct {
	PageID    string            `json:"pageId"`
	Title     string            `json:"title"`     // Optional, empty means no change
	Body      string            `json:"body"`      // Optional, empty means no change
	Checksums map[string]string `json:"checksums"` // Required for conflict detection
}

// ConfluenceAddCommentParams represents parameters for adding a comment to a Confluence page.
type ConfluenceAddCommentParams struct {
	PageID string `json:"pageId"`
	Body   string `json:"body"` // Markdown content
}

// ConfluenceAttachmentInfo represents metadata from a Confluence attachment upload.
type ConfluenceAttachmentInfo struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	FileID   string `json:"fileId"`   // Used in ADF media nodes
	FileSize int64  `json:"fileSize"`
}

// ConfluenceReadVerbHelp maps read verbs to their help text.
var ConfluenceReadVerbHelp = map[string]string{
	"get_page": `Get page content. Param: page ID or URL

Returns: title, status, space, author, version, body (as markdown), checksums.

Roundtrip formats in output (copy into confluence_update_page):
- Mentions: @[Name](accountId:xxx)
- Dates: {date:2024-01-15}
- Status: {status:TODO|color=blue}

Returns __CHECKSUMS__ section with SHA256 hashes for: title, body, version.
Required for confluence_update_page.`,

	"get_comments": `Get page comments. Param: page ID or URL

Returns all comments with author, timestamp, and body in markdown.`,

	"search": `Search pages with CQL. Param: CQL query string

Example: space = DEV AND title ~ 'API'
Returns matching pages with: ID, title, space, status.

CQL Reference: https://developer.atlassian.com/cloud/confluence/cql-functions/`,
}

// ConfluenceFormatDocumentation contains the full extended markdown syntax reference for Confluence.
const ConfluenceFormatDocumentation = `# Extended Markdown Format Reference (Confluence)

This MCP uses an extended markdown format that supports Confluence-specific features.
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

## Confluence Extensions

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

### Inline Cards (Smart Links)
    {card:https://example.com/page}

### Text Color
    {color:#ff0000}Red text{color}

### Emoji
    :smile: :thumbsup: :warning: :check_mark: :cross_mark:

---

## Tips

1. **Roundtrip preservation**: Content from confluence_get_page uses this format.
   Copy as-is into confluence_update_page to preserve formatting.

2. **Nested content**: Panels and expand blocks support full markdown inside,
   including lists, code blocks, and other blocks.

3. **Media uploads**: URLs and local paths are automatically uploaded
   as attachments when you update/create a page.
`

// ConfluenceWriteVerbHelp maps write verbs to their help text.
var ConfluenceWriteVerbHelp = map[string]string{
	"add_comment": "Add comment to page. Param: {\"pageId\": \"123456\", \"body\": \"Comment text\"}\n\nBody supports markdown:\n- Blocks: headings, code blocks, blockquotes, lists, tables\n- Inline: **bold**, *italic*, ~~strike~~, `code`, [link](url)\n- Mentions: @[Name](accountId:xxx)",

	"update_page": `Update page. Param: {"pageId": "123456", "title": "New Title", "body": "Content", "checksums": {...}}

Workflow:
1. Call get_format to learn extended markdown syntax
2. Call confluence_get_page to get current values and checksums
3. Include checksums for fields you're updating
4. If page changed since read, returns conflict error

Checksum fields: title, body, version (all required)

Returns fresh checksums on success.`,

	"create_page": `Create new page. Param: {"spaceId": "123", "title": "Title", "body": "Content", "parentId": "456"}

Workflow:
1. Call get_format to learn extended markdown syntax
2. Create page with fields

Required: spaceId, title
Optional: body (markdown), parentId (for child pages)

Returns created page ID.`,
}
