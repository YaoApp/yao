package llm

import (
	"encoding/json"
	"fmt"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

func init() {
	process.RegisterGroup("yao.llm", map[string]process.Handler{
		// Provider CRUD
		"provider.get":    processProviderGet,
		"provider.create": processProviderCreate,
		"provider.update": processProviderUpdate,
		"provider.delete": processProviderDelete,
		"provider.list":   processProviderList,
		"provider.find":   processProviderFind,

		// Model CRUD
		"model.get":    processModelGet,
		"model.create": processModelCreate,
		"model.update": processModelUpdate,
		"model.delete": processModelDelete,
		"model.list":   processModelList,
		"model.find":   processModelFind,

		// Role bindings
		"role.resolve": processRoleResolve,
		"role.get":     processRoleGet,
		"role.set":     processRoleSet,
		"role.delete":  processRoleDelete,

		// Presets
		"preset.list":   processPresetList,
		"preset.lookup": processPresetLookup,

		// Registry
		"reload": processReload,
	})
}

// ---------------------------------------------------------------------------
// Provider processes
// ---------------------------------------------------------------------------

// processProviderGet returns a provider by auto-increment ID.
// Args[0] int: provider ID
func processProviderGet(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	id := p.ArgsInt(0)
	provider, err := GetProvider(id)
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return provider
}

// processProviderCreate creates a new provider.
// Args[0] map: provider data
func processProviderCreate(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	var prov Provider
	if err := unmarshalArg(p.Args[0], &prov); err != nil {
		exception.New("invalid provider data: "+err.Error(), 400).Throw()
	}
	id, err := CreateProvider(&prov)
	if err != nil {
		exception.New(err.Error(), 400).Throw()
	}
	prov.ID = id
	return prov
}

// processProviderUpdate updates an existing provider.
// Args[0] int: provider ID, Args[1] map: provider data
func processProviderUpdate(p *process.Process) interface{} {
	p.ValidateArgNums(2)
	id := p.ArgsInt(0)
	var prov Provider
	if err := unmarshalArg(p.Args[1], &prov); err != nil {
		exception.New("invalid provider data: "+err.Error(), 400).Throw()
	}
	if err := UpdateProvider(id, &prov); err != nil {
		exception.New(err.Error(), 400).Throw()
	}
	return nil
}

// processProviderDelete deletes a provider by ID.
// Args[0] int: provider ID
func processProviderDelete(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	id := p.ArgsInt(0)
	if err := DeleteProvider(id); err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return nil
}

// processProviderList lists providers with optional scope filtering.
// Args[0] string: scope (optional), Args[1] string: teamID, Args[2] string: userID
func processProviderList(p *process.Process) interface{} {
	scope := Scope(p.ArgsString(0, ""))
	teamID := p.ArgsString(1, "")
	userID := p.ArgsString(2, "")
	providers, err := ListProviders(scope, teamID, userID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return providers
}

// processProviderFind finds a provider by ProviderID key (system scope).
// Args[0] string: providerID
func processProviderFind(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	if Global == nil {
		exception.New("llm registry not initialized", 500).Throw()
	}
	prov, ok := Global.FindProviderByKey(ProviderID(p.ArgsString(0)))
	if !ok {
		exception.New("provider not found", 404).Throw()
	}
	return prov
}

// ---------------------------------------------------------------------------
// Model processes
// ---------------------------------------------------------------------------

// processModelGet returns a model by auto-increment ID.
// Args[0] int: model ID
func processModelGet(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	id := p.ArgsInt(0)
	model, err := GetModel(id)
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return model
}

// processModelCreate creates a new model under a provider.
// Args[0] map: model data (must include provider_ref)
func processModelCreate(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	var m Model
	if err := unmarshalArg(p.Args[0], &m); err != nil {
		exception.New("invalid model data: "+err.Error(), 400).Throw()
	}
	if m.ProviderRef <= 0 {
		exception.New("provider_ref is required and must be a positive integer", 400).Throw()
	}
	if _, err := GetProvider(m.ProviderRef); err != nil {
		exception.New("provider_ref "+fmt.Sprint(m.ProviderRef)+" does not exist", 404).Throw()
	}
	id, err := CreateModel(&m)
	if err != nil {
		exception.New(err.Error(), 400).Throw()
	}
	m.ID = id
	return m
}

// processModelUpdate updates an existing model.
// Args[0] int: model ID, Args[1] map: model data
func processModelUpdate(p *process.Process) interface{} {
	p.ValidateArgNums(2)
	id := p.ArgsInt(0)
	var m Model
	if err := unmarshalArg(p.Args[1], &m); err != nil {
		exception.New("invalid model data: "+err.Error(), 400).Throw()
	}
	if err := UpdateModel(id, &m); err != nil {
		exception.New(err.Error(), 400).Throw()
	}
	return nil
}

// processModelDelete deletes a model by ID.
// Args[0] int: model ID
func processModelDelete(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	id := p.ArgsInt(0)
	if err := DeleteModel(id); err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return nil
}

// processModelList lists models under a provider.
// Args[0] int: provider ref ID
func processModelList(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	providerRef := p.ArgsInt(0)
	models, err := ListModels(providerRef)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return models
}

// processModelFind finds a model by provider ref and model_id string.
// Args[0] int: provider ref, Args[1] string: model_id
func processModelFind(p *process.Process) interface{} {
	p.ValidateArgNums(2)
	providerRef := p.ArgsInt(0)
	modelID := p.ArgsString(1)
	m, err := FindModelByModelID(providerRef, modelID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	if m == nil {
		exception.New("model not found", 404).Throw()
	}
	return m
}

// ---------------------------------------------------------------------------
// Role processes
// ---------------------------------------------------------------------------

// processRoleResolve resolves a role to Provider+Model via scope cascade.
// Args[0] string: role, Args[1] string: teamID, Args[2] string: userID
func processRoleResolve(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	role := p.ArgsString(0)
	teamID := p.ArgsString(1, "")
	userID := p.ArgsString(2, "")
	if Global == nil {
		exception.New("llm registry not initialized", 500).Throw()
	}
	prov, model, err := Global.ResolveRoleModel(role, teamID, userID)
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return map[string]interface{}{
		"provider": prov,
		"model":    model,
	}
}

// processRoleGet returns role bindings for a given scope.
// Args[0] string: scope, Args[1] string: teamID, Args[2] string: userID
func processRoleGet(p *process.Process) interface{} {
	scope := Scope(p.ArgsString(0, "system"))
	teamID := p.ArgsString(1, "")
	userID := p.ArgsString(2, "")
	bindings, err := GetRoleBindings(scope, teamID, userID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return bindings
}

// processRoleSet creates or updates a role binding.
// Args[0] map: RoleBinding data
func processRoleSet(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	var rb RoleBinding
	if err := unmarshalArg(p.Args[0], &rb); err != nil {
		exception.New("invalid role binding data: "+err.Error(), 400).Throw()
	}
	id, err := SetRoleBinding(&rb)
	if err != nil {
		exception.New(err.Error(), 400).Throw()
	}
	return id
}

// processRoleDelete deletes a role binding by ID.
// Args[0] int: binding ID
func processRoleDelete(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	id := p.ArgsInt(0)
	if err := DeleteRoleBinding(id); err != nil {
		exception.New(err.Error(), 404).Throw()
	}
	return nil
}

// ---------------------------------------------------------------------------
// Preset processes
// ---------------------------------------------------------------------------

// processPresetList returns all preset providers, or models for a specific provider.
// Args[0] string (optional): providerID — when provided, returns models for that provider;
// when omitted or empty, returns all preset provider entries.
func processPresetList(p *process.Process) interface{} {
	providerID := p.ArgsString(0, "")
	if providerID != "" {
		return ListPresets(providerID)
	}
	return ListAllPresets()
}

// processPresetLookup looks up a preset model.
// Args[0] string: providerID, Args[1] string: modelID
func processPresetLookup(p *process.Process) interface{} {
	p.ValidateArgNums(2)
	providerID := p.ArgsString(0)
	modelID := p.ArgsString(1)
	pm := LookupPreset(providerID, modelID)
	if pm == nil {
		exception.New("preset not found", 404).Throw()
	}
	return pm
}

// ---------------------------------------------------------------------------
// Registry
// ---------------------------------------------------------------------------

// processReload reloads the global registry from database.
func processReload(p *process.Process) interface{} {
	if Global == nil {
		if err := Init(); err != nil {
			exception.New(err.Error(), 500).Throw()
		}
		return nil
	}
	if err := Global.Reload(); err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return nil
}

// unmarshalArg converts a process argument (map or JSON) to a typed struct.
func unmarshalArg(v interface{}, target interface{}) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}
