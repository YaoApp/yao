package webproxy

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/response"
)

// authMiddleware wraps an http.Handler with JWT cookie/bearer token authentication.
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		svc := oauth.OAuth
		if svc == nil {
			// OAuth not initialized — allow through (dev/testing mode)
			next.ServeHTTP(w, r)
			return
		}

		token := extractToken(r)
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		_, err := svc.VerifyToken(token)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func extractToken(r *http.Request) string {
	// Try Authorization header first
	if auth := r.Header.Get("Authorization"); auth != "" {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	// Try access_token cookie (both with and without __Host- prefix)
	// Note: net/http does NOT URL-decode cookie values (unlike Gin's c.Cookie()),
	// so we must manually decode to handle "Bearer+" → "Bearer ".
	cookieName := response.GetCookieName("access_token")
	if c, err := r.Cookie(cookieName); err == nil && c.Value != "" {
		return decodeBearerCookie(c.Value)
	}

	// Fallback: plain cookie name without prefix
	if c, err := r.Cookie("access_token"); err == nil && c.Value != "" {
		return decodeBearerCookie(c.Value)
	}

	return ""
}

func decodeBearerCookie(raw string) string {
	// URL-decode first (handles "Bearer+" → "Bearer ")
	decoded, err := url.QueryUnescape(raw)
	if err != nil {
		decoded = raw
	}
	return strings.TrimPrefix(decoded, "Bearer ")
}
