package service

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/server/http"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/openapi"
)

// Watch the application code change for hot update
func Watch(srv *http.Server, interrupt chan uint8) (err error) {

	if application.App == nil {
		return fmt.Errorf("Application is not initialized")
	}

	return application.App.Watch(func(event, name string) {
		if strings.Contains(event, "CHMOD") {
			return
		}

		// Reload
		err = engine.Reload(config.Conf, engine.LoadOption{Action: "watch"})
		if err != nil {
			fmt.Println(color.RedString("[Watch] Reload: %s", err.Error()))
			return
		}
		fmt.Println(color.GreenString("[Watch] Reload Completed"))

		// Model
		if strings.HasPrefix(name, "/models") {
			fmt.Println(color.GreenString("[Watch] Model: %s changed (Please run yao migrate manually)", name))
		}

		// API changes: hot reload or restart
		if strings.HasPrefix(name, "/apis") {
			if openapi.Server != nil {
				// OpenAPI mode: hot reload (no server restart needed)
				err = ReloadAPIs()
				if err != nil {
					fmt.Println(color.RedString("[Watch] Reload APIs: %s", err.Error()))
					return
				}
				fmt.Println(color.GreenString("[Watch] APIs Reloaded"))
			} else {
				// Traditional mode: restart server
				err = Restart(srv, config.Conf)
				if err != nil {
					fmt.Println(color.RedString("[Watch] Restart: %s", err.Error()))
					return
				}
				fmt.Println(color.GreenString("[Watch] Restart Completed"))
			}
		}

	}, interrupt)
}
