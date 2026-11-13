package oauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/gou/store/lru"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

func newTestNodeService(t *testing.T) *Service {
	t.Helper()
	signCert, signKey := makeTestCert(t, "signing")
	nodeCert, nodeKey := makeTestCert(t, "node")

	data, err := lru.New(1024)
	require.NoError(t, err)
	cache, err := lru.New(1024)
	require.NoError(t, err)
	svc, err := NewService(&Config{
		Store:     data,
		Cache:     cache,
		IssuerURL: "https://oauth.test.example.com",
		Signing: types.SigningConfig{
			SigningCertPath:  signCert,
			SigningKeyPath:   signKey,
			SigningAlgorithm: "RS256",
		},
		NodeToken: types.NodeTokenConfig{
			SigningCertPath:  nodeCert,
			SigningKeyPath:   nodeKey,
			SigningAlgorithm: "RS256",
			KID:              "nt-test",
			TTL:              time.Hour,
		},
	})
	require.NoError(t, err)
	return svc
}

func makeTestCert(t *testing.T, prefix string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: "Yao Node Test " + prefix},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	require.NoError(t, err)
	certPath := filepath.Join(dir, prefix+"-cert.pem")
	keyPath := filepath.Join(dir, prefix+"-key.pem")
	require.NoError(t, os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644))
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0o600))
	return certPath, keyPath
}

func testIssueRequest() IssueRequest {
	return IssueRequest{
		ComputerID: "cmp_test",
		NodeID:     "node-01234567",
		Scope:      "user",
		Visibility: "private",
		UserID:     "user_1",
		TeamID:     "team_1",
	}
}

func TestNodeTokenLifecycle(t *testing.T) {
	svc := newTestNodeService(t)
	ctx := context.Background()
	first, err := svc.IssueNodeToken(ctx, testIssueRequest())
	require.NoError(t, err)
	require.NotEmpty(t, first.Token)
	require.NotEmpty(t, first.JTI)

	info, err := svc.ValidateNodeToken(ctx, first.Token)
	require.NoError(t, err)
	require.Equal(t, "cmp_test", info.ComputerID)
	require.Equal(t, "node-01234567", info.NodeID)

	_, err = svc.IssueNodeToken(ctx, testIssueRequest())
	require.ErrorIs(t, err, ErrNodeTokenConflict)

	second, err := svc.RefreshNodeToken(ctx, first.Token)
	require.NoError(t, err)
	_, err = svc.ValidateNodeToken(ctx, first.Token)
	require.ErrorIs(t, err, ErrInvalidNodeToken)
	_, err = svc.ValidateNodeToken(ctx, second.Token)
	require.NoError(t, err)

	third, err := svc.RotateNodeToken(ctx, "cmp_test")
	require.NoError(t, err)
	_, err = svc.ValidateNodeToken(ctx, second.Token)
	require.ErrorIs(t, err, ErrInvalidNodeToken)
	_, err = svc.ValidateNodeToken(ctx, third.Token)
	require.NoError(t, err)

	require.NoError(t, svc.RevokeNodeToken(ctx, third.JTI))
	_, err = svc.ValidateNodeToken(ctx, third.Token)
	require.ErrorIs(t, err, ErrInvalidNodeToken)
	require.NoError(t, svc.RevokeNodeToken(ctx, third.JTI))
	require.NoError(t, svc.RevokeNodeToken(ctx, ""))
}

func TestNodeTokenIssueValidation(t *testing.T) {
	svc := newTestNodeService(t)
	cases := []IssueRequest{
		{},
		{ComputerID: "c", NodeID: "n", Scope: "bad", Visibility: "private"},
		{ComputerID: "c", NodeID: "n", Scope: "user", Visibility: "bad", UserID: "u"},
		{ComputerID: "c", NodeID: "n", Scope: "user", Visibility: "private"},
		{ComputerID: "c", NodeID: "n", Scope: "team", Visibility: "team"},
	}
	for _, req := range cases {
		_, err := svc.IssueNodeToken(context.Background(), req)
		require.Error(t, err)
	}
}

