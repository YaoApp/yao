package standard

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/yaoapp/gou/process"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/attachment"
	"github.com/yaoapp/yao/messenger"
	messengerTypes "github.com/yaoapp/yao/messenger/types"
)

// DeliveryCenter handles routing delivery content to configured channels
// It decides which channels to use based on robot/user preferences and executes the delivery
type DeliveryCenter struct {
	httpClient *http.Client
}

// NewDeliveryCenter creates a new DeliveryCenter instance
func NewDeliveryCenter() *DeliveryCenter {
	return &DeliveryCenter{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Deliver sends content to all configured channels based on preferences
// Returns results for each channel target and any error
func (dc *DeliveryCenter) Deliver(
	ctx *robottypes.Context,
	content *robottypes.DeliveryContent,
	deliveryCtx *robottypes.DeliveryContext,
	prefs *robottypes.DeliveryPreferences,
	robotInstance *robottypes.Robot,
) ([]robottypes.ChannelResult, error) {
	if content == nil {
		return nil, fmt.Errorf("delivery content is nil")
	}
	if prefs == nil {
		return nil, nil // No preferences = no delivery
	}

	var results []robottypes.ChannelResult
	var lastErr error

	// Process email targets
	if prefs.Email != nil && prefs.Email.Enabled {
		for _, target := range prefs.Email.Targets {
			result := dc.sendEmail(ctx.Context, content, target, deliveryCtx, robotInstance)
			results = append(results, result)
			if !result.Success && lastErr == nil {
				lastErr = fmt.Errorf("email delivery failed: %s", result.Error)
			}
		}
	}

	// Process webhook targets
	if prefs.Webhook != nil && prefs.Webhook.Enabled {
		for _, target := range prefs.Webhook.Targets {
			result := dc.postWebhook(ctx.Context, content, target, deliveryCtx)
			results = append(results, result)
			if !result.Success && lastErr == nil {
				lastErr = fmt.Errorf("webhook delivery failed: %s", result.Error)
			}
		}
	}

	// Process process targets
	if prefs.Process != nil && prefs.Process.Enabled {
		for _, target := range prefs.Process.Targets {
			result := dc.callProcess(ctx.Context, content, target, deliveryCtx)
			results = append(results, result)
			if !result.Success && lastErr == nil {
				lastErr = fmt.Errorf("process delivery failed: %s", result.Error)
			}
		}
	}

	return results, lastErr
}

// sendEmail sends delivery content to a single email target
func (dc *DeliveryCenter) sendEmail(
	ctx context.Context,
	content *robottypes.DeliveryContent,
	target robottypes.EmailTarget,
	deliveryCtx *robottypes.DeliveryContext,
	robotInstance *robottypes.Robot,
) robottypes.ChannelResult {
	now := time.Now()

	// Build target identifier from recipients
	targetID := strings.Join(target.To, ",")
	if targetID == "" {
		targetID = "no-recipients"
	}

	result := robottypes.ChannelResult{
		Type:   robottypes.DeliveryEmail,
		Target: targetID,
		SentAt: &now,
	}

	// Get messenger service
	svc := messenger.Instance
	if svc == nil {
		result.Error = "messenger service not available"
		return result
	}

	// Build email message
	msg := &messengerTypes.Message{
		To:      target.To,
		Subject: buildEmailSubject(target.Subject, target.Template, content, deliveryCtx),
		Body:    buildEmailBody(target.Template, content),
		Type:    messengerTypes.MessageTypeEmail,
	}

	// Set From address from Robot's email (if configured)
	if robotInstance != nil && robotInstance.RobotEmail != "" {
		msg.From = robotInstance.RobotEmail
	}

	// Convert attachments
	attachments := convertAttachments(ctx, content.Attachments)
	if len(attachments) > 0 {
		msg.Attachments = attachments
	}

	// Send email using global default channel
	channel := robottypes.DefaultEmailChannel()
	err := svc.Send(ctx, channel, msg)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.Success = true
	result.Recipients = target.To

	return result
}

// postWebhook posts delivery content to a single webhook target
func (dc *DeliveryCenter) postWebhook(
	ctx context.Context,
	content *robottypes.DeliveryContent,
	target robottypes.WebhookTarget,
	deliveryCtx *robottypes.DeliveryContext,
) robottypes.ChannelResult {
	now := time.Now()
	result := robottypes.ChannelResult{
		Type:   robottypes.DeliveryWebhook,
		Target: target.URL,
		SentAt: &now,
	}

	// Build webhook payload
	payload := map[string]interface{}{
		"event":        "robot.delivery",
		"timestamp":    now.Format(time.RFC3339),
		"execution_id": deliveryCtx.ExecutionID,
		"member_id":    deliveryCtx.MemberID,
		"team_id":      deliveryCtx.TeamID,
		"trigger_type": deliveryCtx.TriggerType,
		"content": map[string]interface{}{
			"summary": content.Summary,
			"body":    content.Body,
		},
	}

	// Add attachments info (not the actual files)
	if len(content.Attachments) > 0 {
		attachmentInfo := make([]map[string]interface{}, 0, len(content.Attachments))
		for _, att := range content.Attachments {
			attachmentInfo = append(attachmentInfo, map[string]interface{}{
				"title":       att.Title,
				"description": att.Description,
				"task_id":     att.TaskID,
				"file":        att.File,
			})
		}
		payload["attachments"] = attachmentInfo
	}

	// Marshal payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		result.Error = fmt.Sprintf("failed to marshal payload: %v", err)
		return result
	}

	// Build request
	method := target.Method
	if method == "" {
		method = "POST"
	}

	req, err := http.NewRequestWithContext(ctx, method, target.URL, bytes.NewReader(payloadBytes))
	if err != nil {
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		return result
	}

	req.Header.Set("Content-Type", "application/json")

	// Add custom headers
	for key, value := range target.Headers {
		req.Header.Set(key, value)
	}

	// Add secret if configured (for signature verification)
	if target.Secret != "" {
		// TODO: Implement HMAC signature
		req.Header.Set("X-Webhook-Secret", target.Secret)
	}

	// Send request
	resp, err := dc.httpClient.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("request failed: %v", err)
		return result
	}
	defer resp.Body.Close()

	// Read response body
	body, _ := io.ReadAll(resp.Body)

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result.Error = fmt.Sprintf("webhook returned status %d: %s", resp.StatusCode, string(body))
		return result
	}

	result.Success = true
	result.Details = map[string]interface{}{
		"status_code": resp.StatusCode,
		"response":    string(body),
	}

	return result
}

