package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var negativePatterns = regexp.MustCompile(`(?i)(don'?t know|i cannot|no idea|not sure|unable to|i can'?t)`)

// buildValidatorResponse analyses the last user message and returns a JSON
// object with {"passed": bool, "reason": string}.  The heuristic is
// intentionally simple: if the embedded "output" text contains a negative
// indicator phrase the validation fails; otherwise it passes.
func buildValidatorResponse(messages json.RawMessage) string {
	msg := rawLastUserMessage(messages)
	passed := !negativePatterns.MatchString(msg)
	reason := "Mock validation: criteria met"
	if !passed {
		reason = "Mock validation: output does not meet criteria"
	}
	b, _ := json.Marshal(map[string]interface{}{
		"passed": passed,
		"reason": reason,
	})
	return string(b)
}

// buildGeneratorResponse parses the user message to extract a "count" field
// and returns a JSON array of test cases that the test framework can consume.
func buildGeneratorResponse(messages json.RawMessage) string {
	msg := rawLastUserMessage(messages)
	count := 3
	if n := extractCount(msg); n > 0 {
		count = n
	}

	cases := make([]map[string]interface{}, 0, count)
	for i := 1; i <= count; i++ {
		cases = append(cases, map[string]interface{}{
			"id":    fmt.Sprintf("gen-%d", i),
			"input": fmt.Sprintf("Test input %d", i),
			"assert": map[string]interface{}{
				"type":  "regex",
				"value": "(?i)(echo|test|hello|hi|input)",
			},
		})
	}
	b, _ := json.Marshal(cases)
	return string(b)
}

// rawLastUserMessage returns the raw content string of the last user message
// without the "echo: " prefix that extractLastUserMessage adds.
func rawLastUserMessage(messages json.RawMessage) string {
	var msgs []struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(messages, &msgs); err != nil {
		return ""
	}
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			var text string
			if err := json.Unmarshal(msgs[i].Content, &text); err == nil {
				return text
			}
			var blocks []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}
			if err := json.Unmarshal(msgs[i].Content, &blocks); err == nil {
				for _, b := range blocks {
					if b.Type == "text" {
						return b.Text
					}
				}
			}
		}
	}
	return ""
}

var countRe = regexp.MustCompile(`"count"\s*:\s*(\d+)`)

func extractCount(s string) int {
	m := countRe.FindStringSubmatch(s)
	if len(m) < 2 {
		if idx := strings.Index(s, "count="); idx >= 0 {
			rest := s[idx+6:]
			end := strings.IndexAny(rest, "&\" ,}\n")
			if end < 0 {
				end = len(rest)
			}
			if n, err := strconv.Atoi(rest[:end]); err == nil {
				return n
			}
		}
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}
