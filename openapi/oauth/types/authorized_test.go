package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateAccessScope(t *testing.T) {
	info := &AuthorizedInfo{
		UserID:   "user123",
		TeamID:   "team456",
		TenantID: "tenant789",
	}

	scope := info.CreateAccessScope()
	assert.NotNil(t, scope)
	assert.Equal(t, "user123", scope.CreatedBy)
	assert.Empty(t, scope.UpdatedBy)
	assert.Equal(t, "team456", scope.TeamID)
	assert.Equal(t, "tenant789", scope.TenantID)
}

func TestCreateAccessScopeNil(t *testing.T) {
	var info *AuthorizedInfo
	scope := info.CreateAccessScope()
	assert.Nil(t, scope)
}

func TestCreateAccessScopePartial(t *testing.T) {
	info := &AuthorizedInfo{
		UserID: "user123",
	}

	scope := info.CreateAccessScope()
	assert.NotNil(t, scope)
	assert.Equal(t, "user123", scope.CreatedBy)
	assert.Empty(t, scope.UpdatedBy)
	assert.Empty(t, scope.TeamID)
	assert.Empty(t, scope.TenantID)
}

func TestUpdateAccessScope(t *testing.T) {
	info := &AuthorizedInfo{
		UserID:   "user456",
		TeamID:   "team789",
		TenantID: "tenant000",
	}

	scope := info.UpdateAccessScope()
	assert.NotNil(t, scope)
	assert.Empty(t, scope.CreatedBy)
	assert.Equal(t, "user456", scope.UpdatedBy)
	assert.Equal(t, "team789", scope.TeamID)
	assert.Equal(t, "tenant000", scope.TenantID)
}

func TestUpdateAccessScopeNil(t *testing.T) {
	var info *AuthorizedInfo
	scope := info.UpdateAccessScope()
	assert.Nil(t, scope)
}

func TestAccessScope(t *testing.T) {
	info := &AuthorizedInfo{
		UserID:   "user123",
		TeamID:   "team456",
		TenantID: "tenant789",
	}

	scope := info.AccessScope()
	assert.NotNil(t, scope)
	assert.Equal(t, "user123", scope.CreatedBy)
	assert.Equal(t, "user123", scope.UpdatedBy)
	assert.Equal(t, "team456", scope.TeamID)
	assert.Equal(t, "tenant789", scope.TenantID)
}

func TestAccessScopeNil(t *testing.T) {
	var info *AuthorizedInfo
	scope := info.AccessScope()
	assert.Nil(t, scope)
}

func TestAccessScopeEmptyFields(t *testing.T) {
	info := &AuthorizedInfo{}
	scope := info.AccessScope()
	assert.NotNil(t, scope)
	assert.Empty(t, scope.CreatedBy)
	assert.Empty(t, scope.UpdatedBy)
	assert.Empty(t, scope.TeamID)
	assert.Empty(t, scope.TenantID)
}

func TestAccessScopeOnlyUser(t *testing.T) {
	info := &AuthorizedInfo{
		UserID: "user999",
	}

	scope := info.AccessScope()
	assert.NotNil(t, scope)
	assert.Equal(t, "user999", scope.CreatedBy)
	assert.Equal(t, "user999", scope.UpdatedBy)
	assert.Empty(t, scope.TeamID)
	assert.Empty(t, scope.TenantID)
}

func TestAccessScopeOnlyTeam(t *testing.T) {
	info := &AuthorizedInfo{
		TeamID:   "team111",
		TenantID: "tenant222",
	}

	scope := info.AccessScope()
	assert.NotNil(t, scope)
	assert.Empty(t, scope.CreatedBy)
	assert.Empty(t, scope.UpdatedBy)
	assert.Equal(t, "team111", scope.TeamID)
	assert.Equal(t, "tenant222", scope.TenantID)
}

