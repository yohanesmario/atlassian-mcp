package jira

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"atlassian-mcp/internal/adf"
	"atlassian-mcp/internal/client"
)

// FetchIssue fetches an issue by key and returns formatted markdown.
func FetchIssue(issueKey string) (string, error) {
	// Fetch issue with expanded fields
	body, err := client.Request(client.Jira, fmt.Sprintf("/rest/api/3/issue/%s?expand=renderedFields", issueKey))
	if err != nil {
		return "", err
	}

	var issue map[string]any
	if err := json.Unmarshal(body, &issue); err != nil {
		return "", fmt.Errorf("failed to parse issue response")
	}

	return formatIssue(issue), nil
}

// FetchComments fetches comments for an issue.
func FetchComments(issueKey string) (string, error) {
	commentsBody, err := client.Request(client.Jira, fmt.Sprintf("/rest/api/3/issue/%s/comment?orderBy=created&maxResults=50", issueKey))
	if err != nil {
		return "", err
	}

	var comments map[string]any
	if err := json.Unmarshal(commentsBody, &comments); err != nil {
		return "", fmt.Errorf("failed to parse response")
	}

	return formatComments(issueKey, comments), nil
}

func formatComments(issueKey string, comments map[string]any) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Comments for %s\n\n", issueKey))

	commentList, ok := comments["comments"].([]any)
	if !ok || len(commentList) == 0 {
		sb.WriteString("No comments found.\n")
		return sb.String()
	}

	for _, c := range commentList {
		comment, ok := c.(map[string]any)
		if !ok {
			continue
		}

		author := "Unknown"
		authorID := ""
		if a, ok := comment["author"].(map[string]any); ok {
			author, _ = a["displayName"].(string)
			authorID, _ = a["accountId"].(string)
		}
		created, _ := comment["created"].(string)

		authorInfo := author
		if authorID != "" {
			authorInfo = fmt.Sprintf("%s {user:%s}", author, authorID)
		}
		sb.WriteString(fmt.Sprintf("### %s (%s)\n\n", authorInfo, created))
		if body, ok := comment["body"].(map[string]any); ok {
			sb.WriteString(adf.ToMarkdown(body))
		}
		sb.WriteString("\n---\n\n")
	}

	return sb.String()
}

