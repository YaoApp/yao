package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/process"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/agent"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/data"
	"github.com/yaoapp/yao/i18n"
	"github.com/yaoapp/yao/kb"
	kbtypes "github.com/yaoapp/yao/kb/types"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/share"
)

// AppLoad prepares share.App for serving: sets Lang default, AdminRoot, and icon URLs.
func AppLoad(cfg config.Config) error {
	if share.App.Lang == "" {
		share.App.Lang = cfg.Lang
	}
	if err := appReplaceAdminRoot(); err != nil {
		return err
	}
	appSetIcons()
	return nil
}

// AppExport registers yao.app.* processes and API routes.
func AppExport() error {
	appExportProcess()
	return appExportAPI()
}

func appReplaceAdminRoot() error {
	if share.App.AdminRoot == "" {
		share.App.AdminRoot = "yao"
	}
	root := strings.TrimPrefix(share.App.AdminRoot, "/")
	root = strings.TrimSuffix(root, "/")
	return data.ReplaceCUI("__yao_admin_root", root)
}

func appSetIcons() {
	apiBase := appGetAPIBase()
	share.App.Favicon = fmt.Sprintf("%s/__yao/app/icons/app.ico", apiBase)
	share.App.Logo = fmt.Sprintf("%s/__yao/app/icons/app.png", apiBase)
}

func appGetAPIBase() string {
	if openapi.Server != nil && openapi.Server.Config != nil && openapi.Server.Config.BaseURL != "" {
		return openapi.Server.Config.BaseURL
	}
	return "/api"
}

func appExportAPI() error {
	http := api.HTTP{
		Name:        "Widget App API",
		Description: "Widget App API",
		Version:     share.VERSION,
		Guard:       "bearer-jwt",
		Group:       "__yao/app",
		Paths:       []api.Path{},
	}

	settingProcess := "yao.app.Xgen"
	if share.App.Setting != "" {
		settingProcess = share.App.Setting
	}

	http.Paths = append(http.Paths, api.Path{
		Label:       "App Setting",
		Description: "App Setting",
		Guard:       "-",
		Path:        "/setting",
		Method:      "GET",
		Process:     settingProcess,
		In:          []interface{}{},
		Out:         api.Out{Status: 200, Type: "application/json"},
	})

	http.Paths = append(http.Paths, api.Path{
		Label:       "App Setting",
		Description: "App Setting",
		Guard:       "-",
		Path:        "/setting",
		Method:      "POST",
		Process:     settingProcess,
		In:          []interface{}{":payload"},
		Out:         api.Out{Status: 200, Type: "application/json"},
	})

	menuArgs := []interface{}{}
	if share.App.Menu.Args != nil {
		menuArgs = share.App.Menu.Args
	}
	menuArgs = append(menuArgs, "$query.locale")
	http.Paths = append(http.Paths, api.Path{
		Label:       "App Menu",
		Description: "App Menu",
		Path:        "/menu",
		Method:      "GET",
		Process:     "yao.app.Menu",
		In:          menuArgs,
		Out:         api.Out{Status: 200, Type: "application/json"},
	})

	http.Paths = append(http.Paths, api.Path{
		Label:       "App Icons",
		Description: "App Icons",
		Path:        "/icons/:name",
		Guard:       "-",
		Method:      "GET",
		Process:     "yao.app.Icons",
		In:          []interface{}{"$param.name"},
		Out:         api.Out{Status: 200},
	})

	http.Paths = append(http.Paths, api.Path{
		Label:       "Setup",
		Description: "Setup",
		Path:        "/setup",
		Guard:       "-",
		Method:      "POST",
		Process:     "yao.app.Setup",
		In:          []interface{}{":payload"},
		Out:         api.Out{Status: 200, Type: "application/json"},
	})

	http.Paths = append(http.Paths, api.Path{
		Label:       "Check",
		Description: "Check",
		Path:        "/check",
		Guard:       "-",
		Method:      "POST",
		Process:     "yao.app.Check",
		In:          []interface{}{":payload"},
		Out:         api.Out{Status: 200, Type: "application/json"},
	})

	http.Paths = append(http.Paths, api.Path{
		Label:       "Service",
		Description: "Service",
		Path:        "/service/:name",
		Guard:       "bearer-jwt",
		Method:      "POST",
		Process:     "yao.app.Service",
		In:          []interface{}{"$param.name", ":payload"},
		Out:         api.Out{Status: 200, Type: "application/json"},
	})

	source, err := jsoniter.Marshal(http)
	if err != nil {
		return err
	}

	_, err = api.LoadSource("<widget.app>.yao", source, "widgets.app")
	return err
}

