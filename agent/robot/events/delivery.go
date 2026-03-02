package events

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/text"
	agentcontext "github.com/yaoapp/yao/agent/context"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/attachment"
	eventtypes "github.com/yaoapp/yao/event/types"
	"github.com/yaoapp/yao/messenger"
	messengerTypes "github.com/yaoapp/yao/messenger/types"
)

// handleDelivery routes delivery content to configured channels (email, webhook, process).
func (h *robotHandler) handleDelivery(ctx context.Context, ev *eventtypes.Event, resp chan<- eventtypes.Result) {
	var payload DeliveryPayload
	if err := ev.Should(&payload); err != nil {
		log.Error("delivery handler: invalid payload: %v", err)
		if ev.IsCall {
			resp <- eventtypes.Result{Err: err}
		}
		return
	}

	log.Info("delivery handler: execution=%s member=%s", payload.ExecutionID, payload.MemberID)

	content := payload.Content
	prefs := payload.Preferences
	if content == nil {
		log.Warn("delivery handler: nil content for execution=%s", payload.ExecutionID)
		if ev.IsCall {
			resp <- eventtypes.Result{Data: "no content"}
		}
		return
	}
	if prefs == nil {
		if ev.IsCall {
			resp <- eventtypes.Result{Data: "no preferences, skipped"}
		}
		return
	}

	deliveryCtx := &robottypes.DeliveryContext{
		MemberID:    payload.MemberID,
		ExecutionID: payload.ExecutionID,
		TeamID:      payload.TeamID,
	}

	var results []robottypes.ChannelResult
	var lastErr error

	if prefs.Email != nil && prefs.Email.Enabled {
		for _, target := range prefs.Email.Targets {
			r := h.sendEmail(ctx, content, target, deliveryCtx)
			results = append(results, r)
			if !r.Success && lastErr == nil {
				lastErr = fmt.Errorf("email delivery failed: %s", r.Error)
			}
		}
	}

	if prefs.Webhook != nil && prefs.Webhook.Enabled {
		for _, target := range prefs.Webhook.Targets {
			r := h.postWebhook(ctx, content, target, deliveryCtx)
			results = append(results, r)
			if !r.Success && lastErr == nil {
				lastErr = fmt.Errorf("webhook delivery failed: %s", r.Error)
			}
		}
	}

	if prefs.Process != nil && prefs.Process.Enabled {
		for _, target := range prefs.Process.Targets {
			r := h.callProcess(ctx, content, target, deliveryCtx)
			results = append(results, r)
			if !r.Success && lastErr == nil {
				lastErr = fmt.Errorf("process delivery failed: %s", r.Error)
			}
		}
	}

	// Push delivery to integration channels only when the task originated from one
	if reply := getReplyFunc(); reply != nil && payload.ChatID != "" {
		channel, chatID := splitChannelChatID(payload.ChatID)
		if channel != "" && chatID != "" {
			msg := buildDeliveryMessage(content)
			if msg != nil {
				extra := map[string]any{
					"member_id":    payload.MemberID,
					"execution_id": payload.ExecutionID,
				}
				for k, v := range payload.Extra {
					extra[k] = v
				}
				metadata := &MessageMetadata{
					Channel: channel,
					ChatID:  chatID,
					Extra:   extra,
				}
				if err := reply(ctx, msg, metadata); err != nil {
					log.Error("delivery handler: integration reply failed channel=%s execution=%s: %v", channel, payload.ExecutionID, err)
				}
			}
		}
	}

	if lastErr != nil {
		log.Error("delivery handler: partial failure execution=%s: %v", payload.ExecutionID, lastErr)
	}

	if ev.IsCall {
		resp <- eventtypes.Result{
			Data: map[string]interface{}{
				"execution_id": payload.ExecutionID,
				"results":      results,
			},
			Err: lastErr,
		}
	}
}

