package types

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/fs"
	"gopkg.in/yaml.v3"
)

// Prompts is a slice of Prompt with helper methods
type Prompts []Prompt

// SystemVariables defines the available system variables
// These are computed at parse time
var SystemVariables = map[string]func() string{
	"TIME":     func() string { return time.Now().Format("15:04:05") },
	"DATE":     func() string { return time.Now().Format("2006-01-02") },
	"DATETIME": func() string { return time.Now().Format("2006-01-02 15:04:05") },
	"TIMEZONE": func() string { return time.Now().Location().String() },
	"WEEKDAY":  func() string { return time.Now().Weekday().String() },
	"YEAR":     func() string { return time.Now().Format("2006") },
	"MONTH":    func() string { return time.Now().Format("01") },
	"DAY":      func() string { return time.Now().Format("02") },
	"HOUR":     func() string { return time.Now().Format("15") },
	"MINUTE":   func() string { return time.Now().Format("04") },
	"SECOND":   func() string { return time.Now().Format("05") },
	"UNIX":     func() string { return time.Now().Format("1136239445") }, // Unix timestamp
}

// Regular expressions for variable replacement
var (
	reSysVar   = regexp.MustCompile(`\$SYS\.([A-Z_]+)`)
	reEnvVar   = regexp.MustCompile(`\$ENV\.([A-Za-z_][A-Za-z0-9_]*)`)
	reCtxVar   = regexp.MustCompile(`\$CTX\.([A-Za-z_][A-Za-z0-9_]*)`)
	reAssetRef = regexp.MustCompile(`@assets/([^\s]+\.(md|yml|yaml|json|txt))`)
)

// LoadPrompts loads prompts from a YAML file
// Handles @assets/* replacement at load time
// file: prompt file path relative to app root (e.g., "assistants/test/prompts.yml")
// root: resource root directory for assets (e.g., "assistants/test")
// Returns: prompts slice, modification timestamp, error
func LoadPrompts(file string, root string) ([]Prompt, int64, error) {
	app, err := fs.Get("app")
	if err != nil {
		return nil, 0, err
	}

	ts, err := app.ModTime(file)
	if err != nil {
		return nil, 0, err
	}

	content, err := app.ReadFile(file)
	if err != nil {
		return nil, 0, err
	}

	// Replace @assets/xxx references with file content
	content = replaceAssets(content, root, app)

	// Parse prompts
	var prompts []Prompt
	err = yaml.Unmarshal(content, &prompts)
	if err != nil {
		return nil, 0, err
	}

	return prompts, ts.UnixNano(), nil
}

// LoadPromptsRaw loads raw prompt content from a YAML file
// Handles @assets/* replacement at load time
// Returns raw YAML string for further processing
func LoadPromptsRaw(file string, root string) (string, int64, error) {
	app, err := fs.Get("app")
	if err != nil {
		return "", 0, err
	}

	ts, err := app.ModTime(file)
	if err != nil {
		return "", 0, err
	}

	content, err := app.ReadFile(file)
	if err != nil {
		return "", 0, err
	}

	// Replace @assets/xxx references with file content
	content = replaceAssets(content, root, app)

	return string(content), ts.UnixNano(), nil
}

// LoadGlobalPrompts loads global prompts from agent/prompts.yml
// Returns: prompts slice, modification timestamp, error
func LoadGlobalPrompts() ([]Prompt, int64, error) {
	file := filepath.Join("agent", "prompts.yml")

	// Check if file exists
	exists, _ := application.App.Exists(file)
	if !exists {
		return nil, 0, nil
	}

	return LoadPrompts(file, "agent")
}

