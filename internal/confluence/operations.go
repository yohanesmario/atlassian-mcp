package confluence

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"atlassian-mcp/internal/adf"
	"atlassian-mcp/internal/client"
	"atlassian-mcp/internal/config"
	"atlassian-mcp/internal/types"
)

// LRU cache for user display names
const userCacheMaxSize = 100

type lruCache struct {
	capacity int
	items    map[string]*lruItem
	head     *lruItem // most recent
	tail     *lruItem // least recent
}

type lruItem struct {
	key   string
	value string
	prev  *lruItem
	next  *lruItem
}

var userCache = &lruCache{
	capacity: userCacheMaxSize,
	items:    make(map[string]*lruItem),
}

func (c *lruCache) get(key string) (string, bool) {
	if item, ok := c.items[key]; ok {
		c.moveToFront(item)
		return item.value, true
	}
	return "", false
}

func (c *lruCache) set(key, value string) {
	if item, ok := c.items[key]; ok {
		item.value = value
		c.moveToFront(item)
		return
	}

	item := &lruItem{key: key, value: value}
	c.items[key] = item
	c.addToFront(item)

	if len(c.items) > c.capacity {
		c.removeTail()
	}
}

func (c *lruCache) moveToFront(item *lruItem) {
	if item == c.head {
		return
	}
	c.remove(item)
	c.addToFront(item)
}

func (c *lruCache) addToFront(item *lruItem) {
	item.prev = nil
	item.next = c.head
	if c.head != nil {
		c.head.prev = item
	}
	c.head = item
	if c.tail == nil {
		c.tail = item
	}
}

func (c *lruCache) remove(item *lruItem) {
	if item.prev != nil {
		item.prev.next = item.next
	} else {
		c.head = item.next
	}
	if item.next != nil {
		item.next.prev = item.prev
	} else {
		c.tail = item.prev
	}
}

func (c *lruCache) removeTail() {
	if c.tail == nil {
		return
	}
	delete(c.items, c.tail.key)
	c.remove(c.tail)
}

// fetchUserDisplayName fetches user display name with caching.
func fetchUserDisplayName(accountID string) string {
	if accountID == "" {
		return "Unknown"
	}

	if name, ok := userCache.get(accountID); ok {
		return name
	}

	body, err := client.Request(client.Confluence, fmt.Sprintf("/rest/api/user?accountId=%s", accountID))
	if err != nil {
		userCache.set(accountID, accountID)
		return accountID
	}

	var user map[string]any
	if err := json.Unmarshal(body, &user); err != nil {
		userCache.set(accountID, accountID)
		return accountID
	}

	displayName, ok := user["displayName"].(string)
	if !ok || displayName == "" {
		displayName, ok = user["publicName"].(string)
		if !ok || displayName == "" {
			displayName = accountID
		}
	}

	userCache.set(accountID, displayName)
	return displayName
}

// GetPage fetches a page with metadata, body as extended markdown, and checksums.
func GetPage(pageIDOrURL string) (string, error) {
	pageID, err := config.ExtractPageID(pageIDOrURL)
	if err != nil {
		return "", err
	}

	// Fetch page with ADF body format
	body, err := client.Request(client.Confluence, fmt.Sprintf("/api/v2/pages/%s?body-format=atlas_doc_format", pageID))
	if err != nil {
		return "", err
	}

	var page map[string]any
	if err := json.Unmarshal(body, &page); err != nil {
		return "", fmt.Errorf("failed to parse page response")
	}

	return formatPageOutput(page), nil
}

