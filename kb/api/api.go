package api

import (
	"github.com/yaoapp/gou/graphrag/types"
	kbtypes "github.com/yaoapp/yao/kb/types"
)

// NewAPI creates a new API instance with the provided KB dependencies
func NewAPI(graphRag types.GraphRag, config *kbtypes.Config, providers *kbtypes.ProviderConfig) API {
	return &KBInstance{
		GraphRag:  graphRag,
		Config:    config,
		Providers: providers,
	}
}
