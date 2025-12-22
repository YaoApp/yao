package types

import (
	"os"
	"path/filepath"
	"testing"
)

func TestToPath(t *testing.T) {
	tests := []struct {
		name string
		typ  Type
		id   string
		want string
	}{
		{
			name: "Model with dots and underscores",
			typ:  TypeModel,
			id:   "user__profile.admin",
			want: filepath.Join("models", "user.profile", "admin.mod.yao"),
		},
		{
			name: "API with simple id",
			typ:  TypeAPI,
			id:   "user.login",
			want: filepath.Join("apis", "user", "login.http.yao"),
		},
		{
			name: "Unknown type (defaults to .yao)",
			typ:  TypeUnknown,
			id:   "test",
			want: filepath.Join("", "test.yao"),
		},
		{
			name: "Connector with nested path",
			typ:  TypeConnector,
			id:   "database.mysql__config",
			want: filepath.Join("connectors", "database", "mysql.config.conn.yao"),
		},
		{
			name: "Type with no extensions",
			typ:  Type("unknown"),
			id:   "test",
			want: filepath.Join("", "test.yao"),
		},
		{
			name: "Type with empty extensions",
			typ:  Type(""),
			id:   "test",
			want: filepath.Join("", "test.yao"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToPath(tt.typ, tt.id); got != tt.want {
				t.Errorf("ToPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToID(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "Model file path",
			path: filepath.Join("models", "user.mod.yao"),
			want: "user",
		},
		{
			name: "API file path",
			path: filepath.Join("apis", "user", "login.http.yao"),
			want: "user.login",
		},
		{
			name: "Form file path with dots",
			path: filepath.Join("forms", "user.profile", "edit.form.yao"),
			want: "user__profile.edit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToID(tt.path); got != tt.want {
				t.Errorf("ToID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithTypeToID(t *testing.T) {
	tests := []struct {
		name string
		typ  Type
		path string
		want string
	}{
		{
			name: "Path with leading separator",
			typ:  TypeModel,
			path: string(os.PathSeparator) + filepath.Join("models", "user.mod.yao"),
			want: "user",
		},
		{
			name: "Path without leading separator",
			typ:  TypeModel,
			path: filepath.Join("models", "user.mod.yao"),
			want: "user",
		},
		{
			name: "Path with root not matching",
			typ:  TypeModel,
			path: filepath.Join("other", "user.mod.yao"),
			want: "other.user__mod__yao",
		},
		{
			name: "Nested path with dots",
			typ:  TypeForm,
			path: filepath.Join("forms", "user.profile", "edit.form.yao"),
			want: "user__profile.edit",
		},
		{
			name: "Multiple extensions matching",
			typ:  TypeModel,
			path: filepath.Join("models", "user.mod.jsonc"),
			want: "user",
		},
		{
			name: "No extension matching",
			typ:  TypeModel,
			path: filepath.Join("models", "user.txt"),
			want: "user__txt",
		},
		{
			name: "Path with single part",
			typ:  TypeModel,
			path: "user.mod.yao",
			want: "user__mod__yao",
		},
		{
			name: "Store type with multiple extensions",
			typ:  TypeStore,
			path: filepath.Join("stores", "cache.redis.yao"),
			want: "cache",
		},
		{
			name: "Empty path",
			typ:  TypeModel,
			path: "",
			want: "",
		},
		{
			name: "Path with root matching but no parts",
			typ:  TypeModel,
			path: "models",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WithTypeToID(tt.typ, tt.path); got != tt.want {
				t.Errorf("WithTypeToID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectType(t *testing.T) {
	tests := []struct {
		name string
		path string
		want Type
	}{
		// Test by extension
		{
			name: "HTTP API",
			path: filepath.Join("apis", "user.http.yao"),
			want: TypeAPI,
		},
		{
			name: "Schedule",
			path: filepath.Join("schedules", "backup.sch.yao"),
			want: TypeSchedule,
		},
		{
			name: "Table",
			path: filepath.Join("tables", "user.table.yao"),
			want: TypeTable,
		},
		{
			name: "Form",
			path: filepath.Join("forms", "user.form.yao"),
			want: TypeForm,
		},
		{
			name: "List",
			path: filepath.Join("lists", "user.list.yao"),
			want: TypeList,
		},
		{
			name: "Chart",
			path: filepath.Join("charts", "sales.chart.yao"),
			want: TypeChart,
		},
		{
			name: "Dashboard",
			path: filepath.Join("dashboards", "main.dash.yao"),
			want: TypeDashboard,
		},
		{
			name: "Flow",
			path: filepath.Join("flows", "process.flow.yao"),
			want: TypeFlow,
		},
		{
			name: "Pipe",
			path: filepath.Join("pipes", "transform.pipe.yao"),
			want: TypePipe,
		},
		{
			name: "AIGC",
			path: filepath.Join("aigcs", "chat.ai.yao"),
			want: TypeAIGC,
		},
		{
			name: "Model by extension",
			path: filepath.Join("models", "user.mod.yao"),
			want: TypeModel,
		},
		{
			name: "Connector by extension",
			path: filepath.Join("connectors", "db.conn.yao"),
			want: TypeConnector,
		},
		{
			name: "Store LRU",
			path: filepath.Join("stores", "cache.lru.yao"),
			want: TypeStore,
		},
		{
			name: "LRU extension in non-stores directory",
			path: filepath.Join("other", "cache.lru.yao"),
			want: TypeStore,
		},
		{
			name: "Store Redis",
			path: filepath.Join("stores", "cache.redis.yao"),
			want: TypeStore,
		},
		{
			name: "Store Mongo",
			path: filepath.Join("stores", "cache.mongo.yao"),
			want: TypeStore,
		},
		{
			name: "Store by extension",
			path: filepath.Join("stores", "cache.store.yao"),
			want: TypeStore,
		},
		{
			name: "MCP extension in non-apis directory",
			path: filepath.Join("other", "service.mcp.yao"),
			want: TypeUnknown,
		},
		// Test by root path
		{
			name: "Model by root",
			path: filepath.Join("models", "user.yao"),
			want: TypeModel,
		},
		{
			name: "Connector by root",
			path: filepath.Join("connectors", "db.yao"),
			want: TypeConnector,
		},
		{
			name: "MCP Client",
			path: filepath.Join("mcps", "client.yao"),
			want: TypeMCPClient,
		},
		{
			name: "API by root with http ext",
			path: filepath.Join("apis", "user.http.yao"),
			want: TypeAPI,
		},
		{
			name: "MCP Server",
			path: filepath.Join("apis", "server.mcp.yao"),
			want: TypeMCPServer,
		},
		{
			name: "MCP by extension",
			path: filepath.Join("mcps", "client.mcp.yao"),
			want: TypeMCPClient,
		},
		{
			name: "API by root unknown ext",
			path: filepath.Join("apis", "user.unknown.yao"),
			want: TypeUnknown,
		},
		{
			name: "Schedule by root",
			path: filepath.Join("schedules", "backup.yao"),
			want: TypeSchedule,
		},
		{
			name: "Table by root",
			path: filepath.Join("tables", "user.yao"),
			want: TypeTable,
		},
		{
			name: "Form by root",
			path: filepath.Join("forms", "user.yao"),
			want: TypeForm,
		},
		{
			name: "List by root",
			path: filepath.Join("lists", "user.yao"),
			want: TypeList,
		},
		{
			name: "Chart by root",
			path: filepath.Join("charts", "sales.yao"),
			want: TypeChart,
		},
		{
			name: "Dashboard by root",
			path: filepath.Join("dashboards", "main.yao"),
			want: TypeDashboard,
		},
		{
			name: "Flow by root",
			path: filepath.Join("flows", "process.yao"),
			want: TypeFlow,
		},
		{
			name: "Pipe by root",
			path: filepath.Join("pipes", "transform.yao"),
			want: TypePipe,
		},
		{
			name: "AIGC by root",
			path: filepath.Join("aigcs", "chat.yao"),
			want: TypeAIGC,
		},
		{
			name: "Store by root",
			path: filepath.Join("stores", "cache.yao"),
			want: TypeStore,
		},
		{
			name: "Unknown root",
			path: filepath.Join("unknown", "file.yao"),
			want: TypeUnknown,
		},
		// Edge cases
		{
			name: "Path with less than 2 parts",
			path: "file.yao",
			want: TypeUnknown,
		},
		{
			name: "File without extension",
			path: filepath.Join("models", "user"),
			want: TypeUnknown,
		},
		{
			name: "File with single dot",
			path: filepath.Join("models", "user.yao"),
			want: TypeModel,
		},
		{
			name: "Empty path",
			path: "",
			want: TypeUnknown,
		},
		{
			name: "Path with single component",
			path: "file",
			want: TypeUnknown,
		},
		{
			name: "File with extension parts length < 2",
			path: filepath.Join("models", "user"),
			want: TypeUnknown,
		},
		{
			name: "File with extension matching filename",
			path: filepath.Join("models", "http.yao"),
			want: TypeAPI,
		},
		{
			name: "File with extension matching filename - sch",
			path: filepath.Join("schedules", "sch.yao"),
			want: TypeSchedule,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetectType(tt.path); got != tt.want {
				t.Errorf("DetectType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTypeRootAndExts(t *testing.T) {
	tests := []struct {
		name     string
		typ      Type
		wantRoot string
		wantExts []string
	}{
		{
			name:     "Model",
			typ:      TypeModel,
			wantRoot: "models",
			wantExts: []string{".mod.yao", ".mod.jsonc", ".mod.json"},
		},
		{
			name:     "Connector",
			typ:      TypeConnector,
			wantRoot: "connectors",
			wantExts: []string{".conn.yao", ".conn.jsonc", ".conn.json"},
		},
		{
			name:     "MCP Client",
			typ:      TypeMCPClient,
			wantRoot: "mcps",
			wantExts: []string{".mcp.yao", ".mcp.jsonc", ".mcp.json"},
		},
		{
			name:     "MCP Server",
			typ:      TypeMCPServer,
			wantRoot: "mcps",
			wantExts: []string{".mcp.yao", ".mcp.jsonc", ".mcp.json"},
		},
		{
			name:     "API",
			typ:      TypeAPI,
			wantRoot: "apis",
			wantExts: []string{".http.yao", ".http.jsonc", ".http.json"},
		},
		{
			name:     "Schedule",
			typ:      TypeSchedule,
			wantRoot: "schedules",
			wantExts: []string{".sch.yao", ".sch.jsonc", ".sch.json"},
		},
		{
			name:     "Table",
			typ:      TypeTable,
			wantRoot: "tables",
			wantExts: []string{".table.yao", ".table.jsonc", ".table.json"},
		},
		{
			name:     "Form",
			typ:      TypeForm,
			wantRoot: "forms",
			wantExts: []string{".form.yao", ".form.jsonc", ".form.json"},
		},
		{
			name:     "List",
			typ:      TypeList,
			wantRoot: "lists",
			wantExts: []string{".list.yao", ".list.jsonc", ".list.json"},
		},
		{
			name:     "Chart",
			typ:      TypeChart,
			wantRoot: "charts",
			wantExts: []string{".chart.yao", ".chart.jsonc", ".chart.json"},
		},
		{
			name:     "Dashboard",
			typ:      TypeDashboard,
			wantRoot: "dashboards",
			wantExts: []string{".dash.yao", ".dash.jsonc", ".dash.json"},
		},
		{
			name:     "Flow",
			typ:      TypeFlow,
			wantRoot: "flows",
			wantExts: []string{".flow.yao", ".flow.jsonc", ".flow.json"},
		},
		{
			name:     "Pipe",
			typ:      TypePipe,
			wantRoot: "pipes",
			wantExts: []string{".pipe.yao", ".pipe.jsonc", ".pipe.json"},
		},
		{
			name:     "AIGC",
			typ:      TypeAIGC,
			wantRoot: "aigcs",
			wantExts: []string{".ai.yao", ".ai.jsonc", ".ai.json"},
		},
		{
			name:     "Store",
			typ:      TypeStore,
			wantRoot: "stores",
			wantExts: []string{".lru.yao", ".redis.yao", ".mongo.yao", ".xun.yao", ".store.yao", ".store.jsonc", ".store.json"},
		},
		{
			name:     "Unknown",
			typ:      TypeUnknown,
			wantRoot: "",
			wantExts: []string{},
		},
		{
			name:     "Empty type",
			typ:      Type(""),
			wantRoot: "",
			wantExts: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRoot, gotExts := TypeRootAndExts(tt.typ)
			if gotRoot != tt.wantRoot {
				t.Errorf("TypeRootAndExts() root = %v, want %v", gotRoot, tt.wantRoot)
			}
			if len(gotExts) != len(tt.wantExts) {
				t.Errorf("TypeRootAndExts() exts length = %v, want %v", len(gotExts), len(tt.wantExts))
				return
			}
			for i, ext := range gotExts {
				if ext != tt.wantExts[i] {
					t.Errorf("TypeRootAndExts() exts[%d] = %v, want %v", i, ext, tt.wantExts[i])
				}
			}
		})
	}
}

// Test integration scenarios
func TestIntegration(t *testing.T) {
	tests := []struct {
		name     string
		typ      Type
		id       string
		wantPath string
		wantID   string
	}{
		{
			name:     "Model round trip",
			typ:      TypeModel,
			id:       "user__profile.admin",
			wantPath: filepath.Join("models", "user.profile", "admin.mod.yao"),
			wantID:   "user__profile.admin",
		},
		{
			name:     "API round trip",
			typ:      TypeAPI,
			id:       "user.login",
			wantPath: filepath.Join("apis", "user", "login.http.yao"),
			wantID:   "user.login",
		},
		{
			name:     "Complex nested path",
			typ:      TypeForm,
			id:       "admin__panel.user__management.edit",
			wantPath: filepath.Join("forms", "admin.panel", "user.management", "edit.form.yao"),
			wantID:   "admin__panel.user__management.edit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test ID to Path
			path := ToPath(tt.typ, tt.id)
			if path != tt.wantPath {
				t.Errorf("ToPath() = %v, want %v", path, tt.wantPath)
			}

			// Test Path to ID
			id := WithTypeToID(tt.typ, path)
			if id != tt.wantID {
				t.Errorf("WithTypeToID() = %v, want %v", id, tt.wantID)
			}

			// Test DetectType
			detectedType := DetectType(path)
			if detectedType != tt.typ {
				t.Errorf("DetectType() = %v, want %v", detectedType, tt.typ)
			}
		})
	}
}
