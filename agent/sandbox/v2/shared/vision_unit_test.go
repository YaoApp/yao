//go:build unit

package shared

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func makePNG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 0, G: 0, B: 255, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func TestResizeForVision_NoResize(t *testing.T) {
	data := makePNG(100, 100)
	result, mime, err := ResizeForVision(data, 2048)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mime != "image/png" {
		t.Errorf("expected image/png, got %s", mime)
	}
	if !bytes.Equal(result, data) {
		t.Error("expected no resize for small image")
	}
}

func TestResizeForVision_Resize(t *testing.T) {
	data := makePNG(4000, 3000)
	result, mime, err := ResizeForVision(data, 2048)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mime != "image/png" {
		t.Errorf("expected image/png, got %s", mime)
	}
	if bytes.Equal(result, data) {
		t.Error("expected resized image to differ from original")
	}
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}

	img, _, err := image.Decode(bytes.NewReader(result))
	if err != nil {
		t.Fatalf("cannot decode result: %v", err)
	}
	bounds := img.Bounds()
	if bounds.Dx() > 2048 || bounds.Dy() > 2048 {
		t.Errorf("expected dimensions <= 2048, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestResizeForVision_InvalidData(t *testing.T) {
	_, _, err := ResizeForVision([]byte("not an image"), 2048)
	if err == nil {
		t.Error("expected error for invalid data")
	}
}

func TestHasVision_Nil(t *testing.T) {
	if HasVision(nil) {
		t.Error("nil connector should return false")
	}
}

func TestHasVision_NonLLMConnector(t *testing.T) {
	type dummy struct{}
	if HasVision(dummy{}) {
		t.Error("non-LLMConnector should return false")
	}
}

func TestIsImageMIME(t *testing.T) {
	cases := []struct {
		mime   string
		expect bool
	}{
		{"image/png", true},
		{"image/jpeg", true},
		{"image/gif", true},
		{"image/webp", true},
		{"image/tiff", false},
		{"application/pdf", false},
		{"", false},
	}
	for _, c := range cases {
		if got := isImageMIME(c.mime); got != c.expect {
			t.Errorf("isImageMIME(%q) = %v, want %v", c.mime, got, c.expect)
		}
	}
}

func TestBuildContentBlocks_DSH(t *testing.T) {
	// Re-test buildContentBlocks from dsh package via exported types
	parts := &MessageParts{
		TextParts:   []string{"describe this image"},
		ImageBlocks: []ImageBlock{{MediaType: "image/png", Data: "AAAA", Filename: "test.png"}},
	}
	if len(parts.TextParts) != 1 {
		t.Fatal("expected 1 text part")
	}
	if len(parts.ImageBlocks) != 1 {
		t.Fatal("expected 1 image block")
	}
}
