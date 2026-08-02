package volume

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gossh "golang.org/x/crypto/ssh"
)

// updateSSHConfig appends (or replaces) a Host block in ssh/config
// pointing to the named key file.
func updateSSHConfig(wsBase, name, host string) error {
	cfgPath := filepath.Join(wsBase, "ssh", "config")
	data, _ := os.ReadFile(cfgPath)

	keyPath := filepath.Join(wsBase, "ssh", "keys", name+".key")
	block := fmt.Sprintf("Host %s\n\tIdentityFile %s\n\tStrictHostKeyChecking accept-new\n", host, keyPath)

	cleaned := removeSSHHostBlock(string(data), host)
	if cleaned != "" && !strings.HasSuffix(cleaned, "\n") {
		cleaned += "\n"
	}

	return writeSecureFile(cfgPath, []byte(cleaned+block))
}

// removeSSHHostBlock removes the Host block matching the given host.
func removeSSHHostBlock(content, host string) string {
	var result []string
	skip := false
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Host ") {
			h := strings.TrimSpace(strings.TrimPrefix(trimmed, "Host"))
			skip = strings.EqualFold(h, host)
			if skip {
				continue
			}
		} else if skip {
			if trimmed == "" || (!strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, " ")) {
				skip = false
			} else {
				continue
			}
		}
		if !skip {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

// removeSSHHostBlockByKeyName removes Host blocks whose IdentityFile
// references the given key filename (e.g. "github.key").
func removeSSHHostBlockByKeyName(content, keyFileName string) string {
	type block struct {
		lines  []string
		hasKey bool
	}

	var blocks []block
	cur := block{}

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "Host ") && len(cur.lines) > 0 {
			blocks = append(blocks, cur)
			cur = block{}
		}
		cur.lines = append(cur.lines, line)
		if strings.Contains(trimmed, keyFileName) {
			cur.hasKey = true
		}
	}
	if len(cur.lines) > 0 {
		blocks = append(blocks, cur)
	}

	var result []string
	for _, b := range blocks {
		if !b.hasKey {
			result = append(result, b.lines...)
		}
	}
	if len(result) == 0 {
		return ""
	}
	return strings.Join(result, "\n") + "\n"
}

// derivePublicKey extracts the public key from a PEM-encoded private key
// and returns it in authorized_keys format.
func derivePublicKey(pemData []byte) ([]byte, error) {
	raw, err := gossh.ParseRawPrivateKey(pemData)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	var cryptoPub interface{}
	switch k := raw.(type) {
	case *rsa.PrivateKey:
		cryptoPub = &k.PublicKey
	case *ecdsa.PrivateKey:
		cryptoPub = &k.PublicKey
	case ed25519.PrivateKey:
		cryptoPub = k.Public()
	case *ed25519.PrivateKey:
		cryptoPub = k.Public()
	default:
		return nil, fmt.Errorf("unsupported key type: %T", raw)
	}

	pub, err := gossh.NewPublicKey(cryptoPub)
	if err != nil {
		return nil, fmt.Errorf("create ssh public key: %w", err)
	}

	return gossh.MarshalAuthorizedKey(pub), nil
}

// ---------------------------------------------------------------------------
// localStorage: GitSSHKeyImport
// ---------------------------------------------------------------------------

func (l *localStorage) GitSSHKeyImport(_ context.Context, sessionID, name, privateKey, publicKey, host string) error {
	if name == "" || privateKey == "" {
		return fmt.Errorf("name and private_key are required")
	}

	wsBase, err := l.wsBase(sessionID)
	if err != nil {
		return err
	}
	if err := ensureConfigDirs(wsBase); err != nil {
		return fmt.Errorf("ensure dirs: %w", err)
	}

	keyPath := filepath.Join(wsBase, "ssh", "keys", name+".key")
	if err := writeSecureFile(keyPath, []byte(privateKey)); err != nil {
		return fmt.Errorf("write private key: %w", err)
	}

	pubContent := publicKey
	if pubContent == "" {
		derived, err := derivePublicKey([]byte(privateKey))
		if err != nil {
			return fmt.Errorf("derive public key: %w", err)
		}
		pubContent = string(derived)
	}

	pubPath := filepath.Join(wsBase, "ssh", "keys", name+".pub")
	if err := writeSecureFile(pubPath, []byte(pubContent)); err != nil {
		return fmt.Errorf("write public key: %w", err)
	}

	if host != "" {
		if err := updateSSHConfig(wsBase, name, host); err != nil {
			return fmt.Errorf("update ssh config: %w", err)
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// localStorage: GitSSHKeyList
// ---------------------------------------------------------------------------

func (l *localStorage) GitSSHKeyList(_ context.Context, sessionID string) ([]GitSSHKeyEntry, error) {
	wsBase, err := l.wsBase(sessionID)
	if err != nil {
		return nil, err
	}
	keysDir := filepath.Join(wsBase, "ssh", "keys")

	matches, _ := filepath.Glob(filepath.Join(keysDir, "*.pub"))
	var keys []GitSSHKeyEntry
	for _, pubPath := range matches {
		base := filepath.Base(pubPath)
		name := strings.TrimSuffix(base, ".pub")

		data, err := os.ReadFile(pubPath)
		if err != nil {
			continue
		}

		pubContent := strings.TrimSpace(string(data))
		fingerprint := ""
		if pub, _, _, _, err := gossh.ParseAuthorizedKey(data); err == nil {
			fingerprint = gossh.FingerprintSHA256(pub)
		}

		keys = append(keys, GitSSHKeyEntry{
			Name:        name,
			PublicKey:   pubContent,
			Fingerprint: fingerprint,
		})
	}

	return keys, nil
}

// ---------------------------------------------------------------------------
// localStorage: GitSSHKeyDelete
// ---------------------------------------------------------------------------

func (l *localStorage) GitSSHKeyDelete(_ context.Context, sessionID, name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}

	wsBase, err := l.wsBase(sessionID)
	if err != nil {
		return err
	}
	keysDir := filepath.Join(wsBase, "ssh", "keys")

	_ = os.Remove(filepath.Join(keysDir, name+".key"))
	_ = os.Remove(filepath.Join(keysDir, name+".pub"))

	cfgPath := filepath.Join(wsBase, "ssh", "config")
	if data, err := os.ReadFile(cfgPath); err == nil {
		cleaned := removeSSHHostBlockByKeyName(string(data), name+".key")
		_ = writeSecureFile(cfgPath, []byte(cleaned))
	}

	return nil
}
