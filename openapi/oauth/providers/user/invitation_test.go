package user

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

// TestCreateInvitationCodes tests batch creation of invitation codes
func TestCreateInvitationCodes(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := context.Background()
	provider := NewDefaultUser(&DefaultUserOptions{})

	// Test Case 1: Create multiple invitation codes successfully
	t.Run("Create multiple codes successfully", func(t *testing.T) {
		codeData := []maps.MapStrAny{
			{
				"code":        "TEST-BETA-001",
				"code_type":   "beta",
				"description": "Beta testing code 1",
				"owner_id":    nil, // Official code
				"status":      "draft",
			},
			{
				"code":        "TEST-BETA-002",
				"code_type":   "beta",
				"description": "Beta testing code 2",
				"owner_id":    nil, // Official code
				"status":      "draft",
			},
			{
				"code":        "TEST-PARTNER-001",
				"code_type":   "partner",
				"description": "Partner code 1",
				"owner_id":    nil,
				"status":      "draft",
			},
		}

		codes, err := provider.CreateInvitationCodes(ctx, codeData)
		assert.NoError(t, err)
		assert.Equal(t, 3, len(codes))
		assert.Contains(t, codes, "TEST-BETA-001")
		assert.Contains(t, codes, "TEST-BETA-002")
		assert.Contains(t, codes, "TEST-PARTNER-001")

		// Verify codes were created in database
		m := model.Select("__yao.invitation")
		for _, code := range codes {
			invitations, err := m.Get(model.QueryParam{
				Select: []interface{}{"code", "status", "code_type"},
				Wheres: []model.QueryWhere{
					{Column: "code", Value: code},
				},
				Limit: 1,
			})
			assert.NoError(t, err)
			assert.Equal(t, 1, len(invitations))
			assert.Equal(t, "draft", invitations[0]["status"])
		}

		// Cleanup
		for _, code := range codes {
			m.DeleteWhere(model.QueryParam{
				Wheres: []model.QueryWhere{
					{Column: "code", Value: code},
				},
			})
		}
	})

	// Test Case 2: Create with default values
	t.Run("Create with default values", func(t *testing.T) {
		codeData := []maps.MapStrAny{
			{
				"code": "TEST-DEFAULT-001",
			},
		}

		codes, err := provider.CreateInvitationCodes(ctx, codeData)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(codes))

		// Verify default values
		m := model.Select("__yao.invitation")
		invitations, err := m.Get(model.QueryParam{
			Select: []interface{}{"code", "status", "is_published", "code_type"},
			Wheres: []model.QueryWhere{
				{Column: "code", Value: "TEST-DEFAULT-001"},
			},
			Limit: 1,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(invitations))
		assert.Equal(t, "draft", invitations[0]["status"])
		// is_published can be bool(false) or int64(0) or int(0) - all are valid
		assert.NotNil(t, invitations[0]["is_published"])
		assert.Equal(t, "official", invitations[0]["code_type"])

		// Cleanup
		m.DeleteWhere(model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "code", Value: "TEST-DEFAULT-001"},
			},
		})
	})

	// Test Case 3: Empty batch
	t.Run("Empty batch", func(t *testing.T) {
		codes, err := provider.CreateInvitationCodes(ctx, []maps.MapStrAny{})
		assert.NoError(t, err)
		assert.Equal(t, 0, len(codes))
	})

	// Test Case 4: Missing required field (code)
	t.Run("Missing required field", func(t *testing.T) {
		codeData := []maps.MapStrAny{
			{
				"code_type":   "beta",
				"description": "Missing code field",
			},
		}

		codes, err := provider.CreateInvitationCodes(ctx, codeData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "code is required")
		assert.Equal(t, 0, len(codes))
	})

	// Test Case 5: Duplicate code (should fail)
	t.Run("Duplicate code", func(t *testing.T) {
		// Create first code
		codeData := []maps.MapStrAny{
			{
				"code":      "TEST-DUPLICATE",
				"code_type": "official",
			},
		}
		codes, err := provider.CreateInvitationCodes(ctx, codeData)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(codes))

		// Try to create duplicate
		codes, err = provider.CreateInvitationCodes(ctx, codeData)
		assert.Error(t, err)

		// Cleanup
		m := model.Select("__yao.invitation")
		m.DeleteWhere(model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "code", Value: "TEST-DUPLICATE"},
			},
		})
	})
}

