package preprocess

import (
	"bytes"
	stdimage "image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// ResizeImage decodes an image, resizes it if the longest edge exceeds maxSize,
// and re-encodes as JPEG (quality 85). Returns the original data when the image
// is already small enough or cannot be decoded.
func ResizeImage(data []byte, maxSize int) []byte {
	if maxSize <= 0 {
		return data
	}

	img, _, err := stdimage.Decode(bytes.NewReader(data))
	if err != nil {
		return data
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= maxSize && h <= maxSize {
		return data
	}

	ratio := float64(maxSize) / float64(max(w, h))
	newW, newH := int(float64(w)*ratio), int(float64(h)*ratio)
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	dst := stdimage.NewRGBA(stdimage.Rect(0, 0, newW, newH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 85}); err != nil {
		return data
	}
	return buf.Bytes()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
