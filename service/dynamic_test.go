package service_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/service"
)

func TestDynamicAPIHandler(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)

	cfg := config.Conf
	cfg.Port = 0
	_, err := engine.Load(cfg, engine.LoadOption{})
	if err != nil {
		t.Fatal(err)
	}

	// Temporarily disable OpenAPI for this test
	savedOpenAPIServer := openapi.Server
	openapi.Server = nil
	defer func() { openapi.Server = savedOpenAPIServer }()

	// Set up guards
	api.SetGuards(service.Guards)

	// Load and build route table
	api.BuildRouteTable()

	// Create test router
	router := gin.New()
	router.Any("/api/*path", service.DynamicAPIHandler)

	// Test: API not found
	response := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/nonexistent", nil)
	router.ServeHTTP(response, req)
	assert.Equal(t, 404, response.Code)

	// Test: Exact match (app setting API)
	response = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/__yao/app/setting", nil)
	router.ServeHTTP(response, req)
	// Note: This may return 403 if guard is not satisfied, which is expected
	assert.True(t, response.Code == 200 || response.Code == 403)
}

func TestReloadAPIs(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)

	cfg := config.Conf
	cfg.Port = 0
	_, err := engine.Load(cfg, engine.LoadOption{})
	if err != nil {
		t.Fatal(err)
	}

	// Initial build
	api.BuildRouteTable()

	// Reload should not error
	err = service.ReloadAPIs()
	assert.NoError(t, err)
}

func TestGuardSelection(t *testing.T) {
	// Verify traditional Guards exist
	assert.NotNil(t, service.Guards["bearer-jwt"])
	assert.NotNil(t, service.Guards["cookie-jwt"])
	assert.NotNil(t, service.Guards["cross-origin"])

	// Note: OpenAPIGuards() requires oauth.OAuth to be initialized,
	// which happens during engine load with OpenAPI config.
	// The guard mapping is tested implicitly through integration tests.
}
