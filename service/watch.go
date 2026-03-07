package service

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/openapi"
)

// Watch the application code change for hot update
func watch(svc *Service, interrupt chan uint8) error {

	if application.App == nil {
		return fmt.Errorf("Application is not initialized")
	}

	return application.App.Watch(func(event, name string) {
		if strings.Contains(event, "CHMOD") {
			return
		}

		err := engine.Reload(config.Conf, engine.LoadOption{Action: "watch"})
		if err != nil {
			fmt.Println(color.RedString("[Watch] Reload: %s", err.Error()))
			return
		}
		fmt.Println(color.GreenString("[Watch] Reload Completed"))

		if strings.HasPrefix(name, "/models") {
			fmt.Println(color.GreenString("[Watch] Model: %s changed (Please run yao migrate manually)", name))
		}

		if strings.HasPrefix(name, "/apis") {
			if openapi.Server != nil {
				err = ReloadAPIs()
				if err != nil {
					fmt.Println(color.RedString("[Watch] Reload APIs: %s", err.Error()))
					return
				}
				fmt.Println(color.GreenString("[Watch] APIs Reloaded"))
			} else {
				err = Restart(svc, config.Conf)
				if err != nil {
					fmt.Println(color.RedString("[Watch] Restart: %s", err.Error()))
					return
				}
				fmt.Println(color.GreenString("[Watch] Restart Completed"))
			}
		}

	}, interrupt)
}