func TestAccessScopeIntegration(t *testing.T) {
	info := &AuthorizedInfo{
		Subject:  "subject123",
		ClientID: "client456",
		UserID:   "user789",
		TeamID:   "team000",
		TenantID: "tenant111",
	}

	// Test CreateAccessScope
	createScope := info.CreateAccessScope()
	assert.Equal(t, "user789", createScope.CreatedBy)
	assert.Empty(t, createScope.UpdatedBy)
	assert.Equal(t, "team000", createScope.TeamID)
	assert.Equal(t, "tenant111", createScope.TenantID)

	// Test UpdateAccessScope
	updateScope := info.UpdateAccessScope()
	assert.Empty(t, updateScope.CreatedBy)
	assert.Equal(t, "user789", updateScope.UpdatedBy)
	assert.Equal(t, "team000", updateScope.TeamID)
	assert.Equal(t, "tenant111", updateScope.TenantID)

	// Test AccessScope
	queryScope := info.AccessScope()
	assert.Equal(t, "user789", queryScope.CreatedBy)
	assert.Equal(t, "user789", queryScope.UpdatedBy)
	assert.Equal(t, "team000", queryScope.TeamID)
	assert.Equal(t, "tenant111", queryScope.TenantID)
}

func TestWithCreateScope(t *testing.T) {
	info := &AuthorizedInfo{
		UserID:   "user123",
		TeamID:   "team456",
		TenantID: "tenant789",
	}

	data := map[string]interface{}{
		"name":        "Test Team",
		"description": "A test team",
	}

	result := info.WithCreateScope(data)
	assert.Equal(t, "Test Team", result["name"])
	assert.Equal(t, "A test team", result["description"])
	assert.Equal(t, "user123", result["__yao_created_by"])
	assert.Equal(t, "team456", result["__yao_team_id"])
	assert.Equal(t, "tenant789", result["__yao_tenant_id"])
	assert.Nil(t, result["__yao_updated_by"])
}

func TestWithCreateScopeNil(t *testing.T) {
	var info *AuthorizedInfo
	data := map[string]interface{}{
		"name": "Test",
	}

	result := info.WithCreateScope(data)
	assert.Equal(t, "Test", result["name"])
	assert.Nil(t, result["__yao_created_by"])
	assert.Nil(t, result["__yao_team_id"])
}

func TestWithCreateScopePartial(t *testing.T) {
	info := &AuthorizedInfo{
		UserID: "user123",
	}

	data := map[string]interface{}{
		"name": "Test",
	}

	result := info.WithCreateScope(data)
	assert.Equal(t, "Test", result["name"])
	assert.Equal(t, "user123", result["__yao_created_by"])
	assert.Nil(t, result["__yao_team_id"])
	assert.Nil(t, result["__yao_tenant_id"])
}

func TestWithUpdateScope(t *testing.T) {
	info := &AuthorizedInfo{
		UserID:   "user456",
		TeamID:   "team789",
		TenantID: "tenant000",
	}

	data := map[string]interface{}{
		"name":        "Updated Team",
		"description": "An updated team",
	}

	result := info.WithUpdateScope(data)
	assert.Equal(t, "Updated Team", result["name"])
	assert.Equal(t, "An updated team", result["description"])
	assert.Equal(t, "user456", result["__yao_updated_by"])
	assert.Equal(t, "team789", result["__yao_team_id"])
	assert.Equal(t, "tenant000", result["__yao_tenant_id"])
	assert.Nil(t, result["__yao_created_by"])
}

func TestWithUpdateScopeNil(t *testing.T) {
	var info *AuthorizedInfo
	data := map[string]interface{}{
		"name": "Test",
	}

	result := info.WithUpdateScope(data)
	assert.Equal(t, "Test", result["name"])
	assert.Nil(t, result["__yao_updated_by"])
	assert.Nil(t, result["__yao_team_id"])
}

