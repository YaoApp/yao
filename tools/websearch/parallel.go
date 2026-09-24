package websearch

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/yaoapp/gou/mcp/client"
	"github.com/yaoapp/gou/mcp/types"
	"github.com/yaoapp/yao/share"
)

const parallelURL = "https://search.parallel.ai/mcp"

// SearchParallel searches through Parallel's Search MCP. An empty key selects
// anonymous access; an explicit key is never retried anonymously.
func SearchParallel(query string, limit int, apiKey string) ([]SearchResult, error) {
	return parallelSearchAt(parallelURL, query, limit, apiKey)
}

func parallelSearchAt(endpoint, query string, limit int, apiKey string) ([]SearchResult, error) {
	if apiKey != "" && strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("parallel API key is blank")
	}
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("parallel search query is empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, err := client.New(&types.ClientDSL{Name: "Yao", Version: share.VERSION, Transport: types.TransportHTTP, URL: endpoint})
	if err != nil {
		return nil, err
	}
	// Identify aggregate project usage without user or installation identifiers.
	headers := map[string]string{"User-Agent": "Yao/" + share.VERSION}
	if apiKey != "" {
		headers["x-api-key"] = apiKey
	}
	if err := c.Connect(ctx, types.ConnectionOptions{Headers: headers}); err != nil {
		return nil, err
	}
	defer c.Disconnect(ctx)
	if _, err := c.Initialize(ctx); err != nil {
		return nil, err
	}
	tools, err := c.ListTools(ctx, "")
	if err != nil {
		return nil, err
	}
	found := false
	for _, tool := range tools.Tools {
		if tool.Name == "web_search" {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("parallel MCP does not advertise web_search")
	}
	result, err := c.CallTool(ctx, "web_search", map[string]interface{}{"objective": query, "search_queries": []string{query}})
	if err != nil {
		return nil, err
	}
	return parseParallelResult(result, limit)
}

func parseParallelResult(result *types.CallToolResponse, limit int) ([]SearchResult, error) {
	if result == nil {
		return nil, fmt.Errorf("parallel MCP returned no result")
	}
	if result.IsError {
		return nil, fmt.Errorf("parallel MCP search failed")
	}
	for _, content := range result.Content {
		if content.Type != types.ToolContentTypeText {
			continue
		}
		var data struct {
			Results *[]struct {
				Title    string   `json:"title"`
				URL      string   `json:"url"`
				Excerpts []string `json:"excerpts"`
			} `json:"results"`
			Warnings interface{} `json:"warnings"`
		}
		if err := json.Unmarshal([]byte(content.Text), &data); err != nil {
			continue
		}
		if data.Results == nil {
			continue
		}
		items := make([]SearchResult, 0, len(*data.Results))
		for _, r := range *data.Results {
			if r.URL == "" {
				return nil, fmt.Errorf("parallel search result has no URL")
			}
			items = append(items, SearchResult{Title: r.Title, URL: r.URL, Content: strings.Join(r.Excerpts, "\n\n")})
			if limit > 0 && len(items) >= limit {
				break
			}
		}
		if data.Warnings != nil {
			warning, err := json.Marshal(data.Warnings)
			if err == nil && string(warning) != "[]" {
				// Diagnostic warnings must not become cited hits or exceed the limit.
				log.Printf("Parallel search warnings: %s", warning)
			}
		}
		return items, nil
	}
	return nil, fmt.Errorf("parallel MCP returned invalid search results")
}
