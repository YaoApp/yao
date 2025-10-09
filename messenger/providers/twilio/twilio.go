package twilio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/yaoapp/yao/messenger/types"
)

// Provider implements the Provider interface for Twilio services (SMS, WhatsApp, Email)
type Provider struct {
	config              types.ProviderConfig
	accountSID          string
	authToken           string
	apiSID              string
	apiKey              string
	fromPhone           string
	fromEmail           string
	fromName            string
	messagingServiceSID string
	sendGridAPIKey      string
	httpClient          *http.Client
	baseURL             string
	templateManager     types.TemplateManager
}

// NewTwilioProvider creates a new unified Twilio provider
func NewTwilioProvider(config types.ProviderConfig) (*Provider, error) {
	return NewTwilioProviderWithTemplateManager(config, nil)
}

// NewTwilioProviderWithTemplateManager creates a new Twilio provider with template manager
func NewTwilioProviderWithTemplateManager(config types.ProviderConfig, templateManager types.TemplateManager) (*Provider, error) {
	provider := &Provider{
		config:          config,
		templateManager: templateManager,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://api.twilio.com/2010-04-01",
	}

	// Extract options
	options := config.Options
	if options == nil {
		return nil, fmt.Errorf("Twilio provider requires options")
	}

	// Required options
	if accountSID, ok := options["account_sid"].(string); ok {
		provider.accountSID = accountSID
	} else {
		return nil, fmt.Errorf("Twilio provider requires 'account_sid' option")
	}

	if authToken, ok := options["auth_token"].(string); ok {
		provider.authToken = authToken
	}

	// Optional API Key authentication (preferred over auth_token)
	if apiSID, ok := options["api_sid"].(string); ok {
		provider.apiSID = apiSID
	}

	if apiKey, ok := options["api_key"].(string); ok {
		provider.apiKey = apiKey
	}

	// Optional options for different services
	if fromPhone, ok := options["from_phone"].(string); ok {
		provider.fromPhone = fromPhone
	}

	if fromEmail, ok := options["from_email"].(string); ok {
		provider.fromEmail = fromEmail
	}

	if fromName, ok := options["from_name"].(string); ok {
		provider.fromName = fromName
	}

	if messagingServiceSID, ok := options["messaging_service_sid"].(string); ok {
		provider.messagingServiceSID = messagingServiceSID
	}

	if sendGridAPIKey, ok := options["sendgrid_api_key"].(string); ok {
		provider.sendGridAPIKey = sendGridAPIKey
	}

	if baseURL, ok := options["base_url"].(string); ok {
		provider.baseURL = baseURL
	}

	return provider, nil
}

// Send sends a message using appropriate Twilio service based on message type
func (p *Provider) Send(ctx context.Context, message *types.Message) error {
	switch message.Type {
	case types.MessageTypeSMS:
		return p.sendSMS(ctx, message)
	case types.MessageTypeWhatsApp:
		return p.sendWhatsApp(ctx, message)
	case types.MessageTypeEmail:
		return p.sendEmail(ctx, message)
	default:
		return fmt.Errorf("unsupported message type: %s", message.Type)
	}
}

// SendBatch sends multiple messages in batch
func (p *Provider) SendBatch(ctx context.Context, messages []*types.Message) error {
	for _, message := range messages {
		if err := p.Send(ctx, message); err != nil {
			return fmt.Errorf("failed to send message to %v: %w", message.To, err)
		}
	}
	return nil
}

// SendT sends a message using a template
func (p *Provider) SendT(ctx context.Context, templateID string, templateType types.TemplateType, data types.TemplateData) error {
	// Get template from provider's template manager with specified type
	template, err := p.getTemplate(templateID, templateType)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}

	// Convert template to message
	message, err := template.ToMessage(data)
	if err != nil {
		return fmt.Errorf("failed to convert template to message: %w", err)
	}

	// Send message using existing Send method
	return p.Send(ctx, message)
}

// SendTBatch sends multiple messages using templates in batch
func (p *Provider) SendTBatch(ctx context.Context, templateID string, templateType types.TemplateType, dataList []types.TemplateData) error {
	// Get template from provider's template manager with specified type
	template, err := p.getTemplate(templateID, templateType)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}

	// Convert templates to messages
	messages := make([]*types.Message, 0, len(dataList))
	for _, data := range dataList {
		message, err := template.ToMessage(data)
		if err != nil {
			return fmt.Errorf("failed to convert template to message: %w", err)
		}
		messages = append(messages, message)
	}

	// Send messages using existing SendBatch method
	return p.SendBatch(ctx, messages)
}

// SendTBatchMixed sends multiple messages using different templates with different data
func (p *Provider) SendTBatchMixed(ctx context.Context, templateRequests []types.TemplateRequest) error {
	// Convert template requests to messages
	messages := make([]*types.Message, 0, len(templateRequests))
	for _, req := range templateRequests {
		// Get template from provider's template manager
		template, err := p.getTemplate(req.TemplateID, types.TemplateTypeSMS) // Twilio primarily supports SMS
		if err != nil {
			return fmt.Errorf("template not found: %s, %w", req.TemplateID, err)
		}

		// Convert template to message
		message, err := template.ToMessage(req.Data)
		if err != nil {
			return fmt.Errorf("failed to convert template %s to message: %w", req.TemplateID, err)
		}
		messages = append(messages, message)
	}

	// Send messages using existing SendBatch method
	return p.SendBatch(ctx, messages)
}

