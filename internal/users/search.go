package users

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"atlassian-mcp/internal/client"
)

// SearchUsers searches for users by name or email and returns formatted results
// with account IDs ready for mentions.
func SearchUsers(query string) (string, error) {
	if query == "" {
		return "", fmt.Errorf("search query is required")
	}

	// Use the user picker endpoint - designed for finding users to mention
	endpoint := fmt.Sprintf("/rest/api/3/user/picker?query=%s&maxResults=10", url.QueryEscape(query))

	body, err := client.Request(client.Jira, endpoint)
	if err != nil {
		return "", err
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse user search response")
	}

	users, ok := result["users"].([]any)
	if !ok || len(users) == 0 {
		return "No users found matching: " + query + "\n", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# User Search Results (%d found)\n\n", len(users)))
	sb.WriteString("| Name | Account ID | Mention Format |\n")
	sb.WriteString("|------|------------|----------------|\n")

	for _, u := range users {
		user, ok := u.(map[string]any)
		if !ok {
			continue
		}

		displayName, _ := user["displayName"].(string)
		accountID, _ := user["accountId"].(string)

		if displayName == "" || accountID == "" {
			continue
		}

		// Format: @[Name](accountId:xxx)
		mentionFormat := fmt.Sprintf("@[%s](accountId:%s)", displayName, accountID)

		sb.WriteString(fmt.Sprintf("| %s | %s | `%s` |\n", displayName, accountID, mentionFormat))
	}

	sb.WriteString("\n**Usage:** Copy the mention format into comments, descriptions, or page content.\n")

	return sb.String(), nil
}
