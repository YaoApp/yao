package decision

import (
	_ "embed"

	"github.com/yaoapp/gou/process"
	agentLLM "github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/llmprovider"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/tools/image"
)

//go:embed providers_schema.json
var ProvidersSchemaJSON []byte

// ProvidersHandler is the tools.decision_providers process handler.
// It lists providers with the "decision" capability, reusing the
// shared ListProvidersByCapability from the image package.
// Each provider is enriched with an "available" field indicating
// whether the connector can be resolved for the current user.
func ProvidersHandler(proc *process.Process) interface{} {
	capability := proc.ArgsString(0, "decision")

	authInfo := authorized.ProcessAuthInfo(proc)
	if authInfo == nil {
		return map[string]interface{}{"error": "unauthorized: no auth info in request"}
	}

	if llmprovider.Global == nil {
		return map[string]interface{}{"error": "llmprovider registry not initialized"}
	}

	providers, err := image.ListProvidersByCapability(capability, authInfo)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	// Enrich each provider with availability status.
	// Decision providers are expected to be few (1-3); synchronous probing
	// is sufficient. If count exceeds 5 in the future, switch to concurrent
	// probing with errgroup (limit 3).
	enriched := make([]map[string]interface{}, 0, len(providers))
	for _, p := range providers {
		m := map[string]interface{}{
			"key":       p.Key,
			"name":      p.Name,
			"models":    p.Models,
			"available": probeAvailable(p.FirstConnectorID(), authInfo),
		}
		enriched = append(enriched, m)
	}

	return map[string]interface{}{
		"capability": capability,
		"providers":  enriched,
	}
}

// probeAvailable checks whether a connector ID can be resolved for the
// given auth context by attempting ResolveConnector.
func probeAvailable(connectorID string, authInfo *oauthTypes.AuthorizedInfo) bool {
	if connectorID == "" {
		return false
	}
	_, _, err := agentLLM.ResolveConnector(connectorID, authInfo)
	return err == nil
}
