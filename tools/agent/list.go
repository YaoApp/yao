package agent

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	goufs "github.com/yaoapp/gou/fs"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/agent/i18n"
)

// ListHandler handles the agent_list tool.
// Args[0]: namespace (string, optional)
func ListHandler(proc *process.Process) interface{} {
	namespace := ""
	if len(proc.Args) > 0 {
		namespace = proc.ArgsString(0)
	}

	locale := extractLocale(proc)

	app, err := goufs.Get("app")
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("app filesystem: %s", err.Error())}
	}

	root := "/assistants"
	exists, _ := app.Exists(root)
	if !exists {
		return map[string]interface{}{"agents": []agentInfo{}}
	}

	nsDirs, err := app.ReadDir(root, false)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("read assistants dir: %s", err.Error())}
	}

	agents := make([]agentInfo, 0)
	for _, nsDir := range nsDirs {
		nsName := filepath.Base(nsDir)
		if namespace != "" && nsName != namespace {
			continue
		}

		agentDirs, err := app.ReadDir(nsDir, false)
		if err != nil {
			continue
		}

		for _, agentDir := range agentDirs {
			pkgFile := filepath.Join(agentDir, "package.yao")
			pkgExists, _ := app.Exists(pkgFile)
			if !pkgExists {
				continue
			}

			data, err := app.ReadFile(pkgFile)
			if err != nil {
				continue
			}

			var pkg packageDSL
			if err := json.Unmarshal(data, &pkg); err != nil {
				continue
			}

			agentName := filepath.Base(agentDir)
			id := nsName + "." + agentName

			if strings.HasPrefix(id, "__yao.") {
				continue
			}

			name := pkg.Name
			description := pkg.Description
			capabilities := pkg.Capabilities
			resolveLocaleFields(agentDir, locale, &name, &description, &capabilities)

			agents = append(agents, agentInfo{
				ID:           id,
				Name:         name,
				Description:  description,
				Capabilities: capabilities,
			})
		}
	}

	return map[string]interface{}{"agents": agents}
}

// resolveLocaleFields replaces {{ key }} templates in name/description using
// the agent's locales/ directory. Falls back gracefully: exact locale →
// language code (e.g. "zh") → en-us → raw template.
func resolveLocaleFields(agentDir, locale string, fields ...*string) {
	hasTemplate := false
	for _, f := range fields {
		if strings.Contains(*f, "{{") {
			hasTemplate = true
			break
		}
	}
	if !hasTemplate {
		return
	}

	locales, err := i18n.GetLocales(agentDir)
	if err != nil || len(locales) == 0 {
		return
	}
	locales = locales.Flatten()

	li := findLocale(locales, locale)
	if li == nil {
		return
	}

	for _, f := range fields {
		if parsed := li.Parse(*f); parsed != nil {
			if s, ok := parsed.(string); ok {
				*f = s
			}
		}
	}
}

func findLocale(locales i18n.Map, locale string) *i18n.I18n {
	locale = strings.ToLower(locale)
	if li, ok := locales[locale]; ok {
		return &li
	}
	parts := strings.SplitN(locale, "-", 2)
	if len(parts) > 1 {
		if li, ok := locales[parts[0]]; ok {
			return &li
		}
	}
	if li, ok := locales["en-us"]; ok {
		return &li
	}
	if li, ok := locales["en"]; ok {
		return &li
	}
	return nil
}
