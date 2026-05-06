package webfetch

import (
	_ "embed"
	"os"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/setting"
)

//go:embed schema.json
var SchemaJSON []byte

// FetchResponse is the return type for the webfetch tool.
type FetchResponse struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Format  string `json:"format"`
}

type fetchConfig struct {
	Provider       string // "cloud" / "brightdata" / "" (direct)
	APIKey         string
	APIURL         string // cloud mode endpoint
	BrightdataKey  string
	BrightdataZone string
}

// Handler is the tools.web_fetch process handler.
// Args[0]: url (string)
// Args[1]: format (string, default "markdown")
func Handler(proc *process.Process) interface{} {
	url := proc.ArgsString(0)
	format := proc.ArgsString(1, "markdown")
	userID, teamID := getAuthInfo(proc)
	cfg := getConfig(userID, teamID)

	switch cfg.Provider {
	case "cloud":
		return cloudFetch(cfg, url, format)
	default:
		return localFetch(cfg, url, format)
	}
}

func getAuthInfo(proc *process.Process) (userID, teamID string) {
	if proc.Authorized != nil {
		userID = proc.Authorized.UserID
		teamID = proc.Authorized.TeamID
	}
	return
}

func getConfig(userID, teamID string) *fetchConfig {
	cfg := &fetchConfig{}

	if setting.Global != nil {
		assignment, _ := setting.Global.GetMerged(userID, teamID, "search.tool_assignment")
		if v, ok := assignment["web_scrape"].(string); ok && v != "" {
			cfg.Provider = v
		}
	}

	switch cfg.Provider {
	case "cloud":
		cfg.APIKey, cfg.APIURL = getCloudConfig(userID, teamID)
	case "brightdata":
		cfg.BrightdataKey, cfg.BrightdataZone = getBrightdataConfig(userID, teamID)
	default:
		cfg.BrightdataKey, cfg.BrightdataZone = getBrightdataConfig(userID, teamID)
	}
	return cfg
}

func getCloudConfig(userID, teamID string) (apiKey, apiURL string) {
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
	return
}

func getBrightdataConfig(userID, teamID string) (apiKey, zone string) {
	if setting.Global != nil {
		saved, _ := setting.Global.GetMerged(userID, teamID, "search.providers.brightdata")
		if fv, ok := saved["field_values"].(map[string]interface{}); ok {
			if v, ok := fv["api_key"].(string); ok {
				apiKey = config.DecryptValue(v)
			}
			if v, ok := fv["zone"].(string); ok {
				zone = v
			}
		}
	}
	if apiKey == "" {
		apiKey = os.Getenv("BRIGHTDATA_API_KEY")
	}
	if zone == "" {
		zone = os.Getenv("BRIGHTDATA_ZONE")
	}
	return
}

func localFetch(cfg *fetchConfig, url, format string) *FetchResponse {
	switch format {
	case "html":
		return fetchHTML(cfg, url)
	default:
		return fetchMarkdown(cfg, url)
	}
}
