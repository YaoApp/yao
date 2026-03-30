package commercial

import (
	"crypto/x509"
	_ "embed"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"sync"
)

//go:embed roots/root-ca-1.pem
var rootCA1PEM []byte

//go:embed roots/root-ca-2.pem
var rootCA2PEM []byte

//go:embed roots/revoked.json
var revokedJSON []byte

var (
	rootPoolOnce = &sync.Once{}
	rootPool     *x509.CertPool

	revokedOnce    = &sync.Once{}
	revokedSerials []*big.Int
)

// RootPool returns the built-in root certificate pool (primary + backup).
// Returns nil if neither root certificate could be parsed (e.g. dev placeholder).
func RootPool() *x509.CertPool {
	rootPoolOnce.Do(func() {
		pool := x509.NewCertPool()
		added := false
		for _, pemData := range [][]byte{rootCA1PEM, rootCA2PEM} {
			if pool.AppendCertsFromPEM(pemData) {
				added = true
			}
		}
		if added {
			rootPool = pool
		}
	})
	return rootPool
}

// RevokedSerials returns the list of revoked certificate serial numbers
// embedded in the binary.
func RevokedSerials() []*big.Int {
	revokedOnce.Do(func() {
		var data struct {
			Serials []string `json:"serials"`
		}
		if err := json.Unmarshal(revokedJSON, &data); err != nil {
			return
		}
		for _, s := range data.Serials {
			n := new(big.Int)
			if _, ok := n.SetString(s, 0); ok {
				revokedSerials = append(revokedSerials, n)
			}
		}
	})
	return revokedSerials
}

// IsRevoked checks whether the given serial number is in the revocation list.
func IsRevoked(serial *big.Int) bool {
	for _, s := range RevokedSerials() {
		if s.Cmp(serial) == 0 {
			return true
		}
	}
	return false
}

// ParsePEMChain parses a PEM bundle into a list of x509 certificates.
// The first certificate is treated as the leaf; remaining are intermediates.
func ParsePEMChain(pemData []byte) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate
	rest := pemData
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		certs = append(certs, cert)
	}
	return certs, nil
}
