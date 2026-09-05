package tai_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	tai "github.com/yaoapp/yao/openapi/tai"
)

func TestCheckOrigin(t *testing.T) {
	tests := []struct {
		name           string
		origin         string
		host           string
		xForwardedHost string
		want           bool
	}{
		{
			name:   "no origin header",
			origin: "",
			host:   "example.com",
			want:   true,
		},
		{
			name:   "same origin",
			origin: "https://example.com",
			host:   "example.com",
			want:   true,
		},
		{
			name:   "different origin",
			origin: "https://evil.com",
			host:   "example.com",
			want:   false,
		},
		{
			name:   "invalid origin URL",
			origin: "://invalid",
			host:   "example.com",
			want:   false,
		},
		{
			name:           "X-Forwarded-Host match",
			origin:         "https://frontend.example.com",
			host:           "backend.internal",
			xForwardedHost: "frontend.example.com",
			want:           true,
		},
		{
			name:           "X-Forwarded-Host no match",
			origin:         "https://evil.com",
			host:           "backend.internal",
			xForwardedHost: "frontend.example.com",
			want:           false,
		},
		{
			name:   "origin with port",
			origin: "https://example.com:8443",
			host:   "example.com:8443",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{
				Host:   tt.host,
				Header: http.Header{},
			}
			if tt.origin != "" {
				r.Header.Set("Origin", tt.origin)
			}
			if tt.xForwardedHost != "" {
				r.Header.Set("X-Forwarded-Host", tt.xForwardedHost)
			}
			got := tai.ExportCheckOrigin(r)
			assert.Equal(t, tt.want, got)
		})
	}
}
