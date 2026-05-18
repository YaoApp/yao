package sandboxv2

import (
	"time"

	"github.com/yaoapp/yao/agent/sandbox/v2/types"
)

// ExportCanonicalRunner exposes canonicalRunner for testing.
var ExportCanonicalRunner = canonicalRunner

// ExportContainsRunner exposes containsRunner for testing.
var ExportContainsRunner = containsRunner

// Shell kind constants for testing.
const (
	ExportShellSh   = shellSh
	ExportShellPwsh = shellPwsh
	ExportShellPS   = shellPS
	ExportShellCmd  = shellCmd
)

// ExportShellWrap exposes shellWrap for testing.
func ExportShellWrap(kind int, script string) []string {
	return shellWrap(shellKind(kind), script)
}

// ExportCacheKey exposes cacheKey for testing.
var ExportCacheKey = cacheKey

// ExportSetToken exposes setToken for testing.
func ExportSetToken(teamID, userID string, tok *types.SandboxToken, ttl time.Duration) {
	setToken(teamID, userID, tok, ttl)
}

// ExportGetToken exposes getToken for testing.
func ExportGetToken(teamID, userID string) *types.SandboxToken {
	return getToken(teamID, userID)
}

// ExportParseMemory exposes parseMemory for testing.
var ExportParseMemory = parseMemory
