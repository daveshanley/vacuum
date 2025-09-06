// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package cui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
)

// fetchDocsFromDoctorAPI creates a command to fetch documentation for a rule from the doctor API.
func fetchDocsFromDoctorAPI(ruleID string) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		url := fmt.Sprintf("https://localhost:9090/rules/documentation/%s?markdown", ruleID)
		resp, err := client.Get(url)
		if err != nil {
			return docsErrorMsg{ruleID: ruleID, err: err.Error(), is404: false}
		}
		defer resp.Body.Close()

		if resp.StatusCode == 404 {
			return docsErrorMsg{ruleID: ruleID, err: "Documentation not found", is404: true}
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return docsErrorMsg{ruleID: ruleID, err: err.Error(), is404: false}
		}

		if resp.StatusCode != 200 {
			// Try to parse RFC 9457 error response
			var errorResponse struct {
				Type   string `json:"type"`
				Title  string `json:"title"`
				Status int    `json:"status"`
				Detail string `json:"detail"`
			}

			if err = json.Unmarshal(body, &errorResponse); err == nil && errorResponse.Detail != "" {
				// Use the detail from RFC 9457 error response
				return docsErrorMsg{ruleID: ruleID, err: errorResponse.Detail, is404: false}
			}

			// Fallback to generic HTTP error
			return docsErrorMsg{ruleID: ruleID, err: fmt.Sprintf("HTTP %d", resp.StatusCode), is404: false}
		}

		var docResponse struct {
			RuleID   string `json:"ruleId"`
			Category string `json:"category"`
			Body     string `json:"body"`
		}

		if err := json.Unmarshal(body, &docResponse); err != nil {
			return docsErrorMsg{ruleID: ruleID, err: fmt.Sprintf("Failed to parse JSON: %s", err.Error()), is404: false}
		}

		// process shortcodes in docs
		processedContent := ConvertHugoShortcodesToMarkdown(docResponse.Body)

		return docsLoadedMsg{ruleID: ruleID, content: processedContent}
	}
}
