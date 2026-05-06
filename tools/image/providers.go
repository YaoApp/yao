package image

import (
	_ "embed"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/llmprovider"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
)

//go:embed providers_schema.json
var ProvidersSchemaJSON []byte

type providerResult struct {
	Key    string        `json:"key"`
	Name   string        `json:"name"`
	Models []modelResult `json:"models"`
}

type modelResult struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ConnectorID string `json:"connector_id"`
}

// ProvidersHandler is the tools.image_providers process handler.
func ProvidersHandler(proc *process.Process) interface{} {
	capability := proc.ArgsString(0, "image_generation")

	authInfo := authorized.ProcessAuthInfo(proc)
	if authInfo == nil {
		return map[string]interface{}{"error": "unauthorized: no auth info in request"}
	}

	if llmprovider.Global == nil {
		return map[string]interface{}{"error": "llmprovider registry not initialized"}
	}

	providers, err := listProvidersByCapability(capability, authInfo)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{
		"capability": capability,
		"providers":  providers,
	}
}

// listProvidersByCapability returns providers matching the given capability,
// filtered by owner scope (builtin always included, dynamic filtered by auth).
func listProvidersByCapability(capability string, authInfo *oauthTypes.AuthorizedInfo) ([]providerResult, error) {
	enabled := true
	filter := &llmprovider.ProviderFilter{
		Capabilities: []string{capability},
		Enabled:      &enabled,
		Source:       llmprovider.ProviderSourceAll,
	}

	allProviders, err := llmprovider.Global.List(filter)
	if err != nil {
		return nil, err
	}

	var results []providerResult
	for _, p := range allProviders {
		if p.Source == llmprovider.ProviderSourceDynamic && authInfo != nil {
			if !dynamicOwnerMatch(&p.Owner, authInfo) {
				continue
			}
		}

		var models []modelResult
		for _, m := range p.Models {
			if !m.Enabled {
				continue
			}
			if !modelHasCapability(m.Capabilities, capability) {
				continue
			}

			cid := p.ConnectorID
			if p.Source == llmprovider.ProviderSourceDynamic && m.ID != "" {
				cid = p.ConnectorID + ":" + m.ID
			}

			name := m.Name
			if name == "" {
				name = m.ID
			}
			models = append(models, modelResult{
				ID:          m.ID,
				Name:        name,
				ConnectorID: cid,
			})
		}

		if len(models) == 0 {
			continue
		}

		results = append(results, providerResult{
			Key:    p.Key,
			Name:   p.Name,
			Models: models,
		})
	}
	return results, nil
}

// findFirstImageGenConnector returns the connector ID of the first available
// image generation provider, or empty string if none found.
func findFirstImageGenConnector(authInfo *oauthTypes.AuthorizedInfo) string {
	if llmprovider.Global == nil {
		return ""
	}
	providers, err := listProvidersByCapability("image_generation", authInfo)
	if err != nil || len(providers) == 0 {
		return ""
	}
	if len(providers[0].Models) == 0 {
		return ""
	}
	return providers[0].Models[0].ConnectorID
}

func dynamicOwnerMatch(owner *llmprovider.ProviderOwner, authInfo *oauthTypes.AuthorizedInfo) bool {
	if authInfo.GetTeamID() != "" {
		return owner.Type == "team" && owner.TeamID == authInfo.GetTeamID()
	}
	return owner.Type == "user" && owner.UserID == authInfo.GetUserID()
}

func modelHasCapability(caps []string, target string) bool {
	for _, c := range caps {
		if c == target {
			return true
		}
	}
	return false
}
