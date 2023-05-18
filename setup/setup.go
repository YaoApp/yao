package setup

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/data"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/widgets/app"
)

// SetupPort setup port
var SetupPort string = "5099"

// XGenSetupServer XGen Setup
var XGenSetupServer http.Handler = http.FileServer(data.Setup())

// Done check done
var Done = make(chan bool, 1)

// shutdown check done
var shutdown = make(chan bool, 1)

// Canceled check if setup is canceld
var Canceled = make(chan bool, 1)

// Start start the studio api server
func Start() (err error) {

	// recive interrupt signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	errCh := make(chan error, 1)

	// Set router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	if os.Getenv("YAO_SETUP_DEV") != "" {
		fmt.Println(color.WhiteString("\nSETUP DEV: %s\n", os.Getenv("YAO_SETUP_DEV")))
	}

	router.Use(func(c *gin.Context) {
		length := len(c.Request.URL.Path)
		if length >= 5 && c.Request.URL.Path[0:5] == "/api/" {
			c.Next()
			return
		}

		if os.Getenv("YAO_SETUP_DEV") != "" {
			root, err := filepath.Abs(os.Getenv("YAO_SETUP_DEV"))
			if err != nil {
				printError("%s", err)
			}

			static := http.FileServer(http.Dir(root))
			static.ServeHTTP(c.Writer, c.Request)
			return
		}

		// Setup Pages
		XGenSetupServer.ServeHTTP(c.Writer, c.Request)
		return
	}, gin.CustomRecovery(recovered))

	router.POST("/api/__yao/app/check", runCheck)
	router.POST("/api/__yao/app/setup", runSetup)

	// Server setting
	addr := fmt.Sprintf(":%s", SetupPort)

	// Listen
	l, err := net.Listen("tcp4", addr)
	if err != nil {
		addr = ":0"
		l, err = net.Listen("tcp4", addr)
		if err != nil {
			return err
		}
	}

	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}
	defer func() {
		// printInfo("[Setup] %s Close Serve", addr)
		err = srv.Close()
		if err != nil {
			printError("[Setup]  Error (%v)", err)
		}
	}()

	// start serve
	go func() {
		// printInfo("[Setup] Starting: %s", addr)
		if err := srv.Serve(l); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	welcome(l)

	select {

	case <-shutdown:
		printInfo("Setup Shutdown")
		return err

	case <-interrupt:
		printInfo("Setup Interrupt")
		Canceled <- true
		return err

	case err := <-errCh:
		printError("Setup Error: ", err)
		Canceled <- true
		return err
	}
}

// Complete stop the studio api server
func Complete() {
	engine.Unload()
	Done <- true
}

// Stop stop the studio api server
func Stop() {
	shutdown <- true
}

// AdminURL get admin url
func AdminURL(cfg config.Config) ([]string, error) {

	urls, err := URLs(cfg)
	if err != nil {
		return nil, err
	}

	adminRoot := "yao"
	if app.Setting.AdminRoot != "" {
		adminRoot = app.Setting.AdminRoot
	}
	adminRoot = strings.Trim(adminRoot, "/")

	for i := range urls {
		urls[i] = fmt.Sprintf("%s/%s/", urls[i], adminRoot)
	}
	return urls, nil
}

// URLs get admin url
func URLs(cfg config.Config) ([]string, error) {

	ips, err := Ips()
	if err != nil {
		return nil, err
	}

	for i := range ips {
		ips[i] = fmt.Sprintf("http://%s:%d", ips[i], cfg.Port)
	}

	return ips, nil
}

func welcome(l net.Listener) {
	fmt.Println(color.WhiteString("---------------------------------"))
	fmt.Println(color.WhiteString("Yao Application Setup v%s", share.VERSION))
	fmt.Println(color.WhiteString("---------------------------------"))

	ips, err := Ips()
	if err != nil {
		printError("Error: ", err.Error())
	}

	addr := strings.Split(l.Addr().String(), ":")
	if len(addr) != 2 {
		printError("Error: can't get port")
	}

	port := addr[1]
	fmt.Println(color.WhiteString("\nOpen URL in the browser to continue:\n"))
	for _, ip := range ips {
		printInfo("http://%s:%s", ip, port)
	}

	fmt.Println()
	SetupPort = port
}

func printError(message string, args ...interface{}) {
	fmt.Println(color.RedString(message, args...))
	os.Exit(1)
}

func printInfo(message string, args ...interface{}) {
	fmt.Println(color.GreenString(message, args...))
}

// Ips get the local ip list
func Ips() ([]string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	iplist := []string{"127.0.0.1"}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				iplist = append(iplist, ipnet.IP.String())
			}
		}
	}
	return iplist, nil
}
