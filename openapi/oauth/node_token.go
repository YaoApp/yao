package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const (
	nodeTokenPurpose  = "node"
	nodeTokenIssuer   = "yao"
	nodeTokenAudience = "tai"
	nodeTokenLockTTL  = 30 * time.Second
)

var (
	// ErrInvalidNodeToken is returned when a node token cannot be trusted.
	ErrInvalidNodeToken = errors.New("invalid node_token")
	// ErrNodeTokenConflict is returned when a computer already has an active token.
	ErrNodeTokenConflict = errors.New("node_token already active for computer")
	// ErrNodeTokenBusy is returned when a computer token operation is already running.
	ErrNodeTokenBusy = errors.New("node_token operation busy")
	// ErrNodeTokenMissing is returned when a node token does not exist.
	ErrNodeTokenMissing = errors.New("node_token not found")
)

// IssueRequest contains the identity and ownership attached to a node token.
type IssueRequest struct {
	ComputerID string
	NodeID     string
	Scope      string
	Visibility string
	UserID     string
	TeamID     string
}

// NodeToken is a signed node token and its public metadata.
type NodeToken struct {
	Token     string    `json:"node_token"`
	JTI       string    `json:"jti"`
	ExpiresAt time.Time `json:"expires_at"`
}

// NodeInfo is the validated identity carried by a node token.
type NodeInfo struct {
	JTI        string    `json:"jti"`
	ComputerID string    `json:"computer_id"`
	NodeID     string    `json:"node_id"`
	Scope      string    `json:"scope"`
	Visibility string    `json:"visibility"`
	UserID     string    `json:"user_id"`
	TeamID     string    `json:"team_id"`
	KID        string    `json:"kid"`
	IssuedAt   time.Time `json:"issued_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type nodeTokenMetadata struct {
	JTI        string `json:"jti"`
	ComputerID string `json:"computer_id"`
	NodeID     string `json:"node_id"`
	Scope      string `json:"scope"`
	Visibility string `json:"visibility"`
	UserID     string `json:"user_id"`
	TeamID     string `json:"team_id"`
	KID        string `json:"kid"`
	IssuedAt   int64  `json:"issued_at"`
	ExpiresAt  int64  `json:"expires_at"`
}

func (m *nodeTokenMetadata) info() *NodeInfo {
	return &NodeInfo{
		JTI:        m.JTI,
		ComputerID: m.ComputerID,
		NodeID:     m.NodeID,
		Scope:      m.Scope,
		Visibility: m.Visibility,
		UserID:     m.UserID,
		TeamID:     m.TeamID,
		KID:        m.KID,
		IssuedAt:   time.Unix(m.IssuedAt, 0),
		ExpiresAt:  time.Unix(m.ExpiresAt, 0),
	}
}

func (m *nodeTokenMetadata) validate(req IssueRequest) error {
	if req.ComputerID == "" || req.NodeID == "" {
		return fmt.Errorf("%w: computer_id and node_id are required", ErrInvalidNodeToken)
	}
	if req.Scope != "system" && req.Scope != "team" && req.Scope != "user" {
		return fmt.Errorf("%w: invalid scope %q", ErrInvalidNodeToken, req.Scope)
	}
	if req.Visibility != "private" && req.Visibility != "members" && req.Visibility != "team" && req.Visibility != "global" {
		return fmt.Errorf("%w: invalid visibility %q", ErrInvalidNodeToken, req.Visibility)
	}
	if req.Scope == "user" && req.UserID == "" {
		return fmt.Errorf("%w: user_id is required for user scope", ErrInvalidNodeToken)
	}
	if req.Scope == "team" && req.TeamID == "" {
		return fmt.Errorf("%w: team_id is required for team scope", ErrInvalidNodeToken)
	}
	return nil
}

func nodeTokenHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func randomNodeID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "nt_" + hex.EncodeToString(b), nil
}

func (s *Service) nodeTokenKey(raw string) string {
	return s.prefix + "oauth:node_token:" + nodeTokenHash(raw)
}

func (s *Service) nodeTokenJTIKey(jti string) string {
	return s.prefix + "oauth:node_token_jti:" + jti
}

func (s *Service) nodeTokenComputerKey(computerID string) string {
	return s.prefix + "oauth:node_token_by_computer:" + computerID
}

func (s *Service) nodeTokenLockKey(computerID string) string {
	return s.prefix + "oauth:node_token_lock:" + computerID
}

// IssueNodeToken creates the first active node token for a computer.
func (s *Service) IssueNodeToken(ctx context.Context, req IssueRequest) (*NodeToken, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}
	var out *NodeToken
	err := s.withNodeTokenLock(ctx, req.ComputerID, func() error {
		indexKey := s.nodeTokenComputerKey(req.ComputerID)
		if value, ok := s.store.Get(indexKey); ok {
			if jti, _ := value.(string); jti != "" {
				if _, err := s.nodeTokenByJTI(jti); err == nil {
					return ErrNodeTokenConflict
				}
				_ = s.store.Del(indexKey)
			}
		}
		token, err := s.issueNodeTokenLocked(ctx, req)
		if err != nil {
			return err
		}
		out = token
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (req IssueRequest) validate() error {
	return (&nodeTokenMetadata{}).validate(req)
}

// ValidateNodeToken verifies the signature, claims, store state and computer index.
func (s *Service) ValidateNodeToken(ctx context.Context, raw string) (*NodeInfo, error) {
	if ctx != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}
	if s == nil || s.nodeSigningCerts == nil || s.config == nil {
		return nil, ErrInvalidNodeToken
	}
	meta, err := s.parseNodeToken(raw)
	if err != nil {
		return nil, err
	}
	if err := s.validateNodeMetadata(meta); err != nil {
		return nil, err
	}
	return meta.info(), nil
}

func (s *Service) parseNodeToken(raw string) (*nodeTokenMetadata, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, ErrInvalidNodeToken
	}
	token, err := jwt.ParseWithClaims(raw, jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		expected := getSigningMethod(s.config.NodeToken.SigningAlgorithm)
		if token.Method != expected {
			return nil, fmt.Errorf("%w: unexpected signing method", ErrInvalidNodeToken)
		}
		if kid, _ := token.Header["kid"].(string); kid != s.config.NodeToken.KID {
			return nil, fmt.Errorf("%w: unexpected kid", ErrInvalidNodeToken)
		}
		return s.nodeSigningCerts.GetPublicKey(), nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidNodeToken, err)
	}
	if !token.Valid {
		return nil, ErrInvalidNodeToken
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidNodeToken
	}
	if strClaim(claims, "iss") != nodeTokenIssuer || strClaim(claims, "purpose") != nodeTokenPurpose {
		return nil, ErrInvalidNodeToken
	}
	if !audienceContains(claims["aud"], nodeTokenAudience) {
		return nil, ErrInvalidNodeToken
	}
	meta := &nodeTokenMetadata{
		JTI:        strClaim(claims, "jti"),
		ComputerID: strClaim(claims, "computer_id"),
		NodeID:     strClaim(claims, "node_id"),
		Scope:      strClaim(claims, "scope"),
		Visibility: strClaim(claims, "visibility"),
		UserID:     strClaim(claims, "user_id"),
		TeamID:     strClaim(claims, "team_id"),
		KID:        strClaim(claims, "kid"),
		IssuedAt:   intClaim(claims, "iat"),
		ExpiresAt:  intClaim(claims, "exp"),
	}
	if meta.JTI == "" || meta.ComputerID == "" || meta.NodeID == "" || meta.Scope == "" || meta.Visibility == "" {
		return nil, ErrInvalidNodeToken
	}
	if strClaim(claims, "sub") != meta.NodeID {
		return nil, ErrInvalidNodeToken
	}
	if meta.KID != s.config.NodeToken.KID {
		return nil, ErrInvalidNodeToken
	}
	if meta.ExpiresAt <= time.Now().Unix() {
		return nil, ErrInvalidNodeToken
	}
	return meta, nil
}

func (s *Service) validateNodeMetadata(meta *nodeTokenMetadata) error {
	value, ok := s.store.Get(s.nodeTokenJTIKey(meta.JTI))
	if !ok {
		return ErrInvalidNodeToken
	}
	storedHash, _ := value.(string)
	if storedHash == "" {
		return ErrInvalidNodeToken
	}
	tokenValue, ok := s.store.Get(s.prefix + "oauth:node_token:" + storedHash)
	if !ok {
		return ErrInvalidNodeToken
	}
	stored, err := decodeNodeMetadata(tokenValue)
	if err != nil {
		return ErrInvalidNodeToken
	}
	if stored.JTI != meta.JTI || stored.ComputerID != meta.ComputerID || stored.NodeID != meta.NodeID ||
		stored.Scope != meta.Scope || stored.Visibility != meta.Visibility ||
		stored.UserID != meta.UserID || stored.TeamID != meta.TeamID ||
		stored.KID != meta.KID || stored.IssuedAt != meta.IssuedAt || stored.ExpiresAt != meta.ExpiresAt {
		return ErrInvalidNodeToken
	}
	index, ok := s.store.Get(s.nodeTokenComputerKey(meta.ComputerID))
	if !ok {
		return ErrInvalidNodeToken
	}
	indexJTI, _ := index.(string)
	if indexJTI != meta.JTI {
		return ErrInvalidNodeToken
	}
	return nil
}

func (s *Service) issueNodeTokenLocked(ctx context.Context, req IssueRequest) (*NodeToken, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	jti, err := randomNodeID()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	meta := &nodeTokenMetadata{
		JTI:        jti,
		ComputerID: req.ComputerID,
		NodeID:     req.NodeID,
		Scope:      req.Scope,
		Visibility: req.Visibility,
		UserID:     req.UserID,
		TeamID:     req.TeamID,
		KID:        s.config.NodeToken.KID,
		IssuedAt:   now.Unix(),
		ExpiresAt:  now.Add(s.config.NodeToken.TTL).Unix(),
	}
	raw, err := s.signNodeToken(meta)
	if err != nil {
		return nil, err
	}
	tokenKey := s.nodeTokenKey(raw)
	jtiKey := s.nodeTokenJTIKey(meta.JTI)
	indexKey := s.nodeTokenComputerKey(meta.ComputerID)
	hash := nodeTokenHash(raw)
	written := []string{}
	write := func(key string, value interface{}) error {
		if err := s.store.Set(key, value, s.config.NodeToken.TTL); err != nil {
			return err
		}
		written = append(written, key)
		return nil
	}
	if err := write(tokenKey, meta); err != nil {
		return nil, err
	}
	if err := write(jtiKey, hash); err != nil {
		s.cleanupNodeKeys(written)
		return nil, err
	}
	if err := write(indexKey, meta.JTI); err != nil {
		s.cleanupNodeKeys(written)
		return nil, err
	}
	return &NodeToken{Token: raw, JTI: meta.JTI, ExpiresAt: time.Unix(meta.ExpiresAt, 0)}, nil
}

func (s *Service) signNodeToken(meta *nodeTokenMetadata) (string, error) {
	if s.nodeSigningCerts == nil || s.nodeSigningCerts.SigningKey == nil {
		return "", ErrInvalidNodeToken
	}
	claims := jwt.MapClaims{
		"iss":         nodeTokenIssuer,
		"aud":         nodeTokenAudience,
		"sub":         meta.NodeID,
		"purpose":     nodeTokenPurpose,
		"node_id":     meta.NodeID,
		"computer_id": meta.ComputerID,
		"scope":       meta.Scope,
		"visibility":  meta.Visibility,
		"user_id":     meta.UserID,
		"team_id":     meta.TeamID,
		"jti":         meta.JTI,
		"iat":         meta.IssuedAt,
		"exp":         meta.ExpiresAt,
		"kid":         meta.KID,
	}
	token := jwt.NewWithClaims(getSigningMethod(s.config.NodeToken.SigningAlgorithm), claims)
	token.Header["kid"] = meta.KID
	token.Header["typ"] = "JWT"
	return token.SignedString(s.nodeSigningCerts.SigningKey)
}

// RefreshNodeToken validates raw and atomically replaces it with a new token.
func (s *Service) RefreshNodeToken(ctx context.Context, raw string) (*NodeToken, error) {
	old, err := s.parseNodeToken(raw)
	if err != nil {
		return nil, err
	}
	var out *NodeToken
	err = s.withNodeTokenLock(ctx, old.ComputerID, func() error {
		current, err := s.parseNodeToken(raw)
		if err != nil {
			return err
		}
		if err := s.validateNodeMetadata(current); err != nil {
			return err
		}
		req := IssueRequest{ComputerID: current.ComputerID, NodeID: current.NodeID, Scope: current.Scope, Visibility: current.Visibility, UserID: current.UserID, TeamID: current.TeamID}
		next, err := s.replaceNodeTokenLocked(ctx, current, req)
		if err != nil {
			return err
		}
		out = next
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Service) replaceNodeTokenLocked(ctx context.Context, old *nodeTokenMetadata, req IssueRequest) (*NodeToken, error) {
	next, err := s.issueNodeTokenLocked(ctx, req)
	if err != nil {
		return nil, err
	}
	s.deleteNodeTokenByJTI(old.JTI)
	return next, nil
}

func (s *Service) deleteNodeTokenByJTI(jti string) {
	if hash := nodeTokenHashByJTI(s, jti); hash != "" {
		_ = s.store.Del(s.prefix + "oauth:node_token:" + hash)
	}
	_ = s.store.Del(s.nodeTokenJTIKey(jti))
}

// RotateNodeToken replaces the active token for a computer.
func (s *Service) RotateNodeToken(ctx context.Context, computerID string) (*NodeToken, error) {
	if computerID == "" {
		return nil, fmt.Errorf("%w: computer_id is required", ErrInvalidNodeToken)
	}
	var out *NodeToken
	err := s.withNodeTokenLock(ctx, computerID, func() error {
		value, ok := s.store.Get(s.nodeTokenComputerKey(computerID))
		if !ok {
			return ErrNodeTokenConflict
		}
		jti, _ := value.(string)
		old, err := s.nodeTokenByJTI(jti)
		if err != nil {
			return ErrNodeTokenConflict
		}
		req := IssueRequest{ComputerID: old.ComputerID, NodeID: old.NodeID, Scope: old.Scope, Visibility: old.Visibility, UserID: old.UserID, TeamID: old.TeamID}
		next, err := s.issueNodeTokenLocked(ctx, req)
		if err != nil {
			return err
		}
		s.deleteNodeTokenByJTI(old.JTI)
		out = next
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetNodeTokenInfo returns metadata for an active token without changing it.
func (s *Service) GetNodeTokenInfo(ctx context.Context, jti string) (*NodeInfo, error) {
	if ctx != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}
	meta, err := s.nodeTokenByJTI(jti)
	if err != nil {
		return nil, err
	}
	return meta.info(), nil
}

// RevokeNodeToken revokes a token by jti. Missing jti is idempotent success.
func (s *Service) RevokeNodeToken(ctx context.Context, jti string) error {
	if jti == "" {
		return nil
	}
	meta, err := s.nodeTokenByJTI(jti)
	if err != nil {
		return nil
	}
	return s.withNodeTokenLock(ctx, meta.ComputerID, func() error {
		if current, err := s.nodeTokenByJTI(jti); err == nil {
			if value, ok := s.store.Get(s.nodeTokenComputerKey(current.ComputerID)); ok {
				if index, _ := value.(string); index == jti {
					_ = s.store.Del(s.nodeTokenComputerKey(current.ComputerID))
				}
			}
			s.deleteNodeTokenByJTI(jti)
		}
		return nil
	})
}

func (s *Service) nodeTokenByJTI(jti string) (*nodeTokenMetadata, error) {
	value, ok := s.store.Get(s.nodeTokenJTIKey(jti))
	if !ok {
		return nil, ErrNodeTokenMissing
	}
	hash, _ := value.(string)
	if hash == "" {
		return nil, ErrNodeTokenMissing
	}
	metaValue, ok := s.store.Get(s.prefix + "oauth:node_token:" + hash)
	if !ok {
		return nil, ErrNodeTokenMissing
	}
	return decodeNodeMetadata(metaValue)
}

func (s *Service) cleanupNodeKeys(keys []string) {
	for _, key := range keys {
		_ = s.store.Del(key)
	}
}

func (s *Service) withNodeTokenLock(ctx context.Context, computerID string, fn func() error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	local, _ := s.nodeLocks.LoadOrStore(computerID, &sync.Mutex{})
	mutex := local.(*sync.Mutex)
	mutex.Lock()
	defer mutex.Unlock()

	lockKey := s.nodeTokenLockKey(computerID)
	count, err := s.store.Incr(lockKey, 1)
	if err != nil {
		return err
	}
	if count != 1 {
		_, _ = s.store.Decr(lockKey, 1)
		return ErrNodeTokenBusy
	}
	owner := nodeTokenHash(fmt.Sprintf("%s:%d", computerID, time.Now().UnixNano()))
	if err := s.store.Set(lockKey, owner, nodeTokenLockTTL); err != nil {
		_, _ = s.store.Decr(lockKey, 1)
		return err
	}
	defer func() {
		if value, ok := s.store.Get(lockKey); ok {
			if current, _ := value.(string); current == owner {
				_ = s.store.Del(lockKey)
			}
		}
	}()
	return fn()
}

func decodeNodeMetadata(value interface{}) (*nodeTokenMetadata, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var meta nodeTokenMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	if meta.JTI == "" || meta.ComputerID == "" || meta.NodeID == "" {
		return nil, ErrInvalidNodeToken
	}
	return &meta, nil
}

func strClaim(claims jwt.MapClaims, key string) string {
	value, _ := claims[key].(string)
	return value
}

func intClaim(claims jwt.MapClaims, key string) int64 {
	switch value := claims[key].(type) {
	case float64:
		return int64(value)
	case int64:
		return value
	case json.Number:
		n, _ := value.Int64()
		return n
	default:
		return 0
	}
}

func audienceContains(value interface{}, expected string) bool {
	switch aud := value.(type) {
	case string:
		return aud == expected
	case []interface{}:
		for _, item := range aud {
			if item == expected {
				return true
			}
		}
	case []string:
		for _, item := range aud {
			if item == expected {
				return true
			}
		}
	}
	return false
}

func nodeTokenHashByJTI(s *Service, jti string) string {
	value, ok := s.store.Get(s.nodeTokenJTIKey(jti))
	if !ok {
		return ""
	}
	hash, _ := value.(string)
	return hash
}
