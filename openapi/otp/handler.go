package otp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/openapi/user"
	"github.com/yaoapp/yao/openapi/utils"
)

// OTPCreateRequest is the JSON body for POST /otp/create.
type OTPCreateRequest struct {
	MemberID       string `json:"member_id,omitempty"`
	UserID         string `json:"user_id,omitempty"`
	ExpiresIn      int    `json:"expires_in,omitempty"`
	Redirect       string `json:"redirect" binding:"required"`
	Scope          string `json:"scope,omitempty"`
	TokenExpiresIn int    `json:"token_expires_in,omitempty"` // access_token lifetime override (seconds)
	Consume        *bool  `json:"consume,omitempty"`          // revoke code after login; nil means default (true)
}

// OTPLoginRequest is the JSON body for POST /otp/login.
type OTPLoginRequest struct {
	Code   string `json:"code" binding:"required"`
	Locale string `json:"locale,omitempty"`
}

// Attach registers OTP HTTP routes on the given router group.
// The create endpoint requires authentication; login is public.
func Attach(group *gin.RouterGroup, auth types.OAuth) {
	group.POST("/create", auth.Guard, GinOTPCreate)
	group.POST("/login", GinOTPLogin)
}

// GinOTPCreate handles POST /otp/create (protected).
// It forces team_id from the caller's identity and validates that the
// target member belongs to the same team.
func GinOTPCreate(c *gin.Context) {
	authInfo := authorized.GetInfo(c)
	if authInfo == nil || authInfo.TeamID == "" {
		response.RespondWithError(c, http.StatusForbidden, &response.ErrorResponse{
			Code:             "forbidden",
			ErrorDescription: "team context is required",
		})
		return
	}

	var req OTPCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.RespondWithError(c, http.StatusBadRequest, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: err.Error(),
		})
		return
	}

	if req.UserID == "" && req.MemberID == "" {
		response.RespondWithError(c, http.StatusBadRequest, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "user_id or member_id is required",
		})
		return
	}

	teamID := authInfo.TeamID

	// Validate that the target member/user belongs to the caller's team
	if err := validateTeamMembership(teamID, req.UserID, req.MemberID); err != nil {
		response.RespondWithError(c, http.StatusForbidden, &response.ErrorResponse{
			Code:             "forbidden",
			ErrorDescription: err.Error(),
		})
		return
	}

	consume := true
	if req.Consume != nil {
		consume = *req.Consume
	}

	code, err := OTP.Create(&GenerateParams{
		TeamID:         teamID,
		MemberID:       req.MemberID,
		UserID:         req.UserID,
		ExpiresIn:      req.ExpiresIn,
		Redirect:       req.Redirect,
		Scope:          req.Scope,
		TokenExpiresIn: req.TokenExpiresIn,
		Consume:        consume,
	})
	if err != nil {
		response.RespondWithError(c, http.StatusInternalServerError, &response.ErrorResponse{
			Code:             "server_error",
			ErrorDescription: err.Error(),
		})
		return
	}

	response.RespondWithSuccess(c, http.StatusOK, gin.H{"code": code})
}

// GinOTPLogin handles POST /otp/login (public).
// Smart login: checks existing session first, issues tokens only when needed.
func GinOTPLogin(c *gin.Context) {
	var req OTPLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.RespondWithError(c, http.StatusBadRequest, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: err.Error(),
		})
		return
	}

	payload, err := OTP.Verify(req.Code)
	if err != nil {
		response.RespondWithError(c, http.StatusUnauthorized, &response.ErrorResponse{
			Code:             "invalid_otp",
			ErrorDescription: err.Error(),
		})
		return
	}

	var status string

	// Check if the caller already has a valid session
	existingToken := oauth.OAuth.GetAccessToken(c)
	if existingToken != "" {
		if _, verifyErr := oauth.OAuth.VerifyToken(existingToken); verifyErr == nil {
			status = "already_logged_in"
		}
	}

	// No valid session: perform OTP login and set cookies
	if status == "" {
		result, err := OTP.Login(req.Code, req.Locale)
		if err != nil {
			response.RespondWithError(c, http.StatusUnauthorized, &response.ErrorResponse{
				Code:             "otp_login_failed",
				ErrorDescription: err.Error(),
			})
			return
		}

		sid := utils.GetSessionID(c)
		if sid == "" {
			sid = session.ID()
		}

		loginResp := &user.LoginResponse{
			UserID:                result.UserID,
			Subject:               result.Subject,
			AccessToken:           result.AccessToken,
			IDToken:               result.IDToken,
			RefreshToken:          result.RefreshToken,
			ExpiresIn:             result.ExpiresIn,
			RefreshTokenExpiresIn: result.RefreshTokenExpiresIn,
			TokenType:             result.TokenType,
			Scope:                 result.Scope,
			Status:                user.LoginStatusSuccess,
		}
		user.SendLoginCookies(c, loginResp, sid)
		status = "success"
	}

	// Consume OTP code if configured
	if payload.Consume {
		_ = OTP.Revoke(req.Code)
	}

	response.RespondWithSuccess(c, http.StatusOK, gin.H{
		"status":   status,
		"redirect": payload.Redirect,
	})
}

// validateTeamMembership checks that the given user or member belongs to the specified team.
func validateTeamMembership(teamID, userID, memberID string) error {
	userProvider, err := oauth.OAuth.GetUserProvider()
	if err != nil {
		return fmt.Errorf("failed to get user provider: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if memberID != "" {
		member, err := userProvider.GetMemberByMemberID(ctx, memberID)
		if err != nil {
			return fmt.Errorf("member not found: %s", memberID)
		}
		memberTeam := utils.ToString(member["team_id"])
		if memberTeam != teamID {
			return fmt.Errorf("member %s does not belong to team %s", memberID, teamID)
		}
		return nil
	}

	if userID != "" {
		_, err := userProvider.GetMember(ctx, teamID, userID)
		if err != nil {
			return fmt.Errorf("user %s is not a member of team %s", userID, teamID)
		}
		return nil
	}

	return fmt.Errorf("user_id or member_id is required")
}