func TestNodeTokenValidateErrors(t *testing.T) {
	svc := newTestNodeService(t)
	ctx := context.Background()
	_, err := (*Service)(nil).ValidateNodeToken(ctx, "x")
	require.ErrorIs(t, err, ErrInvalidNodeToken)
	_, err = svc.ValidateNodeToken(ctx, "")
	require.ErrorIs(t, err, ErrInvalidNodeToken)

	token, err := svc.IssueNodeToken(ctx, testIssueRequest())
	require.NoError(t, err)

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	_, err = svc.ValidateNodeToken(cancelled, token.Token)
	require.ErrorIs(t, err, context.Canceled)

	// Remove each store entry in turn.
	hash := nodeTokenHash(token.Token)
	require.NoError(t, svc.store.Del(svc.prefix+"oauth:node_token:"+hash))
	_, err = svc.ValidateNodeToken(ctx, token.Token)
	require.ErrorIs(t, err, ErrInvalidNodeToken)

	token, err = svc.IssueNodeToken(ctx, testIssueRequest())
	require.NoError(t, err)
	require.NoError(t, svc.store.Del(svc.nodeTokenJTIKey(token.JTI)))
	_, err = svc.ValidateNodeToken(ctx, token.Token)
	require.ErrorIs(t, err, ErrInvalidNodeToken)

	token, err = svc.IssueNodeToken(ctx, testIssueRequest())
	require.NoError(t, err)
	require.NoError(t, svc.store.Del(svc.nodeTokenComputerKey("cmp_test")))
	_, err = svc.ValidateNodeToken(ctx, token.Token)
	require.ErrorIs(t, err, ErrInvalidNodeToken)

	// Tampered raw token.
	_, err = svc.ValidateNodeToken(ctx, token.Token+"x")
	require.ErrorIs(t, err, ErrInvalidNodeToken)
}

func TestNodeTokenClaimsValidation(t *testing.T) {
	svc := newTestNodeService(t)
	ctx := context.Background()
	base := map[string]interface{}{
		"iss": "yao", "aud": "tai", "sub": "node-01234567", "purpose": "node",
		"node_id": "node-01234567", "computer_id": "cmp_test", "scope": "user",
		"visibility": "private", "user_id": "user_1", "team_id": "team_1",
		"jti": "nt_test", "iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
		"kid": "nt-test",
	}
	sign := func(claims map[string]interface{}, kid string) string {
		tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims(claims))
		tok.Header["kid"] = kid
		raw, err := tok.SignedString(svc.nodeSigningCerts.SigningKey)
		require.NoError(t, err)
		return raw
	}
	cases := []struct {
		name   string
		claims map[string]interface{}
		kid    string
	}{
		{"wrong kid", base, "other"},
		{"wrong iss", withClaim(base, "iss", "other"), "nt-test"},
		{"wrong purpose", withClaim(base, "purpose", "access"), "nt-test"},
		{"wrong audience", withClaim(base, "aud", "other"), "nt-test"},
		{"missing jti", withClaim(base, "jti", ""), "nt-test"},
		{"missing computer", withClaim(base, "computer_id", ""), "nt-test"},
		{"missing node", withClaim(base, "node_id", ""), "nt-test"},
		{"missing scope", withClaim(base, "scope", ""), "nt-test"},
		{"missing visibility", withClaim(base, "visibility", ""), "nt-test"},
		{"sub mismatch", withClaim(base, "sub", "other"), "nt-test"},
		{"expired", withClaim(base, "exp", time.Now().Add(-time.Hour).Unix()), "nt-test"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.ValidateNodeToken(ctx, sign(tc.claims, tc.kid))
			require.Error(t, err)
		})
	}
}

func withClaim(base map[string]interface{}, key string, value interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(base))
	for k, v := range base {
		out[k] = v
	}
	out[key] = value
	return out
}

func TestNodeTokenStoreFailures(t *testing.T) {
	svc := newTestNodeService(t)
	svc.store = &failingStore{Store: svc.store, failSet: true}
	_, err := svc.IssueNodeToken(context.Background(), testIssueRequest())
	require.Error(t, err)
	svc.store = &failingStore{Store: svc.store, failIncr: true}
	_, err = svc.IssueNodeToken(context.Background(), testIssueRequest())
	require.Error(t, err)
}

