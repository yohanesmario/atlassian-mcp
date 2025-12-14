package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Credentials holds Atlassian API credentials.
var (
	Email  string
	Token  string
	Domain string
)

// Pre-compiled regexes for input validation
var (
	// Jira patterns
	issueKeyPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]+-\d+$`)
	issueURLPattern = regexp.MustCompile(`^https://[a-zA-Z0-9-]+\.atlassian\.net/browse/([A-Z][A-Z0-9]+-\d+)$`)

	// Confluence patterns
	pageURLPattern = regexp.MustCompile(`^https://([a-zA-Z0-9-]+)\.atlassian\.net/wiki/spaces/([A-Za-z0-9_-]+)/pages/(\d+)(?:/.*)?$`)
	pageIDPattern  = regexp.MustCompile(`^\d+$`)
)

const (
	maxIssueKeyLength = 50
	maxInputLength    = 500
)

// loadEnvFile loads environment variables from a .env file in the binary's directory.
func loadEnvFile() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exeDir := filepath.Dir(exe)
	envPath := filepath.Join(exeDir, ".env")

	info, err := os.Stat(envPath)
	if err != nil {
		return err
	}

	mode := info.Mode().Perm()
	if mode&0077 != 0 {
		return fmt.Errorf(".env file has insecure permissions (%04o). Run: chmod 600 %s", mode, envPath)
	}

	file, err := os.Open(envPath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
	return scanner.Err()
}

func init() {
	_ = loadEnvFile()

	Email = os.Getenv("ATLASSIAN_EMAIL")
	Token = os.Getenv("ATLASSIAN_API_TOKEN")
	Domain = os.Getenv("ATLASSIAN_DOMAIN")

	if Domain != "" {
		if !strings.HasSuffix(Domain, ".atlassian.net") {
			fmt.Fprintln(os.Stderr, "Error: ATLASSIAN_DOMAIN must be an atlassian.net domain")
			os.Exit(1)
		}
		if strings.Contains(Domain, "/") || strings.Contains(Domain, ":") {
			fmt.Fprintln(os.Stderr, "Error: ATLASSIAN_DOMAIN must be a domain only (no protocol or path)")
			os.Exit(1)
		}
	}
}

// JiraBaseURL returns the base URL for Jira API requests.
func JiraBaseURL() string {
	return fmt.Sprintf("https://%s", Domain)
}

// ConfluenceBaseURL returns the base URL for Confluence API requests.
func ConfluenceBaseURL() string {
	return fmt.Sprintf("https://%s/wiki", Domain)
}

// ExtractIssueKey extracts issue key from URL or returns input if already a key.
// Supports: https://domain.atlassian.net/browse/PROJ-123 or just PROJ-123
func ExtractIssueKey(input string) (string, error) {
	input = strings.TrimSpace(input)

	if issueKeyPattern.MatchString(input) {
		if len(input) > maxIssueKeyLength {
			return "", fmt.Errorf("issue key too long (max %d characters)", maxIssueKeyLength)
		}
		return input, nil
	}

	matches := issueURLPattern.FindStringSubmatch(input)
	if len(matches) == 2 {
		key := matches[1]
		if len(key) > maxIssueKeyLength {
			return "", fmt.Errorf("issue key too long (max %d characters)", maxIssueKeyLength)
		}
		return key, nil
	}

	return "", fmt.Errorf("invalid input: must be PROJ-123 format or full Jira URL")
}

// ExtractPageID extracts page ID from URL or returns input if already an ID.
func ExtractPageID(input string) (string, error) {
	input = strings.TrimSpace(input)

	if len(input) > maxInputLength {
		return "", fmt.Errorf("input too long (max %d characters)", maxInputLength)
	}

	if pageIDPattern.MatchString(input) {
		return input, nil
	}

	matches := pageURLPattern.FindStringSubmatch(input)
	if len(matches) >= 4 {
		return matches[3], nil
	}

	return "", fmt.Errorf("invalid input: must be page ID or full Confluence URL (e.g., https://domain.atlassian.net/wiki/spaces/SPACE/pages/123456/Title)")
}
