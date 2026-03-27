package commercial

import (
	"crypto/x509"
	_ "embed"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

//go:embed testdata/test-intermediate-ca.pem
var testIntermediateCAPEM []byte

//go:embed testdata/test-license.pem
var testLicensePEM []byte

// withTestRootPool temporarily replaces the root pool and revocation list
// for testing, restoring originals on cleanup.
func withTestRootPool(t *testing.T, ca *testCA, revoked []*big.Int) {
	t.Helper()

	origPool := rootPool
	origOnce := rootPoolOnce
	origSerials := revokedSerials
	origRevokedOnce := revokedOnce

	pool := x509.NewCertPool()
	pool.AddCert(ca.Cert)
	rootPool = pool
	doneOnce := &sync.Once{}
	doneOnce.Do(func() {}) // pre-mark as done so RootPool() returns our pool
	rootPoolOnce = doneOnce
	revokedSerials = revoked
	doneOnce2 := &sync.Once{}
	doneOnce2.Do(func() {})
	revokedOnce = doneOnce2

	t.Cleanup(func() {
		rootPool = origPool
		rootPoolOnce = origOnce
		revokedSerials = origSerials
		revokedOnce = origRevokedOnce
	})
}

func TestNoCertificate(t *testing.T) {
	dir := t.TempDir()
	License = DefaultLicense()
	Load(dir, "yao")

	if License.Valid {
		t.Fatal("expected Valid=false with no certificate")
	}
	if License.Source != "none" {
		t.Fatalf("expected Source=none, got %s", License.Source)
	}
	if License.Edition != "community" {
		t.Fatalf("expected Edition=community, got %s", License.Edition)
	}
	if License.MaxUsers != 100 {
		t.Fatalf("expected MaxUsers=100, got %d", License.MaxUsers)
	}
}

func TestValidCertificate(t *testing.T) {
	root, err := generateRootCA("Test Root CA", 10*365*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	withTestRootPool(t, root, nil)

	opts := defaultLicenseOpts()
	leaf, err := generateLicenseCert(root, opts)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "license.pem"), leaf.CertPEM, 0644); err != nil {
		t.Fatal(err)
	}

	Load(dir, "yao")

	if !License.Valid {
		t.Fatalf("expected Valid=true, got error: %s", License.Error)
	}
	if License.Source != "file" {
		t.Fatalf("expected Source=file, got %s", License.Source)
	}
	if License.Edition != "pro" {
		t.Fatalf("expected Edition=pro, got %s", License.Edition)
	}
	if License.LicenseeName != "Test Corp" {
		t.Fatalf("expected LicenseeName=Test Corp, got %s", License.LicenseeName)
	}
	if License.MaxUsers != 500 {
		t.Fatalf("expected MaxUsers=500, got %d", License.MaxUsers)
	}
}

func TestExpiredCertificate(t *testing.T) {
	root, err := generateRootCA("Test Root CA", 10*365*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	withTestRootPool(t, root, nil)

	opts := defaultLicenseOpts()
	opts.NotBefore = time.Now().Add(-30 * time.Minute)
	opts.NotAfter = time.Now().Add(-1 * time.Minute)
	leaf, err := generateLicenseCert(root, opts)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "license.pem"), leaf.CertPEM, 0644)

	Load(dir, "yao")

	if License.Valid {
		t.Fatal("expected Valid=false for expired certificate")
	}
	if !License.IsExpired {
		t.Fatal("expected IsExpired=true")
	}
}

func TestNotYetValidCertificate(t *testing.T) {
	root, err := generateRootCA("Test Root CA", 10*365*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	withTestRootPool(t, root, nil)

	opts := defaultLicenseOpts()
	opts.NotBefore = time.Now().Add(24 * time.Hour)
	opts.NotAfter = time.Now().Add(365 * 24 * time.Hour)
	leaf, err := generateLicenseCert(root, opts)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "license.pem"), leaf.CertPEM, 0644)

	Load(dir, "yao")

	if License.Valid {
		t.Fatal("expected Valid=false for not-yet-valid certificate")
	}
	if License.IsExpired {
		t.Fatal("expected IsExpired=false for future certificate")
	}
}

