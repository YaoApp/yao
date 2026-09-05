package workspace_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	openapiWS "github.com/yaoapp/yao/openapi/workspace"
	ws "github.com/yaoapp/yao/workspace"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestCheckWSOwner_EmptyOwner(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)

	result := openapiWS.ExportCheckWSOwner(c, &ws.Workspace{Owner: "alice"}, "")
	assert.False(t, result, "empty owner must be denied")
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCheckWSOwner_Mismatch(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)

	result := openapiWS.ExportCheckWSOwner(c, &ws.Workspace{Owner: "alice"}, "bob")
	assert.False(t, result, "mismatched owner must be denied")
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCheckWSOwner_Match(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)

	result := openapiWS.ExportCheckWSOwner(c, &ws.Workspace{Owner: "alice"}, "alice")
	assert.True(t, result, "matching owner must be allowed")
}

func TestCheckWSOwner_UnownedWorkspace(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)

	result := openapiWS.ExportCheckWSOwner(c, &ws.Workspace{Owner: ""}, "alice")
	assert.True(t, result, "unowned workspace allows any authenticated user")
}
