package commercial

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const envLicenseCert = "YAO_LICENSE_CERT"

// Load discovers and verifies the commercial license certificate,
// writing the result into the global License variable.
// It never returns an error — failures degrade to community defaults.
func Load(appRoot, product string) {
	License = DefaultLicense()
	License.Product = []string{product}

	pemData, source := findCert(appRoot)
	if pemData == nil {
		log.Printf("[License] No license certificate found, running with community defaults")
		return
	}

	info, err := verify(pemData, product)
	if err != nil {
		License.Source = source
		License.Error = err.Error()
		log.Printf("[License] %v — running with community defaults", err)
		return
	}

	info.Source = source
	info.LoadedAt = time.Now().Unix()
	License = *info

	if License.Valid {
		remaining := time.Until(time.Unix(License.NotAfter, 0))
		log.Printf("[License] Loaded: %s (%s) — valid until %s",
			License.LicenseeName, License.Edition,
			time.Unix(License.NotAfter, 0).UTC().Format("2006-01-02"))
		if remaining < 90*24*time.Hour {
			log.Printf("[License] WARNING: Certificate expires in %d days", int(remaining.Hours()/24))
		}
	}
}

// findCert locates the license PEM data.
// Search order:
//  1. YAO_LICENSE_CERT env (PEM content or file path)
//  2. <appRoot>/license.pem
//  3. <appRoot>/certs/license.pem
func findCert(appRoot string) (pemData []byte, source string) {
	if v := os.Getenv(envLicenseCert); v != "" {
		if strings.HasPrefix(v, "-----BEGIN") {
			return []byte(v), "env"
		}
		data, err := os.ReadFile(v)
		if err == nil {
			return data, "env"
		}
		log.Printf("[License] env %s points to unreadable file: %v", envLicenseCert, err)
	}

	candidates := []string{
		filepath.Join(appRoot, "license.pem"),
		filepath.Join(appRoot, "certs", "license.pem"),
	}
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err == nil {
			return data, "file"
		}
	}
	return nil, "none"
}

// verify parses PEM data, validates the certificate chain against built-in
// roots, checks revocation, time validity, product scope, and extracts
// custom extension fields.
func verify(pemData []byte, product string) (*LicenseInfo, error) {
	certs, err := ParsePEMChain(pemData)
	if err != nil {
		return nil, fmt.Errorf("parse PEM: %w", err)
	}
	if len(certs) == 0 {
		return nil, fmt.Errorf("no certificates found in PEM data")
	}

	leaf := certs[0]

	pool := RootPool()
	if pool == nil {
		return nil, fmt.Errorf("no root certificates available (development build)")
	}

	// Verify the trust chain with a synthetic time within the leaf's validity
	// window. This lets us extract structured info from expired/future
	// certificates instead of returning an opaque x509 error.
	// We use NotAfter-1s (just before expiry) to maximize overlap with CA validity.
	opts := x509.VerifyOptions{
		Roots:       pool,
		CurrentTime: leaf.NotAfter.Add(-time.Second),
		KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}

	if len(certs) > 1 {
		intermediates := x509.NewCertPool()
		for _, c := range certs[1:] {
			intermediates.AddCert(c)
		}
		opts.Intermediates = intermediates
	}

	if _, err := leaf.Verify(opts); err != nil {
		return nil, fmt.Errorf("certificate verification failed: %w", err)
	}

	if IsRevoked(leaf.SerialNumber) {
		return nil, fmt.Errorf("certificate serial %s has been revoked", leaf.SerialNumber.Text(16))
	}

	info := extractIdentity(leaf)
	parseExtensions(leaf, info)

	now := time.Now()
	if now.Before(leaf.NotBefore) {
		info.Valid = false
		info.Error = fmt.Sprintf("certificate not yet valid (starts %s)",
			leaf.NotBefore.UTC().Format("2006-01-02"))
		return info, nil
	}
	if now.After(leaf.NotAfter) {
		info.Valid = false
		info.IsExpired = true
		info.Error = fmt.Sprintf("certificate expired on %s",
			leaf.NotAfter.UTC().Format("2006-01-02"))
		return info, nil
	}

	if !info.HasProduct(product) {
		info.Valid = false
		info.Error = fmt.Sprintf("certificate not licensed for product %q (licensed: %v)",
			product, info.Product)
		return info, nil
	}

	// Machine ID binding: if the certificate specifies a machine ID,
	// it must match the current runtime machine ID.
	// Empty machine_id means no binding — runs on any machine.
	if info.MachineID != "" {
		if got := currentMachineID(); got != info.MachineID {
			info.Valid = false
			info.Error = "certificate machine_id does not match this host"
			return info, nil
		}
	}

	info.Valid = true
	return info, nil
}

