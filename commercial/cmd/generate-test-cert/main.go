// Command generate-test-cert creates a test root CA and a license certificate
// for local development and testing.
//
// Usage:
//
//	go run ./commercial/cmd/generate-test-cert \
//	    -out-cert /path/to/yao-dev-app/license.pem \
//	    -out-root-ca ./commercial/roots/root-ca-1.pem
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/yaoapp/yao/commercial"
)

func main() {
	outCert := flag.String("out-cert", "license.pem", "path to write the license certificate PEM")
	outRootCA := flag.String("out-root-ca", "", "path to write the root CA PEM (optional; for injecting into commercial/roots/)")
	flag.Parse()

	rootKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate root key: %v\n", err)
		os.Exit(1)
	}

	rootSerial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	rootTmpl := &x509.Certificate{
		SerialNumber: rootSerial,
		Subject: pkix.Name{
			CommonName:   "Yao Test Root CA",
			Organization: []string{"Infinite Wisdom Software"},
			Country:      []string{"CN"},
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	rootDER, err := x509.CreateCertificate(rand.Reader, rootTmpl, rootTmpl, &rootKey.PublicKey, rootKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create root cert: %v\n", err)
		os.Exit(1)
	}

	rootCert, _ := x509.ParseCertificate(rootDER)
	rootPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rootDER})

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate leaf key: %v\n", err)
		os.Exit(1)
	}

	leafTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(20001),
		Subject: pkix.Name{
			CommonName:   "Yao Dev App",
			Organization: []string{"Dev Testing"},
			Country:      []string{"CN"},
		},
		EmailAddresses: []string{"dev@yaoapps.com"},
		NotBefore:      time.Now().Add(-1 * time.Hour),
		NotAfter:       time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:       x509.KeyUsageDigitalSignature,
		ExtKeyUsage:    []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		ExtraExtensions: []pkix.Extension{
			{Id: commercial.OIDProduct, Value: []byte("yao,tai")},
			{Id: commercial.OIDEdition, Value: []byte("enterprise")},
			{Id: commercial.OIDMaxUsers, Value: []byte("0")},
			{Id: commercial.OIDMaxTaiNodes, Value: []byte("0")},
			{Id: commercial.OIDMaxAgents, Value: []byte("0")},
			{Id: commercial.OIDMaxSandboxes, Value: []byte("0")},
			{Id: commercial.OIDMaxAPIRPM, Value: []byte("0")},
			{Id: commercial.OIDMaxStorageGB, Value: []byte("0")},
			{Id: commercial.OIDAllowBrandingRemoval, Value: []byte("true")},
			{Id: commercial.OIDAllowWhiteLabel, Value: []byte("true")},
			{Id: commercial.OIDAllowMultiTenant, Value: []byte("true")},
			{Id: commercial.OIDAllowCustomDomain, Value: []byte("true")},
			{Id: commercial.OIDAllowHostExec, Value: []byte("true")},
			{Id: commercial.OIDAllowSSO, Value: []byte("true")},
			{Id: commercial.OIDSupportLevel, Value: []byte("dedicated")},
		},
	}

	leafDER, err := x509.CreateCertificate(rand.Reader, leafTmpl, rootCert, &leafKey.PublicKey, rootKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create leaf cert: %v\n", err)
		os.Exit(1)
	}

	leafPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER})

	if err := os.WriteFile(*outCert, leafPEM, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write cert: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Wrote license certificate to %s\n", *outCert)

	if *outRootCA != "" {
		if err := os.WriteFile(*outRootCA, rootPEM, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "write root CA: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Wrote root CA to %s\n", *outRootCA)
	}
}
