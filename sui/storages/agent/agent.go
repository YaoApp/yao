package agent

import (
	"fmt"
	"path/filepath"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/sui/core"
)

// New create a new agent sui storage
func New(dsl *core.DSL) (*Agent, error) {
	// Use "app" filesystem which is rooted at application source directory
	appFS, err := fs.Get("app")
	if err != nil {
		return nil, err
	}

	// Set default public settings for agent
	if dsl.Public == nil {
		dsl.Public = &core.Public{}
	}

	if dsl.Public.Root == "" {
		dsl.Public.Root = "/agents"
	}

	if dsl.Public.Host == "" {
		dsl.Public.Host = "/"
	}

	if dsl.Public.Index == "" {
		dsl.Public.Index = "/index"
	}

	return &Agent{
		root:           "/agent/template",
		assistantsRoot: "/assistants",
		fs:             appFS,
		DSL:            dsl,
	}, nil
}

// GetTemplates get the templates (returns single agent template)
func (agent *Agent) GetTemplates() ([]core.ITemplate, error) {
	tmpl, err := agent.GetTemplate("agent")
	if err != nil {
		return nil, err
	}
	return []core.ITemplate{tmpl}, nil
}

// GetTemplate get the template
func (agent *Agent) GetTemplate(id string) (core.ITemplate, error) {
	if id != "agent" {
		return nil, fmt.Errorf("Agent storage only supports 'agent' template, got: %s", id)
	}

	// Check if /agent directory exists
	if !agent.fs.IsDir(agent.root) {
		return nil, fmt.Errorf("Agent template directory not found: %s", agent.root)
	}

	// Create agent template
	tmpl := &Template{
		Root:  agent.root,
		agent: agent,
		Template: &core.Template{
			ID:          "agent",
			Name:        "Agent",
			Version:     1,
			Screenshots: []string{},
			Themes:      []core.SelectOption{},
		},
	}

	// Load template.json if exists
	configFile := filepath.Join(agent.root, "template.json")
	if agent.fs.IsFile(configFile) {
		configBytes, err := agent.fs.ReadFile(configFile)
		if err != nil {
			return nil, err
		}
		err = jsoniter.Unmarshal(configBytes, tmpl.Template)
		if err != nil {
			return nil, err
		}
	}

	// Load __document.html
	documentFile := filepath.Join(agent.root, "__document.html")
	if agent.fs.IsFile(documentFile) {
		documentBytes, err := agent.fs.ReadFile(documentFile)
		if err != nil {
			return nil, err
		}
		tmpl.Document = documentBytes
	}

	// Load __data.json
	dataFile := filepath.Join(agent.root, "__data.json")
	if agent.fs.IsFile(dataFile) {
		dataBytes, err := agent.fs.ReadFile(dataFile)
		if err != nil {
			return nil, err
		}
		tmpl.GlobalData = dataBytes
	}

	// Load build script
	err := tmpl.loadBuildScript()
	if err != nil {
		log.Warn("[Agent] Failed to load build script: %v", err)
	}

	return tmpl, nil
}

// UploadTemplate upload the template (not supported for agent)
func (agent *Agent) UploadTemplate(src string, dst string) (core.ITemplate, error) {
	return nil, fmt.Errorf("UploadTemplate is not supported for agent storage")
}

// PublicRootMatcher get the public root matcher
func (agent *Agent) PublicRootMatcher() *core.Matcher {
	return &core.Matcher{Exact: agent.DSL.Public.Root}
}

// Setting get the setting
func (agent *Agent) Setting() (*core.Setting, error) {
	return &core.Setting{
		ID:    agent.DSL.ID,
		Guard: agent.DSL.Guard,
		Option: map[string]interface{}{
			"disableCodeEditor": true,
		},
	}, nil
}

// PublicRoot get the public root
func (agent *Agent) PublicRoot(data map[string]interface{}) (string, error) {
	return agent.DSL.Public.Root, nil
}

// WithSid set the session id
func (agent *Agent) WithSid(sid string) {
	agent.DSL.Sid = sid
}

// getAssistants get all assistant directories that have pages
func (agent *Agent) getAssistants() ([]string, error) {
	if !agent.fs.IsDir(agent.assistantsRoot) {
		return []string{}, nil
	}

	dirs, err := agent.fs.ReadDir(agent.assistantsRoot, false)
	if err != nil {
		return nil, err
	}

	assistants := []string{}
	for _, dir := range dirs {
		if !agent.fs.IsDir(dir) {
			continue
		}

		// Check if this assistant has a pages directory
		pagesDir := filepath.Join(dir, "pages")
		if agent.fs.IsDir(pagesDir) {
			name := filepath.Base(dir)
			assistants = append(assistants, name)
		}
	}

	return assistants, nil
}

// getAssistantPagesRoot get the pages root for an assistant
func (agent *Agent) getAssistantPagesRoot(assistantID string) string {
	return filepath.Join(agent.assistantsRoot, assistantID, "pages")
}

// Exists check if the agent storage is available
func Exists() bool {
	appFS, err := fs.Get("app")
	if err != nil {
		return false
	}
	return appFS.IsDir("/agent/template")
}

// HasAssistantPages check if any assistant has pages
func HasAssistantPages() bool {
	appFS, err := fs.Get("app")
	if err != nil {
		return false
	}

	if !appFS.IsDir("/assistants") {
		return false
	}

	dirs, err := appFS.ReadDir("/assistants", false)
	if err != nil {
		return false
	}

	for _, dir := range dirs {
		if !appFS.IsDir(dir) {
			continue
		}
		pagesDir := filepath.Join(dir, "pages")
		if appFS.IsDir(pagesDir) {
			return true
		}
	}

	return false
}

// log helper
func init() {
	_ = log.Debug
	_ = strings.TrimPrefix
}
