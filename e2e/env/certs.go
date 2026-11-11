package env

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// ensureSigningCerts creates a self-signed signing certificate if missing.
func ensureSigningCerts(appRoot string) error {
	dir := filepath.Join(appRoot, "openapi", "certs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return ensureCertPair(dir, "signing", "Yao E2E Signing")
}

// ensureNodeTokenCerts creates the independent node token key pair.
func ensureNodeTokenCerts(appRoot string) error {
	dir := filepath.Join(appRoot, "openapi", "certs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return ensureCertPair(dir, "node-token", "Yao E2E Node Token")
}

func ensureCertPair(dir, prefix, commonName string) error {
	certPath := filepath.Join(dir, prefix+"-cert.pem")
	keyPath := filepath.Join(dir, prefix+"-key.pem")
	if fileExists(certPath) && fileExists(keyPath) {
		return nil
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	if err := os.WriteFile(certPath, certPEM, 0o644); err != nil {
		return err
	}
	return os.WriteFile(keyPath, keyPEM, 0o600)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
