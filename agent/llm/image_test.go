package llm

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractImageFromResponse_B64(t *testing.T) {
	data := map[string]interface{}{
		"data": []interface{}{
			map[string]interface{}{
				"b64_json": "iVBORw0KGgoAAAANS...",
			},
		},
	}
	resp, err := extractImageFromResponse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Image != "iVBORw0KGgoAAAANS..." {
		t.Errorf("got Image=%q, want %q", resp.Image, "iVBORw0KGgoAAAANS...")
	}
	if resp.Format != "png" {
		t.Errorf("got Format=%q, want %q", resp.Format, "png")
	}
}

func TestExtractImageFromResponse_URL(t *testing.T) {
	fakeImage := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10} // fake JPEG header bytes
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(fakeImage)
	}))
	defer srv.Close()

	data := map[string]interface{}{
		"data": []interface{}{
			map[string]interface{}{
				"b64_json": nil,
				"url":      srv.URL + "/image_0.jpeg",
			},
		},
	}
	resp, err := extractImageFromResponse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := base64.StdEncoding.EncodeToString(fakeImage)
	if resp.Image != expected {
		t.Errorf("got Image=%q, want %q", resp.Image, expected)
	}
	if resp.Format != "jpeg" {
		t.Errorf("got Format=%q, want %q", resp.Format, "jpeg")
	}
}

func TestExtractImageFromResponse_URLPng(t *testing.T) {
	fakeImage := []byte{0x89, 0x50, 0x4E, 0x47} // PNG magic bytes
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(fakeImage)
	}))
	defer srv.Close()

	data := map[string]interface{}{
		"data": []interface{}{
			map[string]interface{}{
				"url": srv.URL + "/output.png",
			},
		},
	}
	resp, err := extractImageFromResponse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Format != "png" {
		t.Errorf("got Format=%q, want %q", resp.Format, "png")
	}
	if resp.Image == "" {
		t.Error("expected non-empty base64 Image")
	}
}

func TestExtractImageFromResponse_URLDownloadFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	data := map[string]interface{}{
		"data": []interface{}{
			map[string]interface{}{
				"url": srv.URL + "/missing.png",
			},
		},
	}
	_, err := extractImageFromResponse(data)
	if err == nil {
		t.Error("expected error for failed download")
	}
}

func TestExtractImageFromResponse_Empty(t *testing.T) {
	data := map[string]interface{}{
		"data": []interface{}{},
	}
	_, err := extractImageFromResponse(data)
	if err == nil {
		t.Error("expected error for empty data array")
	}
}

func TestExtractImageFromResponse_NoData(t *testing.T) {
	data := map[string]interface{}{}
	_, err := extractImageFromResponse(data)
	if err == nil {
		t.Error("expected error for missing data field")
	}
}

func TestExtractImageFromResponse_NullBoth(t *testing.T) {
	data := map[string]interface{}{
		"data": []interface{}{
			map[string]interface{}{
				"b64_json": nil,
				"url":      nil,
			},
		},
	}
	_, err := extractImageFromResponse(data)
	if err == nil {
		t.Error("expected error when both b64_json and url are null")
	}
}

func TestDownloadImageAsBase64(t *testing.T) {
	payload := []byte("fake-png-data")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(payload)
	}))
	defer srv.Close()

	b64, format, err := downloadImageAsBase64(srv.URL + "/test.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if format != "png" {
		t.Errorf("got format=%q, want %q", format, "png")
	}
	decoded, _ := base64.StdEncoding.DecodeString(b64)
	if string(decoded) != string(payload) {
		t.Errorf("decoded content mismatch")
	}
}

func TestDownloadImageAsBase64_FormatFromURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte("data"))
	}))
	defer srv.Close()

	_, format, err := downloadImageAsBase64(srv.URL + "/image.webp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if format != "webp" {
		t.Errorf("got format=%q, want %q (from URL fallback)", format, "webp")
	}
}

func TestExtractAPIError_WithMessage(t *testing.T) {
	data := map[string]interface{}{
		"error": map[string]interface{}{
			"message": "insufficient quota",
		},
	}
	msg := extractAPIError(data)
	if msg != "insufficient quota" {
		t.Errorf("got %q, want %q", msg, "insufficient quota")
	}
}

func TestExtractAPIError_NoMessage(t *testing.T) {
	data := map[string]interface{}{
		"something": "else",
	}
	msg := extractAPIError(data)
	raw, _ := json.Marshal(data)
	if msg != string(raw) {
		t.Errorf("got %q, want raw JSON fallback", msg)
	}
}
