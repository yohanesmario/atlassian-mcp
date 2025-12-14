package jira

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
)

// ComputeFieldChecksum returns a 16-char hex SHA256 checksum of the canonical value.
// Uses first 8 bytes (64 bits) of SHA256 - sufficient for change detection where
// collision resistance against random changes is the goal, not adversarial attacks.
// 64 bits provides ~2^32 expected attempts before collision (birthday bound).
func ComputeFieldChecksum(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:8])
}

// GetCanonicalFieldValue extracts the canonical string for checksum computation.
// Each field type has a specific canonical form to ensure consistent checksums.
func GetCanonicalFieldValue(fieldName string, fields map[string]any) string {
	switch fieldName {
	case "summary":
		if v, ok := fields["summary"].(string); ok {
			return v
		}
	case "description":
		if v, ok := fields["description"].(map[string]any); ok {
			data, _ := json.Marshal(v)
			return string(data)
		}
	case "status":
		if v, ok := fields["status"].(map[string]any); ok {
			if name, ok := v["name"].(string); ok {
				return name
			}
		}
	case "assignee":
		if v, ok := fields["assignee"].(map[string]any); ok {
			if id, ok := v["accountId"].(string); ok {
				return id
			}
		}
	case "priority":
		if v, ok := fields["priority"].(map[string]any); ok {
			if name, ok := v["name"].(string); ok {
				return name
			}
		}
	case "labels":
		if v, ok := fields["labels"].([]any); ok {
			var labels []string
			for _, l := range v {
				if s, ok := l.(string); ok {
					labels = append(labels, s)
				}
			}
			sort.Strings(labels)
			return strings.Join(labels, ",")
		}
	case "components":
		if v, ok := fields["components"].([]any); ok {
			var names []string
			for _, c := range v {
				if comp, ok := c.(map[string]any); ok {
					if name, ok := comp["name"].(string); ok {
						names = append(names, name)
					}
				}
			}
			sort.Strings(names)
			return strings.Join(names, ",")
		}
	}
	return ""
}

// ComputeFieldsChecksums computes checksums for the specified fields.
func ComputeFieldsChecksums(fields map[string]any, fieldNames []string) map[string]string {
	checksums := make(map[string]string)
	for _, name := range fieldNames {
		canonical := GetCanonicalFieldValue(name, fields)
		checksums[name] = ComputeFieldChecksum(canonical)
	}
	return checksums
}
