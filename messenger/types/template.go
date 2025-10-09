package types

import (
	"fmt"
	"regexp"
	"strings"
)

// TemplateType represents the type of template (mail, sms, whatsapp)
type TemplateType string

const (
	TemplateTypeMail     TemplateType = "mail"
	TemplateTypeSMS      TemplateType = "sms"
	TemplateTypeWhatsApp TemplateType = "whatsapp"
)

// templateTypeToMessageType converts TemplateType to MessageType
func templateTypeToMessageType(templateType TemplateType) MessageType {
	switch templateType {
	case TemplateTypeMail:
		return MessageTypeEmail
	case TemplateTypeSMS:
		return MessageTypeSMS
	case TemplateTypeWhatsApp:
		return MessageTypeWhatsApp
	default:
		return ""
	}
}

// Template represents a message template
type Template struct {
	ID       string       `json:"id"`
	Type     TemplateType `json:"type"`
	Language string       `json:"language"`
	Subject  string       `json:"subject,omitempty"`
	Body     string       `json:"body"`
	HTML     string       `json:"html,omitempty"`
}

// TemplateData represents data to be used in template rendering
type TemplateData map[string]interface{}

// Render renders the template with the provided data using simple string replacement
func (t *Template) Render(data TemplateData) (subject, body, html string, err error) {
	// Render subject if available
	if t.Subject != "" {
		subject = renderTemplate(t.Subject, data)
	}

	// Render body
	if t.Body != "" {
		body = renderTemplate(t.Body, data)
	}

	// Render HTML if available
	if t.HTML != "" {
		html = renderTemplate(t.HTML, data)
	}

	return subject, body, html, nil
}

// ToMessage converts template to Message with provided data
func (t *Template) ToMessage(data TemplateData) (*Message, error) {
	// Render template
	subject, body, html, err := t.Render(data)
	if err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	// Get recipients from data
	var recipients []string
	if toData, exists := data["to"]; exists {
		switch v := toData.(type) {
		case []string:
			recipients = v
		case string:
			recipients = []string{v}
		default:
			return nil, fmt.Errorf("'to' field must be string or []string")
		}
	} else {
		return nil, fmt.Errorf("template data must include 'to' field with recipients")
	}

	// Convert TemplateType to MessageType
	messageType := templateTypeToMessageType(t.Type)

	// Create message
	message := &Message{
		Type:    messageType,
		Subject: subject,
		Body:    body,
		HTML:    html,
		To:      recipients,
	}

	// Add optional fields from data
	if from, exists := data["from"]; exists {
		if fromStr, ok := from.(string); ok {
			message.From = fromStr
		}
	}

	return message, nil
}

// renderTemplate renders a template string with data using {{ }} syntax
func renderTemplate(template string, data TemplateData) string {
	// Find all {{ variable }} patterns
	re := regexp.MustCompile(`\{\{\s*([^}]+)\s*\}\}`)

	return re.ReplaceAllStringFunc(template, func(match string) string {
		// Extract variable name from {{ variable }}
		variable := strings.TrimSpace(strings.Trim(match, "{}"))

		// Get value from data using dot notation for nested access
		value := getNestedValue(data, variable)

		// Convert to string
		return fmt.Sprintf("%v", value)
	})
}

// getNestedValue gets a value from data using dot notation (e.g., "user.name", "team.members.count")
func getNestedValue(data TemplateData, key string) interface{} {
	parts := strings.Split(key, ".")

	current := interface{}(data)
	for _, part := range parts {
		part = strings.TrimSpace(part)

		switch v := current.(type) {
		case map[string]interface{}:
			if val, exists := v[part]; exists {
				current = val
			} else {
				return "" // Return empty string if key not found
			}
		case TemplateData:
			if val, exists := v[part]; exists {
				current = val
			} else {
				return "" // Return empty string if key not found
			}
		default:
			return "" // Return empty string if not a map
		}
	}

	return current
}

// TemplateManager manages message templates
type TemplateManager interface {
	// GetTemplate returns a template by ID and type
	GetTemplate(templateID string, templateType TemplateType) (*Template, error)

	// GetAllTemplates returns all loaded templates
	GetAllTemplates() map[string]map[TemplateType]*Template
}
