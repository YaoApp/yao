package openapi

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/commercial"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// attachWellKnown attaches the well-known handlers to the router
func (openapi *OpenAPI) attachWellKnown(router *gin.Engine) {

	// OAuth Discovery and Metadata Endpoints
	wellKnown := router.Group("/.well-known")

	// Yao Configuration Metadata - for client discovery
	wellKnown.GET("/yao", openapi.yaoMetadata)

	// OAuth Authorization Server Metadata - RFC 8414 (Required by MCP)
	wellKnown.GET("/oauth-authorization-server", openapi.oauthServerMetadata)

	// OpenID Connect Discovery - OpenID Connect Discovery 1.0
	wellKnown.GET("/openid_configuration", openapi.oauthOpenIDConfiguration)

	// OAuth Protected Resource Metadata - RFC 9728 (Required by MCP)
	wellKnown.GET("/oauth-protected-resource", openapi.oauthProtectedResourceMetadata)
}

// YaoMetadata represents the Yao server configuration metadata
type YaoMetadata struct {
	// Application information
	Name        string `json:"name,omitempty"`
	Version     string `json:"version,omitempty"`
	Description string `json:"description,omitempty"`

	// OpenAPI configuration
	OpenAPI   string `json:"openapi"`              // OpenAPI base URL (e.g., "/v1")
	IssuerURL string `json:"issuer_url,omitempty"` // OAuth issuer URL

	// Public server URL — the externally accessible base URL for this instance.
	// Clients, proxies, and integrations should use this to construct API endpoints.
	// Set via env YAO_SERVER_URL; falls back to issuer_url (stripped of path), then empty.
	ServerURL string `json:"server_url,omitempty"`

	// Dashboard configuration
	Dashboard string                 `json:"dashboard,omitempty"` // Admin dashboard root path
	GRPC      string                 `json:"grpc,omitempty"`      // gRPC server address (e.g., "127.0.0.1:9099")
	Optional  map[string]interface{} `json:"optional,omitempty"`  // Optional settings

	// Commercial license
	License *commercial.PublicInfo `json:"license,omitempty"`

	// Developer information
	Developer *share.Developer `json:"developer,omitempty"`
}

// yaoMetadata returns Yao server configuration metadata
func (openapi *OpenAPI) yaoMetadata(c *gin.Context) {
	// Get admin root path
	dashboard := share.App.AdminRoot
	if dashboard == "" {
		dashboard = "yao"
	}

	metadata := YaoMetadata{
		Name:        share.App.Name,
		Version:     share.App.Version,
		Description: share.App.Description,
		OpenAPI:     openapi.Config.BaseURL,
		IssuerURL:   openapi.Config.OAuth.IssuerURL,
		ServerURL:   resolveServerURL(openapi.Config.OAuth.IssuerURL),
		Dashboard:   "/" + dashboard,
		GRPC:        resolveGRPCAddr(c),
		Optional:    share.App.Optional,
	}

	// Include license info
	metadata.License = commercial.GetPublicInfo()

	// Include developer info if available
	if share.App.Developer.ID != "" || share.App.Developer.Name != "" {
		metadata.Developer = &share.App.Developer
	}

	c.JSON(200, metadata)
}

// resolveServerURL returns the public server URL for this instance.
// Priority: YAO_SERVER_URL env > issuer_url origin > empty string.
func resolveServerURL(issuerURL string) string {
	if v := strings.TrimRight(os.Getenv("YAO_SERVER_URL"), "/"); v != "" {
		return v
	}
	// Strip path from issuer_url to get just the origin (scheme + host + port)
	if issuerURL != "" {
		if idx := strings.Index(issuerURL, "://"); idx != -1 {
			rest := issuerURL[idx+3:]
			if slash := strings.Index(rest, "/"); slash != -1 {
				return issuerURL[:idx+3] + rest[:slash]
			}
		}
		return issuerURL
	}
	return ""
}

// resolveGRPCAddr returns the gRPC server address for client discovery.
//
// When the listen host includes "internal", "0.0.0.0", or multiple addresses,
// the returned address uses the IP from the incoming HTTP request — if the
// client could reach Yao's HTTP port via that IP, gRPC on the same IP should
// also be reachable. "localhost" is treated as "127.0.0.1".
func resolveGRPCAddr(c *gin.Context) string {
	cfg := config.Conf.GRPC
	if strings.ToLower(cfg.Enabled) == "off" {
		return ""
	}
	port := cfg.Port
	if port == 0 {
		port = 9099
	}

	host := cfg.Host
	useRequestIP := host == "" || host == "0.0.0.0" ||
		strings.Contains(host, ",") ||
		config.HostHasInternal(host)

	if useRequestIP {
		reqHost := c.Request.Host
		h, _, err := net.SplitHostPort(reqHost)
		if err != nil {
			h = reqHost
		}
		if strings.ToLower(h) == "localhost" {
			h = "127.0.0.1"
		}
		host = h
	} else if strings.ToLower(strings.TrimSpace(host)) == "localhost" {
		host = "127.0.0.1"
	}

	return fmt.Sprintf("%s:%s", host, strconv.Itoa(port))
}

// oauthServerMetadata returns authorization server metadata - RFC 8414
func (openapi *OpenAPI) oauthServerMetadata(c *gin.Context) {
	if openapi.OAuth == nil {
		c.JSON(503, gin.H{"error": "OAuth service not available"})
		return
	}
	metadata, err := openapi.OAuth.GetServerMetadata(c.Request.Context())
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, metadata)
}

// oauthOpenIDConfiguration returns OpenID Connect configuration
func (openapi *OpenAPI) oauthOpenIDConfiguration(c *gin.Context) {}

// oauthProtectedResourceMetadata returns protected resource metadata - RFC 9728
func (openapi *OpenAPI) oauthProtectedResourceMetadata(c *gin.Context) {}
