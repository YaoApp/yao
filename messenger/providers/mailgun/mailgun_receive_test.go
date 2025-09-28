package mailgun

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/messenger/types"
)

// createMockGinContext creates a mock gin.Context for testing webhook functionality
func createMockGinContext(formData map[string]interface{}) *gin.Context {
	// Create form values
	values := url.Values{}
	for key, value := range formData {
		if str, ok := value.(string); ok {
			values.Set(key, str)
		}
	}

	// Create request with form data
	req := httptest.NewRequest("POST", "/webhook/mailgun", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Create response recorder
	w := httptest.NewRecorder()

	// Create gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	return c
}

func TestProvider_TriggerWebhook(t *testing.T) {
	// Create a mailgun provider
	config := types.ProviderConfig{
		Name:      "test-mailgun",
		Connector: "mailgun",
		Options: map[string]interface{}{
			"domain":  "test.mailgun.org",
			"api_key": "test-api-key",
			"from":    "test@test.mailgun.org",
		},
	}

	provider, err := NewMailgunProvider(config)
	require.NoError(t, err)

	tests := []struct {
		name     string
		formData map[string]interface{}
		wantErr  bool
		checkFn  func(t *testing.T, msg *types.Message)
	}{
		{
			name: "delivered event",
			formData: map[string]interface{}{
				"event":      "delivered",
				"recipient":  "test@example.com",
				"message-id": "test-message-id",
				"timestamp":  "1234567890",
			},
			wantErr: false,
			checkFn: func(t *testing.T, msg *types.Message) {
				assert.Equal(t, types.MessageTypeEmail, msg.Type)
				assert.Contains(t, msg.To, "test@example.com")
				assert.Equal(t, "Email Delivered", msg.Subject)
				assert.Contains(t, msg.Body, "test@example.com")
				assert.Contains(t, msg.Body, "delivered successfully")
				assert.Equal(t, "mailgun", msg.Metadata["provider"])
				assert.Equal(t, "delivered", msg.Metadata["event"])
			},
		},
		{
			name: "failed event",
			formData: map[string]interface{}{
				"event":     "failed",
				"recipient": "failed@example.com",
				"reason":    "bounce",
			},
			wantErr: false,
			checkFn: func(t *testing.T, msg *types.Message) {
				assert.Equal(t, "Email Failed", msg.Subject)
				assert.Contains(t, msg.Body, "failed@example.com")
				assert.Contains(t, msg.Body, "bounce")
			},
		},
		{
			name: "stored event (incoming email)",
			formData: map[string]interface{}{
				"event":      "stored",
				"sender":     "sender@example.com",
				"recipient":  "inbox@example.com",
				"subject":    "Incoming Email Subject",
				"body-plain": "Email body content",
				"body-html":  "<p>Email HTML content</p>",
			},
			wantErr: false,
			checkFn: func(t *testing.T, msg *types.Message) {
				assert.Equal(t, "Incoming Email Subject", msg.Subject)
				assert.Equal(t, "sender@example.com", msg.From)
				assert.Equal(t, "Email body content", msg.Body)
				assert.Equal(t, "<p>Email HTML content</p>", msg.HTML)
			},
		},
		{
			name: "opened event",
			formData: map[string]interface{}{
				"event":     "opened",
				"recipient": "reader@example.com",
			},
			wantErr: false,
			checkFn: func(t *testing.T, msg *types.Message) {
				assert.Equal(t, "Email Opened", msg.Subject)
				assert.Contains(t, msg.Body, "reader@example.com")
				assert.Contains(t, msg.Body, "opened")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtx := createMockGinContext(tt.formData)

			msg, err := provider.TriggerWebhook(mockCtx)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, msg)

			// Run specific checks
			if tt.checkFn != nil {
				tt.checkFn(t, msg)
			}

			// Common checks
			assert.NotNil(t, msg.Metadata)
			assert.Equal(t, "mailgun", msg.Metadata["provider"])
		})
	}
}

func TestProvider_TriggerWebhook_InvalidInput(t *testing.T) {
	config := types.ProviderConfig{
		Name:      "test-mailgun",
		Connector: "mailgun",
		Options: map[string]interface{}{
			"domain":  "test.mailgun.org",
			"api_key": "test-api-key",
			"from":    "test@test.mailgun.org",
		},
	}

	provider, err := NewMailgunProvider(config)
	require.NoError(t, err)

	// Test with wrong input type
	msg, err := provider.TriggerWebhook("not-gin-context")
	assert.Error(t, err)
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "expected *gin.Context")
}