// formatPageOutput formats page data for output.
func formatPageOutput(page map[string]any) string {
	var sb strings.Builder

	id, _ := page["id"].(string)
	title, _ := page["title"].(string)
	status, _ := page["status"].(string)

	sb.WriteString(fmt.Sprintf("# %s\n\n", title))
	sb.WriteString(fmt.Sprintf("**Page ID:** %s\n", id))
	sb.WriteString(fmt.Sprintf("**Status:** %s\n", status))

	// Space info
	if spaceID, ok := page["spaceId"].(string); ok {
		sb.WriteString(fmt.Sprintf("**Space ID:** %s\n", spaceID))
	}

	// Version info
	if version, ok := page["version"].(map[string]any); ok {
		if number, ok := version["number"].(float64); ok {
			sb.WriteString(fmt.Sprintf("**Version:** %d\n", int(number)))
		}
		if createdAt, ok := version["createdAt"].(string); ok {
			sb.WriteString(fmt.Sprintf("**Last Updated:** %s\n", createdAt))
		}
		if authorID, ok := version["authorId"].(string); ok {
			sb.WriteString(fmt.Sprintf("**Last Author:** %s {user:%s}\n", fetchUserDisplayName(authorID), authorID))
		}
	}

	// Created date
	if createdAt, ok := page["createdAt"].(string); ok {
		sb.WriteString(fmt.Sprintf("**Created:** %s\n", createdAt))
	}

	// Author
	if authorID, ok := page["authorId"].(string); ok {
		sb.WriteString(fmt.Sprintf("**Author:** %s {user:%s}\n", fetchUserDisplayName(authorID), authorID))
	}

	// Parent page
	if parentID, ok := page["parentId"].(string); ok && parentID != "" {
		sb.WriteString(fmt.Sprintf("**Parent Page ID:** %s\n", parentID))
	}

	sb.WriteString("\n")

	// Body content - convert ADF to extended markdown
	if body, ok := page["body"].(map[string]any); ok {
		if adfData, ok := body["atlas_doc_format"].(map[string]any); ok {
			if value, ok := adfData["value"].(string); ok {
				var adfDoc map[string]any
				if err := json.Unmarshal([]byte(value), &adfDoc); err == nil {
					sb.WriteString("__DESCRIPTION__\n")
					sb.WriteString(adf.ToMarkdown(adfDoc))
					sb.WriteString("\n__END_DESCRIPTION__\n")
				}
			}
		}
	}

	// Checksums
	checksums := ComputePageChecksums(page)
	sb.WriteString("\n")
	sb.WriteString(FormatChecksums(checksums))

	return sb.String()
}

// GetComments fetches comments for a page.
func GetComments(pageIDOrURL string) (string, error) {
	pageID, err := config.ExtractPageID(pageIDOrURL)
	if err != nil {
		return "", err
	}

	// Fetch footer comments using v1 API with ADF format
	body, err := client.Request(client.Confluence, fmt.Sprintf("/rest/api/content/%s/child/comment?expand=body.atlas_doc_format,version", pageID))
	if err != nil {
		return "", err
	}

	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response")
	}

	return formatCommentsOutput(pageID, response), nil
}

// formatCommentsOutput formats comments for output.
func formatCommentsOutput(pageID string, response map[string]any) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Comments for Page %s\n\n", pageID))

	results, ok := response["results"].([]any)
	if !ok || len(results) == 0 {
		sb.WriteString("No comments found.\n")
		return sb.String()
	}

	for _, c := range results {
		comment, ok := c.(map[string]any)
		if !ok {
			continue
		}

		author := "Unknown"
		authorID := ""
		if version, ok := comment["version"].(map[string]any); ok {
			if by, ok := version["by"].(map[string]any); ok {
				if displayName, ok := by["displayName"].(string); ok {
					author = displayName
				}
				if accountID, ok := by["accountId"].(string); ok {
					authorID = accountID
				}
			}
			authorInfo := author
			if authorID != "" {
				authorInfo = fmt.Sprintf("%s {user:%s}", author, authorID)
			}
			if when, ok := version["when"].(string); ok {
				sb.WriteString(fmt.Sprintf("### Author: %s (%s)\n\n", authorInfo, when))
			} else {
				sb.WriteString(fmt.Sprintf("### Author: %s\n\n", authorInfo))
			}
		} else {
			sb.WriteString(fmt.Sprintf("### Author: %s\n\n", author))
		}

		if body, ok := comment["body"].(map[string]any); ok {
			if adfData, ok := body["atlas_doc_format"].(map[string]any); ok {
				if value, ok := adfData["value"].(string); ok {
					var adfDoc map[string]any
					if err := json.Unmarshal([]byte(value), &adfDoc); err == nil {
						sb.WriteString("__COMMENT__\n")
						sb.WriteString(adf.ToMarkdown(adfDoc))
						sb.WriteString("\n__END_COMMENT__\n")
					}
				}
			}
		}
		sb.WriteString("---\n\n")
	}

	return sb.String()
}

// SearchPages searches for pages using CQL.
func SearchPages(cql string) (string, error) {
	// URL encode the CQL query
	encoded := url.QueryEscape(cql)
	body, err := client.Request(client.Confluence, fmt.Sprintf("/rest/api/search?cql=%s&limit=50", encoded))
	if err != nil {
		return "", err
	}

	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response")
	}

	return formatSearchResults(response), nil
}

