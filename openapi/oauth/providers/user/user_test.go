package user_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/openapi/oauth/providers/user"
	"github.com/yaoapp/yao/test"
)

// TestUserData represents test user data structure (without UserID - auto-generated)
type TestUserData struct {
	PreferredUsername string                 `json:"preferred_username"`
	Email             string                 `json:"email"`
	Password          string                 `json:"password"`
	Name              string                 `json:"name"`
	GivenName         string                 `json:"given_name"`
	FamilyName        string                 `json:"family_name"`
	Status            string                 `json:"status"`
	RoleID            string                 `json:"role_id"`
	TypeID            string                 `json:"type_id"`
	EmailVerified     bool                   `json:"email_verified"`
	Metadata          map[string]interface{} `json:"metadata"`
}

var (
	testProvider *user.DefaultUser
)

// prepare initializes the test environment for each test function.
//
// PREREQUISITES:
// Before running any tests in this package, you MUST execute the following command in your terminal:
//
//	source $YAO_SOURCE_ROOT/env.local.sh
//
// This loads the required environment variables for the test environment.
//
// WHAT THIS FUNCTION DOES:
// Step 1: Calls test.Prepare(t, config.Conf) to initialize the base Yao test environment
//
//	This sets up database connections, configurations, and other core dependencies
//
// Step 2: Creates test provider with configured options
//
//	This sets up the DefaultUser provider for testing with test-specific configuration
//
// Usage pattern for ALL user provider tests:
//
//	func TestYourFunction(t *testing.T) {
//	    prepare(t)
//	    defer clean()
//
//	    ctx := context.Background() // Each test creates its own context
//
//	    // Your actual test code here...
//	}
func prepare(t *testing.T) {
	// Step 1: Initialize base test environment with all Yao dependencies
	test.Prepare(t, config.Conf)

	// Step 2: Initialize test provider
	testProvider = user.NewDefaultUser(&user.DefaultUserOptions{
		Prefix:     "test:",
		IDStrategy: user.NanoIDStrategy,
		IDPrefix:   "test_",
	})
}

// clean cleans up the test environment after each test function.
//
// WHAT THIS FUNCTION DOES:
// Step 1: Clean up test data from database
// Step 2: Reset global variables
// Step 3: Call test.Clean() to clean up the base test environment
//
// This function should ALWAYS be called with defer to ensure cleanup happens even if tests panic.
func clean() {
	// Step 1: Clean up test data
	cleanupTestData()

	// Step 2: Reset global variables
	testProvider = nil

	// Step 3: Clean up base test environment
	test.Clean()
}

// cleanupTestData removes all test data
func cleanupTestData() {
	if testProvider == nil {
		return
	}

	// Use DestroyWhere (hard delete) to avoid soft delete complications
	// Clean OAuth accounts first (due to foreign key constraints)
	oauthModel := model.Select("__yao.user_oauth_account")
	oauthPatterns := []string{
		"%oauth_test%", "%_list_%", "%oauthlist%", "%oautherror%",
		"%google_%", "%github_%", "%apple_%", "%_delete_%",
		"%discord_%", "%linkedin_%", "%twitter_%",
	}
	for _, pattern := range oauthPatterns {
		oauthModel.DestroyWhere(model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "sub", OP: "like", Value: pattern},
			},
		})
	}

	// Clean roles (should be done before users due to potential role_id references)
	roleModel := model.Select("__yao.user_role")
	rolePatterns := []string{
		"test%", "%testrole%", "%listrole%", "%permrole%", "%adminrole%", "%userrole%",
		"%inactiverole%", "%systemrole%", "%validrole%", "%emptyupdate%", "%emptyperm%",
		"%guestrole%", "%scoperole%", "%test-role-for-exists%",
	}
	for _, pattern := range rolePatterns {
		roleModel.DestroyWhere(model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "role_id", OP: "like", Value: pattern},
			},
		})
	}

	// Clean types (should be done before users due to potential type_id references)
	typeModel := model.Select("__yao.user_type")
	typePatterns := []string{
		"test%", "%testtype%", "%listtype%", "%configtype%", "%basictype%", "%premiumtype%",
		"%inactivetype%", "%validtype%", "%emptyupdate%", "%emptyconfig%", "%scopetype%",
		"%opentype%", "%test-type-for-exists%",
	}
	for _, pattern := range typePatterns {
		typeModel.DestroyWhere(model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "type_id", OP: "like", Value: pattern},
			},
		})
	}

	// Clean users
	userModel := model.Select("__yao.user")

	// Delete test users by pattern (using hard delete)
	userPatterns := []string{
		"test-%", "test_%", "%testuser%", "%oauthtest%", "%oauthlist%",
		"%oautherror%", "%deletetest%", "%roleuser%", "%typeuser%", "%scopeuser%",
		"%erroruser%", "%noroleuser%", "%clearnouser%", "%integuser%", "%notypeuser%",
		"%openuser%", "%clearnotypeuser%", "%test-user-for-exists%", "%perf-test-user%",
	}
	for _, pattern := range userPatterns {
		userModel.DestroyWhere(model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "user_id", OP: "like", Value: pattern},
			},
		})
	}

	// Also clean by username pattern
	usernamePatterns := []string{
		"testuser%", "%oauth_%", "%deletetest%", "%roleuser%", "%typeuser%",
		"%scopeuser%", "%erroruser%", "%noroleuser%", "%clearnouser%", "%integuser%",
		"%notypeuser%", "%openuser%", "%clearnotypeuser%", "%testexistsuser%", "%perfuser%",
	}
	for _, pattern := range usernamePatterns {
		userModel.DestroyWhere(model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "preferred_username", OP: "like", Value: pattern},
			},
		})
	}
}

// setupTestUser creates a user in database for testing
func setupTestUser(t *testing.T, ctx context.Context, userData *TestUserData) (interface{}, string) {
	userMap := maps.MapStrAny{
		// user_id will be auto-generated by CreateUser
		"preferred_username": userData.PreferredUsername,
		"email":              userData.Email,
		"password":           userData.Password, // Will be auto-hashed by Yao
		"name":               userData.Name,
		"given_name":         userData.GivenName,
		"family_name":        userData.FamilyName,
		"status":             userData.Status,
		"role_id":            userData.RoleID,
		"type_id":            userData.TypeID,
		"email_verified":     userData.EmailVerified,
		"metadata":           userData.Metadata,
	}

	id, err := testProvider.CreateUser(ctx, userMap)
	require.NoError(t, err)

	// Return both database ID and auto-generated user_id
	userID := userMap["user_id"].(string)
	return id, userID
}

// createTestUserData creates test user data with unique identifier
func createTestUserData(id string) *TestUserData {
	return &TestUserData{
		// user_id will be auto-generated, not set here
		PreferredUsername: "testuser" + id,
		Email:             "testuser" + id + "@example.com",
		Password:          "TestPass" + id + "!",
		Name:              "Test User " + id,
		GivenName:         "Test",
		FamilyName:        "User " + id,
		Status:            "active",
		RoleID:            "user",
		TypeID:            "regular",
		EmailVerified:     true,
		Metadata:          map[string]interface{}{"source": "test", "id": id},
	}
}
