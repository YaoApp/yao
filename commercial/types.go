package commercial

import "time"

// License is the global commercial license state, populated by Load().
// Read-only after initialization; all modules may read without synchronization.
var License LicenseInfo

// LicenseInfo holds the parsed result of a commercial license certificate.
type LicenseInfo struct {
	Valid    bool   `json:"valid"`
	Source   string `json:"source"` // "none" | "file" | "env"
	LoadedAt int64  `json:"loaded_at"`
	Error    string `json:"error,omitempty"`

	// Identity (from X.509 Subject)
	LicenseeName    string `json:"licensee_name"`
	LicenseeOrg     string `json:"licensee_org"`
	LicenseeCountry string `json:"licensee_country,omitempty"`
	LicenseeEmail   string `json:"licensee_email,omitempty"`
	SerialNumber    string `json:"serial_number"`
	NotBefore       int64  `json:"not_before"`
	NotAfter        int64  `json:"not_after"`
	IsExpired       bool   `json:"is_expired"`
	Issuer          string `json:"issuer"`

	// Scope
	Product   []string `json:"product"`
	Edition   string   `json:"edition"` // "community" | "starter" | "pro" | "enterprise"
	Env       []string `json:"env,omitempty"`
	Domain    string   `json:"domain,omitempty"`
	AppID     string   `json:"app_id,omitempty"`
	MachineID string   `json:"machine_id,omitempty"` // if set, must match runtime machine ID

	// Quota (0 = unlimited)
	MaxUsers     int `json:"max_users"`
	MaxTaiNodes  int `json:"max_tai_nodes"`
	MaxAgents    int `json:"max_agents"`
	MaxSandboxes int `json:"max_sandboxes"`
	MaxAPIRPM    int `json:"max_api_rpm"`
	MaxStorageGB int `json:"max_storage_gb"`

	// Permissions
	Permissions Permissions `json:"permissions"`
}

// Permissions controls feature switches.
type Permissions struct {
	AllowBrandingRemoval bool   `json:"allow_branding_removal"`
	AllowWhiteLabel      bool   `json:"allow_white_label"`
	AllowMultiTenant     bool   `json:"allow_multi_tenant"`
	AllowCustomDomain    bool   `json:"allow_custom_domain"`
	AllowHostExec        bool   `json:"allow_host_exec"`
	AllowSSO             bool   `json:"allow_sso"`
	SupportLevel         string `json:"support_level"` // "none" | "email" | "priority" | "dedicated"
}

// PublicInfo is the subset safe for well-known / public API exposure.
type PublicInfo struct {
	Valid    bool     `json:"valid"`
	Edition  string   `json:"edition"`
	NotAfter int64    `json:"not_after,omitempty"`
	Product  []string `json:"product"`
}

// DefaultLicense returns community-level defaults when no certificate is present.
func DefaultLicense() LicenseInfo {
	return LicenseInfo{
		Source:       "none",
		LoadedAt:     time.Now().Unix(),
		Edition:      "community",
		Product:      []string{"yao"},
		MaxUsers:     100,
		MaxTaiNodes:  1,
		MaxAgents:    3,
		MaxSandboxes: 1,
		MaxAPIRPM:    1000,
		MaxStorageGB: 10,
		Permissions: Permissions{
			SupportLevel: "none",
		},
	}
}

// GetPublicInfo returns the public-safe subset of the current license state.
func GetPublicInfo() *PublicInfo {
	return &PublicInfo{
		Valid:    License.Valid,
		Edition:  License.Edition,
		NotAfter: License.NotAfter,
		Product:  License.Product,
	}
}

var editionRank = map[string]int{
	"community":  0,
	"starter":    1,
	"pro":        2,
	"enterprise": 3,
}

// IsLevel reports whether the license meets or exceeds the given minimum edition.
func (l LicenseInfo) IsLevel(minEdition string) bool {
	return editionRank[l.Edition] >= editionRank[minEdition]
}

// HasProduct reports whether the license covers the given product name.
func (l LicenseInfo) HasProduct(product string) bool {
	for _, p := range l.Product {
		if p == product {
			return true
		}
	}
	return false
}
