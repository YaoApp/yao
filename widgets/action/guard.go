package action

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/kun/log"
)

// UseGuard using the guard in action
func (p *Process) UseGuard(c *gin.Context, id string) error {
	guards := strings.Split(p.Guard, ",")
	for _, guard := range guards {
		guard = strings.TrimSpace(guard)
		log.Trace("Widget: %s Guard: %s", id, guard)
		if guard == "-" {
			return nil
		}

		if guard != "" {
			if middleware, has := api.HTTPGuards[guard]; has {
				middleware(c)
				continue
			}
			api.ProcessGuard(guard)(c)
		}
	}
	return nil
}
