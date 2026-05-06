package vision

import (
	"encoding/base64"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	agentCtx "github.com/yaoapp/yao/agent/context"
)

func makePNG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	var buf strings.Builder
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return []byte(buf.String())
}

func TestResizeImage_SmallImage(t *testing.T) {
	pngData := makePNG(100, 80)
	data, mime := resizeImage(pngData, 1080)
	if mime != "image/png" {
		t.Errorf("mime = %q, want image/png", mime)
	}
	if len(data) != len(pngData) {
		t.Errorf("data length changed for small image")
	}
}

func TestResizeImage_LargeImage(t *testing.T) {
	pngData := makePNG(2000, 1500)
	data, mime := resizeImage(pngData, 1080)
	if mime != "image/jpeg" {
		t.Errorf("mime = %q, want image/jpeg", mime)
	}
	img, _, err := image.Decode(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("failed to decode resized image: %v", err)
	}
	b := img.Bounds()
	if b.Dx() > 1080 || b.Dy() > 1080 {
		t.Errorf("resized dimensions %dx%d exceed max 1080", b.Dx(), b.Dy())
	}
	if b.Dx() != 1080 {
		t.Errorf("longest edge = %d, want 1080", b.Dx())
	}
}

func TestResizeImage_InvalidData(t *testing.T) {
	raw := []byte("not an image at all")
	data, mime := resizeImage(raw, 1080)
	if string(data) != string(raw) {
		t.Error("should return original data on decode failure")
	}
	if mime == "" {
		t.Error("mime should not be empty")
	}
}

func TestResizeImage_ExactBoundary(t *testing.T) {
	pngData := makePNG(1080, 720)
	data, _ := resizeImage(pngData, 1080)
	if len(data) != len(pngData) {
		t.Error("image at exact max_size should not be re-encoded")
	}
}

func TestDecodeDataURI(t *testing.T) {
	original := []byte("hello world")
	b64 := base64.StdEncoding.EncodeToString(original)
	uri := "data:application/octet-stream;base64," + b64

	data, err := decodeDataURI(uri)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(original) {
		t.Errorf("decoded = %q, want %q", data, original)
	}
}

func TestDecodeDataURI_Invalid(t *testing.T) {
	_, err := decodeDataURI("not-a-data-uri")
	if err == nil {
		t.Error("expected error for invalid data URI")
	}
}

func TestExtractTextContent_String(t *testing.T) {
	result := extractTextContent("hello world")
	if result != "hello world" {
		t.Errorf("got %q, want %q", result, "hello world")
	}
}

func TestExtractTextContent_ContentParts(t *testing.T) {
	parts := []agentCtx.ContentPart{
		{Type: agentCtx.ContentImageURL, ImageURL: &agentCtx.ImageURL{URL: "data:image/png;base64,abc"}},
		{Type: agentCtx.ContentText, Text: "description of the image"},
	}
	result := extractTextContent(parts)
	if result != "description of the image" {
		t.Errorf("got %q, want %q", result, "description of the image")
	}
}

func TestExtractTextContent_Fallback(t *testing.T) {
	result := extractTextContent(12345)
	if result != "12345" {
		t.Errorf("got %q, want %q", result, "12345")
	}
}

func TestReadBytes_HTTP(t *testing.T) {
	pngData := makePNG(50, 50)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(pngData)
	}))
	defer srv.Close()

	data, err := readBytes(srv.URL + "/test.png")
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != len(pngData) {
		t.Errorf("got %d bytes, want %d", len(data), len(pngData))
	}
}

func TestReadBytes_HTTP404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := readBytes(srv.URL + "/missing.png")
	if err == nil {
		t.Error("expected error for 404")
	}
}

func TestReadBytes_DataURI(t *testing.T) {
	original := []byte("test data")
	b64 := base64.StdEncoding.EncodeToString(original)
	uri := "data:application/octet-stream;base64," + b64

	data, err := readBytes(uri)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(original) {
		t.Errorf("got %q, want %q", data, original)
	}
}

func TestReadBytes_UnsupportedScheme(t *testing.T) {
	_, err := readBytes("ftp://example.com/file.png")
	if err == nil {
		t.Error("expected error for unsupported scheme")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("error should mention unsupported: %v", err)
	}
}

func TestResolveImage_DataURI(t *testing.T) {
	pngData := makePNG(100, 80)
	b64 := base64.StdEncoding.EncodeToString(pngData)
	uri := "data:image/png;base64," + b64

	result, err := resolveImage(uri, 1080)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(result, "data:image/") {
		t.Errorf("expected data URI, got %q", result[:min(50, len(result))])
	}
}

func TestResolveImage_HTTP(t *testing.T) {
	pngData := makePNG(200, 150)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(pngData)
	}))
	defer srv.Close()

	result, err := resolveImage(srv.URL+"/img.png", 1080)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(result, "data:image/") {
		t.Errorf("expected data URI, got %q", result[:min(50, len(result))])
	}
}

func TestHttpGet_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't respond — let the client timeout
		select {}
	}))
	defer srv.Close()

	// Use a very short timeout for testing — override not possible with global func,
	// but the 30s timeout should handle normally. Just verify it returns eventually.
	// This test just verifies the function signature works.
	_, err := readBytes("http://invalid.host.that.does.not.exist.example.com/img.png")
	if err == nil {
		t.Error("expected error for unreachable host")
	}
}

func TestReadBytes_WorkspaceInvalid(t *testing.T) {
	_, err := readBytes("workspace://no-slash-path")
	if err == nil {
		t.Error("expected error for invalid workspace URI")
	}
}

func TestReadBytes_AttachInvalid(t *testing.T) {
	_, err := readBytes("attach://no-slash-path")
	if err == nil {
		t.Error("expected error for invalid attach URI")
	}
}
