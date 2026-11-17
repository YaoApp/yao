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
