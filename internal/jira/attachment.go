package jira

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"atlassian-mcp/internal/client"
	"atlassian-mcp/internal/config"
	"atlassian-mcp/internal/types"
)

// maxJiraAttachmentSize is the maximum file size for Jira attachments (10MB).
const maxJiraAttachmentSize = 10 * 1024 * 1024

// supportedMediaExtensions lists file extensions supported by Atlassian for media embedding.
// See: https://confluence.atlassian.com/jirasoftwareserver/attaching-files-and-screenshots-to-issues-939938913.html
var supportedMediaExtensions = map[string]bool{
	".gif":  true,
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".bmp":  true,
}

// pendingUpload holds file data collected before validation and upload.
type pendingUpload struct {
	// nodeAttrs is a pointer to the ADF node attributes for post-upload update.
	nodeAttrs map[string]any
	// data is the file contents read into memory.
	data []byte
	// filename is the sanitized filename for upload.
	filename string
	// source is the original source path or URL for error messages.
	source string
}

// UploadAttachment uploads a file to a Jira issue and returns attachment info
func UploadAttachment(issueKey string, fileData []byte, filename string) (*types.JiraAttachmentInfo, error) {
	endpoint := fmt.Sprintf("/rest/api/3/issue/%s/attachments", issueKey)
	reqURL := fmt.Sprintf("https://%s%s", config.Domain, endpoint)

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %v", err)
	}

	if _, err := part.Write(fileData); err != nil {
		return nil, fmt.Errorf("failed to write file data: %v", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %v", err)
	}

	req, err := http.NewRequest("POST", reqURL, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request")
	}

	auth := base64.StdEncoding.EncodeToString([]byte(config.Email + ":" + config.Token))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Atlassian-Token", "no-check") // Required for attachment uploads

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Jira: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("attachment upload failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	// Response is an array of attachments
	var attachments []types.JiraAttachmentInfo
	if err := json.Unmarshal(respBody, &attachments); err != nil {
		return nil, fmt.Errorf("failed to parse attachment response: %v", err)
	}

	if len(attachments) == 0 {
		return nil, fmt.Errorf("no attachment returned from upload")
	}

	att := &attachments[0]
	// Extract media ID from content URL
	att.MediaID = extractMediaIDFromURL(att.Content)

	// If no UUID found in URL, try to get it by following redirect
	if att.MediaID == "" {
		// Try HEAD request to get redirect URL
		req, _ := http.NewRequest("HEAD", att.Content, nil)
		auth := base64.StdEncoding.EncodeToString([]byte(config.Email + ":" + config.Token))
		req.Header.Set("Authorization", "Basic "+auth)

		// Use a client that doesn't follow redirects
		noRedirectClient := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		resp, err := noRedirectClient.Do(req)
		if err == nil {
			defer resp.Body.Close()
			if location := resp.Header.Get("Location"); location != "" {
				att.MediaID = extractMediaIDFromURL(location)
			}
		}
	}

	// Final fallback: use attachment ID if we still don't have a media ID
	if att.MediaID == "" {
		att.MediaID = att.ID
	}

	return att, nil
}

// extractMediaIDFromURL extracts the media UUID from Jira's content URL
// URL format: https://api.media.atlassian.com/file/{mediaId}/binary
func extractMediaIDFromURL(contentURL string) string {
	// Look for /file/{uuid}/ pattern
	re := regexp.MustCompile(`/file/([a-f0-9-]+)/`)
	matches := re.FindStringSubmatch(contentURL)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// UploadPendingMedia walks the ADF tree, validates all pending media, and uploads them.
// All files are validated before any uploads occur to prevent partial uploads.
func UploadPendingMedia(issueKey string, adf map[string]any) error {
	// Phase 1: Collect all pending uploads into memory
	pending, err := collectPendingUploads(adf)
	if err != nil {
		return fmt.Errorf("failed to collect uploads: %w", err)
	}
	if len(pending) == 0 {
		return nil
	}

	// Phase 2: Validate all uploads
	if err := validatePendingUploads(pending, maxJiraAttachmentSize); err != nil {
		return err
	}

	// Phase 3: Upload all files (only reached if validation passed)
	for _, p := range pending {
		attInfo, err := UploadAttachment(issueKey, p.data, p.filename)
		if err != nil {
			return fmt.Errorf("upload failed for %s: %w", p.source, err)
		}

		// Update ADF node with real media ID
		p.nodeAttrs["id"] = attInfo.MediaID
		p.nodeAttrs["collection"] = "mediaServiceAttachments"
		delete(p.nodeAttrs, "_source")
	}

	return nil
}

// downloadFile fetches a file from a URL and returns its contents
func downloadFile(url string) ([]byte, string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download file: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, "", fmt.Errorf("failed to download file (HTTP %d)", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read file data: %v", err)
	}

	// Extract filename from URL or Content-Disposition
	filename := filepath.Base(url)
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if strings.Contains(cd, "filename=") {
			parts := strings.Split(cd, "filename=")
			if len(parts) > 1 {
				filename = strings.Trim(parts[1], `"' `)
			}
		}
	}

	return data, filename, nil
}

// sanitizeFilename removes unsafe characters from a filename.
// Only allows alphanumeric, dash, underscore, and dot characters.
// Consecutive underscores are collapsed to a single underscore.
func sanitizeFilename(name string) string {
	if name == "" {
		return "attachment"
	}

	// Separate extension
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)

	// Replace invalid characters with underscore
	var result strings.Builder
	for _, r := range base {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			result.WriteRune(r)
		} else if r == '.' {
			result.WriteRune('_') // dots in basename become underscores
		} else {
			result.WriteRune('_')
		}
	}

	// Collapse consecutive underscores
	sanitized := result.String()
	for strings.Contains(sanitized, "__") {
		sanitized = strings.ReplaceAll(sanitized, "__", "_")
	}

	// Trim leading/trailing underscores
	sanitized = strings.Trim(sanitized, "_")

	// Fallback if empty after sanitization
	if sanitized == "" {
		sanitized = "attachment"
	}

	// Sanitize extension (remove invalid chars, keep the dot)
	if ext != "" {
		ext = strings.ToLower(ext)
		var extResult strings.Builder
		extResult.WriteRune('.')
		for _, r := range ext[1:] { // skip the leading dot
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				extResult.WriteRune(r)
			}
		}
		ext = extResult.String()
	}

	return sanitized + ext
}