func TestTamperedCertificate(t *testing.T) {
	root, err := generateRootCA("Test Root CA", 10*365*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	withTestRootPool(t, root, nil)

	// Generate with a different root (not in our pool) to simulate tampering
	fakeRoot, err := generateRootCA("Fake Root CA", 10*365*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	opts := defaultLicenseOpts()
	leaf, err := generateLicenseCert(fakeRoot, opts)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "license.pem"), leaf.CertPEM, 0644)

	Load(dir, "yao")

	if License.Valid {
		t.Fatal("expected Valid=false for tampered certificate")
	}
	if License.Error == "" {
		t.Fatal("expected Error to be set")
	}
}

func TestWrongProduct(t *testing.T) {
	root, err := generateRootCA("Test Root CA", 10*365*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	withTestRootPool(t, root, nil)

	opts := defaultLicenseOpts()
	// Only licensed for "tai", not "yao"
	for i, ext := range opts.Extensions {
		if ext.OID.Equal(OIDProduct) {
			opts.Extensions[i].Value = "tai"
		}
	}
	leaf, err := generateLicenseCert(root, opts)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "license.pem"), leaf.CertPEM, 0644)

	Load(dir, "yao")

	if License.Valid {
		t.Fatal("expected Valid=false for wrong product")
	}
}

func TestCertificateChainWithIntermediate(t *testing.T) {
	root, err := generateRootCA("Test Root CA", 10*365*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	withTestRootPool(t, root, nil)

	intermediate, err := generateIntermediateCA("Test Intermediate CA", 3*365*24*time.Hour, root)
	if err != nil {
		t.Fatal(err)
	}

	opts := defaultLicenseOpts()
	leaf, err := generateLicenseCert(intermediate, opts)
	if err != nil {
		t.Fatal(err)
	}

	// PEM chain: leaf + intermediate
	chainPEM := append(leaf.CertPEM, intermediate.CertPEM...)

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "license.pem"), chainPEM, 0644)

	Load(dir, "yao")

	if !License.Valid {
		t.Fatalf("expected Valid=true with intermediate chain, got error: %s", License.Error)
	}
}

func TestRevokedCertificate(t *testing.T) {
	root, err := generateRootCA("Test Root CA", 10*365*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	opts := defaultLicenseOpts()
	opts.Serial = big.NewInt(99999)
	leaf, err := generateLicenseCert(root, opts)
	if err != nil {
		t.Fatal(err)
	}

	withTestRootPool(t, root, []*big.Int{big.NewInt(99999)})

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "license.pem"), leaf.CertPEM, 0644)

	Load(dir, "yao")

	if License.Valid {
		t.Fatal("expected Valid=false for revoked certificate")
	}
}

