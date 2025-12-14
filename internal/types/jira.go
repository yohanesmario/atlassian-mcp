package types

// Jira-specific types

// JiraAddCommentParams represents parameters for adding a comment to a Jira issue.
type JiraAddCommentParams struct {
	Issue string `json:"issue"`
	Body  string `json:"body"`
}

// JiraUpdateIssueParams represents parameters for updating a Jira issue.
type JiraUpdateIssueParams struct {
	Issue     string            `json:"issue"`
	Fields    map[string]any    `json:"fields"`
	Checksums map[string]string `json:"checksums"`
}

// JiraCreateIssueParams represents parameters for creating a Jira issue.
type JiraCreateIssueParams struct {
	Project     string `json:"project"`
	IssueType   string `json:"issuetype"`
	Summary     string `json:"summary"`
	Description string `json:"description,omitempty"`
}

// JiraAttachmentInfo represents metadata from a Jira attachment upload.
type JiraAttachmentInfo struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	MediaID  string // Extracted from content URL
	MimeType string `json:"mimeType"`
	Content  string `json:"content"` // URL to file content
}

// JiraReadVerbHelp maps read verbs to their help text.
var JiraReadVerbHelp = map[string]string{
	"get_issue": `Get issue details. Param: issue key or URL (e.g., PROJ-123)

Returns: summary, status, type, priority, assignee, reporter, labels, components, parent, dates, description, subtasks, linked issues.

Roundtrip formats in output (copy into jira_add_comment/jira_update_issue):
- Mentions: @[Name](accountId:xxx)
- Media: ![alt](jira-media:id:collection:type)

Returns __CHECKSUMS__ section with SHA256 hashes for: summary, description, status, assignee, priority, labels, components. Required for jira_update_issue.`,
	"get_comments": `Get issue comments. Param: issue key or URL

Returns up to 50 comments (oldest first) with author, timestamp, and body in markdown.`,
	"search": `Search issues with JQL. Param: JQL query string

Example: assignee=currentUser() AND status=Open
Returns up to 50 issues with: key, type, summary, status, assignee.

JQL Reference: https://support.atlassian.com/jira-software-cloud/docs/use-advanced-search-with-jira-query-language-jql/`,
}

// JiraFormatDocumentation contains the full extended markdown syntax reference for Jira.
const JiraFormatDocumentation = `# Extended Markdown Format Reference (Jira)

This MCP uses an extended markdown format that supports Jira-specific features.
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

## Jira Extensions

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

**New image from URL (auto-uploaded to issue):**
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

Note: Image uploads work in issue descriptions only, not in comments.

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

### Text Color
    {color:#ff0000}Red text{color}

### Emoji
    :smile: :thumbsup: :warning: :check_mark: :cross_mark:

---

## Tips

1. **Roundtrip preservation**: Content from jira_get_issue uses this format.
   Copy as-is into jira_update_issue to preserve formatting.

2. **Nested content**: Panels and expand blocks support full markdown inside,
   including lists, code blocks, and other blocks.

3. **Media uploads**: URLs and local paths in descriptions are automatically
   uploaded as attachments when you update an issue. Max 10MB per file.
`

// JiraWriteVerbHelp maps write verbs to their help text.
var JiraWriteVerbHelp = map[string]string{
	"add_comment": `Add comment to issue. Param: {"issue": "PROJ-123", "body": "Comment text"}

Body supports markdown:
- Blocks: headings, code blocks (with lang), blockquotes, lists, tables, horizontal rules
- Inline: **bold**, *italic*, ~~strike~~, ` + "`code`" + `, [link](url)
- Mentions: @[Name](accountId:xxx) - use format from jira_get_issue output
- Existing media: ![alt](jira-media:id:collection:type)

Note: Image uploads not supported in comments. To add images, update the issue description.`,
	"update_issue": `Update issue fields. Param: {"issue": "PROJ-123", "fields": {...}, "checksums": {...}}

Workflow:
1. Call get_format to learn extended markdown syntax
2. Call jira_get_issue to get current values and checksums
3. Include checksum for each field you update
4. If field changed since read, returns conflict error

Checksum fields: summary, description, status, assignee, priority, labels, components

Image uploads supported:
- New: ![alt](url) or ![alt](/path) - auto-uploaded as attachment (10MB limit)
- Existing: ![alt](jira-media:id:collection:type) from jira_get_issue

Returns fresh checksums on success.`,
	"create_issue": `Create new issue. Param: {"project": "PROJ", "issuetype": "Task", "summary": "Title", "description": "Details"}

Workflow:
1. Call get_format to learn extended markdown syntax
2. Create issue with fields

Required: project (key), issuetype (name), summary
Optional: description (markdown)

To add images: create issue first, then use jira_update_issue with description containing ![alt](url).
Returns created issue key.`,
}
