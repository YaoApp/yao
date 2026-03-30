package commercial

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"
)

// testCA holds a generated CA certificate and its private key.
type testCA struct {
	Cert    *x509.Certificate
	Key     *ecdsa.PrivateKey
	CertPEM []byte
}

// testLicenseCert holds a generated leaf (license) certificate.
type testLicenseCert struct {
	Cert    *x509.Certificate
	CertPEM []byte
}

// generateRootCA creates a self-signed ECDSA P-384 root CA for testing.
func generateRootCA(cn string, validity time.Duration) (*testCA, error) {
	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, err
	}

	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   cn,
			Organization: []string{"Infinite Wisdom Software"},
			Country:      []string{"CN"},
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(validity),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	return &testCA{Cert: cert, Key: key, CertPEM: certPEM}, nil
}

// generateIntermediateCA creates an intermediate CA signed by the given parent.
func generateIntermediateCA(cn string, validity time.Duration, parent *testCA) (*testCA, error) {
	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, err
	}

	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   cn,
			Organization: []string{"Test Partner Inc."},
			Country:      []string{"US"},
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(validity),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, parent.Cert, &key.PublicKey, parent.Key)
	if err != nil {
		return nil, err
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	return &testCA{Cert: cert, Key: key, CertPEM: certPEM}, nil
}

// licenseOpts configures a test license certificate.
type licenseOpts struct {
	CN         string
	Org        string
	Country    string
	Email      string
	NotBefore  time.Time
	NotAfter   time.Time
	Serial     *big.Int
	Extensions []ExtensionValue
}

func defaultLicenseOpts() licenseOpts {
	return licenseOpts{
		CN:        "Test Corp",
		Org:       "Test Inc.",
		Country:   "CN",
		Email:     "test@example.com",
		NotBefore: time.Now().Add(-1 * time.Hour),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),
		Serial:    big.NewInt(10042),
		Extensions: []ExtensionValue{
			{OID: OIDProduct, Value: "yao"},
			{OID: OIDEdition, Value: "pro"},
			{OID: OIDMaxUsers, Value: "500"},
			{OID: OIDMaxTaiNodes, Value: "10"},
			{OID: OIDMaxAgents, Value: "50"},
			{OID: OIDMaxSandboxes, Value: "20"},
			{OID: OIDMaxAPIRPM, Value: "10000"},
			{OID: OIDMaxStorageGB, Value: "100"},
			{OID: OIDSupportLevel, Value: "priority"},
		},
	}
}

// generateLicenseCert creates a leaf license certificate signed by the given CA.
func generateLicenseCert(signer *testCA, opts licenseOpts) (*testLicenseCert, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	tmpl := &x509.Certificate{
		SerialNumber: opts.Serial,
		Subject: pkix.Name{
			CommonName:   opts.CN,
			Organization: []string{opts.Org},
			Country:      []string{opts.Country},
		},
		EmailAddresses: []string{opts.Email},
		NotBefore:      opts.NotBefore,
		NotAfter:       opts.NotAfter,
		KeyUsage:       x509.KeyUsageDigitalSignature,
		ExtKeyUsage:    []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}

	for _, ext := range opts.Extensions {
		tmpl.ExtraExtensions = append(tmpl.ExtraExtensions, pkix.Extension{
			Id:    ext.OID,
			Value: []byte(ext.Value),
		})
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, signer.Cert, &key.PublicKey, signer.Key)
	if err != nil {
		return nil, err
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	return &testLicenseCert{Cert: cert, CertPEM: certPEM}, nil
}
