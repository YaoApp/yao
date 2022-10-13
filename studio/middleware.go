package studio

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xun"
)

func hdRecovered(c *gin.Context, recovered interface{}) {

	var code = http.StatusInternalServerError

	if err, ok := recovered.(string); ok {
		c.JSON(code, xun.R{
			"code":    code,
			"message": fmt.Sprintf("%s", err),
		})
	} else if err, ok := recovered.(exception.Exception); ok {
		code = err.Code
		c.JSON(code, xun.R{
			"code":    code,
			"message": err.Message,
		})
	} else if err, ok := recovered.(*exception.Exception); ok {
		code = err.Code
		c.JSON(code, xun.R{
			"code":    code,
			"message": err.Message,
		})
	} else {
		c.JSON(code, xun.R{
			"code":    code,
			"message": fmt.Sprintf("%v", recovered),
		})
	}

	c.AbortWithStatus(code)
}
