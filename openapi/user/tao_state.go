package user

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/openapi/oauth"
)

// taoCache is replaceable in unit tests; production uses the OAuth cache.
var taoCache = func() store.Store {
	if oauth.OAuth == nil {
		return nil
	}
	return oauth.OAuth.GetCache()
}

const taoTxnTTL = 10 * time.Minute

type taoLoginTxn struct {
	State       string `json:"state"`
	Nonce       string `json:"nonce"`
	Verifier    string `json:"verifier"`
	Challenge   string `json:"challenge"`
	RedirectURI string `json:"redirect_uri"`
	ReturnURL   string `json:"return_url,omitempty"`
	SID         string `json:"sid,omitempty"`
	Mode        string `json:"mode"`
	Locale      string `json:"locale,omitempty"`
	Prompt      string `json:"prompt,omitempty"`
	LoginHint   string `json:"login_hint,omitempty"`
	MaxAge      string `json:"max_age,omitempty"`
}

func taoTxnKey(state string) string { return "signin:tao:txn:" + state }

func saveTaoTxn(txn *taoLoginTxn) error {
	if txn == nil || txn.State == "" {
		return fmt.Errorf("invalid tao transaction")
	}
	cache := taoCache()
	if cache == nil {
		return fmt.Errorf("oauth cache is not available")
	}
	data, _ := json.Marshal(txn)
	return cache.Set(taoTxnKey(txn.State), string(data), taoTxnTTL)
}

func loadTaoTxn(state string) (*taoLoginTxn, error) {
	if state == "" {
		return nil, fmt.Errorf("state is required")
	}
	cache := taoCache()
	if cache == nil {
		return nil, fmt.Errorf("oauth cache is not available")
	}
	raw, ok := cache.Get(taoTxnKey(state))
	if !ok || raw == nil {
		return nil, fmt.Errorf("tao transaction not found")
	}
	text, ok := raw.(string)
	if !ok {
		return nil, fmt.Errorf("invalid tao transaction")
	}
	var txn taoLoginTxn
	if err := json.Unmarshal([]byte(text), &txn); err != nil {
		return nil, fmt.Errorf("parse tao transaction: %w", err)
	}
	if txn.State != state {
		return nil, fmt.Errorf("invalid tao transaction state")
	}
	return &txn, nil
}

func deleteTaoTxn(state string) {
	if state == "" {
		return
	}
	if cache := taoCache(); cache != nil {
		_ = cache.Del(taoTxnKey(state))
	}
}

const taoHandoffTTL = 2 * time.Minute

// TaoLoginHandoff carries a freshly established Yao login through the
// cross-site redirect from Tao back to /v1/oauth/authorize. The browser may
// drop SameSite=Strict access-token cookies on that redirect, so the authorize
// handler consumes this one-time server-side handoff instead of relying on a
// cookie.
type TaoLoginHandoff struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	SessionID    string `json:"session_id,omitempty"`
}

func taoHandoffKey(id string) string { return "signin:tao:handoff:" + id }

func saveTaoLoginHandoff(accessToken, refreshToken, sessionID string) (string, error) {
	if accessToken == "" {
		return "", fmt.Errorf("login handoff access token is empty")
	}
	id, err := taoRandom(32)
	if err != nil {
		return "", err
	}
	cache := taoCache()
	if cache == nil {
		return "", fmt.Errorf("oauth cache is not available")
	}
	data, err := json.Marshal(&TaoLoginHandoff{AccessToken: accessToken, RefreshToken: refreshToken, SessionID: sessionID})
	if err != nil {
		return "", err
	}
	if err := cache.Set(taoHandoffKey(id), string(data), taoHandoffTTL); err != nil {
		return "", err
	}
	return id, nil
}

// ConsumeTaoLoginHandoff retrieves and deletes a one-time login handoff.
func ConsumeTaoLoginHandoff(id string) (*TaoLoginHandoff, bool) {
	if id == "" {
		return nil, false
	}
	cache := taoCache()
	if cache == nil {
		return nil, false
	}
	raw, ok := cache.Get(taoHandoffKey(id))
	if !ok || raw == nil {
		return nil, false
	}
	_ = cache.Del(taoHandoffKey(id))
	text, ok := raw.(string)
	if !ok {
		return nil, false
	}
	var handoff TaoLoginHandoff
	if err := json.Unmarshal([]byte(text), &handoff); err != nil || handoff.AccessToken == "" {
		return nil, false
	}
	return &handoff, true
}
