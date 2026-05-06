package setting

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/setting"
)

const preferenceNS = "preference"

func preferenceScope(info *oauthTypes.AuthorizedInfo) setting.ScopeID {
	return setting.ScopeID{Scope: setting.ScopeUser, UserID: info.UserID}
}

// handlePreferenceGet returns the current user's preference.
// GET /setting/preference
func handlePreferenceGet(c *gin.Context) {
	info := authorized.GetInfo(c)

	if setting.Global == nil {
		response.RespondWithSuccess(c, http.StatusOK, PreferenceData{})
		return
	}

	merged, _ := setting.Global.GetMerged(info.UserID, info.TeamID, preferenceNS)
	data := preferenceFromMap(merged)
	response.RespondWithSuccess(c, http.StatusOK, data)
}

// handlePreferenceUpdate partially updates the current user's preference.
// PUT /setting/preference
func handlePreferenceUpdate(c *gin.Context) {
	info := authorized.GetInfo(c)

	var body PreferenceData
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if setting.Global == nil {
		respondError(c, http.StatusInternalServerError, "setting registry not initialized")
		return
	}

	scope := preferenceScope(info)
	existing, _ := setting.Global.Get(scope, preferenceNS)

	m := make(map[string]interface{})
	for k, v := range existing {
		m[k] = v
	}

	// Marshal the body to a map so only non-nil fields are included
	bodyBytes, _ := json.Marshal(body)
	var bodyMap map[string]interface{}
	json.Unmarshal(bodyBytes, &bodyMap)
	for k, v := range bodyMap {
		m[k] = v
	}

	if _, err := setting.Global.Set(scope, preferenceNS, m); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	merged, _ := setting.Global.GetMerged(info.UserID, info.TeamID, preferenceNS)
	result := preferenceFromMap(merged)
	response.RespondWithSuccess(c, http.StatusOK, result)
}

func preferenceFromMap(m map[string]interface{}) PreferenceData {
	data := PreferenceData{}
	if m == nil {
		return data
	}
	if v, ok := m["email_notification"].(bool); ok {
		data.EmailNotification = &v
	}
	if v, ok := m["banner_dismissed"].(bool); ok {
		data.BannerDismissed = &v
	}
	if v, ok := m["onboarding_completed"].(bool); ok {
		data.OnboardingCompleted = &v
	}
	return data
}
