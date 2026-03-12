package cmd

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/gou/mcp"
	"github.com/yaoapp/gou/plugin"
	"github.com/yaoapp/gou/schedule"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/gou/task"
	"github.com/yaoapp/gou/websocket"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	yaogrpc "github.com/yaoapp/yao/grpc"
	_ "github.com/yaoapp/yao/grpc/auth"
	sandboxhandler "github.com/yaoapp/yao/grpc/sandbox"
	"github.com/yaoapp/yao/openapi"
	sandbox "github.com/yaoapp/yao/sandbox/v2"
	ischedule "github.com/yaoapp/yao/schedule"
	"github.com/yaoapp/yao/service"
	"github.com/yaoapp/yao/setup"
	"github.com/yaoapp/yao/share"

	itask "github.com/yaoapp/yao/task"
)

var startDebug = false
var startDisableWatching = false

var startCmd = &cobra.Command{
	Use:   "start",
	Short: L("Start Engine"),
	Long:  L("Start Engine"),
	Run: func(cmd *cobra.Command, args []string) {

		defer share.SessionStop()
		defer plugin.KillAll()

		// recive interrupt signal
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

		Boot()

		// Setup
		isnew := false

		// Check if current directory is a Yao app root
		if !setup.IsYaoApp(config.Conf.Root) {

			// Check if we're inside a Yao app (subdirectory)
			if setup.InYaoApp(config.Conf.Root) {
				fmt.Println(color.RedString(L("Please run the command in the root directory of project")))
				os.Exit(1)
			}

			// Not in a Yao app, check if empty to install
			if setup.IsEmptyDir(config.Conf.Root) {
				// Install the init app
				if err := install(); err != nil {
					fmt.Println(color.RedString(L("Install: %s"), err.Error()))
					os.Exit(1)
				}
				isnew = true
			} else {
				// Directory not empty and no app.yao
				fmt.Println(color.RedString("The app.yao file is missing"))
				os.Exit(1)
			}
		}

		// force debug
		if startDebug {
			config.Development()
		}

		// load the application engine
		loadWarnings, err := engine.Load(config.Conf, engine.LoadOption{
			Action: "start",
		})
		if err != nil {
			fmt.Println(color.RedString(L("Load: %s"), err.Error()))
			os.Exit(1)
		}

		port := fmt.Sprintf(":%d", config.Conf.Port)
		if port == ":80" {
			port = ""
		}

		// variables for the service
		fs, err := fs.Get("system")
		if err != nil {
			fmt.Println(color.RedString(L("FileSystem: %s"), err.Error()))
			os.Exit(1)
		}

		mode := config.Conf.Mode
		host := config.Conf.Host
		dataRoot := fs.Root()
		runtimeMode := config.Conf.Runtime.Mode

		fmt.Println(color.WhiteString("\n--------------------------------------------"))
		fmt.Println(
			color.WhiteString(strings.TrimPrefix(share.App.Name, "::")),
			color.WhiteString(share.App.Version),
			mode,
		)
		fmt.Println(color.WhiteString("--------------------------------------------"))
		if !share.BUILDIN {
			root, _ := filepath.Abs(config.Conf.Root)
			fmt.Println(color.WhiteString(L("Root")), color.GreenString(" %s", root))
		}

		fmt.Println(color.WhiteString(L("Runtime")), color.GreenString(" %s", runtimeMode))
		fmt.Println(color.WhiteString(L("Data")), color.GreenString(" %s", dataRoot))
		fmt.Println(color.WhiteString(L("Listening")), color.GreenString(" %s:%d", config.Conf.Host, config.Conf.Port))

		// print the messages under the development mode
		if mode == "development" {
			printApis(false)
			printTasks(false)
			printSchedules(false)
			printConnectors(false)
			printStores(false)
			printMCPs(false)
		}

		root, _ := adminRoot()
		endpoints := []setup.Endpoint{{URL: fmt.Sprintf("http://%s%s", "127.0.0.1", port), Interface: "localhost"}}
		switch host {
		case "0.0.0.0":
			if values, err := setup.Endpoints(config.Conf); err == nil {
				endpoints = append(endpoints, values...)
			}
		case "127.0.0.1":
			// Localhost only
		default:
			matched := false
			endpoints = []setup.Endpoint{}
			if values, err := setup.Endpoints(config.Conf); err == nil {
				for _, value := range values {
					if strings.HasPrefix(value.URL, fmt.Sprintf("http://%s:", host)) {
						endpoints = append(endpoints, value)
						matched = true
					}
				}
			}
			if !matched {
				fmt.Println(color.RedString(L("Host %s not found"), host))
				os.Exit(1)
			}
		}

		// Print welcome message for the new application
		if isnew {
			printWelcome()
		}

		// Start Tasks
		itask.Start()
		defer itask.Stop()

		// Start Schedules
		ischedule.Start()
		defer ischedule.Stop()

		// Pre-flight: detect port conflicts before attempting to start servers.
		if occupied, proc := portOccupied(config.Conf.Host, config.Conf.Port); occupied {
			fmt.Println(color.RedString(L("Fatal: HTTP port %d is already in use%s"), config.Conf.Port, proc))
			return
		}
		if strings.ToLower(config.Conf.GRPC.Enabled) != "off" {
			for _, h := range yaogrpc.ExpandHosts(config.Conf.GRPC.Host) {
				if occupied, proc := portOccupied(h, config.Conf.GRPC.Port); occupied {
					fmt.Println(color.RedString(L("Fatal: gRPC port %d is already in use%s"), config.Conf.GRPC.Port, proc))
					return
				}
			}
		}

		// Wire gRPC heartbeat → sandbox Manager so container liveness is tracked.
		yaogrpc.SetSandboxOnBeat(func(data *sandboxhandler.HeartbeatData) string {
			sandbox.M().Heartbeat(data.SandboxID, true, int(data.RunningProcs))
			return "ok"
		})

		// Start all servers (gRPC + HTTP) as a single unit.
		// Start() blocks until HTTP port is bound (READY) or returns error.
		svc, err := service.Start(config.Conf, service.ServerHooks{
			Start: yaogrpc.StartServer,
			Stop:  yaogrpc.Stop,
			Addrs: yaogrpc.Addr,
		})
		if err != nil {
			fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
			return
		}

		// Access Points (printed after servers are up so addresses are known)
		fmt.Println(color.WhiteString("\n---------------------------------"))
		fmt.Println(color.WhiteString(L("Access Points")))
		fmt.Println(color.WhiteString("---------------------------------"))

		if grpcAddrs := svc.HookAddrs(); len(grpcAddrs) > 0 {
			fmt.Println(color.CyanString("\ngRPC"))
			fmt.Println(color.WhiteString("--------------------------"))
			for _, addr := range grpcAddrs {
				fmt.Println(color.WhiteString(L("Server")), color.GreenString(" %s", addr))
			}
		}

		apiRoot := "/api"
		if openapi.Server != nil {
			apiRoot = openapi.Server.Config.BaseURL
		}
		for _, endpoint := range endpoints {
			fmt.Println(color.CyanString("\n%s", endpoint.Interface))
			fmt.Println(color.WhiteString("--------------------------"))
			fmt.Println(color.WhiteString(L("Website")), color.GreenString(" %s", endpoint.URL))
			fmt.Println(color.WhiteString(L("Dashboard")), color.GreenString(" %s/%s/auth/entry", endpoint.URL, strings.Trim(root, "/")))
			if openapi.Server != nil {
				fmt.Println(color.WhiteString(L("OpenAPI")), color.GreenString(" %s%s", endpoint.URL, apiRoot))
			} else {
				fmt.Println(color.WhiteString(L("API")), color.GreenString(" %s%s", endpoint.URL, apiRoot))
			}
		}
		fmt.Println("")

		// Start watching
		watchDone := make(chan uint8, 1)
		if mode == "development" && !startDisableWatching {
			go svc.Watch(watchDone)
		}

		// Print the messages under the production mode
		if mode == "production" {
			printApis(true)
			printTasks(true)
			printSchedules(true)
			printConnectors(true)
			printStores(true)
			printMCPs(true)
		}

		// Print the warnings
		if len(loadWarnings) > 0 {
			fmt.Println(color.YellowString("---------------------------------"))
			fmt.Println(color.YellowString(L("Warnings")))
			fmt.Println(color.YellowString("---------------------------------"))
			for _, warning := range loadWarnings {
				fmt.Println(color.YellowString("[%s] %s", warning.Widget, warning.Error))
			}
			fmt.Printf("\n")
		}

		fmt.Println(color.GreenString(L("Server is up and running...")))
		fmt.Println(color.GreenString("Ctrl+C to stop"))

		for {
			select {
			case <-interrupt:
				fmt.Println(color.WhiteString("\nShutting down..."))
				svc.Stop()
				fmt.Println(color.GreenString(L("✨Exited successfully!")))
				watchDone <- 1
				return
			}
		}
	},
}

