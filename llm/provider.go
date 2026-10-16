package llm

import (
	"fmt"

	goumodel "github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
)

const providerModelID = "__yao.llm.provider"

func providerModel() *goumodel.Model {
	m, err := goumodel.Get(providerModelID)
	if err != nil {
		panic("llm: provider model not loaded: " + err.Error())
	}
	return m
}

// CreateProvider inserts a new Provider and returns its auto-increment ID.
// The global Registry cache is refreshed on success.
func CreateProvider(p *Provider) (int, error) {
	row := providerToRow(p)
	id, err := providerModel().Create(row)
	if err != nil {
		return 0, fmt.Errorf("llm: create provider: %w", err)
	}
	p.ID = id
	refreshRegistry()
	return id, nil
}

// GetProvider retrieves a Provider by its auto-increment ID.
func GetProvider(id int) (*Provider, error) {
	row, err := providerModel().Find(id, goumodel.QueryParam{})
	if err != nil {
		return nil, fmt.Errorf("llm: get provider %d: %w", id, err)
	}
	return rowToProvider(row), nil
}

// UpdateProvider updates an existing Provider by ID.
// The global Registry cache is refreshed on success.
func UpdateProvider(id int, p *Provider) error {
	row := providerToRow(p)
	row.Del("id")
	if err := providerModel().Update(id, row); err != nil {
		return err
	}
	refreshRegistry()
	return nil
}

// DeleteProvider removes a Provider by ID.
// The global Registry cache is refreshed on success.
func DeleteProvider(id int) error {
	if err := providerModel().Delete(id); err != nil {
		return err
	}
	refreshRegistry()
	return nil
}

// ListProviders returns all providers matching the given scope filter.
// Pass empty scope/teamID/userID to list all.
func ListProviders(scope Scope, teamID, userID string) ([]Provider, error) {
	param := goumodel.QueryParam{
		Wheres: buildScopeWheres(scope, teamID, userID),
		Orders: []goumodel.QueryOrder{{Column: "priority", Option: "asc"}, {Column: "id", Option: "asc"}},
	}
	rows, err := providerModel().Get(param)
	if err != nil {
		return nil, fmt.Errorf("llm: list providers: %w", err)
	}
	out := make([]Provider, len(rows))
	for i, row := range rows {
		out[i] = *rowToProvider(row)
	}
	return out, nil
}

// ListProvidersByScope returns providers visible at the given scope level.
// System providers are always included; team/user providers are added
// based on the provided teamID/userID.
func ListProvidersByScope(teamID, userID string) ([]Provider, error) {
	wheres := []goumodel.QueryWhere{
		{Column: "scope", Value: "system"},
	}
	if teamID != "" {
		wheres = append(wheres, goumodel.QueryWhere{
			Column: "scope", Value: "team", Method: "orwhere",
		})
		wheres = append(wheres, goumodel.QueryWhere{
			Column: "team_id", Value: teamID,
		})
	}
	if userID != "" {
		wheres = append(wheres, goumodel.QueryWhere{
			Column: "scope", Value: "user", Method: "orwhere",
		})
		wheres = append(wheres, goumodel.QueryWhere{
			Column: "user_id", Value: userID,
		})
	}
	param := goumodel.QueryParam{
		Wheres: wheres,
		Orders: []goumodel.QueryOrder{{Column: "priority", Option: "asc"}, {Column: "id", Option: "asc"}},
	}
	rows, err := providerModel().Get(param)
	if err != nil {
		return nil, fmt.Errorf("llm: list providers by scope: %w", err)
	}
	out := make([]Provider, len(rows))
	for i, row := range rows {
		out[i] = *rowToProvider(row)
	}
	return out, nil
}

// FindProviderByKey finds a provider by provider_id within a scope.
func FindProviderByKey(providerID string, scope Scope, teamID, userID string) (*Provider, error) {
	wheres := []goumodel.QueryWhere{
		{Column: "provider_id", Value: providerID},
	}
	wheres = append(wheres, buildScopeWheres(scope, teamID, userID)...)

	param := goumodel.QueryParam{
		Wheres: wheres,
		Limit:  1,
	}
	rows, err := providerModel().Get(param)
	if err != nil {
		return nil, fmt.Errorf("llm: find provider by key: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rowToProvider(rows[0]), nil
}

// ---------------------------------------------------------------------------
// Row conversion helpers
// ---------------------------------------------------------------------------

func providerToRow(p *Provider) maps.MapStrAny {
	row := maps.MapStrAny{
		"name":        p.Name,
		"provider_id": string(p.ProviderID),
		"api":         string(p.API),
		"base_url":    p.BaseURL,
		"api_key":     p.APIKey,
		"enabled":     p.Enabled,
		"status":      p.Status,
		"scope":       string(p.Scope),
		"team_id":     p.TeamID,
		"user_id":     p.UserID,
		"priority":    p.Priority,
	}
	if p.ID > 0 {
		row["id"] = p.ID
	}
	if p.Protocols != nil {
		row["protocols"] = p.Protocols
	}
	if p.Headers != nil {
		row["headers"] = p.Headers
	}
	if p.Compat != nil {
		row["compat"] = p.Compat
	}
	if p.PresetKey != "" {
		row["preset_key"] = p.PresetKey
	}
	if p.Metadata != nil {
		row["metadata"] = p.Metadata
	}
	return row
}

func rowToProvider(row maps.MapStr) *Provider {
	p := &Provider{
		ID:         getInt(row, "id"),
		Name:       getString(row, "name"),
		ProviderID: ProviderID(getString(row, "provider_id")),
		API:        API(getString(row, "api")),
		BaseURL:    getString(row, "base_url"),
		APIKey:     getString(row, "api_key"),
		Enabled:    getBool(row, "enabled"),
		Status:     getString(row, "status"),
		PresetKey:  getString(row, "preset_key"),
		Scope:      Scope(getString(row, "scope")),
		TeamID:     getString(row, "team_id"),
		UserID:     getString(row, "user_id"),
		Priority:   getInt(row, "priority"),
	}
	if v, ok := row["protocols"]; ok && v != nil {
		p.Protocols = toAPISlice(v)
	}
	if v, ok := row["headers"]; ok && v != nil {
		p.Headers = toJSONMap(v)
	}
	if v, ok := row["compat"]; ok && v != nil {
		p.Compat = toJSONMap(v)
	}
	if v, ok := row["metadata"]; ok && v != nil {
		p.Metadata = toJSONMap(v)
	}
	p.CreatedAt = getTime(row, "created_at")
	p.UpdatedAt = getTime(row, "updated_at")
	return p
}
