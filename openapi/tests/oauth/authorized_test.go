package openapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
)

// TestGetInfoOAuthEmail tests that GetInfo correctly extracts __oauth_email from gin context
func TestGetInfoOAuthEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("extracts oauth_email when set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)

		// Set context values
		c.Set("__subject", "test-subject")
		c.Set("__client_id", "test-client")
		c.Set("__user_id", "test-user")
		c.Set("__scope", "openid profile")
		c.Set("__oauth_email", "user@example.com")

		info := authorized.GetInfo(c)

		assert.NotNil(t, info)
		assert.Equal(t, "test-subject", info.Subject)
		assert.Equal(t, "test-client", info.ClientID)
		assert.Equal(t, "test-user", info.UserID)
		assert.Equal(t, "openid profile", info.Scope)
		assert.Equal(t, "user@example.com", info.OAuthEmail)
	})

	t.Run("oauth_email is empty when not set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)

		c.Set("__subject", "test-subject")
		c.Set("__user_id", "test-user")

		info := authorized.GetInfo(c)

		assert.NotNil(t, info)
		assert.Equal(t, "test-user", info.UserID)
		assert.Empty(t, info.OAuthEmail, "OAuthEmail should be empty when __oauth_email is not set in context")
	})

	t.Run("oauth_email handles wrong type gracefully", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)

		c.Set("__subject", "test-subject")
		c.Set("__oauth_email", 12345) // Wrong type (int instead of string)

		info := authorized.GetInfo(c)

		assert.NotNil(t, info)
		assert.Empty(t, info.OAuthEmail, "OAuthEmail should be empty when __oauth_email has wrong type")
	})
}

// TestGetInfoAuthSource tests that GetInfo correctly extracts __auth_source from gin context
func TestGetInfoAuthSource(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("extracts auth_source when set to password", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)

		c.Set("__subject", "test-subject")
		c.Set("__user_id", "test-user")
		c.Set("__auth_source", "password")

		info := authorized.GetInfo(c)

		assert.NotNil(t, info)
		assert.Equal(t, "password", info.AuthSource)
	})

	t.Run("extracts auth_source when set to google", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)

		c.Set("__subject", "test-subject")
		c.Set("__user_id", "test-user")
		c.Set("__auth_source", "google")
		c.Set("__oauth_email", "s***a@gmail.com")

		info := authorized.GetInfo(c)

		assert.NotNil(t, info)
		assert.Equal(t, "google", info.AuthSource)
		assert.Equal(t, "s***a@gmail.com", info.OAuthEmail)
	})

	t.Run("extracts auth_source when set to github", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)

		c.Set("__subject", "test-subject")
		c.Set("__user_id", "test-user")
		c.Set("__auth_source", "github")

		info := authorized.GetInfo(c)

		assert.NotNil(t, info)
		assert.Equal(t, "github", info.AuthSource)
	})

	t.Run("auth_source is empty when not set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)

		c.Set("__subject", "test-subject")

		info := authorized.GetInfo(c)

		assert.NotNil(t, info)
		assert.Empty(t, info.AuthSource, "AuthSource should be empty when __auth_source is not set")
	})

	t.Run("auth_source handles wrong type gracefully", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)

		c.Set("__subject", "test-subject")
		c.Set("__auth_source", true) // Wrong type (bool instead of string)

		info := authorized.GetInfo(c)

		assert.NotNil(t, info)
		assert.Empty(t, info.AuthSource, "AuthSource should be empty when __auth_source has wrong type")
	})
}

// TestGetInfoRememberMe tests that GetInfo correctly extracts __remember_me from gin context
func TestGetInfoRememberMe(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("extracts remember_me when true", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)

		c.Set("__subject", "test-subject")
		c.Set("__remember_me", true)

		info := authorized.GetInfo(c)

		assert.NotNil(t, info)
		assert.True(t, info.RememberMe)
	})

	t.Run("remember_me defaults to false when not set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)

		c.Set("__subject", "test-subject")

		info := authorized.GetInfo(c)

		assert.NotNil(t, info)
		assert.False(t, info.RememberMe)
	})
}

// TestGetInfoTeamContext tests that GetInfo correctly extracts team-related fields
func TestGetInfoTeamContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("extracts full context for OAuth team member", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)

		// Simulate a Google-logged-in user who has selected a team
		c.Set("__subject", "sub-12345")
		c.Set("__client_id", "client-abc")
		c.Set("__user_id", "user-67890")
		c.Set("__scope", "openid profile email")
		c.Set("__team_id", "team-111")
		c.Set("__tenant_id", "tenant-222")
		c.Set("__sid", "session-333")
		c.Set("__remember_me", true)
		c.Set("__auth_source", "google")
		c.Set("__oauth_email", "u***r@gmail.com")

		info := authorized.GetInfo(c)

		assert.NotNil(t, info)
		assert.Equal(t, "sub-12345", info.Subject)
		assert.Equal(t, "client-abc", info.ClientID)
		assert.Equal(t, "user-67890", info.UserID)
		assert.Equal(t, "openid profile email", info.Scope)
		assert.Equal(t, "team-111", info.TeamID)
		assert.Equal(t, "tenant-222", info.TenantID)
		assert.Equal(t, "session-333", info.SessionID)
		assert.True(t, info.RememberMe)
		assert.Equal(t, "google", info.AuthSource)
		assert.Equal(t, "u***r@gmail.com", info.OAuthEmail)
	})

	t.Run("extracts context for password login user", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)

		// Simulate a password-logged-in user
		c.Set("__subject", "sub-admin")
		c.Set("__client_id", "client-abc")
		c.Set("__user_id", "user-admin")
		c.Set("__scope", "openid profile email")
		c.Set("__team_id", "team-default")
		c.Set("__auth_source", "password")
		// No __oauth_email for password login

		info := authorized.GetInfo(c)

		assert.NotNil(t, info)
		assert.Equal(t, "password", info.AuthSource)
		assert.Empty(t, info.OAuthEmail, "OAuthEmail should be empty for password login")
	})
}
