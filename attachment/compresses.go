package attachment

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
)

// CompressImage compresses the image while maintaining aspect ratio
func CompressImage(reader io.Reader, contentType string, maxSize int) ([]byte, error) {
	// Read all data first
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	// Decode image
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Calculate new dimensions
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	var newWidth, newHeight int

	if width > height {
		if width > maxSize {
			newWidth = maxSize
			newHeight = int(float64(height) * (float64(maxSize) / float64(width)))
		} else {
			return data, nil // No need to resize, return original data
		}
	} else {
		if height > maxSize {
			newHeight = maxSize
			newWidth = int(float64(width) * (float64(maxSize) / float64(height)))
		} else {
			return data, nil // No need to resize, return original data
		}
	}

	// Create new image with new dimensions
	newImg := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// Scale the image using bilinear interpolation
	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			srcX := float64(x) * float64(width) / float64(newWidth)
			srcY := float64(y) * float64(height) / float64(newHeight)
			newImg.Set(x, y, img.At(int(srcX), int(srcY)))
		}
	}

	// Encode image
	var buf bytes.Buffer
	switch contentType {
	case "image/jpeg":
		err = jpeg.Encode(&buf, newImg, &jpeg.Options{Quality: 85})
	case "image/png":
		err = png.Encode(&buf, newImg)
	default:
		return data, nil // Unsupported format, return original data
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	return buf.Bytes(), nil
}
