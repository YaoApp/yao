package user

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/response"
)

// RequireTaoIdentity rejects requests from non-Tao identities for Tao-backed
// application APIs. It must run after the OAuth guard.
func RequireTaoIdentity(c *gin.Context) {
	if !TaoConfigured() {
		response.RespondWithError(c, http.StatusServiceUnavailable, &response.ErrorResponse{
			Code:             "tao_not_configured",
			ErrorDescription: "Tao OAuth is not configured; set TAO_OAUTH_ISSUER",
		})
		c.Abort()
		return
	}
	info := authorized.GetInfo(c)
	if info == nil || info.UserID == "" {
		response.RespondWithError(c, http.StatusUnauthorized, &response.ErrorResponse{
			Code:             "unauthorized",
			ErrorDescription: "User not authenticated",
		})
		c.Abort()
		return
	}
	if info.AuthSource == "tao" {
		// A Tao session without a persisted Tao OAuth account cannot call
		// Tao-backed APIs; the user must sign in with Tao again.
		provider, err := taoGetUserProvider()
		if err != nil || provider == nil {
			response.RespondWithError(c, http.StatusServiceUnavailable, &response.ErrorResponse{
				Code:             "tao_identity_required",
				ErrorDescription: "Unable to resolve Tao authorization",
			})
			c.Abort()
			return
		}
		accounts, err := provider.GetUserOAuthAccounts(c.Request.Context(), info.UserID)
		if err == nil {
			for _, account := range accounts {
				if fmt.Sprint(account["provider"]) == "tao" && fmt.Sprint(account["refresh_token"]) != "" {
					c.Next()
					return
				}
			}
		}
		response.RespondWithError(c, http.StatusForbidden, &response.ErrorResponse{
			Code:             "tao_authorization_required",
			ErrorDescription: "Please sign in with Tao again to continue",
		})
		c.Abort()
		return
	}
	provider, err := taoGetUserProvider()
	if err != nil {
		response.RespondWithError(c, http.StatusServiceUnavailable, &response.ErrorResponse{
			Code:             "tao_identity_required",
			ErrorDescription: "Unable to resolve user identity",
		})
		c.Abort()
		return
	}
	u, err := provider.GetUser(context.Background(), info.UserID)
	if err == nil {
		if source, _ := u["identity_source"].(string); source == "tao" {
			c.Next()
			return
		}
	}
	response.RespondWithError(c, http.StatusForbidden, &response.ErrorResponse{
		Code:             "tao_identity_required",
		ErrorDescription: "This feature requires a Tao account",
	})
	c.Abort()
}

// RequireTaoManagedKey ensures Tao Computers/Catalog calls have a managed key.
// The account binding endpoint is intentionally not gated because it is how a
// user can provide a key manually.
func RequireTaoManagedKey(c *gin.Context) {
	info := authorized.GetInfo(c)
	if info == nil || info.UserID == "" {
		response.RespondWithError(c, http.StatusUnauthorized, &response.ErrorResponse{
			Code:             "unauthorized",
			ErrorDescription: "User not authenticated",
		})
		c.Abort()
		return
	}
	if _, err := ResolveTaoBillingKey(c.Request.Context(), info); err != nil {
		var billingErr *TaoBillingError
		if errors.As(err, &billingErr) {
			response.RespondWithError(c, billingErr.Status, &response.ErrorResponse{
				Code:             billingErr.Code,
				ErrorDescription: billingErr.Description,
			})
		} else {
			response.RespondWithError(c, http.StatusForbidden, &response.ErrorResponse{
				Code:             "tao_authorization_required",
				ErrorDescription: "Please sign in with Tao again to continue",
			})
		}
		c.Abort()
		return
	}
	c.Next()
}