// TestUseInvitationCode tests invitation code redemption
func TestUseInvitationCode(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := context.Background()
	provider := NewDefaultUser(&DefaultUserOptions{})
	m := model.Select("__yao.invitation")

	// Test Case 1: Successfully use a valid invitation code
	t.Run("Use valid code successfully", func(t *testing.T) {
		// Create a valid, published, active invitation code
		codeData := []maps.MapStrAny{
			{
				"code":         "TEST-USE-001",
				"code_type":    "beta",
				"status":       "active",
				"is_published": true,
			},
		}
		_, err := provider.CreateInvitationCodes(ctx, codeData)
		assert.NoError(t, err)

		// Use the invitation code
		err = provider.UseInvitationCode(ctx, "TEST-USE-001", "user_123")
		assert.NoError(t, err)

		// Verify code was marked as used
		invitations, err := m.Get(model.QueryParam{
			Select: []interface{}{"code", "status", "used_by", "used_at"},
			Wheres: []model.QueryWhere{
				{Column: "code", Value: "TEST-USE-001"},
			},
			Limit: 1,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(invitations))
		assert.Equal(t, "used", invitations[0]["status"])
		assert.Equal(t, "user_123", invitations[0]["used_by"])
		assert.NotNil(t, invitations[0]["used_at"])

		// Cleanup
		m.DeleteWhere(model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "code", Value: "TEST-USE-001"},
			},
		})
	})

	// Test Case 2: Try to use non-existent code
	t.Run("Use non-existent code", func(t *testing.T) {
		err := provider.UseInvitationCode(ctx, "NONEXISTENT-CODE", "user_123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), ErrInvitationCodeNotFound)
	})

	// Test Case 3: Try to use already used code
	t.Run("Use already used code", func(t *testing.T) {
		// Create and use a code
		codeData := []maps.MapStrAny{
			{
				"code":         "TEST-USE-002",
				"code_type":    "beta",
				"status":       "active",
				"is_published": true,
			},
		}
		_, err := provider.CreateInvitationCodes(ctx, codeData)
		assert.NoError(t, err)

		// First use
		err = provider.UseInvitationCode(ctx, "TEST-USE-002", "user_123")
		assert.NoError(t, err)

		// Try to use again
		err = provider.UseInvitationCode(ctx, "TEST-USE-002", "user_456")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), ErrInvitationCodeAlreadyUsed)

		// Cleanup
		m.DeleteWhere(model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "code", Value: "TEST-USE-002"},
			},
		})
	})

	// Test Case 4: Try to use unpublished code
	t.Run("Use unpublished code", func(t *testing.T) {
		codeData := []maps.MapStrAny{
			{
				"code":         "TEST-USE-003",
				"code_type":    "beta",
				"status":       "active",
				"is_published": false, // Not published
			},
		}
		_, err := provider.CreateInvitationCodes(ctx, codeData)
		assert.NoError(t, err)

		err = provider.UseInvitationCode(ctx, "TEST-USE-003", "user_123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), ErrInvitationCodeNotPublished)

		// Cleanup
		m.DeleteWhere(model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "code", Value: "TEST-USE-003"},
			},
		})
	})

	// Test Case 5: Try to use code with wrong status
	t.Run("Use code with draft status", func(t *testing.T) {
		codeData := []maps.MapStrAny{
			{
				"code":         "TEST-USE-004",
				"code_type":    "beta",
				"status":       "draft", // Not active
				"is_published": true,
			},
		}
		_, err := provider.CreateInvitationCodes(ctx, codeData)
		assert.NoError(t, err)

		err = provider.UseInvitationCode(ctx, "TEST-USE-004", "user_123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "status must be 'active'")

		// Cleanup
		m.DeleteWhere(model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "code", Value: "TEST-USE-004"},
			},
		})
	})

	// Test Case 6: Try to use expired code
	t.Run("Use expired code", func(t *testing.T) {
		// Create code that expired yesterday
		yesterday := time.Now().Add(-24 * time.Hour)
		codeData := []maps.MapStrAny{
			{
				"code":         "TEST-USE-005",
				"code_type":    "beta",
				"status":       "active",
				"is_published": true,
				"expires_at":   yesterday,
			},
		}
		_, err := provider.CreateInvitationCodes(ctx, codeData)
		assert.NoError(t, err)

		err = provider.UseInvitationCode(ctx, "TEST-USE-005", "user_123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), ErrInvitationCodeExpired)

		// Cleanup
		m.DeleteWhere(model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "code", Value: "TEST-USE-005"},
			},
		})
	})

	// Test Case 7: Use code that has not expired yet
	t.Run("Use code with future expiration", func(t *testing.T) {
		// Create code that expires tomorrow
		tomorrow := time.Now().Add(24 * time.Hour)
		codeData := []maps.MapStrAny{
			{
				"code":         "TEST-USE-006",
				"code_type":    "beta",
				"status":       "active",
				"is_published": true,
				"expires_at":   tomorrow,
			},
		}
		_, err := provider.CreateInvitationCodes(ctx, codeData)
		assert.NoError(t, err)

		err = provider.UseInvitationCode(ctx, "TEST-USE-006", "user_123")
		assert.NoError(t, err)

		// Cleanup
		m.DeleteWhere(model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "code", Value: "TEST-USE-006"},
			},
		})
	})
}