func formatIssue(issue map[string]any) string {
	var sb strings.Builder

	key, _ := issue["key"].(string)
	fields, _ := issue["fields"].(map[string]any)

	sb.WriteString(fmt.Sprintf("# %s\n\n", key))

	if summary, ok := fields["summary"].(string); ok {
		sb.WriteString(fmt.Sprintf("**Summary:** %s\n\n", summary))
	}

	if status, ok := fields["status"].(map[string]any); ok {
		if name, ok := status["name"].(string); ok {
			sb.WriteString(fmt.Sprintf("**Status:** %s\n", name))
		}
	}

	if issuetype, ok := fields["issuetype"].(map[string]any); ok {
		if name, ok := issuetype["name"].(string); ok {
			sb.WriteString(fmt.Sprintf("**Type:** %s\n", name))
		}
	}

	if priority, ok := fields["priority"].(map[string]any); ok {
		if name, ok := priority["name"].(string); ok {
			sb.WriteString(fmt.Sprintf("**Priority:** %s\n", name))
		}
	}

	if assignee, ok := fields["assignee"].(map[string]any); ok {
		if name, ok := assignee["displayName"].(string); ok {
			if accountID, ok := assignee["accountId"].(string); ok {
				sb.WriteString(fmt.Sprintf("**Assignee:** %s {user:%s}\n", name, accountID))
			} else {
				sb.WriteString(fmt.Sprintf("**Assignee:** %s\n", name))
			}
		}
	}

	if reporter, ok := fields["reporter"].(map[string]any); ok {
		if name, ok := reporter["displayName"].(string); ok {
			if accountID, ok := reporter["accountId"].(string); ok {
				sb.WriteString(fmt.Sprintf("**Reporter:** %s {user:%s}\n", name, accountID))
			} else {
				sb.WriteString(fmt.Sprintf("**Reporter:** %s\n", name))
			}
		}
	}

	// Labels
	if labels, ok := fields["labels"].([]any); ok && len(labels) > 0 {
		labelStrs := make([]string, 0, len(labels))
		for _, l := range labels {
			if s, ok := l.(string); ok {
				labelStrs = append(labelStrs, s)
			}
		}
		if len(labelStrs) > 0 {
			sb.WriteString(fmt.Sprintf("**Labels:** %s\n", strings.Join(labelStrs, ", ")))
		}
	}

	// Components
	if components, ok := fields["components"].([]any); ok && len(components) > 0 {
		compStrs := make([]string, 0, len(components))
		for _, c := range components {
			if comp, ok := c.(map[string]any); ok {
				if name, ok := comp["name"].(string); ok {
					compStrs = append(compStrs, name)
				}
			}
		}
		if len(compStrs) > 0 {
			sb.WriteString(fmt.Sprintf("**Components:** %s\n", strings.Join(compStrs, ", ")))
		}
	}

	// Epic Link (customfield_10014 is common, but may vary)
	if epic, ok := fields["parent"].(map[string]any); ok {
		if epicKey, ok := epic["key"].(string); ok {
			epicSummary := ""
			if epicFields, ok := epic["fields"].(map[string]any); ok {
				epicSummary, _ = epicFields["summary"].(string)
			}
			if epicSummary != "" {
				sb.WriteString(fmt.Sprintf("**Parent:** %s - %s\n", epicKey, epicSummary))
			} else {
				sb.WriteString(fmt.Sprintf("**Parent:** %s\n", epicKey))
			}
		}
	}

	// Created/Updated dates
	if created, ok := fields["created"].(string); ok {
		sb.WriteString(fmt.Sprintf("**Created:** %s\n", created))
	}
	if updated, ok := fields["updated"].(string); ok {
		sb.WriteString(fmt.Sprintf("**Updated:** %s\n", updated))
	}

	sb.WriteString("\n")

	if description, ok := fields["description"].(map[string]any); ok {
		sb.WriteString("__DESCRIPTION__\n")
		sb.WriteString(adf.ToMarkdown(description))
		sb.WriteString("__END_DESCRIPTION__\n\n")
	}

	// Subtasks
	if subtasks, ok := fields["subtasks"].([]any); ok && len(subtasks) > 0 {
		sb.WriteString("## Subtasks\n\n")
		for _, st := range subtasks {
			if subtask, ok := st.(map[string]any); ok {
				stKey, _ := subtask["key"].(string)
				stFields, _ := subtask["fields"].(map[string]any)
				stSummary, _ := stFields["summary"].(string)
				stStatus := ""
				if status, ok := stFields["status"].(map[string]any); ok {
					stStatus, _ = status["name"].(string)
				}
				sb.WriteString(fmt.Sprintf("- [%s] %s - %s\n", stKey, stSummary, stStatus))
			}
		}
		sb.WriteString("\n")
	}

	// Linked issues
	if issuelinks, ok := fields["issuelinks"].([]any); ok && len(issuelinks) > 0 {
		sb.WriteString("## Linked Issues\n\n")
		for _, link := range issuelinks {
			if l, ok := link.(map[string]any); ok {
				linkType, _ := l["type"].(map[string]any)
				if outward, ok := l["outwardIssue"].(map[string]any); ok {
					linkName, _ := linkType["outward"].(string)
					outKey, _ := outward["key"].(string)
					outFields, _ := outward["fields"].(map[string]any)
					outSummary, _ := outFields["summary"].(string)
					sb.WriteString(fmt.Sprintf("- %s: %s - %s\n", linkName, outKey, outSummary))
				}
				if inward, ok := l["inwardIssue"].(map[string]any); ok {
					linkName, _ := linkType["inward"].(string)
					inKey, _ := inward["key"].(string)
					inFields, _ := inward["fields"].(map[string]any)
					inSummary, _ := inFields["summary"].(string)
					sb.WriteString(fmt.Sprintf("- %s: %s - %s\n", linkName, inKey, inSummary))
				}
			}
		}
		sb.WriteString("\n")
	}

	// Compute and append checksums for optimistic concurrency control
	checksumFields := []string{"summary", "description", "status", "assignee", "priority", "labels", "components"}
	checksums := ComputeFieldsChecksums(fields, checksumFields)

	sb.WriteString("\n__CHECKSUMS__\n")
	for _, field := range checksumFields {
		sb.WriteString(fmt.Sprintf("%s=%s\n", field, checksums[field]))
	}
	sb.WriteString("__END_CHECKSUMS__\n")

	return sb.String()
}

// SearchIssues searches for issues using JQL (enhanced search endpoint)
func SearchIssues(jql string) (string, error) {
	endpoint := "/rest/api/3/search/jql"

	payload := map[string]any{
		"jql":        jql,
		"maxResults": 50,
		"fields":     []string{"key", "summary", "status", "assignee", "issuetype", "priority"},
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal search request")
	}

	body, err := client.Post(client.Jira, endpoint, reqBody)
	if err != nil {
		return "", err
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse search response")
	}

	issues, ok := result["issues"].([]any)
	if !ok {
		return "No issues found.\n", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Search Results (%d issues)\n\n", len(issues)))

	for _, issue := range issues {
		issueMap, ok := issue.(map[string]any)
		if !ok {
			continue
		}

		key, _ := issueMap["key"].(string)
		fields, _ := issueMap["fields"].(map[string]any)
		summary, _ := fields["summary"].(string)

		status := "Unknown"
		if s, ok := fields["status"].(map[string]any); ok {
			status, _ = s["name"].(string)
		}

		assignee := "Unassigned"
		if a, ok := fields["assignee"].(map[string]any); ok {
			name, _ := a["displayName"].(string)
			if accountID, ok := a["accountId"].(string); ok {
				assignee = fmt.Sprintf("%s {user:%s}", name, accountID)
			} else {
				assignee = name
			}
		}

		issueType := ""
		if t, ok := fields["issuetype"].(map[string]any); ok {
			issueType, _ = t["name"].(string)
		}

		sb.WriteString(fmt.Sprintf("- **%s** [%s] %s (%s) - %s\n", key, issueType, summary, status, assignee))
	}

	return sb.String(), nil
}

// AddComment adds a comment to an issue
func AddComment(issueKey, commentBody string) (string, error) {
	endpoint := fmt.Sprintf("/rest/api/3/issue/%s/comment", issueKey)

	payload := map[string]any{
		"body": adf.FromMarkdown(commentBody),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal comment")
	}

	resp, err := client.Post(client.Jira, endpoint, body)
	if err != nil {
		return "", err
	}

	var result map[string]any
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("failed to parse response")
	}

	commentID, _ := result["id"].(string)
	return fmt.Sprintf("Comment added successfully (ID: %s)", commentID), nil
}