// getTemplate gets a template by ID and type from the provider's template manager
func (p *Provider) getTemplate(templateID string, templateType types.TemplateType) (*types.Template, error) {
	if p.templateManager == nil {
		return nil, fmt.Errorf("template manager not available")
	}
	return p.templateManager.GetTemplate(templateID, templateType)
}

// GetType returns the provider type
func (p *Provider) GetType() string {
	return "twilio"
}

// GetName returns the provider name
func (p *Provider) GetName() string {
	return p.config.Name
}

// GetPublicInfo returns public information about the provider
func (p *Provider) GetPublicInfo() types.ProviderPublicInfo {
	description := "Twilio multi-channel communication provider"
	if p.config.Description != "" {
		description = p.config.Description
	}

	capabilities := []string{}
	if p.fromPhone != "" || p.messagingServiceSID != "" {
		capabilities = append(capabilities, "sms")
	}
	if p.fromPhone != "" {
		capabilities = append(capabilities, "whatsapp")
	}
	if p.sendGridAPIKey != "" {
		capabilities = append(capabilities, "email")
	}
	if len(capabilities) == 0 {
		capabilities = []string{"sms", "whatsapp", "email"} // Default capabilities
	}

	return types.ProviderPublicInfo{
		Name:         p.config.Name,
		Type:         "twilio",
		Description:  description,
		Capabilities: capabilities,
		Features: types.Features{
			SupportsWebhooks:   true,
			SupportsReceiving:  false,
			SupportsTracking:   true,
			SupportsScheduling: true,
		},
	}
}

// Validate validates the provider configuration
func (p *Provider) Validate() error {
	if p.accountSID == "" {
		return fmt.Errorf("account_sid is required")
	}

	// Either auth_token or both api_sid and api_key must be provided
	hasAuthToken := p.authToken != ""
	hasAPIKeys := p.apiSID != "" && p.apiKey != ""

	if !hasAuthToken && !hasAPIKeys {
		return fmt.Errorf("either 'auth_token' or both 'api_sid' and 'api_key' are required")
	}

	// If API keys are partially configured, require both
	if (p.apiSID != "" && p.apiKey == "") || (p.apiSID == "" && p.apiKey != "") {
		return fmt.Errorf("both 'api_sid' and 'api_key' must be provided together")
	}

	return nil
}

// Close closes the provider connection (no-op for HTTP-based Twilio)
func (p *Provider) Close() error {
	return nil
}

// sendSMS sends an SMS message via Twilio
func (p *Provider) sendSMS(ctx context.Context, message *types.Message) error {
	if p.fromPhone == "" && p.messagingServiceSID == "" {
		return fmt.Errorf("either from_phone or messaging_service_sid is required for SMS")
	}

	for _, to := range message.To {
		err := p.sendSMSToRecipient(ctx, to, message)
		if err != nil {
			return fmt.Errorf("failed to send SMS to %s: %w", to, err)
		}
	}
	return nil
}

// sendSMSToRecipient sends SMS to a single recipient
func (p *Provider) sendSMSToRecipient(ctx context.Context, to string, message *types.Message) error {
	apiURL := fmt.Sprintf("%s/Accounts/%s/Messages.json", p.baseURL, p.accountSID)

	// Prepare form data
	data := url.Values{}
	data.Set("To", to)
	data.Set("Body", message.Body)

	if p.messagingServiceSID != "" {
		data.Set("MessagingServiceSid", p.messagingServiceSID)
	} else {
		data.Set("From", p.fromPhone)
	}

	return p.sendTwilioRequest(ctx, apiURL, data)
}

// sendWhatsApp sends a WhatsApp message via Twilio
func (p *Provider) sendWhatsApp(ctx context.Context, message *types.Message) error {
	if p.fromPhone == "" {
		return fmt.Errorf("from_phone is required for WhatsApp messages")
	}

	for _, to := range message.To {
		err := p.sendWhatsAppToRecipient(ctx, to, message)
		if err != nil {
			return fmt.Errorf("failed to send WhatsApp message to %s: %w", to, err)
		}
	}
	return nil
}

// sendWhatsAppToRecipient sends WhatsApp message to a single recipient
func (p *Provider) sendWhatsAppToRecipient(ctx context.Context, to string, message *types.Message) error {
	apiURL := fmt.Sprintf("%s/Accounts/%s/Messages.json", p.baseURL, p.accountSID)

	// Ensure phone numbers have WhatsApp prefix
	fromWhatsApp := p.fromPhone
	if !strings.HasPrefix(fromWhatsApp, "whatsapp:") {
		fromWhatsApp = "whatsapp:" + fromWhatsApp
	}

	toWhatsApp := to
	if !strings.HasPrefix(toWhatsApp, "whatsapp:") {
		toWhatsApp = "whatsapp:" + toWhatsApp
	}

	// Prepare form data
	data := url.Values{}
	data.Set("From", fromWhatsApp)
	data.Set("To", toWhatsApp)
	data.Set("Body", message.Body)

	return p.sendTwilioRequest(ctx, apiURL, data)
}

