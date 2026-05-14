package agent

import (
	"net/http"

	"github.com/gin-gonic/gin"
	sandboxv2 "github.com/yaoapp/yao/agent/sandbox/v2"
)

// ListImages returns the curated list of preset sandbox images.
// GET /api/v1/agent/images
func ListImages(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"images": sandboxv2.PresetImages})
}