func TestEnvVarLoading(t *testing.T) {
	root, err := generateRootCA("Test Root CA", 10*365*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	withTestRootPool(t, root, nil)

	opts := defaultLicenseOpts()
	leaf, err := generateLicenseCert(root, opts)
	if err != nil {
		t.Fatal(err)
	}

	// Write cert to a temp file and point env var to it
	certFile := filepath.Join(t.TempDir(), "test-license.pem")
	os.WriteFile(certFile, leaf.CertPEM, 0644)

	t.Setenv(envLicenseCert, certFile)

	Load(t.TempDir(), "yao")

	if !License.Valid {
		t.Fatalf("expected Valid=true via env, got error: %s", License.Error)
	}
	if License.Source != "env" {
		t.Fatalf("expected Source=env, got %s", License.Source)
	}
}

func TestAllExtensions(t *testing.T) {
	root, err := generateRootCA("Test Root CA", 10*365*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	withTestRootPool(t, root, nil)

	opts := defaultLicenseOpts()
	opts.Extensions = []ExtensionValue{
		{OID: OIDProduct, Value: "yao,tai"},
		{OID: OIDEdition, Value: "enterprise"},
		{OID: OIDEnv, Value: "production,staging"},
		{OID: OIDDomain, Value: "*.acme.com"},
		{OID: OIDAppID, Value: "acme-crm"},
		{OID: OIDMaxUsers, Value: "0"},
		{OID: OIDMaxTaiNodes, Value: "0"},
		{OID: OIDMaxAgents, Value: "0"},
		{OID: OIDMaxSandboxes, Value: "0"},
		{OID: OIDMaxAPIRPM, Value: "0"},
		{OID: OIDMaxStorageGB, Value: "0"},
		{OID: OIDAllowBrandingRemoval, Value: "true"},
		{OID: OIDAllowWhiteLabel, Value: "true"},
		{OID: OIDAllowMultiTenant, Value: "true"},
		{OID: OIDAllowCustomDomain, Value: "true"},
		{OID: OIDAllowHostExec, Value: "true"},
		{OID: OIDAllowSSO, Value: "true"},
		{OID: OIDSupportLevel, Value: "dedicated"},
	}

	leaf, err := generateLicenseCert(root, opts)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "license.pem"), leaf.CertPEM, 0644)

	Load(dir, "yao")

	if !License.Valid {
		t.Fatalf("expected Valid=true, got error: %s", License.Error)
	}
	if License.Edition != "enterprise" {
		t.Fatalf("expected enterprise, got %s", License.Edition)
	}
	if !License.HasProduct("yao") || !License.HasProduct("tai") {
		t.Fatalf("expected product yao,tai, got %v", License.Product)
	}
	if len(License.Env) != 2 {
		t.Fatalf("expected 2 envs, got %v", License.Env)
	}
	if License.Domain != "*.acme.com" {
		t.Fatalf("expected domain *.acme.com, got %s", License.Domain)
	}
	if License.AppID != "acme-crm" {
		t.Fatalf("expected app_id acme-crm, got %s", License.AppID)
	}
	if License.MaxUsers != 0 {
		t.Fatalf("expected MaxUsers=0 (unlimited), got %d", License.MaxUsers)
	}
	if !License.Permissions.AllowBrandingRemoval {
		t.Fatal("expected AllowBrandingRemoval=true")
	}
	if !License.Permissions.AllowWhiteLabel {
		t.Fatal("expected AllowWhiteLabel=true")
	}
	if !License.Permissions.AllowMultiTenant {
		t.Fatal("expected AllowMultiTenant=true")
	}
	if !License.Permissions.AllowSSO {
		t.Fatal("expected AllowSSO=true")
	}
	if License.Permissions.SupportLevel != "dedicated" {
		t.Fatalf("expected SupportLevel=dedicated, got %s", License.Permissions.SupportLevel)
	}
}

func TestDefaultLicenseAndHelpers(t *testing.T) {
	def := DefaultLicense()

	if def.Valid {
		t.Fatal("default should not be Valid")
	}
	if def.Edition != "community" {
		t.Fatalf("expected community, got %s", def.Edition)
	}
	if !def.IsLevel("community") {
		t.Fatal("community should satisfy IsLevel(community)")
	}
	if def.IsLevel("starter") {
		t.Fatal("community should not satisfy IsLevel(starter)")
	}
	if def.IsLevel("pro") {
		t.Fatal("community should not satisfy IsLevel(pro)")
	}

	pro := LicenseInfo{Edition: "pro"}
	if !pro.IsLevel("community") {
		t.Fatal("pro should satisfy IsLevel(community)")
	}
	if !pro.IsLevel("starter") {
		t.Fatal("pro should satisfy IsLevel(starter)")
	}
	if !pro.IsLevel("pro") {
		t.Fatal("pro should satisfy IsLevel(pro)")
	}
	if pro.IsLevel("enterprise") {
		t.Fatal("pro should not satisfy IsLevel(enterprise)")
	}

	multi := LicenseInfo{Product: []string{"yao", "tai"}}
	if !multi.HasProduct("yao") {
		t.Fatal("expected HasProduct(yao)=true")
	}
	if !multi.HasProduct("tai") {
		t.Fatal("expected HasProduct(tai)=true")
	}
	if multi.HasProduct("other") {
		t.Fatal("expected HasProduct(other)=false")
	}
}

// TestRealChainIntermediate verifies a certificate signed by a real test intermediate CA,
// which in turn is signed by the real Root CA 1 embedded in the binary.
// Uses testdata/test-intermediate-ca.pem and testdata/test-license.pem generated by
// /Volumes/DATA/Work/Yaobots/keys/gen-test-certs.go.
func TestRealChainIntermediate(t *testing.T) {
	// Use the real embedded root pool (not overridden)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "license.pem"), testLicensePEM, 0644); err != nil {
		t.Fatal(err)
	}

	License = DefaultLicense()
	Load(dir, "yao")

	if !License.Valid {
		t.Fatalf("expected Valid=true with real chain, got error: %s", License.Error)
	}
	if License.Edition != "enterprise" {
		t.Fatalf("expected Edition=enterprise, got %s", License.Edition)
	}
	if !License.HasProduct("yao") {
		t.Fatalf("expected HasProduct(yao)=true, got products: %v", License.Product)
	}
	if !License.Permissions.AllowBrandingRemoval {
		t.Fatal("expected AllowBrandingRemoval=true")
	}
	if !License.Permissions.AllowWhiteLabel {
		t.Fatal("expected AllowWhiteLabel=true")
	}
	if !License.IsLevel("enterprise") {
		t.Fatalf("expected IsLevel(enterprise)=true, edition=%s", License.Edition)
	}
}