func install() error {
	// Copy the app source files from the binary
	err := setup.Install(config.Conf.Root)
	if err != nil {
		return err
	}

	// Reload the application engine
	Boot()

	// load the application engine
	loadWarnings, err := engine.Load(config.Conf, engine.LoadOption{Action: "start"})
	if err != nil {
		return err
	}

	// Print the warnings
	if len(loadWarnings) > 0 {
		for _, warning := range loadWarnings {
			fmt.Println(color.YellowString("[%s] %s", warning.Widget, warning.Error))
		}
		fmt.Printf("\n\n")
	}

	err = setup.Initialize(config.Conf.Root, config.Conf)
	if err != nil {
		return err
	}

	return nil
}

func adminRoot() (string, int) {
	adminRoot := "/yao/"
	if share.App.AdminRoot != "" {
		root := strings.TrimPrefix(share.App.AdminRoot, "/")
		root = strings.TrimSuffix(root, "/")
		adminRoot = fmt.Sprintf("/%s/", root)
	}
	adminRootLen := len(adminRoot)
	return adminRoot, adminRootLen
}

func printWelcome() {
	fmt.Println(color.CyanString("\n---------------------------------"))
	fmt.Println(color.CyanString(L("🎉 Welcome to Yao 🎉 ")))
	fmt.Println(color.CyanString("---------------------------------"))
	fmt.Println(color.WhiteString("📚 Documentation:        "), color.CyanString("https://yaoapps.com/docs"))
	fmt.Println(color.WhiteString("🏡 Join Yao Community:   "), color.CyanString("https://yaoapps.com/community"))
	fmt.Println(color.WhiteString("🤖 Build Your Digital Workforce:"), color.CyanString("https://yaoagents.com"))
	fmt.Println("")
}

