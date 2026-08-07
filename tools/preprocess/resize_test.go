package preprocess

import (
	stdimage "image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
)

func makePNG(w, h int) []byte {
	img := stdimage.NewRGBA(stdimage.Rect(0, 0, w, h))
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
	out := ResizeImage(pngData, 1080)
	if len(out) != len(pngData) {
		t.Error("small image should be returned unchanged")
	}
}

func TestResizeImage_LargeImage(t *testing.T) {
	pngData := makePNG(2000, 1500)
	out := ResizeImage(pngData, 1080)

	img, err := jpeg.Decode(strings.NewReader(string(out)))
	if err != nil {
		t.Fatalf("failed to decode resized JPEG: %v", err)
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
	raw := []byte("not an image")
	out := ResizeImage(raw, 1080)
	if string(out) != string(raw) {
		t.Error("should return original data on decode failure")
	}
}

func TestResizeImage_ExactBoundary(t *testing.T) {
	pngData := makePNG(1080, 720)
	out := ResizeImage(pngData, 1080)
	if len(out) != len(pngData) {
		t.Error("image at exact max_size should not be re-encoded")
	}
}