// callProcess calls a Yao Process with delivery content
func (dc *DeliveryCenter) callProcess(
	ctx context.Context,
	content *robottypes.DeliveryContent,
	target robottypes.ProcessTarget,
	deliveryCtx *robottypes.DeliveryContext,
) robottypes.ChannelResult {
	now := time.Now()
	result := robottypes.ChannelResult{
		Type:   robottypes.DeliveryProcess,
		Target: target.Process,
		SentAt: &now,
	}

	// Build args: DeliveryContent as first arg, then additional args
	args := make([]interface{}, 0, 1+len(target.Args))
	args = append(args, map[string]interface{}{
		"content": map[string]interface{}{
			"summary":     content.Summary,
			"body":        content.Body,
			"attachments": content.Attachments,
		},
		"context": map[string]interface{}{
			"execution_id": deliveryCtx.ExecutionID,
			"member_id":    deliveryCtx.MemberID,
			"team_id":      deliveryCtx.TeamID,
			"trigger_type": deliveryCtx.TriggerType,
		},
	})
	args = append(args, target.Args...)

	// Create and execute process
	proc, err := process.Of(target.Process, args...)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create process: %v", err)
		return result
	}
	proc.Context = ctx

	err = proc.Execute()
	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.Success = true
	result.Details = proc.Value

	return result
}

// buildEmailSubject builds the email subject line
func buildEmailSubject(subject, template string, content *robottypes.DeliveryContent, ctx *robottypes.DeliveryContext) string {
	// Use explicit subject if provided
	if subject != "" {
		return subject
	}

	// Use template-based subject if template is specified
	// TODO: Implement template rendering
	if template != "" {
		return fmt.Sprintf("[Robot] %s", content.Summary)
	}

	// Default: use summary
	if content.Summary != "" {
		return fmt.Sprintf("[Robot] %s", content.Summary)
	}

	return fmt.Sprintf("[Robot] Execution %s Complete", ctx.ExecutionID)
}

// buildEmailBody builds the email body content
func buildEmailBody(template string, content *robottypes.DeliveryContent) string {
	// TODO: Implement template rendering
	// For now, just use the body directly
	if content.Body != "" {
		return content.Body
	}
	return content.Summary
}

// convertAttachments converts DeliveryAttachment to messenger Attachment format
func convertAttachments(ctx context.Context, attachments []robottypes.DeliveryAttachment) []messengerTypes.Attachment {
	if len(attachments) == 0 {
		return nil
	}

	result := make([]messengerTypes.Attachment, 0, len(attachments))

	for _, att := range attachments {
		// Parse file wrapper: __<uploader>://<fileID>
		uploader, fileID, isWrapper := attachment.Parse(att.File)
		if !isWrapper {
			// Skip non-wrapper attachments
			continue
		}

		// Get file info from attachment manager
		manager, ok := attachment.Managers[uploader]
		if !ok {
			continue
		}

		info, err := manager.Info(ctx, fileID)
		if err != nil {
			continue
		}

		// Read file content
		content, err := manager.Read(ctx, fileID)
		if err != nil {
			continue
		}

		// Build messenger attachment
		msgAtt := messengerTypes.Attachment{
			Filename:    info.Filename,
			ContentType: info.ContentType,
			Content:     content,
		}

		result = append(result, msgAtt)
	}

	return result
}