func printConnectors(silent bool) {

	if len(connector.Connectors) == 0 {
		return
	}

	if silent {
		for name := range connector.Connectors {
			log.Info("[Connector] %s loaded", name)
		}
		return
	}

	fmt.Println(color.WhiteString("\n---------------------------------"))
	fmt.Println(color.WhiteString(L("Connectors List (%d)"), len(connector.Connectors)))
	fmt.Println(color.WhiteString("---------------------------------"))
	for name := range connector.Connectors {
		fmt.Print(color.CyanString("[Connector]"))
		fmt.Print(color.WhiteString(" %s\t loaded\n", name))
	}
}

func printStores(silent bool) {
	if len(store.Pools) == 0 {
		return
	}

	if silent {
		for name := range store.Pools {
			log.Info("[Store] %s loaded", name)
		}
		return
	}

	fmt.Println(color.WhiteString("\n---------------------------------"))
	fmt.Println(color.WhiteString(L("Stores List (%d)"), len(store.Pools)))
	fmt.Println(color.WhiteString("---------------------------------"))
	for name := range store.Pools {
		fmt.Print(color.CyanString("[Store]"))
		fmt.Print(color.WhiteString(" %s\t loaded\n", name))
	}
}

func printSchedules(silent bool) {

	if len(schedule.Schedules) == 0 {
		return
	}

	if silent {
		for name, sch := range schedule.Schedules {
			process := fmt.Sprintf("Process: %s", sch.Process)
			if sch.TaskName != "" {
				process = fmt.Sprintf("Task: %s", sch.TaskName)
			}
			log.Info("[Schedule] %s %s %s %s", sch.Schedule, name, sch.Name, process)
		}
		return
	}

	fmt.Println(color.WhiteString("\n---------------------------------"))
	fmt.Println(color.WhiteString(L("Schedules List (%d)"), len(schedule.Schedules)))
	fmt.Println(color.WhiteString("---------------------------------"))
	for name, sch := range schedule.Schedules {
		process := fmt.Sprintf("Process: %s", sch.Process)
		if sch.TaskName != "" {
			process = fmt.Sprintf("Task: %s", sch.TaskName)
		}
		fmt.Print(color.CyanString("[Schedule] %s %s", sch.Schedule, name))
		fmt.Print(color.WhiteString("\t%s\t%s\n", sch.Name, process))
	}
}

