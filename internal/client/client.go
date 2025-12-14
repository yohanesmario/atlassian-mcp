package client

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"time"

	"atlassian-mcp/internal/config"
)

// HTTPClient is the shared HTTP client with timeout and TLS hardening.
var HTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	},
}

// Service identifies which Atlassian service to use.
type Service string

const (
	Jira       Service = "jira"
	Confluence Service = "confluence"
)

func baseURL(svc Service) string {
	switch svc {
	case Confluence:
		return config.ConfluenceBaseURL()
	default:
		return config.JiraBaseURL()
	}
}

func serviceName(svc Service) string {
	switch svc {
	case Confluence:
		return "Confluence"
	default:
		return "Jira"
	}
}

func authHeader() string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(config.Email+":"+config.Token))
}

func handleStatusCode(svc Service, statusCode int) error {
	name := serviceName(svc)
	switch statusCode {
	case 400:
		return fmt.Errorf("bad request (HTTP 400)")
	case 401:
		return fmt.Errorf("authentication failed (HTTP 401)")
	case 403:
		return fmt.Errorf("access denied (HTTP 403)")
	case 404:
		return fmt.Errorf("not found or no permission (HTTP 404)")
	default:
		return fmt.Errorf("%s API error (HTTP %d)", name, statusCode)
	}
}

// Request performs a GET request to the specified service.
func Request(svc Service, endpoint string) ([]byte, error) {
	url := baseURL(svc) + endpoint

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request")
	}

	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Accept", "application/json")

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s", serviceName(svc))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response")
	}

	if resp.StatusCode != 200 {
		return nil, handleStatusCode(svc, resp.StatusCode)
	}

	return body, nil
}

// Post performs a POST request to the specified service.
func Post(svc Service, endpoint string, body []byte) ([]byte, error) {
	url := baseURL(svc) + endpoint

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request")
	}

	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s", serviceName(svc))
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, handleStatusCode(svc, resp.StatusCode)
	}

	return respBody, nil
}

// Put performs a PUT request to the specified service.
func Put(svc Service, endpoint string, body []byte) ([]byte, error) {
	url := baseURL(svc) + endpoint

	req, err := http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request")
	}

	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s", serviceName(svc))
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, handleStatusCode(svc, resp.StatusCode)
	}

	return respBody, nil
}
