package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"atlassian-mcp/internal/config"
	"atlassian-mcp/internal/handler"
	"atlassian-mcp/internal/types"
)

func main() {
	if config.Email == "" || config.Token == "" || config.Domain == "" {
		fmt.Fprintln(os.Stderr, "Error: ATLASSIAN_EMAIL, ATLASSIAN_API_TOKEN, and ATLASSIAN_DOMAIN environment variables must be set")
		os.Exit(1)
	}

	scanner := bufio.NewScanner(os.Stdin)
	// Increase buffer size for large messages
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var req types.Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			continue
		}

		resp := handler.HandleRequest(req)
		if resp.ID == nil && resp.Result == nil && resp.Error == nil {
			// Skip empty responses (notifications)
			continue
		}

		respBytes, _ := json.Marshal(resp)
		fmt.Println(string(respBytes))
	}
}
