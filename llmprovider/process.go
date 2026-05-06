package llmprovider

import (
	"encoding/json"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

func init() {
	process.RegisterGroup("llmprovider", map[string]process.Handler{
		// --- existing ---
		"get":        ProcessGet,
		"getmasked":  ProcessGetMasked,
		"create":     ProcessCreate,
		"update":     ProcessUpdate,
		"delete":     ProcessDelete,
		"list":       ProcessList,
		"getsetting": ProcessGetSetting,
		"getpresets": ProcessGetPresets,
		"getpreset":  ProcessGetPreset,

		// --- roles ---
		"getrole":         ProcessGetRole,
		"getrolebyuser":   ProcessGetRoleByUser,
		"getrolebyteam":   ProcessGetRoleByTeam,
		"listroles":       ProcessListRoles,
		"listrolesbyuser": ProcessListRolesByUser,
		"listrolesbyteam": ProcessListRolesByTeam,

		// --- models ---
		"getmodel":                ProcessGetModel,
		"getrolemodel":            ProcessGetRoleModel,
		"getrolemodelbyuser":      ProcessGetRoleModelByUser,
		"getrolemodelbyteam":      ProcessGetRoleModelByTeam,
		"getdefaultmodel":         ProcessGetDefaultModel,
		"getdefaultmodelbyuser":   ProcessGetDefaultModelByUser,
		"getdefaultmodelbyteam":   ProcessGetDefaultModelByTeam,
		"getvisionmodel":          ProcessGetVisionModel,
		"getvisionmodelbyuser":    ProcessGetVisionModelByUser,
		"getvisionmodelbyteam":    ProcessGetVisionModelByTeam,
		"getaudiomodel":           ProcessGetAudioModel,
		"getaudiomodelbyuser":     ProcessGetAudioModelByUser,
		"getaudiomodelbyteam":     ProcessGetAudioModelByTeam,
		"getembeddingmodel":       ProcessGetEmbeddingModel,
		"getembeddingmodelbyuser": ProcessGetEmbeddingModelByUser,
		"getembeddingmodelbyteam": ProcessGetEmbeddingModelByTeam,

		// --- capabilities ---
		"getcapabilities":           ProcessGetCapabilities,
		"getrolecapabilities":       ProcessGetRoleCapabilities,
		"getrolecapabilitiesbyuser": ProcessGetRoleCapabilitiesByUser,
		"getrolecapabilitiesbyteam": ProcessGetRoleCapabilitiesByTeam,

		// --- list models ---
		"listmodels":       ProcessListModels,
		"listmodelsbyuser": ProcessListModelsByUser,
		"listmodelsbyteam": ProcessListModelsByTeam,
	})
}

func requireGlobal() {
	if Global == nil {
		exception.New("LLM Provider Registry not initialized", 500).Throw()
	}
}

// ProcessGet retrieves a provider by key.
// Args[0] string: provider key
// Args[1] bool:   withKey (optional, default false) — true returns plain-text APIKey
func ProcessGet(p *process.Process) interface{} {
	requireGlobal()
	p.ValidateArgNums(1)
	key := p.ArgsString(0)

	withKey := len(p.Args) > 1 && toBool(p.Args[1])
	provider, err := Global.Get(key, withKey)
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return provider
}

// ProcessGetMasked retrieves a provider with API key masked.
// Args[0] string: provider key
func ProcessGetMasked(p *process.Process) interface{} {
	requireGlobal()
	p.ValidateArgNums(1)
	key := p.ArgsString(0)

	provider, err := Global.GetMasked(key)
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return provider
}

// ProcessCreate adds a new provider.
// Args[0] map: Provider data
func ProcessCreate(p *process.Process) interface{} {
	requireGlobal()
	p.ValidateArgNums(1)

	var provider Provider
	raw, err := json.Marshal(p.Args[0])
	if err != nil {
		exception.New("invalid provider data: "+err.Error(), 400).Throw()
	}
	if err := json.Unmarshal(raw, &provider); err != nil {
		exception.New("invalid provider data: "+err.Error(), 400).Throw()
	}

	result, err := Global.Create(&provider)
	if err != nil {
		exception.New(err.Error(), 400).Throw()
	}
	return result
}

// ProcessUpdate modifies an existing provider.
// Args[0] string: provider key
// Args[1] map: Provider data
func ProcessUpdate(p *process.Process) interface{} {
	requireGlobal()
	p.ValidateArgNums(2)
	key := p.ArgsString(0)

	var provider Provider
	raw, err := json.Marshal(p.Args[1])
	if err != nil {
		exception.New("invalid provider data: "+err.Error(), 400).Throw()
	}
	if err := json.Unmarshal(raw, &provider); err != nil {
		exception.New("invalid provider data: "+err.Error(), 400).Throw()
	}

	result, err := Global.Update(key, &provider)
	if err != nil {
		exception.New(err.Error(), 400).Throw()
	}
	return result
}

// ProcessDelete removes a provider by key.
// Args[0] string: provider key
func ProcessDelete(p *process.Process) interface{} {
	requireGlobal()
	p.ValidateArgNums(1)
	key := p.ArgsString(0)

	if err := Global.Delete(key); err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return nil
}

// ProcessList returns providers matching a filter.
// Args[0] map:  ProviderFilter (optional)
// Args[1] bool: withKey (optional, default false) — true returns plain-text APIKeys
func ProcessList(p *process.Process) interface{} {
	requireGlobal()

	var filter *ProviderFilter
	if len(p.Args) > 0 && p.Args[0] != nil {
		raw, err := json.Marshal(p.Args[0])
		if err == nil {
			var f ProviderFilter
			if json.Unmarshal(raw, &f) == nil {
				filter = &f
			}
		}
	}

	withKey := len(p.Args) > 1 && toBool(p.Args[1])
	result, err := Global.List(filter, withKey)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return result
}

// ProcessGetSetting returns the runtime connector setting map.
// Args[0] string: provider key
func ProcessGetSetting(p *process.Process) interface{} {
	requireGlobal()
	p.ValidateArgNums(1)
	key := p.ArgsString(0)

	setting, err := Global.GetSetting(key)
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return setting
}

// ProcessGetPresets returns all provider presets.
func ProcessGetPresets(p *process.Process) interface{} {
	return GetPresets()
}

// ProcessGetPreset returns a single preset by key.
// Args[0] string: preset key
func ProcessGetPreset(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	key := p.ArgsString(0)

	preset := GetPreset(key)
	if preset == nil {
		exception.New("preset "+key+" not found", 404).Throw()
	}
	return preset
}

// ---------------------------------------------------------------------------
// Roles
// ---------------------------------------------------------------------------

// ProcessGetRole returns the connectorID for a role (system scope).
// Args[0] string: role name
func ProcessGetRole(p *process.Process) interface{} {
	requireGlobal()
	p.ValidateArgNums(1)
	cid, err := Global.GetRole(p.ArgsString(0))
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return cid
}

// ProcessGetRoleByUser returns the connectorID for a role (user > system merge).
// Args[0] string: role, Args[1] string: userID
func ProcessGetRoleByUser(p *process.Process) interface{} {
	requireGlobal()
	p.ValidateArgNums(2)
	cid, err := Global.GetRoleByUser(p.ArgsString(0), p.ArgsString(1))
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return cid
}

// ProcessGetRoleByTeam returns the connectorID for a role (team > system merge).
// Args[0] string: role, Args[1] string: teamID
func ProcessGetRoleByTeam(p *process.Process) interface{} {
	requireGlobal()
	p.ValidateArgNums(2)
	cid, err := Global.GetRoleByTeam(p.ArgsString(0), p.ArgsString(1))
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return cid
}

// ProcessListRoles returns all role assignments (system scope).
func ProcessListRoles(p *process.Process) interface{} {
	requireGlobal()
	roles, err := Global.ListRoles()
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return rolesToMap(roles)
}

// ProcessListRolesByUser returns all role assignments (user > system merge).
// Args[0] string: userID
func ProcessListRolesByUser(p *process.Process) interface{} {
	requireGlobal()
	p.ValidateArgNums(1)
	roles, err := Global.ListRolesByUser(p.ArgsString(0))
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return rolesToMap(roles)
}

// ProcessListRolesByTeam returns all role assignments (team > system merge).
// Args[0] string: teamID
func ProcessListRolesByTeam(p *process.Process) interface{} {
	requireGlobal()
	p.ValidateArgNums(1)
	roles, err := Global.ListRolesByTeam(p.ArgsString(0))
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return rolesToMap(roles)
}

// ---------------------------------------------------------------------------
// Models
// ---------------------------------------------------------------------------

// ProcessGetModel returns the connector setting map by connectorID.
// Args[0] string: connectorID
func ProcessGetModel(p *process.Process) interface{} {
	requireGlobal()
	p.ValidateArgNums(1)
	conn, err := Global.GetModel(p.ArgsString(0))
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return conn.Setting()
}

// ProcessGetRoleModel returns the connector setting map for a role (system scope).
// Args[0] string: role
func ProcessGetRoleModel(p *process.Process) interface{} {
	requireGlobal()
	p.ValidateArgNums(1)
	conn, err := Global.GetRoleModel(p.ArgsString(0))
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return conn.Setting()
}

// ProcessGetRoleModelByUser returns the connector setting map for a role (user scope).
// Args[0] string: role, Args[1] string: userID
func ProcessGetRoleModelByUser(p *process.Process) interface{} {
	requireGlobal()
	p.ValidateArgNums(2)
	conn, err := Global.GetRoleModelByUser(p.ArgsString(0), p.ArgsString(1))
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return conn.Setting()
}

// ProcessGetRoleModelByTeam returns the connector setting map for a role (team scope).
// Args[0] string: role, Args[1] string: teamID
func ProcessGetRoleModelByTeam(p *process.Process) interface{} {
	requireGlobal()
	p.ValidateArgNums(2)
	conn, err := Global.GetRoleModelByTeam(p.ArgsString(0), p.ArgsString(1))
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return conn.Setting()
}

// ProcessGetDefaultModel returns the default model connector setting map.
func ProcessGetDefaultModel(p *process.Process) interface{} {
	requireGlobal()
	conn, err := Global.GetDefaultModel()
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return conn.Setting()
}

// ProcessGetDefaultModelByUser returns the default model for a user.
// Args[0] string: userID
func ProcessGetDefaultModelByUser(p *process.Process) interface{} {
	requireGlobal()
	p.ValidateArgNums(1)
	conn, err := Global.GetDefaultModelByUser(p.ArgsString(0))
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return conn.Setting()
}

// ProcessGetDefaultModelByTeam returns the default model for a team.
// Args[0] string: teamID
func ProcessGetDefaultModelByTeam(p *process.Process) interface{} {
	requireGlobal()
	p.ValidateArgNums(1)
	conn, err := Global.GetDefaultModelByTeam(p.ArgsString(0))
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return conn.Setting()
}

// ProcessGetVisionModel returns the vision model connector setting map.
func ProcessGetVisionModel(p *process.Process) interface{} {
	requireGlobal()
	conn, err := Global.GetVisionModel()
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return conn.Setting()
}

// ProcessGetVisionModelByUser returns the vision model for a user.
// Args[0] string: userID
func ProcessGetVisionModelByUser(p *process.Process) interface{} {
	requireGlobal()
	p.ValidateArgNums(1)
	conn, err := Global.GetVisionModelByUser(p.ArgsString(0))
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return conn.Setting()
}

// ProcessGetVisionModelByTeam returns the vision model for a team.
// Args[0] string: teamID
func ProcessGetVisionModelByTeam(p *process.Process) interface{} {
	requireGlobal()
	p.ValidateArgNums(1)
	conn, err := Global.GetVisionModelByTeam(p.ArgsString(0))
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return conn.Setting()
}

// ProcessGetAudioModel returns the audio model connector setting map.
func ProcessGetAudioModel(p *process.Process) interface{} {
	requireGlobal()
	conn, err := Global.GetAudioModel()
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return conn.Setting()
}

// ProcessGetAudioModelByUser returns the audio model for a user.
// Args[0] string: userID
func ProcessGetAudioModelByUser(p *process.Process) interface{} {
	requireGlobal()
	p.ValidateArgNums(1)
	conn, err := Global.GetAudioModelByUser(p.ArgsString(0))
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return conn.Setting()
}

// ProcessGetAudioModelByTeam returns the audio model for a team.
// Args[0] string: teamID
func ProcessGetAudioModelByTeam(p *process.Process) interface{} {
	requireGlobal()
	p.ValidateArgNums(1)
	conn, err := Global.GetAudioModelByTeam(p.ArgsString(0))
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return conn.Setting()
}

// ProcessGetEmbeddingModel returns the embedding model connector setting map.
func ProcessGetEmbeddingModel(p *process.Process) interface{} {
	requireGlobal()
	conn, err := Global.GetEmbeddingModel()
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return conn.Setting()
}

// ProcessGetEmbeddingModelByUser returns the embedding model for a user.
// Args[0] string: userID
func ProcessGetEmbeddingModelByUser(p *process.Process) interface{} {
	requireGlobal()
	p.ValidateArgNums(1)
	conn, err := Global.GetEmbeddingModelByUser(p.ArgsString(0))
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return conn.Setting()
}

// ProcessGetEmbeddingModelByTeam returns the embedding model for a team.
// Args[0] string: teamID
func ProcessGetEmbeddingModelByTeam(p *process.Process) interface{} {
	requireGlobal()
	p.ValidateArgNums(1)
	conn, err := Global.GetEmbeddingModelByTeam(p.ArgsString(0))
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return conn.Setting()
}

// ---------------------------------------------------------------------------
// Capabilities
// ---------------------------------------------------------------------------

// ProcessGetCapabilities returns capabilities for a connectorID.
// Args[0] string: connectorID
func ProcessGetCapabilities(p *process.Process) interface{} {
	requireGlobal()
	p.ValidateArgNums(1)
	caps, err := Global.GetCapabilities(p.ArgsString(0))
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return capsToMap(caps)
}

// ProcessGetRoleCapabilities returns capabilities for a role (system scope).
// Args[0] string: role
func ProcessGetRoleCapabilities(p *process.Process) interface{} {
	requireGlobal()
	p.ValidateArgNums(1)
	caps, err := Global.GetRoleCapabilities(p.ArgsString(0))
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return capsToMap(caps)
}

// ProcessGetRoleCapabilitiesByUser returns capabilities for a role (user scope).
// Args[0] string: role, Args[1] string: userID
func ProcessGetRoleCapabilitiesByUser(p *process.Process) interface{} {
	requireGlobal()
	p.ValidateArgNums(2)
	caps, err := Global.GetRoleCapabilitiesByUser(p.ArgsString(0), p.ArgsString(1))
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return capsToMap(caps)
}

// ProcessGetRoleCapabilitiesByTeam returns capabilities for a role (team scope).
// Args[0] string: role, Args[1] string: teamID
func ProcessGetRoleCapabilitiesByTeam(p *process.Process) interface{} {
	requireGlobal()
	p.ValidateArgNums(2)
	caps, err := Global.GetRoleCapabilitiesByTeam(p.ArgsString(0), p.ArgsString(1))
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return capsToMap(caps)
}

// ---------------------------------------------------------------------------
// List Models
// ---------------------------------------------------------------------------

// ProcessListModels returns all enabled models as []Option (system scope).
func ProcessListModels(p *process.Process) interface{} {
	requireGlobal()
	return optionsToSlice(Global.ListModels())
}

// ProcessListModelsByUser returns models visible to a user.
// Args[0] string: userID
func ProcessListModelsByUser(p *process.Process) interface{} {
	requireGlobal()
	p.ValidateArgNums(1)
	return optionsToSlice(Global.ListModelsByUser(p.ArgsString(0)))
}

// ProcessListModelsByTeam returns models visible to a team.
// Args[0] string: teamID
func ProcessListModelsByTeam(p *process.Process) interface{} {
	requireGlobal()
	p.ValidateArgNums(1)
	return optionsToSlice(Global.ListModelsByTeam(p.ArgsString(0)))
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func rolesToMap(roles map[string]RoleTarget) map[string]interface{} {
	result := make(map[string]interface{}, len(roles))
	for k, v := range roles {
		result[k] = map[string]interface{}{
			"provider": v.Provider,
			"model":    v.Model,
		}
	}
	return result
}

func optionsToSlice(opts []connector.Option) []interface{} {
	result := make([]interface{}, len(opts))
	for i, o := range opts {
		result[i] = map[string]interface{}{
			"label": o.Label,
			"value": o.Value,
		}
	}
	return result
}

func toBool(v interface{}) bool {
	switch b := v.(type) {
	case bool:
		return b
	case float64:
		return b != 0
	case int:
		return b != 0
	case string:
		return b == "true" || b == "1"
	default:
		return false
	}
}
