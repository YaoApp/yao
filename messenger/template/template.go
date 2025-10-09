package template

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/messenger/types"
)

// Manager manages message templates
type Manager struct {
	templates map[string]map[types.TemplateType]*types.Template // [templateID][type] -> template
	mutex     sync.RWMutex
}

// Global template manager instance
var Global *Manager = &Manager{
	templates: make(map[string]map[types.TemplateType]*types.Template),
}

// GetTemplate returns a template by ID and type
func (m *Manager) GetTemplate(templateID string, templateType types.TemplateType) (*types.Template, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if templates, exists := m.templates[templateID]; exists {
		if template, typeExists := templates[templateType]; typeExists {
			return template, nil
		}
	}
	return nil, fmt.Errorf("template not found: %s.%s", templateID, templateType)
}

// GetAllTemplates returns all loaded templates
func (m *Manager) GetAllTemplates() map[string]map[types.TemplateType]*types.Template {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Return a copy to prevent external modifications
	result := make(map[string]map[types.TemplateType]*types.Template)
	for id, templates := range m.templates {
		result[id] = make(map[types.TemplateType]*types.Template)
		for templateType, template := range templates {
			result[id][templateType] = template
		}
	}
	return result
}

// GetAvailableTypes returns all available template types for a given templateID
// Returns types in a consistent order: mail, sms, whatsapp
func (m *Manager) GetAvailableTypes(templateID string) []types.TemplateType {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if templates, exists := m.templates[templateID]; exists {
		// Return in preferred order
		var result []types.TemplateType
		preferredOrder := []types.TemplateType{
			types.TemplateTypeMail,
			types.TemplateTypeSMS,
			types.TemplateTypeWhatsApp,
		}

		for _, templateType := range preferredOrder {
			if _, exists := templates[templateType]; exists {
				result = append(result, templateType)
			}
		}
		return result
	}
	return []types.TemplateType{}
}

// ReloadTemplates reloads all templates from disk
func (m *Manager) ReloadTemplates() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Clear existing templates
	m.templates = make(map[string]map[types.TemplateType]*types.Template)

	// Load templates from disk
	return loadTemplates(m)
}

// loadTemplates loads all templates from the templates directory
func loadTemplates(m *Manager) error {
	// Check if templates directory exists
	templatesPath := "messengers/templates"
	exists, err := application.App.Exists(templatesPath)
	if err != nil {
		log.Error("[Template] Error checking templates directory: %v", err)
		return err
	}
	if !exists {
		log.Warn("[Template] templates directory not found, skip loading templates")
		return nil
	}
	log.Info("[Template] Templates directory exists, starting to load templates")

	// Walk through template files
	// Pattern: {name}.{type}.html and {name}.{type}.txt
	exts := []string{"*.mail.html", "*.sms.txt", "*.whatsapp.html"}
	log.Info("[Template] Starting to walk templates directory with extensions: %v", exts)
	err = application.App.Walk(templatesPath, func(root, file string, isdir bool) error {
		log.Info("[Template] Walk callback: root=%s, file=%s, isdir=%v", root, file, isdir)
		if isdir {
			return nil
		}

		log.Info("[Template] Processing file: %s", file)
		// Generate template ID manually to avoid share.ID's dot-to-underscore conversion
		// Format: {language}.{name} (e.g., "en.invite_member")
		relativePath := strings.TrimPrefix(file, root+"/")
		pathParts := strings.Split(relativePath, "/")
		language := pathParts[0]
		filename := pathParts[len(pathParts)-1]
		baseName := strings.TrimSuffix(filename, filepath.Ext(filename))
		// Remove type suffix (e.g., "invite_member.mail" -> "invite_member")
		templateName := strings.Split(baseName, ".")[0]
		templateID := fmt.Sprintf("%s.%s", language, templateName)

		log.Info("[Template] Generated templateID: %s for file: %s", templateID, file)
		template, err := loadTemplate(file, templateID)
		if err != nil {
			log.Warn("[Template] Failed to load template %s: %v", file, err)
			return nil // Continue loading other templates
		}

		if template != nil {
			log.Info("[Template] Loaded template: %s.%s", template.ID, template.Type)
			// Initialize template map for this ID if it doesn't exist
			if m.templates[template.ID] == nil {
				m.templates[template.ID] = make(map[types.TemplateType]*types.Template)
			}
			m.templates[template.ID][template.Type] = template
		}
		return nil
	}, exts...)

	if err != nil {
		return err
	}

	log.Info("[Template] Loaded %d templates", len(m.templates))
	return nil
}