// buildDeliveryMessage converts DeliveryContent into a standard assistant Message.
func buildDeliveryMessage(content *robottypes.DeliveryContent) *agentcontext.Message {
	if content == nil {
		return nil
	}

	var parts []interface{}

	text := content.Body
	if text == "" {
		text = content.Summary
	}
	if text != "" {
		parts = append(parts, map[string]interface{}{
			"type": "text",
			"text": text,
		})
	}

	for _, att := range content.Attachments {
		if att.File == "" {
			continue
		}
		part := map[string]interface{}{
			"type": "file",
			"file": map[string]interface{}{
				"url":      att.File,
				"filename": att.Title,
			},
		}
		parts = append(parts, part)
	}

	if len(parts) == 0 {
		return nil
	}

	var msgContent interface{}
	if len(parts) == 1 {
		if tp, ok := parts[0].(map[string]interface{}); ok && tp["type"] == "text" {
			msgContent = tp["text"]
		} else {
			msgContent = parts
		}
	} else {
		msgContent = parts
	}

	return &agentcontext.Message{
		Role:    agentcontext.RoleAssistant,
		Content: msgContent,
	}
}

// ============================================================================
// Email
// ============================================================================

func (h *robotHandler) sendEmail(
	ctx context.Context,
	content *robottypes.DeliveryContent,
	target robottypes.EmailTarget,
	deliveryCtx *robottypes.DeliveryContext,
) robottypes.ChannelResult {
	now := time.Now()
	targetID := strings.Join(target.To, ",")
	if targetID == "" {
		targetID = "no-recipients"
	}

	result := robottypes.ChannelResult{
		Type:   robottypes.DeliveryEmail,
		Target: targetID,
		SentAt: &now,
	}

	svc := messenger.Instance
	if svc == nil {
		result.Error = "messenger service not available"
		return result
	}

	htmlBody, plainBody := buildEmailBody(target.Template, content)
	msg := &messengerTypes.Message{
		To:      target.To,
		Subject: buildEmailSubject(target.Subject, target.Template, content, deliveryCtx),
		Body:    plainBody,
		HTML:    htmlBody,
		Type:    messengerTypes.MessageTypeEmail,
	}

	attachments := convertAttachments(ctx, content.Attachments)
	if len(attachments) > 0 {
		msg.Attachments = attachments
	}

	channel := robottypes.DefaultEmailChannel()
	if err := svc.Send(ctx, channel, msg); err != nil {
		result.Error = err.Error()
		return result
	}

	result.Success = true
	result.Recipients = target.To
	return result
}

// ============================================================================
// Webhook
// ============================================================================

func (h *robotHandler) postWebhook(
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

	if len(content.Attachments) > 0 {
		info := make([]map[string]interface{}, 0, len(content.Attachments))
		for _, att := range content.Attachments {
			info = append(info, map[string]interface{}{
				"title":       att.Title,
				"description": att.Description,
				"task_id":     att.TaskID,
				"file":        att.File,
			})
		}
		payload["attachments"] = info
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		result.Error = fmt.Sprintf("failed to marshal payload: %v", err)
		return result
	}

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
	for key, value := range target.Headers {
		req.Header.Set(key, value)
	}

	if target.Secret != "" {
		signature := ComputeHMACSignature(payloadBytes, target.Secret)
		req.Header.Set("X-Yao-Signature", signature)
		req.Header.Set("X-Yao-Signature-Algorithm", "HMAC-SHA256")
	}

	httpResp, err := h.httpClient.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("request failed: %v", err)
		return result
	}
	defer httpResp.Body.Close()

	body, _ := io.ReadAll(httpResp.Body)

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		result.Error = fmt.Sprintf("webhook returned status %d: %s", httpResp.StatusCode, string(body))
		return result
	}

	result.Success = true
	result.Details = map[string]interface{}{
		"status_code": httpResp.StatusCode,
		"response":    string(body),
	}
	return result
}

// ============================================================================
// Process
// ============================================================================

