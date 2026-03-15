package sandboxv2

import (
	"fmt"
	"time"

	lrustore "github.com/yaoapp/gou/store/lru"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	"github.com/yaoapp/yao/openapi/oauth"
)

const (
	accessTokenTTL  = 2 * time.Hour
	refreshTokenTTL = 30 * 24 * time.Hour // 30 days
	tokenCacheSize  = 1024
)

var tokenCache *lrustore.Cache

func init() {
	c, err := lrustore.New(tokenCacheSize)
	if err != nil {
		panic("sandbox token cache init failed: " + err.Error())
	}
	tokenCache = c
}

func cacheKey(teamID, userID string) string {
	if teamID == "" {
		return userID
	}
	return teamID + "/" + userID
}

func getToken(teamID, userID string) *types.SandboxToken {
	val, ok := tokenCache.Get(cacheKey(teamID, userID))
	if !ok {
		return nil
	}
	tok, _ := val.(*types.SandboxToken)
	return tok
}

func setToken(teamID, userID string, tok *types.SandboxToken, ttl time.Duration) {
	tokenCache.Set(cacheKey(teamID, userID), tok, ttl)
}

// IssueSandboxToken returns a valid identity token for the given user.
// Tokens are cached by (teamID, userID); a new token is only issued on
// cache miss or expiry. Returns nil without error when oauth.OAuth is nil.
func IssueSandboxToken(teamID, userID string) (*types.SandboxToken, error) {
	if tok := getToken(teamID, userID); tok != nil {
		return tok, nil
	}

	svc := oauth.OAuth
	if svc == nil {
		return nil, nil
	}

	subject, err := svc.Subject("__yao.sandbox", userID)
	if err != nil {
		return nil, fmt.Errorf("sandbox token: derive subject: %w", err)
	}

	extraClaims := map[string]interface{}{
		"user_id": userID,
	}
	if teamID != "" {
		extraClaims["team_id"] = teamID
	}

	tokenStr, err := svc.MakeAccessToken("__yao.sandbox", "grpc:mcp", subject,
		int(accessTokenTTL.Seconds()), extraClaims)
	if err != nil {
		return nil, fmt.Errorf("sandbox token: issue access token: %w", err)
	}

	tok := &types.SandboxToken{Token: tokenStr}

	refreshStr, err := svc.MakeRefreshToken("__yao.sandbox", "grpc:mcp", subject,
		int(refreshTokenTTL.Seconds()), extraClaims)
	if err != nil {
		return nil, fmt.Errorf("sandbox token: issue refresh token: %w", err)
	}
	tok.RefreshToken = refreshStr

	setToken(teamID, userID, tok, accessTokenTTL)
	return tok, nil
}
