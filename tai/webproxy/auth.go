package webproxy

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/yaoapp/yao/openapi/oauth"
)

const proxyAuthCookie = "_yao_proxy"

// authMiddleware wraps an http.Handler with JWT cookie authentication.
// Uses a dedicated cookie (_yao_proxy) to avoid conflicts with target app auth.
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		svc := oauth.OAuth
		if svc == nil {
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
	if c, err := r.Cookie(proxyAuthCookie); err == nil && c.Value != "" {
		return decodeBearerCookie(c.Value)
	}
	return ""
}

// stripProxyCookie removes the proxy's own auth cookie from the request
// before forwarding to the target application.
func stripProxyCookie(r *http.Request) {
	cookies := r.Cookies()
	r.Header.Del("Cookie")
	for _, c := range cookies {
		if c.Name != proxyAuthCookie {
			r.AddCookie(c)
		}
	}
}

func decodeBearerCookie(raw string) string {
	decoded, err := url.QueryUnescape(raw)
	if err != nil {
		decoded = raw
	}
	return strings.TrimPrefix(decoded, "Bearer ")
}