func TestWithScopesIntegration(t *testing.T) {
	info := &AuthorizedInfo{
		UserID:   "user999",
		TeamID:   "team888",
		TenantID: "tenant777",
	}

	// Create scenario
	createData := map[string]interface{}{
		"name": "New Record",
	}
	createResult := info.WithCreateScope(createData)
	assert.Equal(t, "New Record", createResult["name"])
	assert.Equal(t, "user999", createResult["__yao_created_by"])
	assert.Equal(t, "team888", createResult["__yao_team_id"])
	assert.Equal(t, "tenant777", createResult["__yao_tenant_id"])
	assert.Nil(t, createResult["__yao_updated_by"])

	// Update scenario
	updateData := map[string]interface{}{
		"name":   "Updated Record",
		"status": "active",
	}
	updateResult := info.WithUpdateScope(updateData)
	assert.Equal(t, "Updated Record", updateResult["name"])
	assert.Equal(t, "active", updateResult["status"])
	assert.Equal(t, "user999", updateResult["__yao_updated_by"])
	assert.Equal(t, "team888", updateResult["__yao_team_id"])
	assert.Equal(t, "tenant777", updateResult["__yao_tenant_id"])
	assert.Nil(t, updateResult["__yao_created_by"])
}

func TestCopyCreateScope(t *testing.T) {
	source := map[string]interface{}{
		"id":               1,
		"name":             "Original Record",
		"__yao_created_by": "user123",
		"__yao_team_id":    "team456",
		"__yao_tenant_id":  "tenant789",
		"__yao_updated_by": "user999", // Should not be copied
	}

	dest := map[string]interface{}{
		"name":        "New Record",
		"description": "A new record",
	}

	result := CopyCreateScope(source, dest)
	assert.Equal(t, "New Record", result["name"])
	assert.Equal(t, "A new record", result["description"])
	assert.Equal(t, "user123", result["__yao_created_by"])
	assert.Equal(t, "team456", result["__yao_team_id"])
	assert.Equal(t, "tenant789", result["__yao_tenant_id"])
	assert.Nil(t, result["__yao_updated_by"]) // Should not be copied
}

func TestCopyCreateScopePartial(t *testing.T) {
	source := map[string]interface{}{
		"name":             "Original",
		"__yao_created_by": "user123",
	}

	dest := map[string]interface{}{
		"name": "New",
	}

	result := CopyCreateScope(source, dest)
	assert.Equal(t, "New", result["name"])
	assert.Equal(t, "user123", result["__yao_created_by"])
	assert.Nil(t, result["__yao_team_id"])
	assert.Nil(t, result["__yao_tenant_id"])
}

func TestCopyCreateScopeEmpty(t *testing.T) {
	source := map[string]interface{}{
		"name": "Original",
	}

	dest := map[string]interface{}{
		"name": "New",
	}

	result := CopyCreateScope(source, dest)
	assert.Equal(t, "New", result["name"])
	assert.Nil(t, result["__yao_created_by"])
	assert.Nil(t, result["__yao_team_id"])
	assert.Nil(t, result["__yao_tenant_id"])
}

func TestCopyUpdateScope(t *testing.T) {
	source := map[string]interface{}{
		"id":               1,
		"name":             "Original Record",
		"__yao_updated_by": "user456",
		"__yao_team_id":    "team789",
		"__yao_tenant_id":  "tenant000",
		"__yao_created_by": "user123", // Should not be copied
	}

	dest := map[string]interface{}{
		"name":   "Updated Record",
		"status": "active",
	}

	result := CopyUpdateScope(source, dest)
	assert.Equal(t, "Updated Record", result["name"])
	assert.Equal(t, "active", result["status"])
	assert.Equal(t, "user456", result["__yao_updated_by"])
	assert.Equal(t, "team789", result["__yao_team_id"])
	assert.Equal(t, "tenant000", result["__yao_tenant_id"])
	assert.Nil(t, result["__yao_created_by"]) // Should not be copied
}

func TestCopyUpdateScopePartial(t *testing.T) {
	source := map[string]interface{}{
		"name":             "Original",
		"__yao_updated_by": "user456",
		"__yao_team_id":    "team789",
	}

	dest := map[string]interface{}{
		"name": "Updated",
	}

	result := CopyUpdateScope(source, dest)
	assert.Equal(t, "Updated", result["name"])
	assert.Equal(t, "user456", result["__yao_updated_by"])
	assert.Equal(t, "team789", result["__yao_team_id"])
	assert.Nil(t, result["__yao_tenant_id"])
}

