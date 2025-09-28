package twilio

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
	req := httptest.NewRequest("POST", "/webhook/twilio", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Create response recorder
	w := httptest.NewRecorder()

	// Create gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	return c
}

func TestProvider_TriggerWebhook(t *testing.T) {
	// Create a twilio provider
	config := types.ProviderConfig{
		Name:      "test-twilio",
		Connector: "twilio",
		Options: map[string]interface{}{
			"account_sid": "test-account-sid",
			"auth_token":  "test-auth-token",
			"from_phone":  "+1234567890",
		},
	}

	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)

	tests := []struct {
		name     string
		formData map[string]interface{}
		wantErr  bool
		checkFn  func(t *testing.T, msg *types.Message)
	}{
		{
			name: "SMS received",
			formData: map[string]interface{}{
				"MessageSid": "test-message-sid",
				"SmsStatus":  "received",
				"From":       "+1234567890",
				"To":         "+0987654321",
				"Body":       "Hello from SMS",
			},
			wantErr: false,
			checkFn: func(t *testing.T, msg *types.Message) {
				assert.Equal(t, types.MessageTypeSMS, msg.Type)
				assert.Equal(t, "+1234567890", msg.From)
				assert.Contains(t, msg.To, "+0987654321")
				assert.Equal(t, "Hello from SMS", msg.Body)
				assert.Equal(t, "Incoming Message", msg.Subject)
				assert.Equal(t, "twilio", msg.Metadata["provider"])
				assert.Equal(t, "test-message-sid", msg.Metadata["message_sid"])
				assert.Equal(t, "received", msg.Metadata["sms_status"])
			},
		},
		{
			name: "WhatsApp received",
			formData: map[string]interface{}{
				"MessageSid": "whatsapp-message-sid",
				"SmsStatus":  "received",
				"From":       "whatsapp:+1234567890",
				"To":         "whatsapp:+0987654321",
				"Body":       "Hello from WhatsApp",
			},
			wantErr: false,
			checkFn: func(t *testing.T, msg *types.Message) {
				assert.Equal(t, types.MessageTypeWhatsApp, msg.Type)
				assert.Equal(t, "whatsapp:+1234567890", msg.From)
				assert.Contains(t, msg.To, "whatsapp:+0987654321")
				assert.Equal(t, "Hello from WhatsApp", msg.Body)
				assert.Equal(t, "Incoming Message", msg.Subject)
			},
		},
		{
			name: "SMS delivered",
			formData: map[string]interface{}{
				"MessageSid": "delivered-message-sid",
				"SmsStatus":  "delivered",
				"From":       "+1234567890",
				"To":         "+0987654321",
			},
			wantErr: false,
			checkFn: func(t *testing.T, msg *types.Message) {
				assert.Equal(t, "Message Delivered", msg.Subject)
				assert.Contains(t, msg.Body, "delivered")
			},
		},
		{
			name: "SMS failed",
			formData: map[string]interface{}{
				"MessageSid": "failed-message-sid",
				"SmsStatus":  "failed",
				"From":       "+1234567890",
				"To":         "+0987654321",
				"ErrorCode":  "30001",
			},
			wantErr: false,
			checkFn: func(t *testing.T, msg *types.Message) {
				assert.Equal(t, "Message Failed", msg.Subject)
				assert.Contains(t, msg.Body, "failed")
				assert.Contains(t, msg.Body, "30001")
				assert.Equal(t, "30001", msg.Metadata["error_code"])
			},
		},
		{
			name: "SMS queued",
			formData: map[string]interface{}{
				"MessageSid":  "queued-message-sid",
				"SmsStatus":   "queued",
				"From":        "+1234567890",
				"To":          "+0987654321",
				"NumSegments": "1",
			},
			wantErr: false,
			checkFn: func(t *testing.T, msg *types.Message) {
				assert.Equal(t, "Message Queued", msg.Subject)
				assert.Contains(t, msg.Body, "queued")
				assert.Equal(t, "1", msg.Metadata["num_segments"])
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
			assert.Equal(t, "twilio", msg.Metadata["provider"])
		})
	}
}

func TestProvider_TriggerWebhook_InvalidInput(t *testing.T) {
	config := types.ProviderConfig{
		Name:      "test-twilio",
		Connector: "twilio",
		Options: map[string]interface{}{
			"account_sid": "test-account-sid",
			"auth_token":  "test-auth-token",
			"from_phone":  "+1234567890",
		},
	}

	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)

	// Test with wrong input type
	msg, err := provider.TriggerWebhook("not-gin-context")
	assert.Error(t, err)
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "expected *gin.Context")
}