func printTasks(silent bool) {

	if len(task.Tasks) == 0 {
		return
	}

	if silent {
		for _, t := range task.Tasks {
			log.Info("[Task] %s workers:%d", t.Option.Name, t.Option.WorkerNums)
		}
		return
	}

	fmt.Println(color.WhiteString("\n---------------------------------"))
	fmt.Println(color.WhiteString(L("Tasks List (%d)"), len(task.Tasks)))
	fmt.Println(color.WhiteString("---------------------------------"))
	for _, t := range task.Tasks {
		fmt.Print(color.CyanString("[Task] %s", t.Option.Name))
		fmt.Print(color.WhiteString("\t workers: %d\n", t.Option.WorkerNums))
	}
}

func printApis(silent bool) {
	// Determine API root based on OpenAPI mode
	apiRoot := "/api"
	if openapi.Server != nil {
		apiRoot = openapi.Server.Config.BaseURL
	}

	if silent {
		for _, api := range api.APIs {
			if len(api.HTTP.Paths) <= 0 {
				continue
			}
			log.Info("[API] %s(%d)", api.ID, len(api.HTTP.Paths))
			for _, p := range api.HTTP.Paths {
				log.Info("%s %s %s", p.Method, filepath.Join(apiRoot, api.HTTP.Group, p.Path), p.Process)
			}
		}
		for name, upgrader := range websocket.Upgraders { // WebSocket
			log.Info("[WebSocket] GET  /websocket/%s process:%s", name, upgrader.Process)
		}
		return
	}

	// Skip detailed API list when OpenAPI is enabled
	if openapi.Server != nil {
		return
	}

	fmt.Println(color.WhiteString("\n---------------------------------"))
	fmt.Println(color.WhiteString(L("APIs List")))
	fmt.Println(color.WhiteString("---------------------------------"))

	for _, api := range api.APIs { // API info
		if len(api.HTTP.Paths) <= 0 {
			continue
		}

		deprecated := ""
		if strings.HasPrefix(api.ID, "xiang.") {
			deprecated = " WILL BE DEPRECATED"
		}

		fmt.Printf("%s%s\n", color.CyanString("\n%s(%d)", api.ID, len(api.HTTP.Paths)), color.RedString(deprecated))
		for _, p := range api.HTTP.Paths {
			fmt.Println(
				colorMehtod(p.Method),
				color.WhiteString(filepath.Join(apiRoot, api.HTTP.Group, p.Path)),
				"\tprocess:", p.Process)
		}
	}

	if len(websocket.Upgraders) > 0 {
		fmt.Print(color.CyanString(fmt.Sprintf("\n%s(%d)\n", "WebSocket", len(websocket.Upgraders))))
		for name, upgrader := range websocket.Upgraders { // WebSocket
			fmt.Println(
				colorMehtod("GET"),
				color.WhiteString(filepath.Join("/websocket", name)),
				"\tprocess:", upgrader.Process)
		}
	}
}