func TestCopyUpdateScopeEmpty(t *testing.T) {
	source := map[string]interface{}{
		"name": "Original",
	}

	dest := map[string]interface{}{
		"name": "Updated",
	}

	result := CopyUpdateScope(source, dest)
	assert.Equal(t, "Updated", result["name"])
	assert.Nil(t, result["__yao_updated_by"])
	assert.Nil(t, result["__yao_team_id"])
	assert.Nil(t, result["__yao_tenant_id"])
}

func TestCopyCreateScopeRealWorld(t *testing.T) {
	// Simulate real-world scenario from team.go
	authInfo := &AuthorizedInfo{
		UserID:   "063254760529",
		TeamID:   "242182710786",
		TenantID: "tenant789",
	}

	// This mimics: teamData := authInfo.WithCreateScope(...)
	teamData := authInfo.WithCreateScope(map[string]interface{}{
		"name":        "A",
		"description": "AAAA",
	})

	// Verify teamData has the scope fields
	assert.Equal(t, "063254760529", teamData["__yao_created_by"])
	assert.Equal(t, "242182710786", teamData["__yao_team_id"])
	assert.Equal(t, "tenant789", teamData["__yao_tenant_id"])

	// Now copy from teamData to ownerMemberData
	ownerMemberData := CopyCreateScope(teamData, map[string]interface{}{
		"team_id":     "242182710786",
		"user_id":     "063254760529",
		"member_type": "user",
		"role_id":     "owner:free",
		"status":      "active",
	})

	// Verify the scope fields were copied
	assert.Equal(t, "063254760529", ownerMemberData["__yao_created_by"], "created_by should be copied from source")
	assert.Equal(t, "242182710786", ownerMemberData["__yao_team_id"], "team_id should be copied from source")
	assert.Equal(t, "tenant789", ownerMemberData["__yao_tenant_id"], "tenant_id should be copied from source")

	// Verify dest data is preserved
	assert.Equal(t, "242182710786", ownerMemberData["team_id"])
	assert.Equal(t, "063254760529", ownerMemberData["user_id"])
	assert.Equal(t, "user", ownerMemberData["member_type"])
	assert.Equal(t, "owner:free", ownerMemberData["role_id"])
	assert.Equal(t, "active", ownerMemberData["status"])
}

func TestCopyScopesIntegration(t *testing.T) {
	// Simulate a create operation
	originalRecord := map[string]interface{}{
		"id":               1,
		"name":             "Original Team",
		"__yao_created_by": "user123",
		"__yao_team_id":    "team456",
		"__yao_tenant_id":  "tenant789",
	}

	// Copy to create a child record
	childData := map[string]interface{}{
		"name":      "Child Record",
		"parent_id": 1,
	}
	childResult := CopyCreateScope(originalRecord, childData)
	assert.Equal(t, "Child Record", childResult["name"])
	assert.Equal(t, 1, childResult["parent_id"])
	assert.Equal(t, "user123", childResult["__yao_created_by"])
	assert.Equal(t, "team456", childResult["__yao_team_id"])
	assert.Equal(t, "tenant789", childResult["__yao_tenant_id"])

	// Simulate an update operation
	updateRecord := map[string]interface{}{
		"id":               1,
		"name":             "Updated Team",
		"__yao_created_by": "user123",
		"__yao_updated_by": "user999",
		"__yao_team_id":    "team456",
		"__yao_tenant_id":  "tenant789",
	}

	updateData := map[string]interface{}{
		"description": "Updated description",
	}
	updateResult := CopyUpdateScope(updateRecord, updateData)
	assert.Equal(t, "Updated description", updateResult["description"])
	assert.Equal(t, "user999", updateResult["__yao_updated_by"])
	assert.Equal(t, "team456", updateResult["__yao_team_id"])
	assert.Equal(t, "tenant789", updateResult["__yao_tenant_id"])
	assert.Nil(t, updateResult["__yao_created_by"]) // Should not be copied
}

