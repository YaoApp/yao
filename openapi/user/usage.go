package user

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/openapi/utils"
)

// UsageResponse is the stable account-menu contract for the current user.
// A successful response always contains the real Tao account balance.
type UsageResponse struct {
	Plan      string `json:"plan"`
	PlanName  string `json:"plan_name,omitempty"`
	Remaining int64  `json:"remaining"`
	Unit      string `json:"unit"`
}

type taoAccountBalanceResponse struct {
	Balance *int64 `json:"balance"`
}

// GinUsage returns the current user's plan and Tao account balance.
func GinUsage(c *gin.Context) {
	authInfo := authorized.GetInfo(c)
	if authInfo == nil || authInfo.UserID == "" {
		response.RespondWithError(c, http.StatusUnauthorized, &response.ErrorResponse{
			Code:             response.ErrInvalidClient.Code,
			ErrorDescription: "User not authenticated",
		})
		return
	}

	if !TaoConfigured() {
		response.RespondWithError(c, http.StatusServiceUnavailable, &response.ErrorResponse{
			Code:             "tao_not_configured",
			ErrorDescription: "Tao OAuth is not configured",
		})
		return
	}

	identity, err := ResolveTaoBillingIdentity(c.Request.Context(), authInfo)
	if err != nil {
		respondTaoBillingError(c, err)
		return
	}
	if identity.Mode == TaoBillingKeyModeOwner && !identity.IsOwner {
		response.RespondWithError(c, http.StatusForbidden, &response.ErrorResponse{
			Code:             "owner_balance_hidden",
			ErrorDescription: "Owner balance is only visible to the team owner",
		})
		return
	}

	provider, err := taoGetUserProvider()
	if err != nil {
		response.RespondWithError(c, http.StatusInternalServerError, &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to retrieve usage",
		})
		return
	}

	user, err := provider.GetUser(c.Request.Context(), identity.UserID)
	if err != nil {
		response.RespondWithError(c, http.StatusInternalServerError, &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to retrieve usage",
		})
		return
	}

	balance, err := fetchTaoAccountBalance(c.Request.Context(), identity.UserID)
	if err != nil {
		response.RespondWithError(c, http.StatusBadGateway, &response.ErrorResponse{
			Code:             "tao_balance_unavailable",
			ErrorDescription: "Failed to retrieve Tao account balance",
			Reason:           err.Error(),
		})
		return
	}

	result := UsageResponse{
		Plan:      utils.ToString(user["type_id"]),
		PlanName:  utils.ToString(user["type_id"]),
		Remaining: balance,
		Unit:      "credits",
	}
	if result.Plan == "" {
		result.Plan = "free"
	}
	if result.PlanName == "" {
		result.PlanName = result.Plan
	}

	response.RespondWithSuccess(c, http.StatusOK, result)
}

// fetchTaoAccountBalance reads the account-level balance through Tao's OAuth
// bridge. It is the only source for account usage; it does not use an sk-tao
// service key.
func fetchTaoAccountBalance(ctx context.Context, userID string) (int64, error) {
	token, err := taoRefreshUserToken(ctx, userID)
	if err != nil {
		return 0, err
	}
	if token == nil || token.AccessToken == "" {
		return 0, fmt.Errorf("tao access token is empty")
	}
	return fetchTaoAccountBalanceWithToken(ctx, token.AccessToken)
}

func fetchTaoAccountBalanceWithToken(ctx context.Context, accessToken string) (int64, error) {
	provider, err := taoProvider()
	if err != nil {
		return 0, err
	}
	endpoint := strings.TrimRight(provider.Issuer, "/") + "/api/v1/users/me/balance"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("tao account balance endpoint returned %d", resp.StatusCode)
	}
	var body taoAccountBalanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, err
	}
	if body.Balance == nil {
		return 0, fmt.Errorf("tao account balance is unavailable")
	}
	return *body.Balance, nil
}