// formatSearchResults formats search results for output.
func formatSearchResults(response map[string]any) string {
	var sb strings.Builder

	sb.WriteString("# Search Results\n\n")

	results, ok := response["results"].([]any)
	if !ok || len(results) == 0 {
		sb.WriteString("No results found.\n")
		return sb.String()
	}

	for _, r := range results {
		result, ok := r.(map[string]any)
		if !ok {
			continue
		}

		content, _ := result["content"].(map[string]any)
		if content == nil {
			continue
		}

		id, _ := content["id"].(string)
		title, _ := content["title"].(string)
		contentType, _ := content["type"].(string)

		// Get space key from _expandable or space
		spaceKey := ""
		if space, ok := content["space"].(map[string]any); ok {
			spaceKey, _ = space["key"].(string)
		}

		sb.WriteString(fmt.Sprintf("- **%s** (ID: %s, Type: %s", title, id, contentType))
		if spaceKey != "" {
			sb.WriteString(fmt.Sprintf(", Space: %s", spaceKey))
		}
		sb.WriteString(")\n")
	}

	// Show total size if available
	if totalSize, ok := response["totalSize"].(float64); ok {
		sb.WriteString(fmt.Sprintf("\n**Total results:** %d (showing first 50)\n", int(totalSize)))
	}

	return sb.String()
}

