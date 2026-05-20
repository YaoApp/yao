//go:build unit

package llm_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yaoapp/yao/agent/llm"
)

func TestExtractImageFromResponse_B64(t *testing.T) {
	data := map[string]interface{}{
		"data": []interface{}{
			map[string]interface{}{"b64_json": "iVBORw0KGgoAAAANS..."},
		},
	}
	resp, err := llm.ExportExtractImageFromResponse(data)
	require.NoError(t, err)
	assert.Equal(t, "iVBORw0KGgoAAAANS...", resp.Image)
	assert.Equal(t, "png", resp.Format)
}

func TestExtractImageFromResponse_URL(t *testing.T) {
	fakeImage := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(fakeImage)
	}))
	defer srv.Close()

	data := map[string]interface{}{
		"data": []interface{}{
			map[string]interface{}{"b64_json": nil, "url": srv.URL + "/image_0.jpeg"},
		},
	}
	resp, err := llm.ExportExtractImageFromResponse(data)
	require.NoError(t, err)
	assert.Equal(t, base64.StdEncoding.EncodeToString(fakeImage), resp.Image)
	assert.Equal(t, "jpeg", resp.Format)
}

func TestExtractImageFromResponse_URLPng(t *testing.T) {
	fakeImage := []byte{0x89, 0x50, 0x4E, 0x47}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(fakeImage)
	}))
	defer srv.Close()

	data := map[string]interface{}{
		"data": []interface{}{
			map[string]interface{}{"url": srv.URL + "/output.png"},
		},
	}
	resp, err := llm.ExportExtractImageFromResponse(data)
	require.NoError(t, err)
	assert.Equal(t, "png", resp.Format)
	assert.NotEmpty(t, resp.Image)
}

func TestExtractImageFromResponse_URLDownloadFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	data := map[string]interface{}{
		"data": []interface{}{
			map[string]interface{}{"url": srv.URL + "/missing.png"},
		},
	}
	_, err := llm.ExportExtractImageFromResponse(data)
	assert.Error(t, err)
}

func TestExtractImageFromResponse_Empty(t *testing.T) {
	data := map[string]interface{}{"data": []interface{}{}}
	_, err := llm.ExportExtractImageFromResponse(data)
	assert.Error(t, err)
}

func TestExtractImageFromResponse_NoData(t *testing.T) {
	data := map[string]interface{}{}
	_, err := llm.ExportExtractImageFromResponse(data)
	assert.Error(t, err)
}

func TestExtractImageFromResponse_NullBoth(t *testing.T) {
	data := map[string]interface{}{
		"data": []interface{}{
			map[string]interface{}{"b64_json": nil, "url": nil},
		},
	}
	_, err := llm.ExportExtractImageFromResponse(data)
	assert.Error(t, err)
}

func TestDownloadImageAsBase64(t *testing.T) {
	payload := []byte("fake-png-data")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(payload)
	}))
	defer srv.Close()

	b64, format, err := llm.ExportDownloadImageAsBase64(srv.URL + "/test.png")
	require.NoError(t, err)
	assert.Equal(t, "png", format)
	decoded, err := base64.StdEncoding.DecodeString(b64)
	require.NoError(t, err)
	assert.Equal(t, payload, decoded)
}

func TestDownloadImageAsBase64_FormatFromURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte("data"))
	}))
	defer srv.Close()

	_, format, err := llm.ExportDownloadImageAsBase64(srv.URL + "/image.webp")
	require.NoError(t, err)
	assert.Equal(t, "webp", format)
}

func TestExtractAPIError_WithMessage(t *testing.T) {
	data := map[string]interface{}{
		"error": map[string]interface{}{"message": "insufficient quota"},
	}
	assert.Equal(t, "insufficient quota", llm.ExportExtractAPIError(data))
}

func TestExtractAPIError_NoMessage(t *testing.T) {
	data := map[string]interface{}{"something": "else"}
	raw, _ := json.Marshal(data)
	assert.Equal(t, string(raw), llm.ExportExtractAPIError(data))
}
