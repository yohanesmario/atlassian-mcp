package handler

import (
	"encoding/json"

	"atlassian-mcp/internal/config"
	"atlassian-mcp/internal/jira"
	"atlassian-mcp/internal/types"
)

// handleJiraRead handles Jira read operations.
func handleJiraRead(operation, param string) any {
	switch operation {
	case "get_issue":
		issueKey, err := config.ExtractIssueKey(param)
		if err != nil {
			return errorResult(err.Error())
		}
		result, err := jira.FetchIssue(issueKey)
		if err != nil {
			return errorResult(err.Error())
		}
		return successResult(result)

	case "get_comments":
		issueKey, err := config.ExtractIssueKey(param)
		if err != nil {
			return errorResult(err.Error())
		}
		result, err := jira.FetchComments(issueKey)
		if err != nil {
			return errorResult(err.Error())
		}
		return successResult(result)

	case "search":
		result, err := jira.SearchIssues(param)
		if err != nil {
			return errorResult(err.Error())
		}
		return successResult(result)

	default:
		return errorResult("Unknown Jira read operation: " + operation + ". Valid: get_issue, get_comments, search")
	}
}

// handleJiraWrite handles Jira write operations.
func handleJiraWrite(operation, param string) any {
	switch operation {
	case "add_comment":
		var p types.JiraAddCommentParams
		if err := json.Unmarshal([]byte(param), &p); err != nil {
			return errorResult("Invalid JSON params: " + err.Error() + "\n\n" + types.JiraWriteVerbHelp["add_comment"])
		}
		issueKey, err := config.ExtractIssueKey(p.Issue)
		if err != nil {
			return errorResult(err.Error())
		}
		result, err := jira.AddComment(issueKey, p.Body)
		if err != nil {
			return errorResult(err.Error())
		}
		return successResult(result)

	case "update_issue":
		var p types.JiraUpdateIssueParams
		if err := json.Unmarshal([]byte(param), &p); err != nil {
			return errorResult("Invalid JSON params: " + err.Error() + "\n\n" + types.JiraWriteVerbHelp["update_issue"])
		}
		issueKey, err := config.ExtractIssueKey(p.Issue)
		if err != nil {
			return errorResult(err.Error())
		}
		result, err := jira.UpdateIssue(issueKey, p.Fields, p.Checksums)
		if err != nil {
			return errorResult(err.Error())
		}
		return successResult(result)

	case "create_issue":
		var p types.JiraCreateIssueParams
		if err := json.Unmarshal([]byte(param), &p); err != nil {
			return errorResult("Invalid JSON params: " + err.Error() + "\n\n" + types.JiraWriteVerbHelp["create_issue"])
		}
		result, err := jira.CreateIssue(p.Project, p.IssueType, p.Summary, p.Description)
		if err != nil {
			return errorResult(err.Error())
		}
		return successResult(result)

	default:
		return errorResult("Unknown Jira write operation: " + operation + ". Valid: add_comment, update_issue, create_issue")
	}
}
