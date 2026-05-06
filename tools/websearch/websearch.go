package websearch

import (
	_ "embed"
	"os"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/setting"
)

//go:embed schema.json
var SchemaJSON []byte

// SearchResult is the unified return type for all search providers.
type SearchResult struct {
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Content string  `json:"content"`
	Score   float64 `json:"score,omitempty"`
}

type searchConfig struct {
	Provider  string // "tavily" / "serper" / "cloud"
	APIKey    string
	APIURL    string // cloud mode endpoint
	CloudTool string // cloud search tool name, e.g. "serper-search", "tavily-search"
}

// Handler is the tools.web_search process handler.
// Args[0]: query (string)
// Args[1]: limit (int, default 10)
func Handler(proc *process.Process) interface{} {
	query := proc.ArgsString(0)
	limit := proc.ArgsInt(1, 10)
	userID, teamID := getAuthInfo(proc)
	return Search(query, limit, userID, teamID)
}

// Search executes a web search using the configured provider.
// Reads provider/key from Settings (with ENV fallback).
func Search(query string, limit int, userID, teamID string) []SearchResult {
	cfg := getConfig(userID, teamID)
	switch cfg.Provider {
	case "cloud":
		return cloudSearch(cfg, query, limit)
	case "serper":
		return serperSearch(cfg.APIKey, query, limit)
	default:
		return tavilySearch(cfg.APIKey, query, limit)
	}
}

func getAuthInfo(proc *process.Process) (userID, teamID string) {
	if proc.Authorized != nil {
		userID = proc.Authorized.UserID
		teamID = proc.Authorized.TeamID
	}
	return
}

func getConfig(userID, teamID string) *searchConfig {
	cfg := &searchConfig{Provider: "tavily"}

	if setting.Global != nil {
		assignment, _ := setting.Global.GetMerged(userID, teamID, "search.tool_assignment")
		if v, ok := assignment["web_search"].(string); ok && v != "" {
			cfg.Provider = v
		}
	}

	switch cfg.Provider {
	case "cloud":
		cfg.APIKey, cfg.APIURL, cfg.CloudTool = getCloudConfig(userID, teamID)
	case "tavily":
		cfg.APIKey = getProviderKey(userID, teamID, "tavily")
		if cfg.APIKey == "" {
			cfg.APIKey = os.Getenv("TAVILY_API_KEY")
		}
	case "serper":
		cfg.APIKey = getProviderKey(userID, teamID, "serper")
		if cfg.APIKey == "" {
			cfg.APIKey = os.Getenv("SERPER_API_KEY")
		}
	}
	return cfg
}

func getCloudConfig(userID, teamID string) (apiKey, apiURL, cloudTool string) {
	if setting.Global == nil {
		return
	}
	saved, _ := setting.Global.GetMerged(userID, teamID, "cloud")
	if v, ok := saved["api_url"].(string); ok {
		apiURL = v
	}
	if v, ok := saved["api_key"].(string); ok {
		apiKey = config.DecryptValue(v)
	}
	if v, ok := saved["search_tool"].(string); ok && v != "" {
		cloudTool = v
	}
	return
}

func getProviderKey(userID, teamID, presetKey string) string {
	if setting.Global == nil {
		return ""
	}
	saved, _ := setting.Global.GetMerged(userID, teamID, "search.providers."+presetKey)
	if fv, ok := saved["field_values"].(map[string]interface{}); ok {
		if v, ok := fv["api_key"].(string); ok {
			return config.DecryptValue(v)
		}
	}
	return ""
}
