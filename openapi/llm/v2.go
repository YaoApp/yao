package llm

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	yaollm "github.com/yaoapp/yao/llm"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/response"
)

func respondErr(c *gin.Context, status int, msg string) {
	response.RespondWithError(c, status, &response.ErrorResponse{
		Code:             "server_error",
		ErrorDescription: msg,
	})
}

// maskAPIKey replaces an API key with a masked form: first 4 + "..." + last 4.
// Keys shorter than 12 characters become "sk-****".
func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) < 12 {
		return "sk-****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

// maskProviderKey returns a shallow copy of the Provider with APIKey masked.
func maskProviderKey(p *yaollm.Provider) yaollm.Provider {
	out := *p
	out.APIKey = maskAPIKey(out.APIKey)
	return out
}

// maskProvidersKey returns copies of providers with APIKey masked.
func maskProvidersKey(providers []yaollm.Provider) []yaollm.Provider {
	out := make([]yaollm.Provider, len(providers))
	for i := range providers {
		out[i] = providers[i]
		out[i].APIKey = maskAPIKey(out[i].APIKey)
	}
	return out
}

// ---------------------------------------------------------------------------
// Provider CRUD (2.0)
// ---------------------------------------------------------------------------

func handleListProvidersV2(c *gin.Context) {
	scope := c.Query("scope")
	teamID := c.Query("team_id")
	userID := c.Query("user_id")

	providers, err := yaollm.ListProviders(yaollm.Scope(scope), teamID, userID)
	if err != nil {
		respondErr(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, maskProvidersKey(providers))
}

func handleCreateProvider(c *gin.Context) {
	var p yaollm.Provider
	if err := c.ShouldBindJSON(&p); err != nil {
		respondErr(c, http.StatusBadRequest, err.Error())
		return
	}
	id, err := yaollm.CreateProvider(&p)
	if err != nil {
		respondErr(c, http.StatusInternalServerError, err.Error())
		return
	}
	p.ID = id
	masked := maskProviderKey(&p)
	response.RespondWithSuccess(c, http.StatusCreated, masked)
}

func handleGetProvider(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		respondErr(c, http.StatusBadRequest, "invalid provider id")
		return
	}
	p, err := yaollm.GetProvider(id)
	if err != nil {
		respondErr(c, http.StatusNotFound, err.Error())
		return
	}
	masked := maskProviderKey(p)
	response.RespondWithSuccess(c, http.StatusOK, masked)
}

func handleUpdateProvider(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		respondErr(c, http.StatusBadRequest, "invalid provider id")
		return
	}
	var p yaollm.Provider
	if err := c.ShouldBindJSON(&p); err != nil {
		respondErr(c, http.StatusBadRequest, err.Error())
		return
	}
	p.ID = id
	if err := yaollm.UpdateProvider(id, &p); err != nil {
		respondErr(c, http.StatusInternalServerError, err.Error())
		return
	}
	masked := maskProviderKey(&p)
	response.RespondWithSuccess(c, http.StatusOK, masked)
}

func handleDeleteProvider(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		respondErr(c, http.StatusBadRequest, "invalid provider id")
		return
	}
	if err := yaollm.DeleteProvider(id); err != nil {
		respondErr(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, nil)
}

// ---------------------------------------------------------------------------
// Model CRUD (2.0)
// ---------------------------------------------------------------------------

func handleListModelsV2(c *gin.Context) {
	providerID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		respondErr(c, http.StatusBadRequest, "invalid provider id")
		return
	}
	models, err := yaollm.ListModels(providerID)
	if err != nil {
		respondErr(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, models)
}

func handleCreateModel(c *gin.Context) {
	providerID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		respondErr(c, http.StatusBadRequest, "invalid provider id")
		return
	}
	if _, err := yaollm.GetProvider(providerID); err != nil {
		respondErr(c, http.StatusNotFound, "provider not found")
		return
	}
	var m yaollm.Model
	if err := c.ShouldBindJSON(&m); err != nil {
		respondErr(c, http.StatusBadRequest, err.Error())
		return
	}
	m.ProviderRef = providerID
	id, err := yaollm.CreateModel(&m)
	if err != nil {
		respondErr(c, http.StatusInternalServerError, err.Error())
		return
	}
	m.ID = id
	response.RespondWithSuccess(c, http.StatusCreated, m)
}

func handleGetModelV2(c *gin.Context) {
	providerID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		respondErr(c, http.StatusBadRequest, "invalid provider id")
		return
	}
	modelID := c.Param("model_id")
	m, err := yaollm.FindModelByModelID(providerID, modelID)
	if err != nil {
		respondErr(c, http.StatusInternalServerError, err.Error())
		return
	}
	if m == nil {
		respondErr(c, http.StatusNotFound, "model not found")
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, m)
}