func TestNodeTokenRefreshRotateErrors(t *testing.T) {
	svc := newTestNodeService(t)
	ctx := context.Background()
	_, err := svc.RefreshNodeToken(ctx, "bad")
	require.Error(t, err)
	_, err = svc.RotateNodeToken(ctx, "")
	require.Error(t, err)
	_, err = svc.RotateNodeToken(ctx, "missing")
	require.ErrorIs(t, err, ErrNodeTokenConflict)
	_, err = svc.RefreshNodeToken(ctx, mustIssue(t, svc).Token)
	require.NoError(t, err)
}

func mustIssue(t *testing.T, svc *Service) *NodeToken {
	t.Helper()
	return mustIssueForComputer(t, svc, "cmp_test")
}

func mustIssueForComputer(t *testing.T, svc *Service, computerID string) *NodeToken {
	t.Helper()
	req := testIssueRequest()
	req.ComputerID = computerID
	token, err := svc.IssueNodeToken(context.Background(), req)
	require.NoError(t, err)
	return token
}

func TestNodeTokenLockBusy(t *testing.T) {
	svc := newTestNodeService(t)
	svc.store = &failingStore{Store: svc.store, busyIncr: true}
	err := svc.withNodeTokenLock(context.Background(), "cmp_test", func() error { return nil })
	require.ErrorIs(t, err, ErrNodeTokenBusy)
}

type failingStore struct {
	store.Store
	failSet  bool
	failIncr bool
	busyIncr bool
}

func (s *failingStore) Incr(key string, delta int64) (int64, error) {
	if s.failIncr {
		return 0, errors.New("incr failed")
	}
	if s.busyIncr {
		return 2, nil
	}
	return s.Store.Incr(key, delta)
}

func (s *failingStore) Set(key string, value interface{}, ttl time.Duration) error {
	if s.failSet {
		return errors.New("set failed")
	}
	return s.Store.Set(key, value, ttl)
}

func TestNodeTokenSignatureMutations(t *testing.T) {
	svc := newTestNodeService(t)
	ctx := context.Background()
	token := mustIssue(t, svc)
	parts := strings.Split(token.Token, ".")
	require.Len(t, parts, 3)
	_, err := svc.ValidateNodeToken(ctx, parts[0]+"."+parts[1]+".bad")
	require.Error(t, err)
	_, err = svc.ValidateNodeToken(ctx, "not-a-jwt")
	require.Error(t, err)
}

func TestNodeTokenStaleAndCorruptState(t *testing.T) {
	svc := newTestNodeService(t)
	ctx := context.Background()

	// Stale computer index is cleaned before a new issue.
	require.NoError(t, svc.store.Set(svc.nodeTokenComputerKey("cmp_test"), "nt_missing", time.Hour))
	first, err := svc.IssueNodeToken(ctx, testIssueRequest())
	require.NoError(t, err)

	// Empty jti hash.
	require.NoError(t, svc.store.Set(svc.nodeTokenJTIKey(first.JTI), "", time.Hour))
	_, err = svc.ValidateNodeToken(ctx, first.Token)
	require.ErrorIs(t, err, ErrInvalidNodeToken)

	second := mustIssueForComputer(t, svc, "cmp_second")
	// Missing token metadata behind a valid jti hash.
	require.NoError(t, svc.store.Set(svc.nodeTokenJTIKey(second.JTI), "missing-hash", time.Hour))
	_, err = svc.ValidateNodeToken(ctx, second.Token)
	require.ErrorIs(t, err, ErrInvalidNodeToken)

	third := mustIssueForComputer(t, svc, "cmp_third")
	// Metadata mismatch.
	hash := nodeTokenHash(third.Token)
	value, _ := svc.store.Get(svc.prefix + "oauth:node_token:" + hash)
	bad := *value.(*nodeTokenMetadata)
	bad.NodeID = "other-node"
	require.NoError(t, svc.store.Set(svc.prefix+"oauth:node_token:"+hash, &bad, time.Hour))
	_, err = svc.ValidateNodeToken(ctx, third.Token)
	require.ErrorIs(t, err, ErrInvalidNodeToken)

	fourth := mustIssueForComputer(t, svc, "cmp_fourth")
	require.NoError(t, svc.store.Set(svc.nodeTokenComputerKey("cmp_fourth"), "other-jti", time.Hour))
	_, err = svc.ValidateNodeToken(ctx, fourth.Token)
	require.ErrorIs(t, err, ErrInvalidNodeToken)
}

