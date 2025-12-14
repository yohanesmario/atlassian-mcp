package confluence

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"atlassian-mcp/internal/client"
)

// ChecksumFields are the fields tracked for conflict detection on pages.
var ChecksumFields = []string{"title", "body", "version"}

// ComputePageChecksums computes SHA256 checksums for page fields.
func ComputePageChecksums(page map[string]any) map[string]string {
	checksums := make(map[string]string)

	// Title checksum
	if title, ok := page["title"].(string); ok {
		checksums["title"] = hashString(title)
	}

	// Body checksum (ADF JSON)
	if body, ok := page["body"].(map[string]any); ok {
		if adf, ok := body["atlas_doc_format"].(map[string]any); ok {
			if value, ok := adf["value"].(string); ok {
				checksums["body"] = hashString(value)
			}
		}
	}

	// Version checksum
	if version, ok := page["version"].(map[string]any); ok {
		if number, ok := version["number"].(float64); ok {
			checksums["version"] = hashString(fmt.Sprintf("%d", int(number)))
		}
	}

	return checksums
}

// ValidatePageChecksums validates provided checksums against current page state.
// Returns: current checksums, list of conflicting fields, error
func ValidatePageChecksums(pageID string, provided map[string]string) (map[string]string, []string, error) {
	// Fetch current page to get current checksums
	body, err := client.Request(client.Confluence, fmt.Sprintf("/api/v2/pages/%s?body-format=atlas_doc_format", pageID))
	if err != nil {
		return nil, nil, err
	}

	var page map[string]any
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, nil, fmt.Errorf("failed to parse page response")
	}

	current := ComputePageChecksums(page)
	var conflicts []string

	// Compare provided checksums with current
	for field, providedHash := range provided {
		if currentHash, ok := current[field]; ok {
			if currentHash != providedHash {
				conflicts = append(conflicts, field)
			}
		}
	}

	return current, conflicts, nil
}

// FormatChecksums formats checksums for output.
func FormatChecksums(checksums map[string]string) string {
	var sb strings.Builder
	sb.WriteString("__CHECKSUMS__\n")
	for _, field := range ChecksumFields {
		if hash, ok := checksums[field]; ok {
			sb.WriteString(fmt.Sprintf("%s=%s\n", field, hash))
		}
	}
	sb.WriteString("__END_CHECKSUMS__")
	return sb.String()
}

// hashString computes a truncated SHA256 hash of a string.
// Uses first 8 bytes (64 bits) of SHA256 - sufficient for change detection where
// collision resistance against random changes is the goal, not adversarial attacks.
// 64 bits provides ~2^32 expected attempts before collision (birthday bound).
func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h[:8]) // First 8 bytes = 16 hex chars
}

// GetCurrentVersion fetches the current version number for a page.
func GetCurrentVersion(pageID string) (int, error) {
	body, err := client.Request(client.Confluence, fmt.Sprintf("/api/v2/pages/%s", pageID))
	if err != nil {
		return 0, err
	}

	var page map[string]any
	if err := json.Unmarshal(body, &page); err != nil {
		return 0, fmt.Errorf("failed to parse page response")
	}

	if version, ok := page["version"].(map[string]any); ok {
		if number, ok := version["number"].(float64); ok {
			return int(number), nil
		}
	}

	return 0, fmt.Errorf("could not determine page version")
}