// AddComment adds a comment to a page.
func AddComment(params types.ConfluenceAddCommentParams) (string, error) {
	pageID, err := config.ExtractPageID(params.PageID)
	if err != nil {
		return "", err
	}

	// Convert markdown to ADF
	adfDoc := adf.FromMarkdown(params.Body)
	adfJSON, err := json.Marshal(adfDoc)
	if err != nil {
		return "", fmt.Errorf("failed to convert markdown to ADF")
	}

	// Create comment using v1 API (v2 doesn't support comments well yet)
	payload := map[string]any{
		"type": "comment",
		"container": map[string]any{
			"id":   pageID,
			"type": "page",
		},
		"body": map[string]any{
			"atlas_doc_format": map[string]any{
				"value":          string(adfJSON),
				"representation": "atlas_doc_format",
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload")
	}

	_, err = client.Post(client.Confluence, "/rest/api/content", payloadBytes)
	if err != nil {
		return "", fmt.Errorf("failed to add comment: %w", err)
	}

	return fmt.Sprintf("Comment added to page %s successfully.", pageID), nil
}

// UpdatePage updates a page with checksum validation.
func UpdatePage(params types.ConfluenceUpdatePageParams) (string, error) {
	pageID, err := config.ExtractPageID(params.PageID)
	if err != nil {
		return "", err
	}

	// Validate checksums
	if len(params.Checksums) == 0 {
		return "", fmt.Errorf("checksums required for update_page. Use get_page first to obtain checksums")
	}

	_, conflicts, err := ValidatePageChecksums(pageID, params.Checksums)
	if err != nil {
		return "", fmt.Errorf("failed to validate checksums: %w", err)
	}

	if len(conflicts) > 0 {
		return "", fmt.Errorf("conflict: fields modified since read: %s", strings.Join(conflicts, ", "))
	}

	// Get current version
	currentVersion, err := GetCurrentVersion(pageID)
	if err != nil {
		return "", fmt.Errorf("failed to get current version: %w", err)
	}

	// Build update payload
	payload := map[string]any{
		"id":      pageID,
		"status":  "current",
		"version": map[string]any{"number": currentVersion + 1},
	}

	// Add title if provided
	if params.Title != "" {
		payload["title"] = params.Title
	} else {
		// Fetch current title
		body, err := client.Request(client.Confluence, fmt.Sprintf("/api/v2/pages/%s", pageID))
		if err != nil {
			return "", fmt.Errorf("failed to fetch current page: %w", err)
		}
		var page map[string]any
		if err := json.Unmarshal(body, &page); err != nil {
			return "", fmt.Errorf("failed to parse page: %w", err)
		}
		if title, ok := page["title"].(string); ok {
			payload["title"] = title
		}
	}

	// Add body if provided
	if params.Body != "" {
		adfDoc := adf.FromMarkdown(params.Body)

		// Upload any pending media (images from URLs or local paths)
		if err := UploadPendingMedia(pageID, adfDoc); err != nil {
			return "", fmt.Errorf("failed to upload media: %w", err)
		}

		adfJSON, err := json.Marshal(adfDoc)
		if err != nil {
			return "", fmt.Errorf("failed to convert markdown to ADF")
		}

		payload["body"] = map[string]any{
			"representation": "atlas_doc_format",
			"value":          string(adfJSON),
		}
	}

	// Update page
	expectedVersion := currentVersion + 1
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload")
	}

	_, err = client.Put(client.Confluence, fmt.Sprintf("/api/v2/pages/%s", pageID), payloadBytes)
	if err != nil {
		return "", fmt.Errorf("failed to update page: %w", err)
	}

	// Wait for version to propagate before fetching
	delays := []time.Duration{200 * time.Millisecond, 500 * time.Millisecond, 1 * time.Second}
	for _, delay := range delays {
		if v, _ := GetCurrentVersion(pageID); v == expectedVersion {
			break
		}
		time.Sleep(delay)
	}

	// Fetch updated page to get new checksums
	result, err := GetPage(pageID)
	if err != nil {
		return fmt.Sprintf("Page %s updated successfully, but failed to fetch updated checksums.", pageID), nil
	}

	return fmt.Sprintf("Page %s updated successfully.\n\n%s", pageID, result), nil
}

// CreatePage creates a new page in a space.
func CreatePage(params types.ConfluenceCreatePageParams) (string, error) {
	if params.SpaceID == "" {
		return "", fmt.Errorf("spaceId is required")
	}
	if params.Title == "" {
		return "", fmt.Errorf("title is required")
	}

	// Convert markdown body to ADF (or empty doc if no body)
	var adfDoc map[string]any
	if params.Body != "" {
		adfDoc = adf.FromMarkdown(params.Body)
	} else {
		adfDoc = map[string]any{
			"type":    "doc",
			"version": 1,
			"content": []any{},
		}
	}

	// Check if there are pending media uploads
	hasPendingMedia := checkPendingMedia(adfDoc)

	adfJSON, err := json.Marshal(adfDoc)
	if err != nil {
		return "", fmt.Errorf("failed to convert markdown to ADF")
	}

	payload := map[string]any{
		"spaceId": params.SpaceID,
		"status":  "current",
		"title":   params.Title,
		"body": map[string]any{
			"representation": "atlas_doc_format",
			"value":          string(adfJSON),
		},
	}

	// Add parent if specified
	if params.ParentID != "" {
		payload["parentId"] = params.ParentID
	}

	// Create page
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload")
	}

	body, err := client.Post(client.Confluence, "/api/v2/pages", payloadBytes)
	if err != nil {
		return "", fmt.Errorf("failed to create page: %w", err)
	}

	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response")
	}

	pageID, _ := response["id"].(string)

	// If there were pending media, upload them and update the page
	if hasPendingMedia && pageID != "" {
		// Re-parse the markdown to get fresh ADF with placeholders
		adfDoc = adf.FromMarkdown(params.Body)

		// Upload pending media to the newly created page
		if err := UploadPendingMedia(pageID, adfDoc); err != nil {
			return fmt.Sprintf("Page created but media upload failed: %v\n**Page ID:** %s\n**Title:** %s", err, pageID, params.Title), nil
		}

		// Update page with the media IDs
		adfJSON, _ = json.Marshal(adfDoc)

		// Get current version for update
		currentVersion, err := GetCurrentVersion(pageID)
		if err != nil {
			return fmt.Sprintf("Page created but failed to get version for media update: %v\n**Page ID:** %s\n**Title:** %s", err, pageID, params.Title), nil
		}

		updatePayload := map[string]any{
			"id":      pageID,
			"status":  "current",
			"title":   params.Title,
			"version": map[string]any{"number": currentVersion + 1},
			"body": map[string]any{
				"representation": "atlas_doc_format",
				"value":          string(adfJSON),
			},
		}

		updateBytes, _ := json.Marshal(updatePayload)
		_, err = client.Put(client.Confluence, fmt.Sprintf("/api/v2/pages/%s", pageID), updateBytes)
		if err != nil {
			return fmt.Sprintf("Page created but media update failed: %v\n**Page ID:** %s\n**Title:** %s", err, pageID, params.Title), nil
		}
	}

	return fmt.Sprintf("Page created successfully.\n**Page ID:** %s\n**Title:** %s", pageID, params.Title), nil
}

// checkPendingMedia checks if an ADF document has any pending media uploads.
func checkPendingMedia(adf map[string]any) bool {
	content, ok := adf["content"].([]any)
	if !ok {
		return false
	}

	for _, node := range content {
		nodeMap, ok := node.(map[string]any)
		if !ok {
			continue
		}

		nodeType, _ := nodeMap["type"].(string)

		if nodeType == "mediaSingle" {
			innerContent, ok := nodeMap["content"].([]any)
			if !ok || len(innerContent) == 0 {
				continue
			}

			mediaNode, ok := innerContent[0].(map[string]any)
			if !ok {
				continue
			}

			attrs, ok := mediaNode["attrs"].(map[string]any)
			if !ok {
				continue
			}

			id, _ := attrs["id"].(string)
			if strings.HasPrefix(id, "__PENDING_UPLOAD_") {
				return true
			}
		}

		// Check nested content
		if innerContent, ok := nodeMap["content"].([]any); ok {
			innerADF := map[string]any{"content": innerContent}
			if checkPendingMedia(innerADF) {
				return true
			}
		}
	}

	return false
}
