package studio

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/gou/fs/dsl"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

var shutdownSignal = make(chan bool, 1)
var dfs fs.FileSystem
var scripts = map[string][]byte{}

type cfunc struct {
	Method string        `json:"method"`
	Args   []interface{} `json:"args,omitempty"`
}

// Start start the studio api server
func Start(cfg config.Config) (err error) {

	// recive interrupt signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	errCh := make(chan error, 1)

	// Set router
	router := gin.New()
	setRouter(router)

	// Server setting
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Studio.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// Listen
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	defer func() {
		log.Info("[Studio] %s Close Serve", addr)
		err = srv.Close()
		if err != nil {
			log.Error("[Studio] Close Serve Error (%v)", err)
		}
	}()

	// start serve
	go func() {
		log.Info("[Studio] Starting: %s", addr)
		if err := srv.Serve(l); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {

	case <-shutdownSignal:
		log.Info("[Studio] %s Exit (Manual)", addr)
		return err

	case <-interrupt:
		log.Info("[Studio] %s Exit (Interrupt) ", addr)
		return err

	case err := <-errCh:
		log.Error("[Studio] %s Error (%v)", addr, err)
		return err
	}
}

// Stop stop the studio api server
func Stop() {
	shutdownSignal <- true
}

// Load studio config
func Load(cfg config.Config) error {
	err := loadDSL(cfg)
	if err != nil {
		return err
	}
	return loadScripts(cfg)
}

func loadDSL(cfg config.Config) error {

	root, err := filepath.Abs(cfg.Root)
	if err != nil {
		return err
	}

	scriptRoot := filepath.Join(root, "scripts")
	dataRoot := filepath.Join(root, "data")
	dslDenyList := []string{scriptRoot, dataRoot}
	dfs = dsl.New(root).DenyAbs(dslDenyList...)
	return nil
}

func loadScripts(cfg config.Config) error {
	root, err := filepath.Abs(cfg.Root)
	if err != nil {
		return err
	}

	studioRoot := filepath.Join(root, "studio")
	return loadScriptFrom(studioRoot)
}

// Load script From dir
func loadScriptFrom(dir string) error {

	if share.DirNotExists(dir) {
		log.Warn("[Studio] Load %s does not exists", dir)
		return nil
	}

	messages := []string{}
	err := share.Walk(dir, ".js", func(root, filename string) {
		name := share.SpecName(root, filename)
		_, err := v8.LoadRoot(filename, name)
		if err != nil {
			messages = append(messages, err.Error())
		}
	})

	if len(messages) > 0 {
		return fmt.Errorf("[Studio] Load %s", strings.Join(messages, ";"))
	}
	return err
}
