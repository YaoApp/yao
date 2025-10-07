package user_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi/user"
)

// TestConfigTypes tests that our new configuration types are properly defined
func TestConfigTypes(t *testing.T) {
	// Test TeamConfig type
	teamConfig := &user.TeamConfig{
		Roles: []*user.TeamRole{
			{
				RoleID:      "team_owner",
				Label:       "Owner",
				Description: "Full access to team settings",
			},
		},
		Invite: &user.InviteConfig{
			Channel: "default",
			Expiry:  "1d",
			Templates: map[string]string{
				"mail": "en.invite_message",
				"sms":  "en.invite_message",
			},
		},
	}

	assert.NotNil(t, teamConfig, "TeamConfig should not be nil")
	assert.Len(t, teamConfig.Roles, 1, "Should have 1 role")
	assert.Equal(t, "team_owner", teamConfig.Roles[0].RoleID, "Role ID should match")
	assert.Equal(t, "Owner", teamConfig.Roles[0].Label, "Role label should match")
	assert.NotNil(t, teamConfig.Invite, "Invite config should not be nil")
	assert.Equal(t, "default", teamConfig.Invite.Channel, "Channel should match")
	assert.Equal(t, "1d", teamConfig.Invite.Expiry, "Expiry should match")
	assert.Len(t, teamConfig.Invite.Templates, 2, "Should have 2 templates")

	// Test TeamRole type
	role := &user.TeamRole{
		RoleID:      "team_admin",
		Label:       "Admin",
		Description: "Manage team members",
	}

	assert.Equal(t, "team_admin", role.RoleID, "Role ID should match")
	assert.Equal(t, "Admin", role.Label, "Role label should match")
	assert.Equal(t, "Manage team members", role.Description, "Description should match")

	// Test InviteConfig type
	inviteConfig := &user.InviteConfig{
		Channel: "email",
		Expiry:  "24h",
		Templates: map[string]string{
			"mail": "zh-cn.invite_message",
		},
	}

	assert.Equal(t, "email", inviteConfig.Channel, "Channel should match")
	assert.Equal(t, "24h", inviteConfig.Expiry, "Expiry should match")
	assert.Len(t, inviteConfig.Templates, 1, "Should have 1 template")
	assert.Equal(t, "zh-cn.invite_message", inviteConfig.Templates["mail"], "Template should match")
}

// TestConfigTypeCompatibility tests that our types are compatible with JSON marshaling
func TestConfigTypeCompatibility(t *testing.T) {
	// Test TeamConfig JSON marshaling
	teamConfig := &user.TeamConfig{
		Roles: []*user.TeamRole{
			{
				RoleID:      "test_role",
				Label:       "Test Role",
				Description: "A test role",
			},
		},
		Invite: &user.InviteConfig{
			Channel: "test",
			Expiry:  "1h",
			Templates: map[string]string{
				"test": "test_template",
			},
		},
	}

	// Test that the struct can be marshaled to JSON using standard library
	jsonData, err := json.Marshal(teamConfig)
	assert.NoError(t, err, "Should marshal to JSON without error")
	assert.NotEmpty(t, jsonData, "JSON data should not be empty")

	// Test that the struct can be unmarshaled from JSON
	var unmarshaledConfig user.TeamConfig
	err = json.Unmarshal(jsonData, &unmarshaledConfig)
	assert.NoError(t, err, "Should unmarshal from JSON without error")
	assert.Equal(t, teamConfig.Roles[0].RoleID, unmarshaledConfig.Roles[0].RoleID, "Role ID should match after unmarshaling")
	assert.Equal(t, teamConfig.Invite.Channel, unmarshaledConfig.Invite.Channel, "Channel should match after unmarshaling")
}