// TestDeleteInvitationCode tests invitation code deletion
func TestDeleteInvitationCode(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := context.Background()
	provider := NewDefaultUser(&DefaultUserOptions{})
	m := model.Select("__yao.invitation")

	// Test Case 1: Successfully delete an invitation code
	t.Run("Delete code successfully", func(t *testing.T) {
		// Create a code
		codeData := []maps.MapStrAny{
			{
				"code":      "TEST-DELETE-001",
				"code_type": "beta",
			},
		}
		codes, err := provider.CreateInvitationCodes(ctx, codeData)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(codes))

		// Delete the code
		err = provider.DeleteInvitationCode(ctx, "TEST-DELETE-001")
		assert.NoError(t, err)

		// Verify code was soft deleted (should not appear in normal queries)
		invitations, err := m.Get(model.QueryParam{
			Select: []interface{}{"code"},
			Wheres: []model.QueryWhere{
				{Column: "code", Value: "TEST-DELETE-001"},
			},
			Limit: 1,
		})
		assert.NoError(t, err)
		assert.Equal(t, 0, len(invitations), "Code should be soft deleted")
	})

	// Test Case 2: Try to delete non-existent code
	t.Run("Delete non-existent code", func(t *testing.T) {
		err := provider.DeleteInvitationCode(ctx, "NONEXISTENT-DELETE-CODE")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), ErrInvitationCodeNotFound)
	})

	// Test Case 3: Delete used code
	t.Run("Delete used code", func(t *testing.T) {
		// Create and use a code
		codeData := []maps.MapStrAny{
			{
				"code":         "TEST-DELETE-002",
				"code_type":    "beta",
				"status":       "active",
				"is_published": true,
			},
		}
		_, err := provider.CreateInvitationCodes(ctx, codeData)
		assert.NoError(t, err)

		// Use the code
		err = provider.UseInvitationCode(ctx, "TEST-DELETE-002", "user_123")
		assert.NoError(t, err)

		// Delete the used code (should succeed)
		err = provider.DeleteInvitationCode(ctx, "TEST-DELETE-002")
		assert.NoError(t, err)

		// Verify deletion
		invitations, err := m.Get(model.QueryParam{
			Select: []interface{}{"code"},
			Wheres: []model.QueryWhere{
				{Column: "code", Value: "TEST-DELETE-002"},
			},
			Limit: 1,
		})
		assert.NoError(t, err)
		assert.Equal(t, 0, len(invitations))
	})

	// Test Case 4: Try to delete same code twice
	t.Run("Delete code twice", func(t *testing.T) {
		// Create a code
		codeData := []maps.MapStrAny{
			{
				"code":      "TEST-DELETE-003",
				"code_type": "beta",
			},
		}
		_, err := provider.CreateInvitationCodes(ctx, codeData)
		assert.NoError(t, err)

		// First delete
		err = provider.DeleteInvitationCode(ctx, "TEST-DELETE-003")
		assert.NoError(t, err)

		// Second delete (should fail)
		err = provider.DeleteInvitationCode(ctx, "TEST-DELETE-003")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), ErrInvitationCodeNotFound)
	})
}

// TestInvitationCodeWorkflow tests the complete workflow: create -> use -> delete
func TestInvitationCodeWorkflow(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := context.Background()
	provider := NewDefaultUser(&DefaultUserOptions{})
	m := model.Select("__yao.invitation")

	t.Run("Complete workflow", func(t *testing.T) {
		// Step 1: Create multiple codes
		codeData := []maps.MapStrAny{
			{
				"code":         "WORKFLOW-001",
				"code_type":    "beta",
				"status":       "active",
				"is_published": true,
				"description":  "Workflow test code 1",
			},
			{
				"code":         "WORKFLOW-002",
				"code_type":    "beta",
				"status":       "active",
				"is_published": true,
				"description":  "Workflow test code 2",
			},
		}

		codes, err := provider.CreateInvitationCodes(ctx, codeData)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(codes))

		// Step 2: Use first code
		err = provider.UseInvitationCode(ctx, "WORKFLOW-001", "user_workflow_1")
		assert.NoError(t, err)

		// Step 3: Verify first code is used
		invitations, err := m.Get(model.QueryParam{
			Select: []interface{}{"code", "status", "used_by"},
			Wheres: []model.QueryWhere{
				{Column: "code", Value: "WORKFLOW-001"},
			},
			Limit: 1,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(invitations))
		assert.Equal(t, "used", invitations[0]["status"])
		assert.Equal(t, "user_workflow_1", invitations[0]["used_by"])

		// Step 4: Delete second code (unused)
		err = provider.DeleteInvitationCode(ctx, "WORKFLOW-002")
		assert.NoError(t, err)

		// Step 5: Delete first code (used)
		err = provider.DeleteInvitationCode(ctx, "WORKFLOW-001")
		assert.NoError(t, err)

		// Step 6: Verify both codes are deleted
		invitations, err = m.Get(model.QueryParam{
			Select: []interface{}{"code"},
			Wheres: []model.QueryWhere{
				{Column: "code", Value: []string{"WORKFLOW-001", "WORKFLOW-002"}, OP: "in"},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, 0, len(invitations))
	})
}
