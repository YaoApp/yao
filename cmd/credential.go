package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Credential represents the stored OAuth credential for gRPC mode.
type Credential struct {
	Server       string `json:"server"`
	GRPCAddr     string `json:"grpc_addr,omitempty"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
	User         string `json:"user,omitempty"`
	ExpiresAt    string `json:"expires_at,omitempty"`
}

// Expired returns true if the credential has an expires_at in the past.
func (c *Credential) Expired() bool {
	if c.ExpiresAt == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339, c.ExpiresAt)
	if err != nil {
		return false
	}
	return time.Now().After(t)
}

func credentialPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".yao", "credentials"), nil
}

// LoadCredential reads and decodes ~/.yao/credentials. Returns nil if the file
// does not exist.
func LoadCredential() (*Credential, error) {
	path, err := credentialPath()
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read credentials: %w", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(string(raw))
	if err != nil {
		return nil, fmt.Errorf("decode credentials: %w", err)
	}

	var cred Credential
	if err := json.Unmarshal(decoded, &cred); err != nil {
		return nil, fmt.Errorf("unmarshal credentials: %w", err)
	}
	return &cred, nil
}

// LoadCredentialFrom reads and decodes a credential file from a custom path.
func LoadCredentialFrom(path string) (*Credential, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read credentials from %s: %w", path, err)
	}
	decoded, err := base64.StdEncoding.DecodeString(string(raw))
	if err != nil {
		return nil, fmt.Errorf("decode credentials: %w", err)
	}
	var cred Credential
	if err := json.Unmarshal(decoded, &cred); err != nil {
		return nil, fmt.Errorf("unmarshal credentials: %w", err)
	}
	return &cred, nil
}

// SaveCredential encodes and writes the credential to ~/.yao/credentials.
func SaveCredential(cred *Credential) error {
	path, err := credentialPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}
	data, err := json.Marshal(cred)
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	if err := os.WriteFile(path, []byte(encoded), 0600); err != nil {
		return fmt.Errorf("write credentials: %w", err)
	}
	return nil
}

// RemoveCredential deletes ~/.yao/credentials.
func RemoveCredential() error {
	path, err := credentialPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove credentials: %w", err)
	}
	return nil
}
