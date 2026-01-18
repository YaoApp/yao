package mailgun

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/messenger/types"
)

// TriggerWebhook processes Mailgun webhook requests and converts to Message
func (p *Provider) TriggerWebhook(c interface{}) (*types.Message, error) {
	// Cast to gin.Context
	ginCtx, ok := c.(*gin.Context)
	if !ok {
		return nil, fmt.Errorf("expected *gin.Context, got %T", c)
	}

	// Parse form data (Mailgun sends application/x-www-form-urlencoded)
	if err := ginCtx.Request.ParseForm(); err != nil {
		return nil, fmt.Errorf("failed to parse form data: %w", err)
	}

	// Create message from Mailgun webhook data
	message := &types.Message{
		Metadata: make(map[string]interface{}),
	}

	// Extract common Mailgun webhook fields
	event := ginCtx.Request.FormValue("event")
	recipient := ginCtx.Request.FormValue("recipient")
	messageID := ginCtx.Request.FormValue("message-id")
	timestamp := ginCtx.Request.FormValue("timestamp")
	token := ginCtx.Request.FormValue("token")
	signature := ginCtx.Request.FormValue("signature")

	// Map to standard message format
	message.Type = types.MessageTypeEmail
	if recipient != "" {
		message.To = []string{recipient}
	}
	if messageID != "" {
		message.Metadata["message_id"] = messageID
	}

	// Store webhook-specific data
	message.Metadata["provider"] = "mailgun"
	message.Metadata["event"] = event
	message.Metadata["timestamp"] = timestamp
	message.Metadata["token"] = token
	message.Metadata["signature"] = signature
	message.Metadata["webhook_data"] = ginCtx.Request.Form

	// Handle different event types
	switch event {
	case "delivered":
		message.Subject = "Email Delivered"
		message.Body = fmt.Sprintf("Email to %s was delivered successfully", recipient)
	case "failed":
		message.Subject = "Email Failed"
		message.Body = fmt.Sprintf("Email to %s failed to deliver", recipient)
		if reason := ginCtx.Request.FormValue("reason"); reason != "" {
			message.Body += ": " + reason
		}
	case "opened":
		message.Subject = "Email Opened"
		message.Body = fmt.Sprintf("Email to %s was opened", recipient)
	case "clicked":
		message.Subject = "Email Clicked"
		message.Body = fmt.Sprintf("Link in email to %s was clicked", recipient)
	case "unsubscribed":
		message.Subject = "Email Unsubscribed"
		message.Body = fmt.Sprintf("Recipient %s unsubscribed", recipient)
	case "complained":
		message.Subject = "Email Complained"
		message.Body = fmt.Sprintf("Recipient %s marked email as spam", recipient)
	case "stored":
		// Incoming email
		message.Subject = ginCtx.Request.FormValue("subject")
		message.Body = ginCtx.Request.FormValue("body-plain")
		message.HTML = ginCtx.Request.FormValue("body-html")
		message.From = ginCtx.Request.FormValue("sender")
		if message.Subject == "" {
			message.Subject = "Incoming Email"
		}
	default:
		message.Subject = "Mailgun Webhook Event"
		message.Body = fmt.Sprintf("Received %s event for %s", event, recipient)
	}

	return message, nil
}