// collectPendingUploads walks the ADF tree and collects all pending media uploads.
// It downloads URLs and reads local files into memory.
func collectPendingUploads(adf map[string]any) ([]pendingUpload, error) {
	var uploads []pendingUpload

	content, ok := adf["content"].([]any)
	if !ok {
		return uploads, nil
	}

	for _, node := range content {
		nodeMap, ok := node.(map[string]any)
		if !ok {
			continue
		}

		nodeType, _ := nodeMap["type"].(string)

		// Process mediaSingle nodes
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
			if !strings.HasPrefix(id, "__PENDING_UPLOAD_") {
				continue
			}

			source, _ := attrs["_source"].(string)
			alt, _ := attrs["alt"].(string)
			if source == "" {
				continue
			}

			var fileData []byte
			var filename string
			var err error

			if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
				fileData, filename, err = downloadFile(source)
				if err != nil {
					return nil, fmt.Errorf("failed to download %s: %w", source, err)
				}
			} else {
				fileData, err = os.ReadFile(source)
				if err != nil {
					return nil, fmt.Errorf("failed to read %s: %w", source, err)
				}
				filename = filepath.Base(source)
			}

			// Use alt text as filename if available
			if alt != "" && alt != "attachment" {
				ext := filepath.Ext(filename)
				if ext == "" {
					ext = ".png"
				}
				filename = alt + ext
			}

			uploads = append(uploads, pendingUpload{
				nodeAttrs: attrs,
				data:      fileData,
				filename:  sanitizeFilename(filename),
				source:    source,
			})
		}

		// Recursively process nested content
		if innerContent, ok := nodeMap["content"].([]any); ok {
			innerADF := map[string]any{"content": innerContent}
			innerUploads, err := collectPendingUploads(innerADF)
			if err != nil {
				return nil, err
			}
			uploads = append(uploads, innerUploads...)
		}
	}

	return uploads, nil
}

// validatePendingUploads validates all pending uploads and returns an aggregated error.
// Returns nil if all uploads are valid.
func validatePendingUploads(uploads []pendingUpload, maxSize int) error {
	var errors []string

	for _, u := range uploads {
		// Check for empty data
		if len(u.data) == 0 {
			errors = append(errors, fmt.Sprintf("%s: empty file", u.source))
			continue
		}

		// Check size limit
		if len(u.data) > maxSize {
			errors = append(errors, fmt.Sprintf("%s: exceeds %dMB limit", u.source, maxSize/(1024*1024)))
			continue
		}

		// Check filename is valid after sanitization
		if u.filename == "" {
			errors = append(errors, fmt.Sprintf("%s: invalid filename", u.source))
			continue
		}

		// Check for supported media extension
		ext := strings.ToLower(filepath.Ext(u.filename))
		if !supportedMediaExtensions[ext] {
			errors = append(errors, fmt.Sprintf("%s: unsupported file type %q (supported: gif, jpg, jpeg, png, bmp)", u.source, ext))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}
