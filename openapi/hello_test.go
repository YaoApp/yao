package openapi

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/share"
)

func TestHelloWorldHello(t *testing.T) {
	serverURL := Prepare(t)
	defer Clean()

	// Get base URL from server config
	baseURL := ""
	if Server != nil && Server.Config != nil {
		baseURL = Server.Config.BaseURL
	}

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{
			name:   "GET hello endpoint",
			method: "GET",
			path:   baseURL + "/helloworld/hello",
		},
		{
			name:   "POST hello endpoint",
			method: "POST",
			path:   baseURL + "/helloworld/hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make HTTP request
			var resp *http.Response
			var err error

			if tt.method == "GET" {
				resp, err = http.Get(serverURL + tt.path)
			} else {
				resp, err = http.Post(serverURL+tt.path, "application/json", nil)
			}

			assert.NoError(t, err)
			assert.NotNil(t, resp)
			defer resp.Body.Close()

			// Check status code
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Parse JSON response
			var response map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&response)
			assert.NoError(t, err)

			// Verify response structure and content
			assert.Equal(t, "HELLO, WORLD", response["MESSAGE"])
			assert.NotEmpty(t, response["SERVER_TIME"])
			assert.Equal(t, share.VERSION, response["VERSION"])
			assert.Equal(t, share.PRVERSION, response["PRVERSION"])
			assert.Equal(t, share.CUI, response["CUI"])
			assert.Equal(t, share.PRCUI, response["PRCUI"])
			assert.Equal(t, share.App.Name, response["APP"])
			assert.Equal(t, share.App.Version, response["APP_VERSION"])

			// Check that SERVER_TIME is a valid timestamp format
			serverTime, ok := response["SERVER_TIME"].(string)
			assert.True(t, ok)
			assert.NotEmpty(t, serverTime)
		})
	}
}
