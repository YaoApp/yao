package user

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

// TaoBillingIdentity is the user whose Tao managed key pays for the current
// request. The key is still stored per user; only the selected billing user
// changes with the deployment-level owner/self policy.
type TaoBillingIdentity struct {
	UserID  string
	TeamID  string
	Mode    string
	IsOwner bool
}

// TaoBillingError is a stable error contract for Tao billing/auth failures.
type TaoBillingError struct {
	Code        string
	Status      int
	Description string
	Err         error
}

func (e *TaoBillingError) Error() string {
	if e == nil {
		return ""
	}
	if e.Description != "" {
		return e.Description
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Code
}

func (e *TaoBillingError) Unwrap() error { return e.Err }

// ResolveTaoBillingIdentity resolves the user whose managed key is used for
// billing. owner is the default; no team context falls back to self.
func ResolveTaoBillingIdentity(ctx context.Context, info *types.AuthorizedInfo) (*TaoBillingIdentity, error) {
	if info == nil || info.UserID == "" {
		return nil, &TaoBillingError{Code: "unauthorized", Status: 401, Description: "User not authenticated"}
	}
	mode := TaoBillingKeyMode()
	if mode == TaoBillingKeyModeSelf || info.TeamID == "" {
		return &TaoBillingIdentity{UserID: info.UserID, TeamID: info.TeamID, Mode: TaoBillingKeyModeSelf}, nil
	}
	provider, err := taoGetUserProvider()
	if err != nil || provider == nil {
		return nil, &TaoBillingError{Code: "tao_billing_unavailable", Status: 503, Description: "Tao billing identity is unavailable", Err: err}
	}
	team, err := provider.GetTeam(ctx, info.TeamID)
	if err != nil {
		return nil, &TaoBillingError{Code: "tao_billing_unavailable", Status: 503, Description: "Unable to resolve team owner", Err: err}
	}
	ownerID := fmt.Sprint(team["owner_id"])
	if ownerID == "" || ownerID == "<nil>" {
		return nil, &TaoBillingError{Code: "tao_billing_unavailable", Status: 503, Description: "Team owner is not available"}
	}
	return &TaoBillingIdentity{
		UserID:  ownerID,
		TeamID:  info.TeamID,
		Mode:    TaoBillingKeyModeOwner,
		IsOwner: ownerID == info.UserID,
	}, nil
}

// ResolveTaoBillingKey resolves and validates the selected billing user's key.
// It never creates a key for another user.
func ResolveTaoBillingKey(ctx context.Context, info *types.AuthorizedInfo) (string, error) {
	identity, err := ResolveTaoBillingIdentity(ctx, info)
	if err != nil {
		return "", err
	}
	provider, err := taoGetUserProvider()
	if err != nil || provider == nil {
		return "", &TaoBillingError{Code: "tao_billing_unavailable", Status: 503, Description: "Tao billing identity is unavailable", Err: err}
	}
	user, err := provider.GetUserForAuth(ctx, identity.UserID, "user_id")
	if err != nil {
		if identity.Mode == TaoBillingKeyModeOwner {
			return "", &TaoBillingError{Code: "owner_key_missing", Status: 403, Description: "Team owner Tao key is missing", Err: err}
		}
		return "", &TaoBillingError{Code: "tao_key_missing", Status: 403, Description: "Tao managed key is missing", Err: err}
	}
	key, _ := user["tao_api_key"].(string)
	if key == "" {
		if identity.Mode == TaoBillingKeyModeOwner {
			return "", &TaoBillingError{Code: "owner_key_missing", Status: 403, Description: "Team owner Tao key is missing"}
		}
		return "", &TaoBillingError{Code: "tao_key_missing", Status: 403, Description: "Tao managed key is missing"}
	}
	return key, nil
}

func respondTaoBillingError(c *gin.Context, err error) {
	var billingErr *TaoBillingError
	if errors.As(err, &billingErr) {
		response.RespondWithError(c, billingErr.Status, &response.ErrorResponse{
			Code:             billingErr.Code,
			ErrorDescription: billingErr.Description,
		})
		return
	}
	response.RespondWithError(c, http.StatusBadGateway, &response.ErrorResponse{
		Code:             "tao_billing_unavailable",
		ErrorDescription: "Tao billing is unavailable",
	})
}
