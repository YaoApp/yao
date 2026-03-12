package tunnel

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	oauth "github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/tai/types"
)

func extractBearer(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if len(auth) > 7 && strings.EqualFold(auth[:7], "bearer ") {
		return auth[7:]
	}
	return ""
}

var authenticateBearerFunc = authenticateBearerDefault

func authenticateBearerDefault(token string) (types.AuthInfo, error) {
	svc := oauth.OAuth
	if svc == nil {
		return types.AuthInfo{}, fmt.Errorf("oauth service not initialized")
	}

	result, err := svc.AuthenticateToken(oauth.AuthInput{
		AccessToken: token,
	})
	if err != nil {
		return types.AuthInfo{}, err
	}

	info := types.AuthInfo{}
	if result.Info != nil {
		info.Subject = result.Info.Subject
		info.UserID = result.Info.UserID
		info.ClientID = result.Info.ClientID
		info.Scope = result.Info.Scope
		info.TeamID = result.Info.TeamID
		info.TenantID = result.Info.TenantID
	}

	slog.Info("[tunnel-auth] info from token",
		"subject", info.Subject, "user_id", info.UserID,
		"client_id", info.ClientID, "team_id", info.TeamID,
		"scope", info.Scope)

	if result.Claims != nil {
		slog.Info("[tunnel-auth] claims",
			"claims.TeamID", result.Claims.TeamID,
			"claims.ClientID", result.Claims.ClientID,
			"extra", fmt.Sprintf("%+v", result.Claims.Extra))

		if info.TeamID == "" && result.Claims.TeamID != "" {
			info.TeamID = result.Claims.TeamID
		}
		if info.TeamID == "" {
			switch v := result.Claims.Extra["team_id"].(type) {
			case string:
				info.TeamID = v
			case float64:
				info.TeamID = fmt.Sprintf("%.0f", v)
			}
		}
		if info.TenantID == "" {
			if v, ok := result.Claims.Extra["tenant_id"].(string); ok {
				info.TenantID = v
			}
		}
	}

	slog.Info("[tunnel-auth] final", "team_id", info.TeamID, "client_id", info.ClientID)
	return info, nil
}

func portsFromMap(m map[string]int) types.Ports {
	return types.Ports{
		GRPC:   m["grpc"],
		HTTP:   m["http"],
		VNC:    m["vnc"],
		Docker: m["docker"],
		K8s:    m["k8s"],
	}
}

func capsFromMap(m map[string]bool) types.Capabilities {
	return types.Capabilities{
		Docker:   m["docker"],
		K8s:      m["k8s"],
		HostExec: m["host_exec"],
	}
}
