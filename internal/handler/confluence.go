package handler

import (
	"encoding/json"

	"atlassian-mcp/internal/confluence"
	"atlassian-mcp/internal/types"
)

// handleConfluenceRead handles Confluence read operations.
func handleConfluenceRead(operation, param string) any {
	switch operation {
	case "get_page":
		result, err := confluence.GetPage(param)
		if err != nil {
			return errorResult(err.Error())
		}
		return successResult(result)

	case "get_comments":
		result, err := confluence.GetComments(param)
		if err != nil {
			return errorResult(err.Error())
		}
		return successResult(result)

	case "search":
		result, err := confluence.SearchPages(param)
		if err != nil {
			return errorResult(err.Error())
		}
		return successResult(result)

	default:
		return errorResult("Unknown Confluence read operation: " + operation + ". Valid: get_page, get_comments, search")
	}
}

// handleConfluenceWrite handles Confluence write operations.
func handleConfluenceWrite(operation, param string) any {
	switch operation {
	case "add_comment":
		var p types.ConfluenceAddCommentParams
		if err := json.Unmarshal([]byte(param), &p); err != nil {
			return errorResult("Invalid JSON params: " + err.Error() + "\n\n" + types.ConfluenceWriteVerbHelp["add_comment"])
		}
		result, err := confluence.AddComment(p)
		if err != nil {
			return errorResult(err.Error())
		}
		return successResult(result)

	case "update_page":
		var p types.ConfluenceUpdatePageParams
		if err := json.Unmarshal([]byte(param), &p); err != nil {
			return errorResult("Invalid JSON params: " + err.Error() + "\n\n" + types.ConfluenceWriteVerbHelp["update_page"])
		}
		result, err := confluence.UpdatePage(p)
		if err != nil {
			return errorResult(err.Error())
		}
		return successResult(result)

	case "create_page":
		var p types.ConfluenceCreatePageParams
		if err := json.Unmarshal([]byte(param), &p); err != nil {
			return errorResult("Invalid JSON params: " + err.Error() + "\n\n" + types.ConfluenceWriteVerbHelp["create_page"])
		}
		result, err := confluence.CreatePage(p)
		if err != nil {
			return errorResult(err.Error())
		}
		return successResult(result)

	default:
		return errorResult("Unknown Confluence write operation: " + operation + ". Valid: add_comment, update_page, create_page")
	}
}