// loadTemplate loads a single template file
func loadTemplate(file string, templateID string) (*types.Template, error) {
	raw, err := application.App.Read(file)
	if err != nil {
		return nil, err
	}

	// Extract filename from file path to determine template type
	filename := filepath.Base(file)
	baseName := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Parse template type from filename
	// Format: {name}.{type}.{ext} -> {type}
	templateType, _ := parseTemplateType(baseName)

	// Use the provided templateID (already in format: language.name)
	fullTemplateID := templateID

	// Extract language from templateID (format: language.name)
	parts := strings.Split(templateID, ".")
	language := parts[0]

	// Determine template type
	var msgType types.TemplateType
	switch templateType {
	case "mail":
		msgType = types.TemplateTypeMail
	case "sms":
		msgType = types.TemplateTypeSMS
	case "whatsapp":
		msgType = types.TemplateTypeWhatsApp
	default:
		return nil, fmt.Errorf("unsupported template type: %s", templateType)
	}

	// Parse template content
	subject, body, html, err := parseTemplateContent(string(raw), msgType)
	if err != nil {
		return nil, err
	}

	// No need to compile templates - we'll use simple string replacement

	return &types.Template{
		ID:       fullTemplateID,
		Type:     msgType,
		Language: language,
		Subject:  subject,
		Body:     body,
		HTML:     html,
	}, nil
}

// parseTemplateType parses template type from filename
// Example: "invite_member.mail" -> "mail", "invite_member"
func parseTemplateType(filename string) (templateType, templateName string) {
	parts := strings.Split(filename, ".")
	if len(parts) < 2 {
		return "", filename
	}

	// Last part is the type
	templateType = parts[len(parts)-1]

	// Everything before the last part is the name
	templateName = strings.Join(parts[:len(parts)-1], ".")

	return templateType, templateName
}

// parseTemplateContent parses template content based on type
func parseTemplateContent(content string, templateType types.TemplateType) (subject, body, html string, err error) {
	content = strings.TrimSpace(content)

	switch templateType {
	case types.TemplateTypeMail:
		return parseMailTemplate(content)
	case types.TemplateTypeSMS:
		return parseSMSTemplate(content)
	case types.TemplateTypeWhatsApp:
		return parseWhatsAppTemplate(content)
	default:
		return "", "", "", fmt.Errorf("unsupported template type: %s", templateType)
	}
}

// parseMailTemplate parses mail template with HTML structure using goquery
func parseMailTemplate(content string) (subject, body, html string, err error) {
	// Parse HTML content with goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return "", "", "", fmt.Errorf("failed to parse mail template HTML: %w", err)
	}

	// Extract subject from <Subject> or <subject> tag (HTML parsers normalize to lowercase)
	subject = strings.TrimSpace(doc.Find("subject").Text())

	// Extract body content from <content> tag (HTML parsers normalize to lowercase)
	bodySelection := doc.Find("content")
	if bodySelection.Length() == 0 {
		return "", "", "", fmt.Errorf("no <content> tag found in mail template")
	}

	// Get the HTML content of the Content tag
	body, err = bodySelection.Html()
	if err != nil {
		return "", "", "", fmt.Errorf("failed to extract content HTML: %w", err)
	}

	body = strings.TrimSpace(body)

	// For mail templates, body is HTML content
	html = body

	return subject, body, html, nil
}

// parseSMSTemplate parses SMS template (plain text)
func parseSMSTemplate(content string) (subject, body, html string, err error) {
	// SMS templates are just plain text
	body = content
	return "", body, "", nil
}

// parseWhatsAppTemplate parses WhatsApp template (similar to mail)
func parseWhatsAppTemplate(content string) (subject, body, html string, err error) {
	// WhatsApp templates use similar structure to mail
	return parseMailTemplate(content)
}

// LoadTemplates loads all templates during messenger initialization
func LoadTemplates() error {
	return Global.ReloadTemplates()
}