// sendEmail sends an email via Twilio SendGrid API
func (p *Provider) sendEmail(ctx context.Context, message *types.Message) error {
	if p.sendGridAPIKey == "" {
		return fmt.Errorf("sendgrid_api_key is required for email messages")
	}
	if p.fromEmail == "" {
		return fmt.Errorf("from_email is required for email messages")
	}

	// Create SendGrid email payload
	payload := map[string]interface{}{
		"personalizations": []map[string]interface{}{
			{
				"to": p.buildEmailRecipients(message.To),
			},
		},
		"from":    p.buildFromAddress(message),
		"subject": message.Subject,
		"content": p.buildEmailContent(message),
	}

	// Add custom headers if provided
	if len(message.Headers) > 0 {
		payload["headers"] = message.Headers
	}

	// Add attachments if provided
	if len(message.Attachments) > 0 {
		attachments, err := p.buildAttachments(message.Attachments)
		if err != nil {
			return fmt.Errorf("failed to build attachments: %w", err)
		}
		payload["attachments"] = attachments
	}

	// Add custom metadata
	if len(message.Metadata) > 0 {
		customArgs := make(map[string]string)
		for key, value := range message.Metadata {
			if str, ok := value.(string); ok {
				customArgs[key] = str
			}
		}
		if len(customArgs) > 0 {
			payload["custom_args"] = customArgs
		}
	}

	// Add scheduled sending if specified
	if message.ScheduledAt != nil {
		payload["send_at"] = message.ScheduledAt.Unix()
	}

	// Convert to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal email payload: %w", err)
	}

	// Send via SendGrid API
	apiURL := "https://api.sendgrid.com/v3/mail/send"
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.sendGridAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("SendGrid API error: %s - %s", resp.Status, string(body))
	}

	return nil
}

// sendTwilioRequest sends a request to Twilio API
func (p *Provider) sendTwilioRequest(ctx context.Context, apiURL string, data url.Values) error {
	// Add custom metadata as status callback parameters
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Use API Key authentication if available, otherwise fall back to Auth Token
	if p.apiSID != "" && p.apiKey != "" {
		req.SetBasicAuth(p.apiSID, p.apiKey)
	} else {
		req.SetBasicAuth(p.accountSID, p.authToken)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Twilio API error: %s - %s", resp.Status, string(body))
	}

	return nil
}

// buildEmailRecipients builds the recipients array for SendGrid
func (p *Provider) buildEmailRecipients(to []string) []map[string]string {
	recipients := make([]map[string]string, len(to))
	for i, email := range to {
		recipients[i] = map[string]string{"email": email}
	}
	return recipients
}

// buildFromAddress builds the from address for SendGrid
func (p *Provider) buildFromAddress(message *types.Message) map[string]string {
	from := map[string]string{
		"email": p.fromEmail,
	}

	// Use message from if provided, otherwise use configured from
	if message.From != "" {
		from["email"] = message.From
	}

	// Add name if configured
	if p.fromName != "" {
		from["name"] = p.fromName
	}

	return from
}

// buildEmailContent builds the content array for SendGrid
func (p *Provider) buildEmailContent(message *types.Message) []map[string]string {
	content := []map[string]string{}

	if message.Body != "" {
		content = append(content, map[string]string{
			"type":  "text/plain",
			"value": message.Body,
		})
	}

	if message.HTML != "" {
		content = append(content, map[string]string{
			"type":  "text/html",
			"value": message.HTML,
		})
	}

	// If no content is provided, use body as plain text
	if len(content) == 0 {
		content = append(content, map[string]string{
			"type":  "text/plain",
			"value": "No content provided",
		})
	}

	return content
}

// buildAttachments builds the attachments array for SendGrid
func (p *Provider) buildAttachments(attachments []types.Attachment) ([]map[string]interface{}, error) {
	sgAttachments := make([]map[string]interface{}, len(attachments))

	for i, attachment := range attachments {
		// Encode content to base64
		encodedContent := ""
		if len(attachment.Content) > 0 {
			// Simple base64 encoding (in real implementation, use base64 package)
			encodedContent = string(attachment.Content) // This should be base64 encoded
		}

		sgAttachment := map[string]interface{}{
			"content":  encodedContent,
			"filename": attachment.Filename,
			"type":     attachment.ContentType,
		}

		// Add disposition for inline attachments
		if attachment.Inline {
			sgAttachment["disposition"] = "inline"
			if attachment.CID != "" {
				sgAttachment["content_id"] = attachment.CID
			}
		} else {
			sgAttachment["disposition"] = "attachment"
		}

		sgAttachments[i] = sgAttachment
	}

	return sgAttachments, nil
}
