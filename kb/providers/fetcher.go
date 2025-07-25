package providers

import (
	"github.com/yaoapp/gou/graphrag/fetcher"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/yao/kb/providers/factory"
	kbtypes "github.com/yaoapp/yao/kb/types"
)

// FetcherHTTP is a fetcher provider for HTTP/HTTPS URLs
type FetcherHTTP struct{}

// FetcherMCP is a fetcher provider for MCP-based URL fetching
type FetcherMCP struct{}

// AutoRegister registers the fetcher providers
func init() {
	factory.Fetchers["__yao.http"] = &FetcherHTTP{}
	factory.Fetchers["__yao.mcp"] = &FetcherMCP{}
}

// === FetcherHTTP ===

// Make creates a new HTTP fetcher
func (f *FetcherHTTP) Make(option *kbtypes.ProviderOption) (types.Fetcher, error) {
	// TODO: Map kbtypes.ProviderOption to fetcher.HTTPOptions
	httpOptions := &fetcher.HTTPOptions{
		// Headers:   nil, // TODO: Get headers from option
		// UserAgent: "", // Will use default
		// Timeout:   0,  // Will use default
	}
	return fetcher.NewHTTPFetcher(httpOptions), nil
}

// Schema returns the schema for the HTTP fetcher
func (f *FetcherHTTP) Schema(provider *kbtypes.Provider) (*kbtypes.ProviderSchema, error) {
	return nil, nil
}

// === FetcherMCP ===

// Make creates a new MCP fetcher
func (f *FetcherMCP) Make(option *kbtypes.ProviderOption) (types.Fetcher, error) {
	// TODO: Map kbtypes.ProviderOption to fetcher.MCPOptions
	mcpOptions := &fetcher.MCPOptions{
		// ID:                  "", // TODO: Get ID from option
		// Tool:                "", // TODO: Get tool from option
		// ArgumentsMapping:    nil, // TODO: Get arguments mapping from option
		// ResultMapping:       nil, // TODO: Get result mapping from option
		// NotificationMapping: nil, // TODO: Get notification mapping from option
	}
	return fetcher.NewMCP(mcpOptions)
}

// Schema returns the schema for the MCP fetcher
func (f *FetcherMCP) Schema(provider *kbtypes.Provider) (*kbtypes.ProviderSchema, error) {
	return nil, nil
}
