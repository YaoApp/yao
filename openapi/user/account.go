package user

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/response"
)

// ChangePasswordRequest represents the request body for changing password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}

// GinChangePassword handles PUT /account/password - Change current user's password
func GinChangePassword(c *gin.Context) {
	// 1. Get authorized user info from Guard
	authInfo := authorized.GetInfo(c)
	if authInfo == nil || authInfo.UserID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidClient.Code,
			ErrorDescription: "User not authenticated",
		}
		response.RespondWithError(c, response.StatusUnauthorized, errorResp)
		return
	}

	// 2. Parse request body
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: fmt.Sprintf("Invalid request body: %s", err.Error()),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// 3. Validate new_password == confirm_password
	if req.NewPassword != req.ConfirmPassword {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "New password and confirm password do not match",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// 4. Validate new password format (reuse validatePassword from entry.go)
	if err := validatePassword(req.NewPassword); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// 5. Get user provider
	ctx := c.Request.Context()
	userProvider, err := oauth.OAuth.GetUserProvider()
	if err != nil {
		log.Error("Failed to get user provider: %v", err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Internal server error",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// 6. Get user auth data (includes password_hash)
	user, err := userProvider.GetUserForAuth(ctx, authInfo.UserID, "user_id")
	if err != nil {
		log.Error("Failed to get user for auth: %v", err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to verify current password",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// 7. Get password hash and verify current password
	// Note: OAuth-only users (Google/GitHub) have no password_hash and no email.
	// The frontend should not show the change password option for these users.
	passwordHash, ok := user["password_hash"].(string)
	if !ok || passwordHash == "" {
		log.Warn("User %s has no password hash (likely OAuth-only user)", authInfo.UserID)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Password change is not available for this account",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	valid, err := userProvider.VerifyPassword(ctx, req.CurrentPassword, passwordHash)
	if err != nil || !valid {
		log.Warn("Password verification failed for user %s during password change", authInfo.UserID)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Current password is incorrect",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// 8. Update password
	if err := userProvider.UpdatePassword(ctx, authInfo.UserID, req.NewPassword); err != nil {
		log.Error("Failed to update password for user %s: %v", authInfo.UserID, err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to update password",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	log.Info("Password changed successfully for user %s", authInfo.UserID)

	// 9. Return success
	c.JSON(http.StatusOK, gin.H{
		"message": "Password changed successfully",
	})
}