func TestNodeTokenRefreshInvalidInsideLock(t *testing.T) {
	svc := newTestNodeService(t)
	ctx := context.Background()
	token := mustIssue(t, svc)
	// The first parse succeeds, the in-lock validation must fail.
	require.NoError(t, svc.store.Del(svc.nodeTokenComputerKey("cmp_test")))
	_, err := svc.RefreshNodeToken(ctx, token.Token)
	require.ErrorIs(t, err, ErrInvalidNodeToken)
}

func TestNodeTokenSigningUnavailable(t *testing.T) {
	svc := newTestNodeService(t)
	svc.nodeSigningCerts = nil
	_, err := svc.IssueNodeToken(context.Background(), testIssueRequest())
	require.ErrorIs(t, err, ErrInvalidNodeToken)
}

func TestNodeTokenLockSetFailureAndNilContext(t *testing.T) {
	svc := newTestNodeService(t)
	var nilCtx context.Context // exercised intentionally: callers may pass an unset context
	err := svc.withNodeTokenLock(nilCtx, "cmp_nil", func() error { return nil })
	require.NoError(t, err)
	svc.store = &failingStore{Store: svc.store, failSet: true}
	err = svc.withNodeTokenLock(context.Background(), "cmp_set", func() error { return nil })
	require.Error(t, err)
}

func TestNodeTokenHelperEdges(t *testing.T) {
	require.Equal(t, int64(1), intClaim(jwt.MapClaims{"x": int64(1)}, "x"))
	require.Equal(t, int64(2), intClaim(jwt.MapClaims{"x": json.Number("2")}, "x"))
	require.Equal(t, int64(0), intClaim(jwt.MapClaims{"x": "bad"}, "x"))
	require.True(t, audienceContains([]string{"tai"}, "tai"))
	require.True(t, audienceContains([]interface{}{"a", "tai"}, "tai"))
	require.False(t, audienceContains(1, "tai"))
	require.Equal(t, "", nodeTokenHashByJTI(newTestNodeService(t), "missing"))

	_, err := decodeNodeMetadata(map[string]interface{}{})
	require.ErrorIs(t, err, ErrInvalidNodeToken)
	_, err = decodeNodeMetadata(make(chan int))
	require.Error(t, err)

	svc := newTestNodeService(t)
	ctx := context.Background()
	_, err = svc.parseNodeToken("")
	require.Error(t, err)
	claims := jwt.MapClaims{
		"iss": "yao", "aud": "tai", "sub": "n", "purpose": "node",
		"node_id": "n", "computer_id": "c", "scope": "user", "visibility": "private",
		"jti": "j", "iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(), "kid": "nt-test",
	}
	hs := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	hs.Header["kid"] = "nt-test"
	raw, err := hs.SignedString([]byte("secret"))
	require.NoError(t, err)
	_, err = svc.ValidateNodeToken(ctx, raw)
	require.Error(t, err)
}

func TestNodeTokenInfoAndLockBranches(t *testing.T) {
	svc := newTestNodeService(t)
	ctx := context.Background()
	token := mustIssue(t, svc)
	info, err := svc.GetNodeTokenInfo(ctx, token.JTI)
	require.NoError(t, err)
	require.Equal(t, token.JTI, info.JTI)
	_, err = svc.GetNodeTokenInfo(ctx, "missing")
	require.ErrorIs(t, err, ErrNodeTokenMissing)
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	_, err = svc.GetNodeTokenInfo(cancelled, token.JTI)
	require.ErrorIs(t, err, context.Canceled)

	_, err = svc.nodeTokenByJTI("missing")
	require.ErrorIs(t, err, ErrNodeTokenMissing)
	require.NoError(t, svc.store.Set(svc.nodeTokenJTIKey("empty"), "", time.Hour))
	_, err = svc.nodeTokenByJTI("empty")
	require.ErrorIs(t, err, ErrNodeTokenMissing)

	_, err = decodeNodeMetadata(map[string]interface{}{"jti": 1})
	require.Error(t, err)

	cancelled, cancel = context.WithCancel(ctx)
	cancel()
	err = svc.withNodeTokenLock(cancelled, "cmp_cancelled", func() error { return nil })
	require.ErrorIs(t, err, context.Canceled)
}

