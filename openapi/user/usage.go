package user

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/openapi/utils"
)

// UsageResponse is the stable account-menu contract for the current user.
// remaining is null when Tao cannot report the balance; callers must treat
// balance_available=false as unknown, not as zero.
type UsageResponse struct {
	Plan             string `json:"plan"`
	PlanName         string `json:"plan_name,omitempty"`
	Remaining        *int64 `json:"remaining"`
	Unit             string `json:"unit"`
	BalanceAvailable bool   `json:"balance_available"`
}

type taoKeyBalanceResponse struct {
	Object           string `json:"object"`
	Valid            bool   `json:"valid"`
	Balance          *int64 `json:"balance"`
	BalanceAvailable bool   `json:"balance_available"`
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

	provider, err := getUserProvider()
	if err != nil {
		response.RespondWithError(c, http.StatusInternalServerError, &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to retrieve usage",
		})
		return
	}

	user, err := provider.GetUser(c.Request.Context(), authInfo.UserID)
	if err != nil {
		response.RespondWithError(c, http.StatusInternalServerError, &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to retrieve usage",
		})
		return
	}

	result := UsageResponse{
		Plan: utils.ToString(user["type_id"]),
		Unit: "credits",
	}
	if result.Plan == "" {
		result.Plan = "free"
	}
	result.PlanName = result.Plan

	apiKey := utils.ToString(user["tao_api_key"])
	if apiKey == "" {
		// GetUser is the public profile shape and may not include the managed
		// Tao key; the auth shape does.
		if authUser, authErr := provider.GetUserForAuth(c.Request.Context(), authInfo.UserID, "user_id"); authErr == nil {
			apiKey = utils.ToString(authUser["tao_api_key"])
		}
	}
	if TaoConfigured() {
		var balance *int64
		var available bool
		var balanceErr error
		if apiKey != "" {
			balance, available, balanceErr = fetchTaoKeyBalance(c.Request.Context(), apiKey)
		}
		if balance == nil || !available || balanceErr != nil {
			// Older Tao users may have a managed key that is no longer stored
			// locally (Tao only returns plaintext once). Use their OAuth account
			// balance instead, which does not require a key.
			accountBalance, accountAvailable, accountErr := fetchTaoAccountBalance(c.Request.Context(), authInfo.UserID)
			if accountErr != nil {
				log.Warn("Failed to fetch Tao account balance: %v", accountErr)
			} else if accountAvailable {
				balance = accountBalance
				available = accountAvailable
				balanceErr = nil
			}
		}
		if balanceErr != nil {
			log.Warn("Failed to fetch Tao balance: %v", balanceErr)
		}
		result.BalanceAvailable = available
		result.Remaining = balance
	}

	response.RespondWithSuccess(c, http.StatusOK, result)
}

func fetchTaoKeyBalance(ctx context.Context, apiKey string) (*int64, bool, error) {
	provider, err := taoProvider()
	if err != nil {
		return nil, false, err
	}
	baseURL := strings.TrimRight(provider.Issuer, "/")
	if baseURL == "" {
		return nil, false, fmt.Errorf("tao API base URL is not configured")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/key", nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("tao key endpoint returned %d", resp.StatusCode)
	}

	var body taoKeyBalanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, false, err
	}
	if !body.BalanceAvailable || body.Balance == nil {
		return nil, false, nil
	}
	return body.Balance, true, nil
}

type taoAccountBalanceResponse struct {
	Balance *int64 `json:"balance"`
}

// fetchTaoAccountBalance reads the account-level Ding balance through Tao's
// OAuth bridge. It is used when the user's managed Tao key cannot be recovered
// or was created before Yao started persisting the plaintext.
func fetchTaoAccountBalance(ctx context.Context, userID string) (*int64, bool, error) {
	token, err := taoRefreshUserToken(ctx, userID)
	if err != nil {
		return nil, false, err
	}
	if token == nil || token.AccessToken == "" {
		return nil, false, fmt.Errorf("tao access token is empty")
	}
	return fetchTaoAccountBalanceWithToken(ctx, token.AccessToken)
}

func fetchTaoAccountBalanceWithToken(ctx context.Context, accessToken string) (*int64, bool, error) {
	provider, err := taoProvider()
	if err != nil {
		return nil, false, err
	}
	endpoint := strings.TrimRight(provider.Issuer, "/") + "/api/v1/users/me/balance"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("tao account balance endpoint returned %d", resp.StatusCode)
	}
	var body taoAccountBalanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, false, err
	}
	if body.Balance == nil {
		return nil, false, nil
	}
	return body.Balance, true, nil
}
