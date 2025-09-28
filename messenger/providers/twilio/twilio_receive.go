package twilio

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/messenger/types"
)

// TriggerWebhook processes Twilio webhook requests and converts to Message
func (p *Provider) TriggerWebhook(c interface{}) (*types.Message, error) {
	// Cast to gin.Context
	ginCtx, ok := c.(*gin.Context)
	if !ok {
		return nil, fmt.Errorf("expected *gin.Context, got %T", c)
	}

	// Parse form data (Twilio sends application/x-www-form-urlencoded)
	if err := ginCtx.Request.ParseForm(); err != nil {
		return nil, fmt.Errorf("failed to parse form data: %w", err)
	}

	// Create message from Twilio webhook data
	message := &types.Message{
		Metadata: make(map[string]interface{}),
	}

	// Extract common Twilio webhook fields
	messageSid := ginCtx.Request.FormValue("MessageSid")
	smsStatus := ginCtx.Request.FormValue("SmsStatus")
	from := ginCtx.Request.FormValue("From")
	to := ginCtx.Request.FormValue("To")
	body := ginCtx.Request.FormValue("Body")
	numSegments := ginCtx.Request.FormValue("NumSegments")
	errorCode := ginCtx.Request.FormValue("ErrorCode")

	// Map to standard message format
	if from != "" {
		message.From = from
	}
	if to != "" {
		message.To = []string{to}
	}
	if body != "" {
		message.Body = body
	}

	// Determine message type based on phone number format
	if strings.HasPrefix(to, "whatsapp:") || strings.HasPrefix(from, "whatsapp:") {
		message.Type = types.MessageTypeWhatsApp
	} else {
		message.Type = types.MessageTypeSMS
	}

	// Store webhook-specific data
	message.Metadata["provider"] = "twilio"
	message.Metadata["message_sid"] = messageSid
	message.Metadata["sms_status"] = smsStatus
	message.Metadata["num_segments"] = numSegments
	message.Metadata["error_code"] = errorCode
	message.Metadata["webhook_data"] = ginCtx.Request.Form

	// Handle different status types
	switch smsStatus {
	case "queued":
		message.Subject = "Message Queued"
		message.Body = fmt.Sprintf("Message from %s to %s is queued for delivery", from, to)
	case "sent":
		message.Subject = "Message Sent"
		message.Body = fmt.Sprintf("Message from %s to %s was sent", from, to)
	case "received":
		// Incoming message
		message.Subject = "Incoming Message"
		if message.Body == "" {
			message.Body = "Received message from " + from
		}
	case "delivered":
		message.Subject = "Message Delivered"
		message.Body = fmt.Sprintf("Message from %s to %s was delivered", from, to)
	case "undelivered":
		message.Subject = "Message Undelivered"
		message.Body = fmt.Sprintf("Message from %s to %s was not delivered", from, to)
		if errorCode != "" {
			message.Body += " (Error: " + errorCode + ")"
		}
	case "failed":
		message.Subject = "Message Failed"
		message.Body = fmt.Sprintf("Message from %s to %s failed", from, to)
		if errorCode != "" {
			message.Body += " (Error: " + errorCode + ")"
		}
	default:
		message.Subject = "Twilio Webhook Event"
		message.Body = fmt.Sprintf("Received %s status for message from %s to %s", smsStatus, from, to)
	}

	return message, nil
}