func (h *robotHandler) callProcess(
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

	proc, err := process.Of(target.Process, args...)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create process: %v", err)
		return result
	}
	proc.Context = ctx

	if err = proc.Execute(); err != nil {
		result.Error = err.Error()
		return result
	}

	result.Success = true
	result.Details = toJSONSerializable(proc.Value)
	return result
}

// ============================================================================
// Helpers
// ============================================================================

func toJSONSerializable(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	if _, err := json.Marshal(v); err != nil {
		return fmt.Sprintf("%v", v)
	}
	return v
}

func buildEmailSubject(subject, template string, content *robottypes.DeliveryContent, ctx *robottypes.DeliveryContext) string {
	if subject != "" {
		return subject
	}
	if content.Summary != "" {
		return content.Summary
	}
	return fmt.Sprintf("Execution %s Complete", ctx.ExecutionID)
}

func buildEmailBody(template string, content *robottypes.DeliveryContent) (string, string) {
	markdown := content.Body
	if markdown == "" {
		markdown = content.Summary
	}
	html, err := text.MarkdownToHTML(markdown)
	if err != nil {
		return markdown, markdown
	}
	return html, markdown
}

func convertAttachments(ctx context.Context, attachments []robottypes.DeliveryAttachment) []messengerTypes.Attachment {
	if len(attachments) == 0 {
		return nil
	}

	result := make([]messengerTypes.Attachment, 0, len(attachments))
	for _, att := range attachments {
		uploader, fileID, isWrapper := attachment.Parse(att.File)
		if !isWrapper {
			log.Warn("convertAttachments: skipping non-wrapper file value=%q title=%q", att.File, att.Title)
			continue
		}
		manager, ok := attachment.Managers[uploader]
		if !ok {
			log.Warn("convertAttachments: manager not found uploader=%q file=%q title=%q (available: %v)",
				uploader, att.File, att.Title, attachmentManagerKeys())
			continue
		}
		info, err := manager.Info(ctx, fileID)
		if err != nil {
			log.Warn("convertAttachments: failed to get file info fileID=%q uploader=%q: %v", fileID, uploader, err)
			continue
		}
		content, err := manager.Read(ctx, fileID)
		if err != nil {
			log.Warn("convertAttachments: failed to read file fileID=%q uploader=%q: %v", fileID, uploader, err)
			continue
		}

		filename := info.Filename
		if att.Title != "" {
			ext := ""
			if idx := strings.LastIndex(info.Filename, "."); idx >= 0 {
				ext = info.Filename[idx:]
			}
			titleExt := ""
			if idx := strings.LastIndex(att.Title, "."); idx >= 0 {
				titleExt = att.Title[idx:]
			}
			if titleExt != "" {
				filename = att.Title
			} else {
				filename = att.Title + ext
			}
		}

		log.Info("convertAttachments: added attachment filename=%q contentType=%q size=%d", filename, info.ContentType, len(content))
		result = append(result, messengerTypes.Attachment{
			Filename:    filename,
			ContentType: info.ContentType,
			Content:     content,
		})
	}
	return result
}

func attachmentManagerKeys() []string {
	keys := make([]string, 0, len(attachment.Managers))
	for k := range attachment.Managers {
		keys = append(keys, k)
	}
	return keys
}

// ComputeHMACSignature computes HMAC-SHA256 signature for webhook payload.
func ComputeHMACSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyHMACSignature verifies the HMAC-SHA256 signature of a webhook payload.
func VerifyHMACSignature(payload []byte, secret, signature string) bool {
	expected := ComputeHMACSignature(payload, secret)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// splitChannelChatID splits a composite "channel:chatID" string (e.g. "telegram:8134167376")
// into its channel and chatID parts. If no colon is present, channel is empty.
func splitChannelChatID(composite string) (channel, chatID string) {
	if idx := strings.Index(composite, ":"); idx >= 0 {
		return composite[:idx], composite[idx+1:]
	}
	return "", composite
}
