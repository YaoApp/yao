package llm

import (
	"fmt"

	goumodel "github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
)

const roleBindingModelID = "__yao.llm.role_binding"

func roleModel() *goumodel.Model {
	m, err := goumodel.Get(roleBindingModelID)
	if err != nil {
		panic("llm: role_binding model not loaded: " + err.Error())
	}
	return m
}

// SetRoleBinding creates or updates a role binding.
// When rb.ID is zero, looks up an existing binding by the unique key
// (role, scope, team_id, user_id) to avoid UNIQUE constraint violations.
func SetRoleBinding(rb *RoleBinding) (int, error) {
	mod := roleModel()

	if rb.ID == 0 {
		existing, err := findRoleBinding(mod, rb.Role, rb.Scope, rb.TeamID, rb.UserID)
		if err != nil {
			return 0, err
		}
		if existing != nil {
			rb.ID = existing.ID
		}
	}

	row := roleBindingToRow(rb)
	id, err := mod.Save(row)
	if err != nil {
		return 0, fmt.Errorf("llm: set role binding: %w", err)
	}
	intID := toIntFromAny(id)
	rb.ID = intID
	return intID, nil
}

// GetRoleBindings returns all role bindings for the given scope.
func GetRoleBindings(scope Scope, teamID, userID string) ([]RoleBinding, error) {
	param := goumodel.QueryParam{
		Wheres: buildScopeWheres(scope, teamID, userID),
		Orders: []goumodel.QueryOrder{{Column: "role", Option: "asc"}},
	}
	rows, err := roleModel().Get(param)
	if err != nil {
		return nil, fmt.Errorf("llm: get role bindings: %w", err)
	}
	out := make([]RoleBinding, len(rows))
	for i, row := range rows {
		out[i] = *rowToRoleBinding(row)
	}
	return out, nil
}

// DeleteRoleBinding removes a role binding by ID.
func DeleteRoleBinding(id int) error {
	return roleModel().Delete(id)
}

// ResolveRole resolves a role binding using the three-level scope cascade:
// User > Team > System. Returns nil if no binding exists at any level.
func ResolveRole(role, teamID, userID string) (*RoleBinding, error) {
	mod := roleModel()

	// User level
	if userID != "" {
		rb, err := findRoleBinding(mod, role, ScopeUser, "", userID)
		if err != nil {
			return nil, err
		}
		if rb != nil {
			return rb, nil
		}
	}

	// Team level
	if teamID != "" {
		rb, err := findRoleBinding(mod, role, ScopeTeam, teamID, "")
		if err != nil {
			return nil, err
		}
		if rb != nil {
			return rb, nil
		}
	}

	// System level
	return findRoleBinding(mod, role, ScopeSystem, "", "")
}

func findRoleBinding(mod *goumodel.Model, role string, scope Scope, teamID, userID string) (*RoleBinding, error) {
	wheres := []goumodel.QueryWhere{
		{Column: "role", Value: role},
		{Column: "scope", Value: string(scope)},
	}
	if scope == ScopeTeam {
		wheres = append(wheres, goumodel.QueryWhere{Column: "team_id", Value: teamID})
	}
	if scope == ScopeUser {
		wheres = append(wheres, goumodel.QueryWhere{Column: "user_id", Value: userID})
	}

	param := goumodel.QueryParam{
		Wheres: wheres,
		Limit:  1,
	}
	rows, err := mod.Get(param)
	if err != nil {
		return nil, fmt.Errorf("llm: find role binding: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rowToRoleBinding(rows[0]), nil
}

// ---------------------------------------------------------------------------
// Row conversion helpers
// ---------------------------------------------------------------------------

func roleBindingToRow(rb *RoleBinding) maps.MapStrAny {
	row := maps.MapStrAny{
		"role":         rb.Role,
		"provider_ref": rb.ProviderRef,
		"model_id":     rb.ModelID,
		"scope":        string(rb.Scope),
		"team_id":      rb.TeamID,
		"user_id":      rb.UserID,
	}
	if rb.ID > 0 {
		row["id"] = rb.ID
	}
	return row
}

func rowToRoleBinding(row maps.MapStr) *RoleBinding {
	return &RoleBinding{
		ID:          getInt(row, "id"),
		Role:        getString(row, "role"),
		ProviderRef: getInt(row, "provider_ref"),
		ModelID:     getString(row, "model_id"),
		Scope:       Scope(getString(row, "scope")),
		TeamID:      getString(row, "team_id"),
		UserID:      getString(row, "user_id"),
		UpdatedAt:   getTime(row, "updated_at"),
	}
}