func printMCPs(silent bool) {
	clients := mcp.ListClients()
	if len(clients) == 0 {
		return
	}

	if silent {
		for _, clientID := range clients {
			log.Info("[MCP] %s loaded", clientID)
		}
		return
	}

	// Separate agent MCPs from standard MCPs by Type field
	agentClients := []string{}
	standardClients := []string{}
	for _, clientID := range clients {
		client, err := mcp.Select(clientID)
		if err != nil {
			standardClients = append(standardClients, clientID)
			continue
		}

		info := client.Info()
		if info != nil && info.Type == "agent" {
			agentClients = append(agentClients, clientID)
		} else {
			standardClients = append(standardClients, clientID)
		}
	}

	fmt.Println(color.WhiteString("\n---------------------------------"))
	fmt.Println(color.WhiteString(L("MCP Clients List (%d)"), len(clients)))
	fmt.Println(color.WhiteString("---------------------------------"))

	if len(standardClients) > 0 {
		fmt.Println(color.WhiteString("\n%s (%d)", "Standard MCPs", len(standardClients)))
		fmt.Println(color.WhiteString("--------------------------"))
		for _, clientID := range standardClients {
			client, err := mcp.Select(clientID)
			if err != nil {
				fmt.Print(color.CyanString("[MCP] %s", clientID))
				fmt.Print(color.WhiteString("\tloaded\n"))
				continue
			}

			info := client.Info()
			transport := "unknown"
			label := clientID
			if info != nil {
				if info.Transport != "" {
					transport = string(info.Transport)
				}
				if info.Label != "" {
					label = info.Label
				}
			}

			fmt.Print(color.CyanString("[MCP] %s", label))
			fmt.Print(color.WhiteString("\t%s\tid: %s", transport, clientID))

			// Only show tools count for process transport
			if transport == "process" {
				toolsCount := 0
				mapping, err := mcp.GetClientMapping(clientID)
				if err == nil && mapping.Tools != nil {
					toolsCount = len(mapping.Tools)
				}
				fmt.Print(color.WhiteString("\ttools: %d", toolsCount))
			}
			fmt.Print("\n")
		}
	}

	if len(agentClients) > 0 {
		fmt.Println(color.WhiteString("\n%s (%d)", "Agent MCPs", len(agentClients)))
		fmt.Println(color.WhiteString("--------------------------"))
		for _, clientID := range agentClients {
			client, err := mcp.Select(clientID)
			if err != nil {
				fmt.Print(color.CyanString("[MCP] %s", clientID))
				fmt.Print(color.WhiteString("\tloaded\n"))
				continue
			}

			info := client.Info()
			transport := "unknown"
			label := clientID
			if info != nil {
				if info.Transport != "" {
					transport = string(info.Transport)
				}
				if info.Label != "" {
					label = info.Label
				}
			}

			fmt.Print(color.CyanString("[MCP] %s", label))
			fmt.Print(color.WhiteString("\t%s\tid: %s", transport, clientID))

			// Only show tools count for process transport
			if transport == "process" {
				toolsCount := 0
				mapping, err := mcp.GetClientMapping(clientID)
				if err == nil && mapping.Tools != nil {
					toolsCount = len(mapping.Tools)
				}
				fmt.Print(color.WhiteString("\ttools: %d", toolsCount))
			}
			fmt.Print("\n")
		}
	}
}

func colorMehtod(method string) string {
	method = strings.ToUpper(method)
	switch method {
	case "GET":
		return color.GreenString("GET")
	case "POST":
		return color.YellowString("POST")
	default:
		return color.WhiteString(method)
	}
}

// portOccupied probes whether host:port is already bound.
// Returns (true, " (pid XXXX)") when occupied, (false, "") otherwise.
func portOccupied(host string, port int) (bool, string) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return true, fmt.Sprintf(" (%s)", err.Error())
	}
	ln.Close()
	return false, ""
}

func init() {
	startCmd.PersistentFlags().BoolVarP(&startDebug, "debug", "", false, L("Development mode"))
	startCmd.PersistentFlags().BoolVarP(&startDisableWatching, "disable-watching", "", false, L("Disable watching"))
}
