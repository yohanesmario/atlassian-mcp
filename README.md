# _(Custom)_ Atlassian MCP

Unified Jira + Confluence Custom MCP server for Claude Code.

> [!NOTE]
> Weekend project - issues and PRs welcome, but patience appreciated :heart:

## :bulb: Design Philosophy

This MCP server exposes only **2 tools** (`atlassian_read` and `atlassian_write`) to minimize context usage at initial load. Detailed help for each verb is loaded on-demand by passing `param="help"`. The agent fetches documentation only when needed rather than having it embedded in tool descriptions.

This approach:
- Reduces initial token overhead from tool definitions
- Scales to many verbs without bloating context
- Lets the agent discover capabilities as needed

See [Available Tools](#hammer_and_wrench-available-tools) for the full list of supported verbs.

## :thinking: Why This Over the Official MCP?

### :white_check_mark: What You Gain

| Feature | This MCP | Official MCP |
|---------|----------|--------------|
| **Tool count** | 2 tools | 20+ tools |
| **On-demand help** | `param="help"` loads docs | All docs in tool descriptions |

**Key benefits:**
- **80%+ context savings** at startup - more room for actual work
- **Markdown in, Markdown out** - bi-directional Markdown to Atlassian Document Format (ADF) conversion with round-trip preservations, and it works with images too.
- **Checksum-based updates** - prevents accidental overwrites while requiring less token usage compared to the official MCP

### :warning: What You Lose

| Feature | Impact |
|---------|--------|
| **Granular permissions** | Can't allow Jira but deny Confluence - approving `atlassian_read` grants access to both |
| **Sprint/board operations** | Not implemented (PRs welcome) |
| **Jira issue creation fields** | Basic fields only (summary, description, type, project) |

## :gear: Setup

**Required environment variables:**

| Variable | Description |
|----------|-------------|
| `ATLASSIAN_EMAIL` | Your Atlassian account email |
| `ATLASSIAN_API_TOKEN` | API token from [Atlassian](https://id.atlassian.com/manage-profile/security/api-tokens) |
| `ATLASSIAN_DOMAIN` | e.g., `company.atlassian.net` |

Provide these via shell exports, systemd, or any method that makes them available when the binary runs.

**Optional:** Place them in a `.env` file in the binary's directory (environment variables take precedence over `.env`):

```bash
make setup-env  # Creates .env with secure permissions
# Then edit .env with your credentials
```

## :package: Build & Install

Build the binary first before installing your MCP server:

```bash
make build
```

With that done, you can add this to your `~/.claude.json` or `mcp.json`:

```json
{
  "mcpServers": {
    "atlassian-mcp": {
      "type": "stdio",
      "command": "</path/to/atlassian-mcp-dir>/atlassian-mcp-bin"
    }
  }
}
```

## :hammer_and_wrench: Available Tools

### `atlassian_read`

Read from Jira/Confluence. Pass `param="help"` to any verb for detailed usage.

| Verb | Description |
|------|-------------|
| `jira_get_issue` | Get issue details with checksums |
| `jira_get_comments` | Get issue comments |
| `jira_search` | Search issues with JQL |
| `confluence_get_page` | Get page content with checksums |
| `confluence_get_comments` | Get page comments |
| `confluence_search` | Search pages with CQL |
| `get_format` | Extended markdown syntax reference |
| `search_users` | Search users by name (for mentions) |

### `atlassian_write`

Write to Jira/Confluence. Pass `param="help"` to any verb for detailed usage.

| Verb | Description |
|------|-------------|
| `jira_add_comment` | Add comment to issue |
| `jira_update_issue` | Update issue fields (requires checksums) |
| `jira_create_issue` | Create new issue |
| `confluence_add_comment` | Add comment to page |
| `confluence_update_page` | Update page content (requires checksums) |
| `confluence_create_page` | Create new page |

## :sos: Troubleshooting

| Error | Cause | Solution |
|-------|-------|----------|
| `HTTP 401` | Invalid credentials | Verify `ATLASSIAN_EMAIL` and `ATLASSIAN_API_TOKEN` are correct |
| `HTTP 403` | No permission | Ensure API token has access to the project/space |
| Credentials not loaded from `.env` | `.env` has insecure permissions (silently skipped) | Run `chmod 600 .env` or `make setup-env` |
| `ATLASSIAN_DOMAIN must be an atlassian.net domain` | Wrong domain format | Use `company.atlassian.net`, not full URL |
| `ATLASSIAN_DOMAIN must be a domain only` | Included protocol or path | Remove `https://` and any path from domain |
| Checksum conflict error | Content changed since read | Re-read the content to get fresh checksums |
| `file exceeds size limit` | Attachment too large | Jira: 10MB max, Confluence: 25MB max |

## License

:balance_scale: [MIT](./LICENSE)
