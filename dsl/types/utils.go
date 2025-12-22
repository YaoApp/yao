package types

import (
	"os"
	"path/filepath"
	"strings"
)

// ToPath convert id to path
func ToPath(typ Type, id string) string {

	// Get the root path and the extensions of the type
	root, exts := TypeRootAndExts(typ)
	ext := ".yao"
	if len(exts) > 0 {
		ext = exts[0]
	}

	// 1. Replace all . to /
	path := strings.ReplaceAll(id, ".", string(os.PathSeparator))
	// 2. Replace all __ to .
	path = strings.ReplaceAll(path, "__", ".")
	// 3. Join the root path
	return filepath.Join(root, path) + ext
}

// ToID convert file path to id
func ToID(path string) string {
	typ := DetectType(path)
	return WithTypeToID(typ, path)
}

// WithTypeToID convert file path to id
func WithTypeToID(typ Type, path string) string {

	// Get the root path and the extensions of the type
	root, exts := TypeRootAndExts(typ)

	// 0. if the first character is /, remove it
	if strings.HasPrefix(path, string(os.PathSeparator)) {
		path = strings.TrimPrefix(path, string(os.PathSeparator))
	}

	// 1. Split the path by /
	parts := strings.Split(path, string(os.PathSeparator))
	if len(parts) > 0 && parts[0] == root {
		// Skip the root path
		parts = parts[1:]

		// Remove the extension only if parts is not empty
		if len(parts) > 0 {
			last := parts[len(parts)-1]
			for _, ext := range exts {
				if strings.HasSuffix(last, ext) {
					parts[len(parts)-1] = strings.TrimSuffix(last, ext)
				}
			}
		}

		// Join the parts
		path = strings.Join(parts, string(os.PathSeparator))
	}

	// 2. Replace All . to __
	path = strings.ReplaceAll(path, ".", "__")

	// 3. Replace all / to .
	path = strings.ReplaceAll(path, string(os.PathSeparator), ".")

	return path
}

// DetectType detect the type by the file path
func DetectType(path string) Type {
	parts := strings.Split(path, string(os.PathSeparator))
	if len(parts) < 2 {
		return TypeUnknown
	}

	root := parts[0]
	last := parts[len(parts)-1]
	extParts := strings.Split(last, ".")
	if len(extParts) < 2 {
		return TypeUnknown
	}
	ext := extParts[len(extParts)-2]

	// Detect the type by the extension
	switch ext {
	case "http":
		return TypeAPI
	case "sch":
		return TypeSchedule
	case "table":
		return TypeTable
	case "form":
		return TypeForm
	case "list":
		return TypeList
	case "chart":
		return TypeChart
	case "dash":
		return TypeDashboard
	case "flow":
		return TypeFlow
	case "pipe":
		return TypePipe
	case "ai":
		return TypeAIGC
	case "mod":
		return TypeModel
	case "conn":
		return TypeConnector
	case "lru", "redis", "mongo", "xun":
		return TypeStore
	}

	// Detect the type by the root path
	switch root {
	case "models":
		return TypeModel
	case "connectors":
		return TypeConnector
	case "mcps":
		return TypeMCPClient
	case "apis":
		if ext == "http" {
			return TypeAPI
		}
		if ext == "mcp" {
			return TypeMCPServer
		}
		return TypeUnknown
	case "schedules":
		return TypeSchedule
	case "tables":
		return TypeTable
	case "forms":
		return TypeForm
	case "lists":
		return TypeList
	case "charts":
		return TypeChart
	case "dashboards":
		return TypeDashboard
	case "flows":
		return TypeFlow
	case "pipes":
		return TypePipe
	case "aigcs":
		return TypeAIGC
	case "stores":
		return TypeStore
	default:
		return TypeUnknown
	}

}

// TypeRootAndExts return the root path and the extensions of the type
func TypeRootAndExts(typ Type) (string, []string) {
	switch typ {
	case TypeModel:
		return "models", []string{".mod.yao", ".mod.jsonc", ".mod.json"}
	case TypeConnector:
		return "connectors", []string{".conn.yao", ".conn.jsonc", ".conn.json"}
	case TypeMCPClient, TypeMCPServer:
		return "mcps", []string{".mcp.yao", ".mcp.jsonc", ".mcp.json"}
	case TypeAPI:
		return "apis", []string{".http.yao", ".http.jsonc", ".http.json"}
	case TypeSchedule:
		return "schedules", []string{".sch.yao", ".sch.jsonc", ".sch.json"}
	case TypeTable:
		return "tables", []string{".table.yao", ".table.jsonc", ".table.json"}
	case TypeForm:
		return "forms", []string{".form.yao", ".form.jsonc", ".form.json"}
	case TypeList:
		return "lists", []string{".list.yao", ".list.jsonc", ".list.json"}
	case TypeChart:
		return "charts", []string{".chart.yao", ".chart.jsonc", ".chart.json"}
	case TypeDashboard:
		return "dashboards", []string{".dash.yao", ".dash.jsonc", ".dash.json"}
	case TypeFlow:
		return "flows", []string{".flow.yao", ".flow.jsonc", ".flow.json"}
	case TypePipe:
		return "pipes", []string{".pipe.yao", ".pipe.jsonc", ".pipe.json"}
	case TypeAIGC:
		return "aigcs", []string{".ai.yao", ".ai.jsonc", ".ai.json"}
	case TypeStore:
		return "stores", []string{".lru.yao", ".redis.yao", ".mongo.yao", ".xun.yao", ".store.yao", ".store.jsonc", ".store.json"}
	default:
		return "", []string{}
	}
}
