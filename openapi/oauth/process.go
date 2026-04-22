package oauth

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

func init() {
	process.RegisterGroup("oauth", map[string]process.Handler{
		"token.Make":       processTokenMake,
		"token.MakeByUser": processTokenMakeByUser,
	})
}

// processTokenMake generates an OAuth access token with explicit parameters.
//
// Args:
//
//	[0] clientID  string   – OAuth client ID embedded in the token
//	[1] scope     string   – token scope (space-separated)
//	[2] subject   string   – JWT subject claim
//	[3] expiresIn int      – token lifetime in seconds
//	[4] extraClaims map    – (optional) additional JWT claims (e.g. user_id, team_id)
//
// Returns: token string
//
// Example: Process("oauth.token.Make", "tai-agent-smith", "tai:tunnel", "ci-tai", 86400)
func processTokenMake(p *process.Process) interface{} {
	p.ValidateArgNums(4)
	clientID := p.ArgsString(0)
	scope := p.ArgsString(1)
	subject := p.ArgsString(2)
	expiresIn := p.ArgsInt(3)

	if OAuth == nil {
		exception.New("oauth service not initialized", 500).Throw()
	}

	var extraClaims map[string]interface{}
	if p.NumOfArgs() > 4 {
		if claims, ok := p.Args[4].(map[string]interface{}); ok {
			extraClaims = claims
		}
	}

	token, err := OAuth.MakeAccessToken(clientID, scope, subject, expiresIn, extraClaims)
	if err != nil {
		exception.New(fmt.Sprintf("oauth.token.Make: %v", err), 500).Throw()
	}
	return token
}

// processTokenMakeByUser generates an OAuth access token for a team member.
// Looks up user and team from the database, automatically filling clientID, scope, subject, and claims.
//
// Args:
//
//	[0] teamID    string – team ID
//	[1] memberID  string – member ID (business ID)
//	[2] expiresIn int    – token lifetime in seconds (optional, default 86400 = 24h)
//
// Returns: token string
//
// Example: Process("oauth.token.MakeByUser", "team-abc", "member-xyz")
// Example: Process("oauth.token.MakeByUser", "team-abc", "member-xyz", 3600)
func processTokenMakeByUser(p *process.Process) interface{} {
	p.ValidateArgNums(2)
	teamID := p.ArgsString(0)
	memberID := p.ArgsString(1)

	expiresIn := 86400 // default 24h
	if p.NumOfArgs() > 2 {
		if v := p.ArgsInt(2); v > 0 {
			expiresIn = v
		}
	}

	if OAuth == nil {
		exception.New("oauth service not initialized", 500).Throw()
	}

	userProvider, err := OAuth.GetUserProvider()
	if err != nil {
		exception.New(fmt.Sprintf("oauth.token.MakeByUser: failed to get user provider: %v", err), 500).Throw()
	}

	ctx := context.Background()

	member, err := userProvider.GetMemberByMemberID(ctx, memberID)
	if err != nil {
		exception.New(fmt.Sprintf("oauth.token.MakeByUser: member not found: %v", err), 404).Throw()
	}

	memberTeamID := ""
	if v, ok := member["team_id"].(string); ok {
		memberTeamID = v
	}
	if memberTeamID != teamID {
		exception.New(fmt.Sprintf("oauth.token.MakeByUser: member %s does not belong to team %s", memberID, teamID), 403).Throw()
	}

	userID := ""
	if v, ok := member["user_id"].(string); ok {
		userID = v
	}
	if userID == "" {
		exception.New(fmt.Sprintf("oauth.token.MakeByUser: member %s has no user_id", memberID), 500).Throw()
	}

	subject := userID

	extraClaims := map[string]interface{}{
		"user_id":   userID,
		"team_id":   teamID,
		"member_id": memberID,
	}

	token, err := OAuth.MakeAccessToken("yao-admin", "openid profile", subject, expiresIn, extraClaims)
	if err != nil {
		exception.New(fmt.Sprintf("oauth.token.MakeByUser: %v", err), 500).Throw()
	}
	return token
}
