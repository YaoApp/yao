//go:build unit

package image_test

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/content/image"
)

func TestEncodeToBase64DataURI(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		contentType string
		wantPrefix  string
	}{
		{"PNG image", []byte{0x89, 0x50, 0x4E, 0x47}, "image/png", "data:image/png;base64,"},
		{"JPEG image", []byte{0xFF, 0xD8, 0xFF}, "image/jpeg", "data:image/jpeg;base64,"},
		{"Empty content type defaults to PNG", []byte{0x01, 0x02, 0x03}, "", "data:image/png;base64,"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := image.EncodeToBase64DataURI(tt.data, tt.contentType)

			assert.True(t, strings.HasPrefix(result, tt.wantPrefix))

			base64Part := result[len(tt.wantPrefix):]
			decoded, err := base64.StdEncoding.DecodeString(base64Part)
			require.NoError(t, err)
			assert.Equal(t, tt.data, decoded)
		})
	}
}