func handleUpdateModel(c *gin.Context) {
	providerID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		respondErr(c, http.StatusBadRequest, "invalid provider id")
		return
	}
	modelID := c.Param("model_id")

	existing, err := yaollm.FindModelByModelID(providerID, modelID)
	if err != nil {
		respondErr(c, http.StatusInternalServerError, err.Error())
		return
	}
	if existing == nil {
		respondErr(c, http.StatusNotFound, "model not found")
		return
	}

	var m yaollm.Model
	if err := c.ShouldBindJSON(&m); err != nil {
		respondErr(c, http.StatusBadRequest, err.Error())
		return
	}
	m.ProviderRef = providerID
	if err := yaollm.UpdateModel(existing.ID, &m); err != nil {
		respondErr(c, http.StatusInternalServerError, err.Error())
		return
	}
	m.ID = existing.ID
	response.RespondWithSuccess(c, http.StatusOK, m)
}

func handleDeleteModel(c *gin.Context) {
	providerID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		respondErr(c, http.StatusBadRequest, "invalid provider id")
		return
	}
	modelID := c.Param("model_id")
	m, err := yaollm.FindModelByModelID(providerID, modelID)
	if err != nil {
		respondErr(c, http.StatusInternalServerError, err.Error())
		return
	}
	if m == nil {
		respondErr(c, http.StatusNotFound, "model not found")
		return
	}
	if err := yaollm.DeleteModel(m.ID); err != nil {
		respondErr(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, nil)
}

// ---------------------------------------------------------------------------
// Role Bindings (2.0)
// ---------------------------------------------------------------------------

func handleGetRoles(c *gin.Context) {
	info := authorized.GetInfo(c)
	teamID, userID := "", ""
	if info != nil {
		teamID = info.GetTeamID()
		userID = info.GetUserID()
	}

	type roleResult struct {
		Role     string           `json:"role"`
		Provider *yaollm.Provider `json:"provider"`
		Model    *yaollm.Model    `json:"model"`
	}

	roles := []string{"default", "heavy", "light", "vision", "audio", "embedding", "decision"}
	results := make([]roleResult, 0, len(roles))
	for _, role := range roles {
		if yaollm.Global == nil {
			continue
		}
		p, m, err := yaollm.Global.ResolveRoleModel(role, teamID, userID)
		if err != nil {
			continue
		}
		masked := maskProviderKey(p)
		results = append(results, roleResult{Role: role, Provider: &masked, Model: m})
	}
	response.RespondWithSuccess(c, http.StatusOK, results)
}

func handleSetRoles(c *gin.Context) {
	var bindings []yaollm.RoleBinding
	if err := c.ShouldBindJSON(&bindings); err != nil {
		respondErr(c, http.StatusBadRequest, err.Error())
		return
	}
	for i := range bindings {
		if _, err := yaollm.SetRoleBinding(&bindings[i]); err != nil {
			respondErr(c, http.StatusInternalServerError, err.Error())
			return
		}
	}
	response.RespondWithSuccess(c, http.StatusOK, bindings)
}

// ---------------------------------------------------------------------------
// Aggregate queries (2.0)
// ---------------------------------------------------------------------------

func handleListAllModels(c *gin.Context) {
	scope := c.Query("scope")
	teamID := c.Query("team_id")
	userID := c.Query("user_id")

	providers, err := yaollm.ListProviders(yaollm.Scope(scope), teamID, userID)
	if err != nil {
		respondErr(c, http.StatusInternalServerError, err.Error())
		return
	}

	type modelWithProvider struct {
		yaollm.Model
		ProviderName string            `json:"provider_name"`
		ProviderKey  yaollm.ProviderID `json:"provider_key"`
	}

	var all []modelWithProvider
	for _, p := range providers {
		if !p.Enabled {
			continue
		}
		models, err := yaollm.ListEnabledModels(p.ID)
		if err != nil {
			continue
		}
		for _, m := range models {
			all = append(all, modelWithProvider{
				Model:        m,
				ProviderName: p.Name,
				ProviderKey:  p.ProviderID,
			})
		}
	}
	response.RespondWithSuccess(c, http.StatusOK, all)
}

func handleResolveRole(c *gin.Context) {
	role := c.Param("role")
	info := authorized.GetInfo(c)
	teamID, userID := "", ""
	if info != nil {
		teamID = info.GetTeamID()
		userID = info.GetUserID()
	}

	if yaollm.Global == nil {
		respondErr(c, http.StatusInternalServerError, "llm registry not initialized")
		return
	}
	p, m, err := yaollm.Global.ResolveRoleModel(role, teamID, userID)
	if err != nil {
		respondErr(c, http.StatusNotFound, err.Error())
		return
	}
	masked := maskProviderKey(p)
	response.RespondWithSuccess(c, http.StatusOK, map[string]interface{}{
		"role":     role,
		"provider": masked,
		"model":    m,
	})
}
