package studio

import (
	"fmt"

	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/yao/config"
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
	return fmt.Errorf("studio is deprecated")

	// // recive interrupt signal
	// interrupt := make(chan os.Signal, 1)
	// signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	// errCh := make(chan error, 1)

	// // Set router
	// router := gin.New()
	// setRouter(router)

	// // Server setting
	// addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Studio.Port)
	// srv := &http.Server{
	// 	Addr:    addr,
	// 	Handler: router,
	// }

	// // Listen
	// l, err := net.Listen("tcp", addr)
	// if err != nil {
	// 	return err
	// }

	// defer func() {
	// 	log.Info("[Studio] %s Close Serve", addr)
	// 	err = srv.Close()
	// 	if err != nil {
	// 		log.Error("[Studio] Close Serve Error (%v)", err)
	// 	}
	// }()

	// // start serve
	// go func() {
	// 	log.Info("[Studio] Starting: %s", addr)
	// 	if err := srv.Serve(l); err != nil && err != http.ErrServerClosed {
	// 		errCh <- err
	// 	}
	// }()

	// select {

	// case <-shutdownSignal:
	// 	log.Info("[Studio] %s Exit (Manual)", addr)
	// 	return err

	// case <-interrupt:
	// 	log.Info("[Studio] %s Exit (Interrupt) ", addr)
	// 	return err

	// case err := <-errCh:
	// 	log.Error("[Studio] %s Error (%v)", addr, err)
	// 	return err
	// }
}

// Stop stop the studio api server
func Stop() {
	shutdownSignal <- true
}

// Load studio config
func Load(cfg config.Config) error {
	return fmt.Errorf("studio is deprecated")

	// err := loadDSL(cfg)
	// if err != nil {
	// 	return err
	// }
	// return loadScripts()
}

func loadDSL(cfg config.Config) error {
	return fmt.Errorf("studio is deprecated")
	// dslDenyList := []string{cfg.DataRoot}
	// dfs = dsl.New(cfg.AppSource).DenyAbs(dslDenyList...)
	// return nil
}

func loadScripts() error {
	return fmt.Errorf("studio is deprecated")
	// exts := []string{"*.js"}
	// return application.App.Walk("studio", func(root, file string, isdir bool) error {
	// 	if isdir {
	// 		return nil
	// 	}
	// 	_, err := v8.LoadRoot(file, share.ID(root, file))
	// 	return err
	// }, exts...)
}