func appExportProcess() {
	process.Register("yao.app.setting", processAppSetting)
	process.Register("yao.app.xgen", processAppXgen)
	process.Register("yao.app.menu", processAppMenu)
	process.Register("yao.app.icons", processAppIcons)
	process.Register("yao.app.setup", processAppSetup)
	process.Register("yao.app.check", processAppCheck)
	process.Register("yao.app.service", processAppService)
}

func processAppService(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	service := fmt.Sprintf("__yao_service.%s", process.ArgsString(0))
	payload := process.ArgsMap(1)
	if len(payload) == 0 {
		exception.New("content is required", 400).Throw()
	}

	method, ok := payload["method"].(string)
	if !ok || service == "" {
		exception.New("method is required", 400).Throw()
	}

	args := []interface{}{}
	if v, ok := payload["args"].([]interface{}); ok {
		args = v
	}

	// Forward: Agent confirm Command
	if service == "__yao_service.__agent" && method == "ExecCommand" {
		if len(args) < 4 {
			exception.New("args is required (%v)", 400, args).Throw()
		}

		id := args[0].(string)
		ctx := args[3].(map[string]interface{})
		processName := args[1].(string)
		processArgs := append(args[2].([]interface{}), ctx)
		result := forwardAgentExecCommand(process, processName, processArgs...)
		return map[string]interface{}{"id": id, "result": result, "context": ctx}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	script, err := v8.Select(service)
	if err != nil {
		exception.New("services.%s not loaded", 404, process.ArgsString(0)).Throw()
		return nil
	}

	v8ctx, err := script.NewContext(process.Sid, process.Global)
	if err != nil {
		message := fmt.Sprintf("services.%s failed to create context. %s", process.ArgsString(0), err.Error())
		log.Error("[V8] process error. %s", message)
		exception.New(message, 500).Throw()
		return nil
	}
	defer v8ctx.Close()

	res, err := v8ctx.CallWith(ctx, method, args...)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return res
}

func forwardAgentExecCommand(p *process.Process, name string, args ...interface{}) interface{} {
	newProc, err := process.Of(name, args...)
	if err != nil {
		exception.New(err.Error(), 400).Throw()
	}

	res, err := newProc.WithGlobal(p.Global).WithSID(p.Sid).Exec()
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return res
}

func processAppCheck(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	payload := process.ArgsMap(0)
	time.Sleep(3 * time.Second)

	if _, has := payload["error"]; has {
		exception.New("Something error", 500).Throw()
	}
	return nil
}

func processAppSetup(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	payload := process.ArgsMap(0)
	time.Sleep(3 * time.Second)

	if _, has := payload["error"]; has {
		exception.New("Something error", 500).Throw()
	}

	lang := session.Lang(process)
	if sid, has := payload["sid"].(string); has {
		lang, err := session.Global().ID(sid).Get("__yao_lang")
		if err != nil {
			lang = strings.ToLower(lang.(string))
		}
	}

	root := "yao"
	if share.App.AdminRoot != "" {
		root = share.App.AdminRoot
	}

	setting, err := i18n.Trans(lang, []string{"app.app"}, &share.App)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return map[string]interface{}{
		"home":    fmt.Sprintf("http://127.0.0.1:%d", config.Conf.Port),
		"admin":   fmt.Sprintf("http://127.0.0.1:%d/%s/", config.Conf.Port, root),
		"setting": setting,
	}
}

func processAppIcons(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	name := process.ArgsString(0)
	file := filepath.Join("icons", name)
	content, err := application.App.Read(file)
	if err != nil {
		exception.New(err.Error(), 400).Throw()
	}
	return string(content)
}

func processAppMenu(p *process.Process) interface{} {
	if share.App.Menu.Process == "" {
		exception.New("menu.process is required", 400).Throw()
	}

	handle, err := process.Of(share.App.Menu.Process, p.Args...)
	if err != nil {
		exception.New(err.Error(), 400).Throw()
	}

	handle.WithGlobal(p.Global).WithSID(p.Sid)
	if p.Authorized != nil {
		handle = handle.WithAuthorized(p.Authorized)
	}

	err = handle.Execute()
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	defer handle.Dispose()
	return handle.Value()
}

func processAppSetting(process *process.Process) interface{} {
	sid := process.Sid
	if sid == "" {
		sid = session.ID()
	}

	if process.NumOfArgs() > 0 {
		payload := process.ArgsMap(0, map[string]interface{}{
			"now":  time.Now().Unix(),
			"lang": "en-us",
			"sid":  "",
		})

		if v, ok := payload["sid"].(string); ok && v != "" {
			sid = v
		}

		lang := strings.ToLower(fmt.Sprintf("%v", payload["lang"]))
		session.Global().ID(sid).Set("__yao_lang", lang)
	}

	setting, err := i18n.Trans(session.Lang(process, config.Conf.Lang), []string{"app.app"}, &share.App)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	result := setting.(*share.AppInfo)
	result.Sid = sid
	return *result
}

func processAppXgen(process *process.Process) interface{} {
	sid := process.Sid
	if sid == "" {
		sid = session.ID()
	}

	lang := config.Conf.Lang
	if process.NumOfArgs() > 0 {
		payload := process.ArgsMap(0, map[string]interface{}{
			"now":  time.Now().Unix(),
			"lang": "en-us",
			"sid":  "",
		})

		if v, ok := payload["sid"].(string); ok && v != "" {
			sid = v
		}
		lang = strings.ToLower(fmt.Sprintf("%v", payload["lang"]))
		session.Global().ID(sid).Set("__yao_lang", lang)
	}

	mode := os.Getenv("YAO_ENV")
	if mode == "" {
		mode = "production"
	}

	// CUI 1.0 compatibility: empty login entry
	xgenLogin := map[string]map[string]interface{}{
		"entry": {"admin": "/x/Welcome"},
	}

	// The default assistant
	agentConfig := map[string]interface{}{}
	ag := agent.GetAgent()
	if ag != nil {
		if ag.Uses != nil {
			agentConfig["uses"] = ag.Uses
		}

		if ast, ok := ag.Assistant.(*assistant.Assistant); ok {
			agentConfig["default"] = map[string]interface{}{
				"assistant_id":         ast.ID,
				"assistant_name":       ast.Name,
				"assistant_avatar":     ast.Avatar,
				"assistant_deleteable": false,
				"placeholder":          ast.GetPlaceholder(lang),
			}
		}
	}

	// External tools availability (safe subset for frontend)
	toolsConfig := map[string]interface{}{}
	if share.Tools != nil {
		safeTool := func(info *share.ExtToolInfo) map[string]interface{} {
			if info == nil {
				return map[string]interface{}{"available": false}
			}
			return map[string]interface{}{
				"available": info.Available,
				"name":      info.Name,
			}
		}
		toolsConfig["ffmpeg"] = safeTool(share.Tools.FFmpeg)
		toolsConfig["ffprobe"] = safeTool(share.Tools.FFprobe)
		toolsConfig["pdftoppm"] = safeTool(share.Tools.Pdftoppm)
		toolsConfig["mutool"] = safeTool(share.Tools.Mutool)
		toolsConfig["imagemagick"] = safeTool(share.Tools.ImageMagick)

		if share.Tools.Docker != nil {
			docker := map[string]interface{}{
				"available": share.Tools.Docker.Available,
				"name":      "docker",
			}
			if share.Tools.Docker.Mode != "" {
				docker["mode"] = share.Tools.Docker.Mode
			}
			toolsConfig["docker"] = docker
		}

		if share.Tools.Runners != nil {
			runnersConfig := map[string]interface{}{}
			for name, info := range share.Tools.Runners {
				if info == nil {
					continue
				}
				runnersConfig[name] = map[string]interface{}{
					"available": info.Available,
					"name":      info.Name,
					"version":   info.Version,
				}
			}
			toolsConfig["runners"] = runnersConfig
		}
	}

	// OpenAPI Settings
	openapiConfig := map[string]interface{}{}
	if openapi.Server != nil {
		openapiConfig = map[string]interface{}{
			"baseURL": openapi.Server.Config.BaseURL,
		}
	}

	// Knowledge Base Settings
	kbConfig := map[string]interface{}{}
	if kb.Instance != nil {
		if knowledgebase, ok := kb.Instance.(*kb.KnowledgeBase); ok && knowledgebase.Config != nil {
			currentLang := lang
			if currentLang == "" {
				currentLang = "en"
			}

			extractProviderIDs := func(providerMap map[string][]*kbtypes.Provider) []string {
				ids := []string{}
				if providerMap == nil {
					return ids
				}

				if providers, exists := providerMap[currentLang]; exists {
					for _, provider := range providers {
						ids = append(ids, provider.ID)
					}
					return ids
				}

				if currentLang != "en" {
					if providers, exists := providerMap["en"]; exists {
						for _, provider := range providers {
							ids = append(ids, provider.ID)
						}
						return ids
					}
				}

				for _, providers := range providerMap {
					for _, provider := range providers {
						ids = append(ids, provider.ID)
					}
					break
				}

				return ids
			}

			var chunkings, embeddings, converters, extractions, fetchers []string
			var searchers, rerankers, votes, weights, scores []string

			if knowledgebase.Providers != nil {
				chunkings = extractProviderIDs(knowledgebase.Providers.Chunkings)
				embeddings = extractProviderIDs(knowledgebase.Providers.Embeddings)
				converters = extractProviderIDs(knowledgebase.Providers.Converters)
				extractions = extractProviderIDs(knowledgebase.Providers.Extractions)
				fetchers = extractProviderIDs(knowledgebase.Providers.Fetchers)
				searchers = extractProviderIDs(knowledgebase.Providers.Searchers)
				rerankers = extractProviderIDs(knowledgebase.Providers.Rerankers)
				votes = extractProviderIDs(knowledgebase.Providers.Votes)
				weights = extractProviderIDs(knowledgebase.Providers.Weights)
				scores = extractProviderIDs(knowledgebase.Providers.Scores)
			}

			kbConfig = map[string]interface{}{
				"features":    knowledgebase.Config.Features,
				"chunkings":   chunkings,
				"embeddings":  embeddings,
				"converters":  converters,
				"extractions": extractions,
				"fetchers":    fetchers,
				"searchers":   searchers,
				"rerankers":   rerankers,
				"votes":       votes,
				"weights":     weights,
				"scores":      scores,
				"uploader":    knowledgebase.Config.Uploader,
			}
		}
	}

	xgenSetting := map[string]interface{}{
		"name":        share.App.Name,
		"description": share.App.Description,
		"developer":   share.App.Developer,
		"version":     share.App.Version,
		"yao": map[string]interface{}{
			"version":   share.VERSION,
			"prversion": share.PRVERSION,
		},
		"cui": map[string]interface{}{
			"version":   share.CUI,
			"prversion": share.PRCUI,
		},
		"theme":     share.App.Theme,
		"lang":      share.App.Lang,
		"mode":      mode,
		"apiPrefix": "__yao",
		"token":     share.App.Token,
		"optional":  share.App.Optional,
		"login":     xgenLogin,
		"agent":     agentConfig,
		"tools":     toolsConfig,
		"openapi":   openapiConfig,
		"kb":        kbConfig,
	}

	apiBase := appGetAPIBase()
	if share.App.Logo != "" {
		logo := share.App.Logo
		if strings.HasPrefix(logo, "/api/") {
			logo = apiBase + strings.TrimPrefix(logo, "/api")
		}
		xgenSetting["logo"] = logo
	}

	if share.App.Favicon != "" {
		favicon := share.App.Favicon
		if strings.HasPrefix(favicon, "/api/") {
			favicon = apiBase + strings.TrimPrefix(favicon, "/api")
		}
		xgenSetting["favicon"] = favicon
	}

	setting, err := i18n.Trans(session.Lang(process, config.Conf.Lang), []string{"app.app"}, xgenSetting)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	setting.(map[string]interface{})["sid"] = sid
	return setting.(map[string]interface{})
}
