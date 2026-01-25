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

// getAssistants get all assistant directories that have pages (supports nested assistants)
// Returns assistant IDs like: ["expense", "tasks", "tests.nested.demo"]
// Nested paths are joined with "." to form the assistant ID
func (agent *Agent) getAssistants() ([]string, error) {
	if !agent.fs.IsDir(agent.assistantsRoot) {
		return []string{}, nil
	}

	assistants := []string{}
	err := agent.scanAssistantsRecursive(agent.assistantsRoot, "", &assistants)
	if err != nil {
		return nil, err
	}

	return assistants, nil
}

// scanAssistantsRecursive recursively scans directories for assistants with pages
// prefix is the accumulated path prefix (e.g., "tests.nested")
func (agent *Agent) scanAssistantsRecursive(dir string, prefix string, assistants *[]string) error {
	dirs, err := agent.fs.ReadDir(dir, false)
	if err != nil {
		return err
	}

	for _, subdir := range dirs {
		if !agent.fs.IsDir(subdir) {
			continue
		}

		name := filepath.Base(subdir)
		// Skip hidden directories and special directories
		if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "__") {
			continue
		}

		// Build the assistant ID with prefix
		assistantID := name
		if prefix != "" {
			assistantID = prefix + "." + name
		}

		// Check if this directory has a pages subdirectory
		pagesDir := filepath.Join(subdir, "pages")
		if agent.fs.IsDir(pagesDir) {
			*assistants = append(*assistants, assistantID)
		}

		// Recursively scan subdirectories for nested assistants
		// Only scan if there's no pages directory (to avoid scanning inside pages/)
		// or if there are other subdirectories that might contain nested assistants
		if !agent.fs.IsDir(pagesDir) {
			err := agent.scanAssistantsRecursive(subdir, assistantID, assistants)
			if err != nil {
				log.Warn("[Agent] Error scanning subdirectory %s: %v", subdir, err)
				continue
			}
		} else {
			// Even if this has pages, check for nested assistants in other subdirectories
			subdirs, _ := agent.fs.ReadDir(subdir, false)
			for _, nested := range subdirs {
				nestedName := filepath.Base(nested)
				if agent.fs.IsDir(nested) && nestedName != "pages" &&
					!strings.HasPrefix(nestedName, ".") &&
					!strings.HasPrefix(nestedName, "__") {
					err := agent.scanAssistantsRecursive(nested, assistantID+"."+nestedName, assistants)
					if err != nil {
						log.Warn("[Agent] Error scanning nested directory %s: %v", nested, err)
						continue
					}
				}
			}
		}
	}

	return nil
}

// getAssistantPagesRoot get the pages root for an assistant
// assistantID can be "expense" or "tests.nested.demo"
// Returns the actual filesystem path like "/assistants/tests/nested/demo/pages"
func (agent *Agent) getAssistantPagesRoot(assistantID string) string {
	// Convert dot notation to path: "tests.nested.demo" -> "tests/nested/demo"
	pathParts := strings.Split(assistantID, ".")
	assistantPath := filepath.Join(pathParts...)
	return filepath.Join(agent.assistantsRoot, assistantPath, "pages")
}

// Exists check if the agent storage is available
func Exists() bool {
	appFS, err := fs.Get("app")
	if err != nil {
		return false
	}
	return appFS.IsDir("/agent/template")
}

// HasAssistantPages check if any assistant has pages (supports nested assistants)
func HasAssistantPages() bool {
	appFS, err := fs.Get("app")
	if err != nil {
		return false
	}

	if !appFS.IsDir("/assistants") {
		return false
	}

	return hasAssistantPagesRecursive(appFS, "/assistants")
}

// hasAssistantPagesRecursive recursively checks for assistants with pages
func hasAssistantPagesRecursive(appFS fs.FileSystem, dir string) bool {
	dirs, err := appFS.ReadDir(dir, false)
	if err != nil {
		return false
	}

	for _, subdir := range dirs {
		if !appFS.IsDir(subdir) {
			continue
		}

		name := filepath.Base(subdir)
		// Skip hidden directories and special directories
		if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "__") {
			continue
		}

		// Check if this directory has a pages subdirectory
		pagesDir := filepath.Join(subdir, "pages")
		if appFS.IsDir(pagesDir) {
			return true
		}

		// Recursively check subdirectories
		if hasAssistantPagesRecursive(appFS, subdir) {
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
