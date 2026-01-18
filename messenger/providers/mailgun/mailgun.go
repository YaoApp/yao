package mailgun

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"time"

	"github.com/yaoapp/yao/messenger/types"
)

// Provider implements the Provider interface for Mailgun email sending
type Provider struct {
	config          types.ProviderConfig
	domain          string
	apiKey          string
	from            string
	baseURL         string
	httpClient      *http.Client
	templateManager types.TemplateManager
}

// NewMailgunProvider creates a new Mailgun provider
func NewMailgunProvider(config types.ProviderConfig) (*Provider, error) {
	return NewMailgunProviderWithTemplateManager(config, nil)
}

// NewMailgunProviderWithTemplateManager creates a new Mailgun provider with template manager
func NewMailgunProviderWithTemplateManager(config types.ProviderConfig, templateManager types.TemplateManager) (*Provider, error) {
	provider := &Provider{
		config:          config,
		templateManager: templateManager,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Extract options
	options := config.Options
	if options == nil {
		return nil, fmt.Errorf("Mailgun provider requires options")
	}

	// Required options
	if domain, ok := options["domain"].(string); ok {
		provider.domain = domain
	} else {
		return nil, fmt.Errorf("Mailgun provider requires 'domain' option")
	}

	if apiKey, ok := options["api_key"].(string); ok {
		provider.apiKey = apiKey
	} else {
		return nil, fmt.Errorf("Mailgun provider requires 'api_key' option")
	}

	if from, ok := options["from"].(string); ok {
		provider.from = from
	} else {
		return nil, fmt.Errorf("Mailgun provider requires 'from' option")
	}

	// Optional options
	if baseURL, ok := options["base_url"].(string); ok {
		provider.baseURL = baseURL
	} else {
		// Default to US region
		provider.baseURL = "https://api.mailgun.net/v3"
	}

	return provider, nil
}

// Send sends a message using Mailgun
func (p *Provider) Send(ctx context.Context, message *types.Message) error {
	if message.Type != types.MessageTypeEmail {
		return fmt.Errorf("Mailgun provider only supports email messages")
	}

	return p.sendEmail(ctx, message)
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
		template, err := p.getTemplate(req.TemplateID, types.TemplateTypeMail) // Mailgun supports email
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
	return "mailgun"
}

// GetName returns the provider name
func (p *Provider) GetName() string {
	return p.config.Name
}

// GetPublicInfo returns public information about the provider
func (p *Provider) GetPublicInfo() types.ProviderPublicInfo {
	description := "Mailgun email service provider"
	if p.config.Description != "" {
		description = p.config.Description
	}

	return types.ProviderPublicInfo{
		Name:         p.config.Name,
		Type:         "mailgun",
		Description:  description,
		Capabilities: []string{"email", "webhooks", "tracking"},
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
	if p.domain == "" {
		return fmt.Errorf("domain is required")
	}
	if p.apiKey == "" {
		return fmt.Errorf("api_key is required")
	}
	if p.from == "" {
		return fmt.Errorf("from address is required")
	}
	return nil
}

// Close closes the provider connection (no-op for HTTP-based Mailgun)
func (p *Provider) Close() error {
	return nil
}

// sendEmail sends an email via Mailgun API
func (p *Provider) sendEmail(ctx context.Context, message *types.Message) error {
	apiURL := fmt.Sprintf("%s/%s/messages", p.baseURL, p.domain)

	// Check if we have attachments - use multipart/form-data if so
	if len(message.Attachments) > 0 {
		return p.sendEmailWithAttachments(ctx, apiURL, message)
	}

	// No attachments - use simple URL-encoded form
	return p.sendEmailSimple(ctx, apiURL, message)
}

// sendEmailSimple sends email without attachments using URL-encoded form
func (p *Provider) sendEmailSimple(ctx context.Context, apiURL string, message *types.Message) error {
	// Prepare form data
	data := url.Values{}

	// From address
	from := message.From
	if from == "" {
		from = p.from
	}
	data.Set("from", from)

	// To addresses
	for _, to := range message.To {
		data.Add("to", to)
	}

	// Subject and content
	data.Set("subject", message.Subject)

	if message.Body != "" {
		data.Set("text", message.Body)
	}

	if message.HTML != "" {
		data.Set("html", message.HTML)
	}

	// Custom headers
	if message.Headers != nil {
		for key, value := range message.Headers {
			data.Set("h:"+key, value)
		}
	}

	// Custom variables (metadata)
	if message.Metadata != nil {
		for key, value := range message.Metadata {
			if str, ok := value.(string); ok {
				data.Set("v:"+key, str)
			}
		}
	}

	// Priority
	if message.Priority > 0 {
		data.Set("o:priority", fmt.Sprintf("%d", message.Priority))
	}

	// Scheduled sending
	if message.ScheduledAt != nil {
		data.Set("o:deliverytime", message.ScheduledAt.Format(time.RFC1123Z))
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set authentication
	req.SetBasicAuth("api", p.apiKey)
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
		return fmt.Errorf("Mailgun API error: %s - %s", resp.Status, string(body))
	}

	return nil
}

// sendEmailWithAttachments sends email with attachments using multipart/form-data
func (p *Provider) sendEmailWithAttachments(ctx context.Context, apiURL string, message *types.Message) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// From address
	from := message.From
	if from == "" {
		from = p.from
	}
	if err := writer.WriteField("from", from); err != nil {
		return fmt.Errorf("failed to write from field: %w", err)
	}

	// To addresses
	for _, to := range message.To {
		if err := writer.WriteField("to", to); err != nil {
			return fmt.Errorf("failed to write to field: %w", err)
		}
	}

	// Subject
	if err := writer.WriteField("subject", message.Subject); err != nil {
		return fmt.Errorf("failed to write subject field: %w", err)
	}

	// Text body
	if message.Body != "" {
		if err := writer.WriteField("text", message.Body); err != nil {
			return fmt.Errorf("failed to write text field: %w", err)
		}
	}

	// HTML body
	if message.HTML != "" {
		if err := writer.WriteField("html", message.HTML); err != nil {
			return fmt.Errorf("failed to write html field: %w", err)
		}
	}

	// Custom headers
	if message.Headers != nil {
		for key, value := range message.Headers {
			if err := writer.WriteField("h:"+key, value); err != nil {
				return fmt.Errorf("failed to write header field: %w", err)
			}
		}
	}

	// Custom variables (metadata)
	if message.Metadata != nil {
		for key, value := range message.Metadata {
			if str, ok := value.(string); ok {
				if err := writer.WriteField("v:"+key, str); err != nil {
					return fmt.Errorf("failed to write metadata field: %w", err)
				}
			}
		}
	}

	// Priority
	if message.Priority > 0 {
		if err := writer.WriteField("o:priority", fmt.Sprintf("%d", message.Priority)); err != nil {
			return fmt.Errorf("failed to write priority field: %w", err)
		}
	}

	// Scheduled sending
	if message.ScheduledAt != nil {
		if err := writer.WriteField("o:deliverytime", message.ScheduledAt.Format(time.RFC1123Z)); err != nil {
			return fmt.Errorf("failed to write deliverytime field: %w", err)
		}
	}

	// Add attachments
	for _, attachment := range message.Attachments {
		fieldName := "attachment"
		if attachment.Inline {
			fieldName = "inline"
		}

		// Create form file with proper headers
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, attachment.Filename))
		if attachment.ContentType != "" {
			h.Set("Content-Type", attachment.ContentType)
		} else {
			h.Set("Content-Type", "application/octet-stream")
		}

		part, err := writer.CreatePart(h)
		if err != nil {
			return fmt.Errorf("failed to create attachment part: %w", err)
		}

		if _, err := part.Write(attachment.Content); err != nil {
			return fmt.Errorf("failed to write attachment content: %w", err)
		}
	}

	// Close multipart writer
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, &body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set authentication and content type
	req.SetBasicAuth("api", p.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Mailgun API error: %s - %s", resp.Status, string(respBody))
	}

	return nil
}
