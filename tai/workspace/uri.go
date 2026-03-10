package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// hostURI holds the parsed result of a host URI.
type hostURI struct {
	Scheme string // "local" or "tmp"; empty for workspace paths
	Path   string // resolved absolute path (host) or relative path (workspace)
	IsHost bool
}

// parseHostURI extracts scheme and path from a host URI string.
//
//	"local:///abs/path"   -> {Scheme:"local", Path:"/abs/path", IsHost:true}
//	"tmp:///rel/path"     -> {Scheme:"tmp",   Path:"rel/path",  IsHost:true}
//	"some/workspace/path" -> {Scheme:"",      Path:"some/workspace/path", IsHost:false}
func parseHostURI(raw string) hostURI {
	switch {
	case strings.HasPrefix(raw, "local:///"):
		return hostURI{Scheme: "local", Path: strings.TrimPrefix(raw, "local:///"), IsHost: true}
	case strings.HasPrefix(raw, "tmp:///"):
		return hostURI{Scheme: "tmp", Path: strings.TrimPrefix(raw, "tmp:///"), IsHost: true}
	default:
		return hostURI{Path: raw}
	}
}

// resolveAbsHostPath converts a parsed hostURI into an absolute filesystem path.
// For "local" scheme, Path is already absolute (rooted at /).
// For "tmp" scheme, Path is relative to os.TempDir().
func resolveAbsHostPath(u hostURI) (string, error) {
	switch u.Scheme {
	case "local":
		abs := filepath.Clean("/" + u.Path)
		return abs, nil
	case "tmp":
		if strings.Contains(u.Path, "..") {
			return "", fmt.Errorf("path traversal not allowed in tmp:// URI")
		}
		return filepath.Join(os.TempDir(), u.Path), nil
	default:
		return "", fmt.Errorf("not a host URI: %q", u.Path)
	}
}