func TestNodeTokenIssueCleanup(t *testing.T) {
	svc := newTestNodeService(t)
	svc.store = &failAfterStore{Store: svc.store, failAt: 2}
	_, err := svc.IssueNodeToken(context.Background(), testIssueRequest())
	require.Error(t, err)
	svc = newTestNodeService(t)
	svc.store = &failAfterStore{Store: svc.store, failAt: 3}
	_, err = svc.IssueNodeToken(context.Background(), testIssueRequest())
	require.Error(t, err)
}

type failAfterStore struct {
	store.Store
	failAt int
	sets   int
}

func (s *failAfterStore) Set(key string, value interface{}, ttl time.Duration) error {
	s.sets++
	if s.failAt > 0 && s.sets >= s.failAt {
		return errors.New("set failed")
	}
	return s.Store.Set(key, value, ttl)
}

func TestNodeTokenConcurrentRefresh(t *testing.T) {
	svc := newTestNodeService(t)
	token := mustIssue(t, svc)
	ctx := context.Background()
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			_, err := svc.RefreshNodeToken(ctx, token.Token)
			results <- err
		}()
	}
	success := 0
	for i := 0; i < 2; i++ {
		if <-results == nil {
			success++
		}
	}
	require.Equal(t, 1, success)
}

func TestNodeTokenRevokeByComputer(t *testing.T) {
	svc := newTestNodeService(t)
	ctx := context.Background()

	// Empty computer ID is a no-op.
	require.NoError(t, svc.RevokeNodeTokenByComputer(ctx, ""))

	// No index is a no-op.
	require.NoError(t, svc.RevokeNodeTokenByComputer(ctx, "cmp_missing"))

	// Empty or non-string index is removed without touching token metadata.
	require.NoError(t, svc.store.Set(svc.nodeTokenComputerKey("cmp_empty"), "", time.Hour))
	require.NoError(t, svc.RevokeNodeTokenByComputer(ctx, "cmp_empty"))
	_, ok := svc.store.Get(svc.nodeTokenComputerKey("cmp_empty"))
	require.False(t, ok)

	require.NoError(t, svc.store.Set(svc.nodeTokenComputerKey("cmp_nonstring"), 42, time.Hour))
	require.NoError(t, svc.RevokeNodeTokenByComputer(ctx, "cmp_nonstring"))
	_, ok = svc.store.Get(svc.nodeTokenComputerKey("cmp_nonstring"))
	require.False(t, ok)

	// A valid token is revoked and the computer index is removed.
	token := mustIssueForComputer(t, svc, "cmp_revoke")
	info, err := svc.GetNodeTokenInfo(ctx, token.JTI)
	require.NoError(t, err)
	require.Equal(t, "cmp_revoke", info.ComputerID)
	require.NoError(t, svc.RevokeNodeTokenByComputer(ctx, "cmp_revoke"))
	_, err = svc.GetNodeTokenInfo(ctx, token.JTI)
	require.Error(t, err)
	_, ok = svc.store.Get(svc.nodeTokenComputerKey("cmp_revoke"))
	require.False(t, ok)
	require.NoError(t, svc.RevokeNodeTokenByComputer(ctx, "cmp_revoke"))

	// A stale index pointing at missing metadata is cleaned up.
	require.NoError(t, svc.store.Set(svc.nodeTokenComputerKey("cmp_stale"), "missing-jti", time.Hour))
	require.NoError(t, svc.RevokeNodeTokenByComputer(ctx, "cmp_stale"))
	_, ok = svc.store.Get(svc.nodeTokenComputerKey("cmp_stale"))
	require.False(t, ok)

	// Canceled context is propagated by the computer lock.
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, svc.RevokeNodeTokenByComputer(canceled, "cmp_canceled"), context.Canceled)
}