// UpdateIssue updates fields on an issue with optimistic concurrency control.
// Checksums are required for all fields being updated.
func UpdateIssue(issueKey string, fields map[string]any, checksums map[string]string) (string, error) {
	// Validate: checksums required for all fields being updated
	var missingChecksums []string
	for fieldName := range fields {
		if _, ok := checksums[fieldName]; !ok {
			missingChecksums = append(missingChecksums, fieldName)
		}
	}
	if len(missingChecksums) > 0 {
		sort.Strings(missingChecksums)
		return "", fmt.Errorf("missing checksums for fields: %s", strings.Join(missingChecksums, ", "))
	}

	// Fetch current issue to verify checksums
	currentBody, err := client.Request(client.Jira, fmt.Sprintf("/rest/api/3/issue/%s", issueKey))
	if err != nil {
		return "", err
	}

	var currentIssue map[string]any
	if err := json.Unmarshal(currentBody, &currentIssue); err != nil {
		return "", fmt.Errorf("failed to parse issue for verification")
	}

	currentFields, _ := currentIssue["fields"].(map[string]any)

	// Check each field being updated against its checksum
	var mismatched []string
	for fieldName := range fields {
		expectedChecksum := checksums[fieldName]
		currentCanonical := GetCanonicalFieldValue(fieldName, currentFields)
		currentChecksum := ComputeFieldChecksum(currentCanonical)
		if currentChecksum != expectedChecksum {
			mismatched = append(mismatched, fieldName)
		}
	}

	if len(mismatched) > 0 {
		sort.Strings(mismatched)
		return "", fmt.Errorf("conflict: fields modified since read: %s", strings.Join(mismatched, ", "))
	}

	// Proceed with update
	endpoint := fmt.Sprintf("/rest/api/3/issue/%s", issueKey)

	// Convert description to ADF if it's a string
	if desc, ok := fields["description"].(string); ok {
		adfDoc := adf.FromMarkdown(desc)

		// Upload any pending media (images from URLs or local paths)
		if err := UploadPendingMedia(issueKey, adfDoc); err != nil {
			return "", fmt.Errorf("failed to upload media: %v", err)
		}

		fields["description"] = adfDoc
	}

	payload := map[string]any{
		"fields": fields,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal update")
	}

	_, err = client.Put(client.Jira, endpoint, body)
	if err != nil {
		return "", err
	}

	// Re-fetch issue to get fresh checksums
	updatedBody, err := client.Request(client.Jira, fmt.Sprintf("/rest/api/3/issue/%s", issueKey))
	if err != nil {
		// Update succeeded but couldn't fetch fresh checksums
		return fmt.Sprintf("Issue %s updated successfully (could not fetch fresh checksums)", issueKey), nil
	}

	var updatedIssue map[string]any
	if err := json.Unmarshal(updatedBody, &updatedIssue); err != nil {
		return fmt.Sprintf("Issue %s updated successfully (could not parse fresh checksums)", issueKey), nil
	}

	updatedFields, _ := updatedIssue["fields"].(map[string]any)

	// Compute checksums only for the fields that were updated
	var updatedFieldNames []string
	for fieldName := range fields {
		updatedFieldNames = append(updatedFieldNames, fieldName)
	}
	sort.Strings(updatedFieldNames)
	newChecksums := ComputeFieldsChecksums(updatedFields, updatedFieldNames)

	checksumJSON, _ := json.Marshal(newChecksums)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Issue %s updated successfully\n\n", issueKey))
	sb.WriteString("## Checksums\n\n")
	sb.WriteString("```json\n")
	sb.WriteString(string(checksumJSON))
	sb.WriteString("\n```\n")

	return sb.String(), nil
}

// CreateIssue creates a new issue
func CreateIssue(project, issueType, summary, description string) (string, error) {
	endpoint := "/rest/api/3/issue"

	fields := map[string]any{
		"project":   map[string]any{"key": project},
		"issuetype": map[string]any{"name": issueType},
		"summary":   summary,
	}

	if description != "" {
		fields["description"] = adf.FromMarkdown(description)
	}

	payload := map[string]any{
		"fields": fields,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal issue")
	}

	resp, err := client.Post(client.Jira, endpoint, body)
	if err != nil {
		return "", err
	}

	var result map[string]any
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("failed to parse response")
	}

	key, _ := result["key"].(string)
	return fmt.Sprintf("Issue created: %s", key), nil
}
