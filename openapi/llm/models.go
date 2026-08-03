package llm

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/connector"
	agentllm "github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/llmprovider"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
)

// openaiModel mirrors the OpenAI /v1/models object.
type openaiModel struct {
	ID           string                 `json:"id"`
	Object       string                 `json:"object"`
	OwnedBy      string                 `json:"owned_by"`
	Capabilities map[string]interface{} `json:"capabilities,omitempty"`
}

// handleListModels implements GET /llm/models (OpenAI SDK compatible).
func handleListModels(c *gin.Context) {
	info := authorized.GetInfo(c)

	var opts []connector.Option
	if llmprovider.Global != nil {
		opts = llmprovider.Global.ListModelsBy(info)
	} else {
		opts = connector.AIConnectors
	}

	models := make([]openaiModel, 0, len(opts))
	for _, opt := range opts {
		var conn connector.Connector
		var err error
		if llmprovider.Global != nil {
			conn, err = llmprovider.Global.GetModel(opt.Value)
		} else {
			conn, err = connector.Select(opt.Value)
		}
		if err != nil {
			continue
		}

		connType := connectorType(conn)
		if connType != "openai" && connType != "anthropic" {
			continue
		}

		caps := agentllm.GetCapabilitiesFromConn(conn)
		capsMap := agentllm.ToMap(caps)
		if isNonChatModel(capsMap) {
			continue
		}

		ownedBy := "system"
		if conn.GetMetaInfo().Builtin == false {
			if llmprovider.Global != nil {
				if p, err := llmprovider.Global.GetByConnectorID(opt.Value); err == nil {
					switch p.Owner.Type {
					case "user":
						ownedBy = "user"
					case "team":
						ownedBy = "team"
					}
				}
			}
		}

		models = append(models, openaiModel{
			ID:           opt.Value,
			Object:       "model",
			OwnedBy:      ownedBy,
			Capabilities: capsMap,
		})
	}

	// Append role-based aliases (use::default, use::light, etc.)
	if llmprovider.Global != nil {
		roles, err := llmprovider.Global.ListRolesBy(info)
		if err == nil {
			for role := range roles {
				roleID := agentllm.RolePrefix + role

				var capsMap map[string]interface{}
				if conn, _, err := agentllm.ResolveConnector(roleID, info); err == nil {
					caps := agentllm.GetCapabilitiesFromConn(conn)
					capsMap = agentllm.ToMap(caps)
				}

				models = append(models, openaiModel{
					ID:           roleID,
					Object:       "model",
					OwnedBy:      "role",
					Capabilities: capsMap,
				})
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   models,
	})
}
