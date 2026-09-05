package oauth_test

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

func TestKnownDefaultKeyFingerprint_MatchesYaoInit(t *testing.T) {
	// Read the actual default key from the yao-init sibling repo.
	// CI may not have yao-init checked out — skip gracefully.
	yaoInitKey := filepath.Join("..", "..", "..", "yao-init", "openapi", "certs", "signing-key.pem")
	keyPEM, err := os.ReadFile(yaoInitKey)
	if err != nil {
		t.Skipf("yao-init default signing-key.pem not found (%s): %v", yaoInitKey, err)
	}

	block, _ := pem.Decode(keyPEM)
	require.NotNil(t, block, "failed to decode PEM")

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		require.NoError(t, err, "failed to parse private key")
	}

	rsaKey, ok := privateKey.(*rsa.PrivateKey)
	require.True(t, ok, "key is not RSA")

	pubDER, err := x509.MarshalPKIXPublicKey(&rsaKey.PublicKey)
	require.NoError(t, err)

	fingerprint := sha256.Sum256(pubDER)
	got := hex.EncodeToString(fingerprint[:])
	assert.Equal(t, oauth.ExportKnownDefaultKeyFingerprint, got,
		"knownDefaultKeyFingerprint must match the SHA-256 of yao-init's signing-key.pem public key")
}

func TestGenerateAndReplaceCertificates(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "signing-cert.pem")
	keyPath := filepath.Join(dir, "signing-key.pem")

	config := &types.SigningConfig{
		SigningCertPath: certPath,
		SigningKeyPath:  keyPath,
	}

	certs, err := oauth.ExportGenerateAndReplaceCertificates(config)
	require.NoError(t, err)
	require.NotNil(t, certs)

	// Files must be persisted
	certData, err := os.ReadFile(certPath)
	require.NoError(t, err)
	assert.Contains(t, string(certData), "BEGIN CERTIFICATE")

	keyData, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	assert.Contains(t, string(keyData), "BEGIN PRIVATE KEY")

	// Generated key fingerprint must differ from the default
	rsaKey, ok := certs.SigningKey.(*rsa.PrivateKey)
	require.True(t, ok)
	pubDER, err := x509.MarshalPKIXPublicKey(&rsaKey.PublicKey)
	require.NoError(t, err)
	fingerprint := sha256.Sum256(pubDER)
	got := hex.EncodeToString(fingerprint[:])
	assert.NotEqual(t, oauth.ExportKnownDefaultKeyFingerprint, got,
		"generated key must NOT match the default fingerprint")

	// Key file permission check (unix only)
	info, err := os.Stat(keyPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm(),
		"private key file must have 0600 permissions")
}