func TestAuthorizedToMap(t *testing.T) {
	tests := []struct {
		name     string
		auth     *AuthorizedInfo
		expected map[string]interface{}
	}{
		{
			name: "Full AuthorizedInfo",
			auth: &AuthorizedInfo{
				Subject:    "user123",
				ClientID:   "client456",
				Scope:      "read write",
				SessionID:  "session789",
				UserID:     "user123",
				TeamID:     "team456",
				TenantID:   "tenant789",
				RememberMe: true,
				Constraints: DataConstraints{
					OwnerOnly:   true,
					CreatorOnly: false,
					EditorOnly:  false,
					TeamOnly:    true,
					Extra: map[string]interface{}{
						"department": "engineering",
					},
				},
			},
			expected: map[string]interface{}{
				"sub":         "user123",
				"client_id":   "client456",
				"scope":       "read write",
				"session_id":  "session789",
				"user_id":     "user123",
				"team_id":     "team456",
				"tenant_id":   "tenant789",
				"remember_me": true,
				"constraints": map[string]interface{}{
					"owner_only": true,
					"team_only":  true,
					"extra": map[string]interface{}{
						"department": "engineering",
					},
				},
			},
		},
		{
			name: "Partial AuthorizedInfo",
			auth: &AuthorizedInfo{
				UserID: "user123",
				TeamID: "team456",
			},
			expected: map[string]interface{}{
				"user_id": "user123",
				"team_id": "team456",
			},
		},
		{
			name: "AuthorizedInfo with only constraints",
			auth: &AuthorizedInfo{
				UserID: "user123",
				Constraints: DataConstraints{
					TeamOnly: true,
					Extra: map[string]interface{}{
						"region": "us-west",
					},
				},
			},
			expected: map[string]interface{}{
				"user_id": "user123",
				"constraints": map[string]interface{}{
					"team_only": true,
					"extra": map[string]interface{}{
						"region": "us-west",
					},
				},
			},
		},
		{
			name:     "Nil AuthorizedInfo",
			auth:     nil,
			expected: nil,
		},
		{
			name:     "Empty AuthorizedInfo",
			auth:     &AuthorizedInfo{},
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.auth.AuthorizedToMap()

			if tt.expected == nil {
				assert.Nil(t, result)
				return
			}

			assert.NotNil(t, result)

			// Check all expected keys
			for key, expectedValue := range tt.expected {
				actualValue, ok := result[key]
				if !ok {
					t.Errorf("Key %s not found in result", key)
					continue
				}

				// Special handling for nested maps (constraints)
				if key == "constraints" {
					expectedConstraints, _ := expectedValue.(map[string]interface{})
					actualConstraints, ok := actualValue.(map[string]interface{})
					assert.True(t, ok, "constraints should be map[string]interface{}")

					for cKey, cExpectedValue := range expectedConstraints {
						cActualValue, ok := actualConstraints[cKey]
						assert.True(t, ok, "Constraint key %s should exist", cKey)

						// Special handling for nested extra map
						if cKey == "extra" {
							expectedExtra, _ := cExpectedValue.(map[string]interface{})
							actualExtra, ok := cActualValue.(map[string]interface{})
							assert.True(t, ok, "extra should be map[string]interface{}")

							for eKey, eExpectedValue := range expectedExtra {
								eActualValue, ok := actualExtra[eKey]
								assert.True(t, ok, "Extra key %s should exist", eKey)
								assert.Equal(t, eExpectedValue, eActualValue)
							}
						} else {
							assert.Equal(t, cExpectedValue, cActualValue)
						}
					}
				} else {
					assert.Equal(t, expectedValue, actualValue)
				}
			}

			// Check no unexpected keys (except for empty maps)
			if len(tt.expected) > 0 {
				for key := range result {
					_, ok := tt.expected[key]
					assert.True(t, ok, "Unexpected key %s in result", key)
				}
			}
		})
	}
}