// LoadPromptPresets loads prompt presets from a directory
// Supports multi-level directories, key is path with "/" replaced by "."
// e.g., prompts/chat/friendly.yml -> "chat.friendly"
func LoadPromptPresets(dir string, root string) (map[string][]Prompt, int64, error) {
	app, err := fs.Get("app")
	if err != nil {
		return nil, 0, err
	}

	// Check if directory exists
	exists, _ := app.Exists(dir)
	if !exists {
		return nil, 0, nil
	}

	// Read directory recursively
	files, err := app.ReadDir(dir, true)
	if err != nil {
		return nil, 0, err
	}

	presets := make(map[string][]Prompt)
	var latestTs int64

	for _, file := range files {
		// Only process .yml/.yaml files
		if !strings.HasSuffix(file, ".yml") && !strings.HasSuffix(file, ".yaml") {
			continue
		}

		ts, err := app.ModTime(file)
		if err != nil {
			return nil, 0, err
		}
		if ts.UnixNano() > latestTs {
			latestTs = ts.UnixNano()
		}

		// Read file content
		content, err := app.ReadFile(file)
		if err != nil {
			return nil, 0, err
		}

		// Replace @assets/xxx references with file content
		content = replaceAssets(content, root, app)

		// Parse prompts
		var prompts []Prompt
		err = yaml.Unmarshal(content, &prompts)
		if err != nil {
			return nil, 0, err
		}

		// Build key: get relative path from dir, remove extension and replace "/" with "."
		relPath := strings.TrimPrefix(file, dir+"/")
		key := strings.TrimSuffix(relPath, filepath.Ext(relPath))
		key = strings.ReplaceAll(key, "/", ".")
		presets[key] = prompts
	}

	return presets, latestTs, nil
}

// replaceAssets replaces @assets/xxx references with file content
func replaceAssets(content []byte, root string, app fs.FileSystem) []byte {
	return reAssetRef.ReplaceAllFunc(content, func(s []byte) []byte {
		matches := reAssetRef.FindStringSubmatch(string(s))
		if len(matches) < 2 {
			return s
		}

		asset := matches[1]
		assetFile := filepath.Join(root, "assets", asset)
		assetContent, err := app.ReadFile(assetFile)
		if err != nil {
			return []byte("")
		}

		// Add proper YAML formatting for content (multiline string)
		lines := strings.Split(string(assetContent), "\n")
		formattedContent := "|\n"
		for _, line := range lines {
			formattedContent += "    " + line + "\n"
		}
		return []byte(formattedContent)
	})
}

// Parse parses a single prompt, replacing variables
// ctx: context variables map, key corresponds to $CTX.{key}
// Returns a new Prompt with variables replaced
func (p Prompt) Parse(ctx map[string]string) Prompt {
	result := Prompt{
		Role:    p.Role,
		Content: parseVariables(p.Content, ctx),
		Name:    p.Name,
	}
	return result
}

// Parse parses all prompts in the slice, replacing variables
// ctx: context variables map, key corresponds to $CTX.{key}
// Returns a new Prompts slice with variables replaced
func (ps Prompts) Parse(ctx map[string]string) Prompts {
	result := make(Prompts, len(ps))
	for i, p := range ps {
		result[i] = p.Parse(ctx)
	}
	return result
}

// parseVariables replaces all variable types in content
func parseVariables(content string, ctx map[string]string) string {
	// Replace $SYS.* variables
	content = reSysVar.ReplaceAllStringFunc(content, func(s string) string {
		matches := reSysVar.FindStringSubmatch(s)
		if len(matches) < 2 {
			return s
		}
		varName := matches[1]
		if fn, ok := SystemVariables[varName]; ok {
			return fn()
		}
		return s // Keep original if not found
	})

	// Replace $ENV.* variables
	content = reEnvVar.ReplaceAllStringFunc(content, func(s string) string {
		matches := reEnvVar.FindStringSubmatch(s)
		if len(matches) < 2 {
			return s
		}
		varName := matches[1]
		return os.Getenv(varName)
	})

	// Replace $CTX.* variables
	if ctx != nil {
		content = reCtxVar.ReplaceAllStringFunc(content, func(s string) string {
			matches := reCtxVar.FindStringSubmatch(s)
			if len(matches) < 2 {
				return s
			}
			varName := matches[1]
			if val, ok := ctx[varName]; ok {
				return val
			}
			return "" // Empty string if not found in ctx
		})
	}

	return content
}

// Merge merges two prompt slices
// globalPrompts are prepended to assistantPrompts
func Merge(globalPrompts, assistantPrompts []Prompt) []Prompt {
	if len(globalPrompts) == 0 {
		return assistantPrompts
	}
	if len(assistantPrompts) == 0 {
		return globalPrompts
	}
	result := make([]Prompt, 0, len(globalPrompts)+len(assistantPrompts))
	result = append(result, globalPrompts...)
	result = append(result, assistantPrompts...)
	return result
}