// TestRealIntermediateCACert parses the embedded test intermediate CA cert
// and verifies it is signed by the real Root CA 1.
func TestRealIntermediateCACert(t *testing.T) {
	intCert, err := ParsePEMChain(testIntermediateCAPEM)
	if err != nil || len(intCert) == 0 {
		t.Fatalf("failed to parse test intermediate CA: %v", err)
	}

	opts := x509.VerifyOptions{
		Roots:       RootPool(),
		CurrentTime: time.Now(),
	}
	// Intermediate CA certs are not end-entity; relax KeyUsages check
	opts.KeyUsages = []x509.ExtKeyUsage{x509.ExtKeyUsageAny}
	if _, err := intCert[0].Verify(opts); err != nil {
		t.Fatalf("test intermediate CA not verified by real root pool: %v", err)
	}
}

func TestCertsSubdirectoryFallback(t *testing.T) {
	root, err := generateRootCA("Test Root CA", 10*365*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	withTestRootPool(t, root, nil)

	opts := defaultLicenseOpts()
	leaf, err := generateLicenseCert(root, opts)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	certsDir := filepath.Join(dir, "certs")
	os.MkdirAll(certsDir, 0755)
	os.WriteFile(filepath.Join(certsDir, "license.pem"), leaf.CertPEM, 0644)

	Load(dir, "yao")

	if !License.Valid {
		t.Fatalf("expected Valid=true from certs/ fallback, got error: %s", License.Error)
	}
	if License.Source != "file" {
		t.Fatalf("expected Source=file, got %s", License.Source)
	}
}

func TestMachineIDEmpty(t *testing.T) {
	// No machine_id in cert → valid on any machine
	root, err := generateRootCA("Test Root CA", 10*365*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	withTestRootPool(t, root, nil)

	opts := defaultLicenseOpts()
	// defaultLicenseOpts has no OIDMachineID → empty
	leaf, err := generateLicenseCert(root, opts)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "license.pem"), leaf.CertPEM, 0644)
	Load(dir, "yao")

	if !License.Valid {
		t.Fatalf("expected Valid=true when machine_id is empty, got: %s", License.Error)
	}
}

func TestMachineIDMatch(t *testing.T) {
	// machine_id in cert matches current machine → valid
	root, err := generateRootCA("Test Root CA", 10*365*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	withTestRootPool(t, root, nil)

	thisID := currentMachineID()
	opts := defaultLicenseOpts()
	opts.Extensions = append(opts.Extensions, ExtensionValue{OID: OIDMachineID, Value: thisID})
	leaf, err := generateLicenseCert(root, opts)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "license.pem"), leaf.CertPEM, 0644)
	Load(dir, "yao")

	if !License.Valid {
		t.Fatalf("expected Valid=true when machine_id matches, got: %s", License.Error)
	}
	if License.MachineID != thisID {
		t.Fatalf("expected MachineID=%s, got %s", thisID, License.MachineID)
	}
}

func TestMachineIDMismatch(t *testing.T) {
	// machine_id in cert does not match current machine → invalid
	root, err := generateRootCA("Test Root CA", 10*365*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	withTestRootPool(t, root, nil)

	opts := defaultLicenseOpts()
	opts.Extensions = append(opts.Extensions, ExtensionValue{OID: OIDMachineID, Value: "000000000000000000000000deadbeef"})
	leaf, err := generateLicenseCert(root, opts)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "license.pem"), leaf.CertPEM, 0644)
	Load(dir, "yao")

	if License.Valid {
		t.Fatal("expected Valid=false when machine_id does not match")
	}
	if License.Error == "" {
		t.Fatal("expected Error to be set")
	}
}
