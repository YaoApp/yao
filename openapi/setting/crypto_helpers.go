package setting

import (
	"strings"

	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/setting"
)

const maskChars = 4

// encryptValue encrypts a plaintext value for durable storage.
func encryptValue(plaintext string) string {
	return setting.Encrypt(plaintext)
}

// decryptValue decrypts a stored value.
func decryptValue(value string) string {
	return setting.Decrypt(value)
}

// DecryptValue decrypts a value encrypted by encryptValue.
func DecryptValue(s string) string {
	return setting.Decrypt(s)
}

// maskKey returns a masked representation of a secret key (prefix...suffix).
func maskKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= maskChars {
		return strings.Repeat("*", len(key))
	}
	prefix := key[:3]
	suffix := key[len(key)-maskChars:]
	return prefix + "..." + suffix
}

// scopeFromAuth derives a setting ScopeID from authorized info.
func scopeFromAuth(info *oauthTypes.AuthorizedInfo) setting.ScopeID {
	if info.TeamID != "" {
		return setting.ScopeID{Scope: setting.ScopeTeam, TeamID: info.TeamID}
	}
	return setting.ScopeID{Scope: setting.ScopeUser, UserID: info.UserID}
}
