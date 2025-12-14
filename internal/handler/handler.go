package handler

import (
	"encoding/json"
	"strings"

	"atlassian-mcp/internal/types"
	"atlassian-mcp/internal/users"
)

// HandleRequest routes MCP requests to appropriate handlers.
func HandleRequest(req types.Request) types.Response {
	switch req.Method {
	case "initialize":
		return types.Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]any{
					"tools": map[string]any{},
				},
				"serverInfo": map[string]any{
					"name":    "atlassian-mcp",
					"version": "1.0.0",
				},
			},
		}

	case "notifications/initialized":
		return types.Response{}

	case "tools/list":
		return types.Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  handleToolsList(),
		}

	case "tools/call":
		var params types.ToolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return types.Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &types.Error{Code: -32602, Message: "Invalid params"},
			}
		}
		return types.Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  handleToolCall(params),
		}

	default:
		return types.Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &types.Error{Code: -32601, Message: "Method not found"},
		}
	}
}

// handleToolsList returns the list of available tools.
func handleToolsList() any {
	return map[string]any{
		"tools": []types.Tool{
			{
				Name:        "atlassian_read",
				Description: "Read from Jira/Confluence. Verbs: jira_get_issue, jira_get_comments, jira_search, confluence_get_page, confluence_get_comments, confluence_search, get_format, search_users. IMPORTANT: Call with param=\"help\" first to learn verb usage.",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"verb": map[string]any{
							"type":        "string",
							"description": "Operation: jira_get_issue, jira_get_comments, jira_search, confluence_get_page, confluence_get_comments, confluence_search, get_format, search_users",
						},
						"param": map[string]any{
							"type":        "string",
							"description": "Issue key/URL, page ID/URL, query, or \"help\" for usage",
						},
					},
					"required": []string{"verb", "param"},
				},
			},
			{
				Name:        "atlassian_write",
				Description: "Write to Jira/Confluence. Verbs: jira_add_comment, jira_update_issue, jira_create_issue, confluence_add_comment, confluence_update_page, confluence_create_page. IMPORTANT: Call with param=\"help\" first to learn verb usage.",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"verb": map[string]any{
							"type":        "string",
							"description": "Operation: jira_add_comment, jira_update_issue, jira_create_issue, confluence_add_comment, confluence_update_page, confluence_create_page",
						},
						"param": map[string]any{
							"type":        "string",
							"description": "JSON params or \"help\" for usage",
						},
					},
					"required": []string{"verb", "param"},
				},
			},
		},
	}
}

// handleToolCall dispatches tool calls to appropriate handlers.
func handleToolCall(params types.ToolCallParams) any {
	var args types.VerbArgs
	if err := json.Unmarshal(params.Arguments, &args); err != nil {
		return errorResult("Invalid arguments: must provide verb and param")
	}

	switch params.Name {
	case "atlassian_read":
		return handleAtlassianRead(args)
	case "atlassian_write":
		return handleAtlassianWrite(args)
	default:
		return errorResult("Unknown tool: " + params.Name)
	}
}

// handleAtlassianRead routes read operations to the appropriate service.
func handleAtlassianRead(args types.VerbArgs) any {
	// Help handling - show all available verbs
	if args.Param == "help" {
		return handleReadHelp(args.Verb)
	}

	// Handle unified verbs (no service prefix needed)
	if args.Verb == "get_format" {
		return successResult(types.FormatDocumentation)
	}
	if args.Verb == "search_users" {
		result, err := users.SearchUsers(args.Param)
		if err != nil {
			return errorResult(err.Error())
		}
		return successResult(result)
	}

	// Parse service prefix from verb (e.g., "jira_get_issue" -> "jira", "get_issue")
	service, operation := parseVerb(args.Verb)

	switch service {
	case "jira":
		return handleJiraRead(operation, args.Param)
	case "confluence":
		return handleConfluenceRead(operation, args.Param)
	default:
		return errorResult("Unknown service prefix in verb: " + args.Verb + ". Use jira_ or confluence_ prefix, or use get_format.")
	}
}

// handleAtlassianWrite routes write operations to the appropriate service.
func handleAtlassianWrite(args types.VerbArgs) any {
	// Help handling - show all available verbs
	if args.Param == "help" {
		return handleWriteHelp(args.Verb)
	}

	// Parse service prefix from verb
	service, operation := parseVerb(args.Verb)

	switch service {
	case "jira":
		return handleJiraWrite(operation, args.Param)
	case "confluence":
		return handleConfluenceWrite(operation, args.Param)
	default:
		return errorResult("Unknown service prefix in verb: " + args.Verb + ". Use jira_ or confluence_ prefix.")
	}
}

// parseVerb splits "jira_get_issue" into ("jira", "get_issue")
func parseVerb(verb string) (service, operation string) {
	parts := strings.SplitN(verb, "_", 2)
	if len(parts) != 2 {
		return "", verb
	}
	return parts[0], parts[1]
}

// handleReadHelp returns help for read verbs.
func handleReadHelp(verb string) any {
	// If a specific verb is given, show its help
	if verb != "" {
		// Handle unified verbs
		if verb == "get_format" {
			return successResult("Get extended markdown format documentation. Param: ignored\n\nReturns full syntax reference for the extended markdown format used by this MCP.")
		}
		if verb == "search_users" {
			return successResult(types.SearchUsersHelp)
		}

		service, operation := parseVerb(verb)
		switch service {
		case "jira":
			if help, ok := types.JiraReadVerbHelp[operation]; ok {
				return successResult(help)
			}
		case "confluence":
			if help, ok := types.ConfluenceReadVerbHelp[operation]; ok {
				return successResult(help)
			}
		}
	}

	// Show all available read verbs
	var sb strings.Builder
	sb.WriteString("Available read verbs:\n\n")

	sb.WriteString("**Jira:**\n")
	for v := range types.JiraReadVerbHelp {
		sb.WriteString("- jira_" + v + "\n")
	}

	sb.WriteString("\n**Confluence:**\n")
	for v := range types.ConfluenceReadVerbHelp {
		sb.WriteString("- confluence_" + v + "\n")
	}

	sb.WriteString("\n**Shared:**\n")
	sb.WriteString("- get_format\n")
	sb.WriteString("- search_users\n")

	return successResult(sb.String())
}

// handleWriteHelp returns help for write verbs.
func handleWriteHelp(verb string) any {
	// If a specific verb is given, show its help
	if verb != "" {
		service, operation := parseVerb(verb)
		switch service {
		case "jira":
			if help, ok := types.JiraWriteVerbHelp[operation]; ok {
				return successResult(help)
			}
		case "confluence":
			if help, ok := types.ConfluenceWriteVerbHelp[operation]; ok {
				return successResult(help)
			}
		}
	}

	// Show all available write verbs
	var sb strings.Builder
	sb.WriteString("Available write verbs:\n\n")

	sb.WriteString("**Jira:**\n")
	for v := range types.JiraWriteVerbHelp {
		sb.WriteString("- jira_" + v + "\n")
	}

	sb.WriteString("\n**Confluence:**\n")
	for v := range types.ConfluenceWriteVerbHelp {
		sb.WriteString("- confluence_" + v + "\n")
	}

	return successResult(sb.String())
}

// successResult creates a successful tool result.
func successResult(text string) map[string]any {
	return map[string]any{
		"content": []types.TextContent{{Type: "text", Text: text}},
	}
}

// errorResult creates an error tool result.
func errorResult(text string) map[string]any {
	return map[string]any{
		"isError": true,
		"content": []types.TextContent{{Type: "text", Text: text}},
	}
}