func extractIdentity(cert *x509.Certificate) *LicenseInfo {
	info := &LicenseInfo{
		LicenseeName: cert.Subject.CommonName,
		SerialNumber: cert.SerialNumber.Text(16),
		NotBefore:    cert.NotBefore.Unix(),
		NotAfter:     cert.NotAfter.Unix(),
		Issuer:       cert.Issuer.CommonName,
		Edition:      "community",
		Product:      []string{},
		Permissions:  Permissions{SupportLevel: "none"},
	}
	if len(cert.Subject.Organization) > 0 {
		info.LicenseeOrg = cert.Subject.Organization[0]
	}
	if len(cert.Subject.Country) > 0 {
		info.LicenseeCountry = cert.Subject.Country[0]
	}
	if len(cert.EmailAddresses) > 0 {
		info.LicenseeEmail = cert.EmailAddresses[0]
	}
	return info
}

func parseExtensions(cert *x509.Certificate, info *LicenseInfo) {
	for _, ext := range cert.Extensions {
		val := string(ext.Value)

		switch {
		// Scope
		case ext.Id.Equal(OIDProduct):
			info.Product = splitCSV(val)
		case ext.Id.Equal(OIDEdition):
			info.Edition = val
		case ext.Id.Equal(OIDEnv):
			if val != "" {
				info.Env = splitCSV(val)
			}
		case ext.Id.Equal(OIDDomain):
			info.Domain = val
		case ext.Id.Equal(OIDAppID):
			info.AppID = val
		case ext.Id.Equal(OIDMachineID):
			info.MachineID = val

		// Quota
		case ext.Id.Equal(OIDMaxUsers):
			info.MaxUsers = atoi(val)
		case ext.Id.Equal(OIDMaxTaiNodes):
			info.MaxTaiNodes = atoi(val)
		case ext.Id.Equal(OIDMaxAgents):
			info.MaxAgents = atoi(val)
		case ext.Id.Equal(OIDMaxSandboxes):
			info.MaxSandboxes = atoi(val)
		case ext.Id.Equal(OIDMaxAPIRPM):
			info.MaxAPIRPM = atoi(val)
		case ext.Id.Equal(OIDMaxStorageGB):
			info.MaxStorageGB = atoi(val)

		// Permissions
		case ext.Id.Equal(OIDAllowBrandingRemoval):
			info.Permissions.AllowBrandingRemoval = toBool(val)
		case ext.Id.Equal(OIDAllowWhiteLabel):
			info.Permissions.AllowWhiteLabel = toBool(val)
		case ext.Id.Equal(OIDAllowMultiTenant):
			info.Permissions.AllowMultiTenant = toBool(val)
		case ext.Id.Equal(OIDAllowCustomDomain):
			info.Permissions.AllowCustomDomain = toBool(val)
		case ext.Id.Equal(OIDAllowHostExec):
			info.Permissions.AllowHostExec = toBool(val)
		case ext.Id.Equal(OIDAllowSSO):
			info.Permissions.AllowSSO = toBool(val)
		case ext.Id.Equal(OIDSupportLevel):
			info.Permissions.SupportLevel = val
		}
	}
}

// MakeExtension creates a pkix.Extension for embedding in a certificate.
func MakeExtension(oid asn1.ObjectIdentifier, value string) ExtensionValue {
	return ExtensionValue{OID: oid, Value: value}
}

// ExtensionValue pairs an OID with its string value for certificate generation.
type ExtensionValue struct {
	OID   asn1.ObjectIdentifier
	Value string
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			result = append(result, t)
		}
	}
	return result
}

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

func toBool(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	return s == "true" || s == "1" || s == "yes"
}

// currentMachineID returns a deterministic identifier for the current host,
// using the same algorithm as tai/machine.ID():
//   - macOS:   IOPlatformUUID via ioreg
//   - Linux:   /etc/machine-id
//   - Windows: HKLM MachineGuid registry value
//   - fallback: sha256("tai-fallback:" + hostname)[:16]
//
// Implemented via platform-specific machine_{os}.go files in this package.
func currentMachineID() string {
	if id := platformMachineID(); id != "" {
		return id
	}
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "unknown"
	}
	h := sha256.Sum256([]byte("tai-fallback:" + hostname))
	return fmt.Sprintf("%x", h[:16])
}
