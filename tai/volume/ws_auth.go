package volume

import (
	"bufio"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

// resolveAuth reads workspace config to resolve authentication
// for the given remote URL. Returns nil, nil for public repos
// or when no credentials are configured.
func (l *localStorage) resolveAuth(sessionID, remoteURL string) (transport.AuthMethod, error) {
	wsBase, err := l.wsBase(sessionID)
	if err != nil {
		return nil, nil
	}

	if isSSH(remoteURL) {
		host := extractSSHHost(remoteURL)

		sshCfgPath := filepath.Join(wsBase, "ssh", "config")
		if keyPath, err := matchSSHKey(sshCfgPath, host); err == nil {
			if keyData, err := os.ReadFile(keyPath); err == nil {
				if auth, err := gitssh.NewPublicKeys("git", keyData, ""); err == nil {
					return auth, nil
				}
			}
		}

		keysDir := filepath.Join(wsBase, "ssh", "keys")
		if keys, _ := filepath.Glob(filepath.Join(keysDir, "*.key")); len(keys) > 0 {
			if keyData, err := os.ReadFile(keys[0]); err == nil {
				if auth, err := gitssh.NewPublicKeys("git", keyData, ""); err == nil {
					return auth, nil
				}
			}
		}

		if auth, err := gitssh.NewSSHAgentAuth("git"); err == nil {
			return auth, nil
		}
	}

	if isHTTPS(remoteURL) {
		credPath := filepath.Join(wsBase, "git", "credentials")
		host := extractHost(remoteURL)
		entries, err := readCredentialFile(credPath)
		if err == nil {
			for _, e := range entries {
				if strings.EqualFold(e.Host, host) {
					return &githttp.BasicAuth{
						Username: e.Username,
						Password: e.Token,
					}, nil
				}
			}
		}
	}

	return nil, nil
}

// isSSH returns true for SSH-style git URLs.
func isSSH(rawURL string) bool {
	return strings.HasPrefix(rawURL, "git@") || strings.HasPrefix(rawURL, "ssh://")
}

// isHTTPS returns true for HTTPS git URLs.
func isHTTPS(rawURL string) bool {
	return strings.HasPrefix(rawURL, "https://")
}

// extractSSHHost extracts the hostname from an SSH URL.
func extractSSHHost(rawURL string) string {
	if strings.HasPrefix(rawURL, "ssh://") {
		if u, err := url.Parse(rawURL); err == nil {
			return u.Hostname()
		}
	}
	if idx := strings.Index(rawURL, "@"); idx >= 0 {
		rest := rawURL[idx+1:]
		if colonIdx := strings.Index(rest, ":"); colonIdx >= 0 {
			return rest[:colonIdx]
		}
	}
	return ""
}

// extractHost extracts the hostname from an HTTPS URL.
func extractHost(rawURL string) string {
	if u, err := url.Parse(rawURL); err == nil {
		return u.Hostname()
	}
	return ""
}

// matchSSHKey parses an SSH config file and returns the IdentityFile
// path for the given host.
func matchSSHKey(sshCfgPath, host string) (string, error) {
	f, err := os.Open(sshCfgPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var currentHost string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "Host ") {
			currentHost = strings.TrimSpace(strings.TrimPrefix(line, "Host"))
		} else if strings.HasPrefix(line, "IdentityFile ") && strings.EqualFold(currentHost, host) {
			return strings.TrimSpace(strings.TrimPrefix(line, "IdentityFile")), nil
		}
	}
	return "", os.ErrNotExist
}

// sanitizeRemoteURL removes userinfo (credentials) from a URL.
func sanitizeRemoteURL(rawURL string) string {
	if !strings.Contains(rawURL, "@") || isSSH(rawURL) {
		return rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	u.User = nil
	return u.String()
}
