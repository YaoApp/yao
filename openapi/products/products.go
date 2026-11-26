// Package products exposes the source-neutral product catalog API.
package products

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/modules/product"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

// Handler exposes GET /v1/products.
type Handler struct {
	Service *product.Service
}

// Attach registers the product catalog route.
func Attach(group *gin.RouterGroup, oauthSvc oauthtypes.OAuth, service *product.Service) {
	h := &Handler{Service: service}
	group.Use(oauthSvc.Guard)
	group.GET("", h.List)
}

// List lists products from the requested source only. The API is deliberately
// independent from Tao; Tao is one registered provider.
func (h *Handler) List(c *gin.Context) {
	info := authorized.GetInfo(c)
	if info == nil || info.UserID == "" {
		response.RespondWithError(c, http.StatusUnauthorized, &response.ErrorResponse{
			Code: "unauthorized", ErrorDescription: "User not authenticated",
		})
		return
	}
	source := c.Query("source")
	if source == "" {
		response.RespondWithError(c, http.StatusBadRequest, &response.ErrorResponse{
			Code: "invalid_request", ErrorDescription: "source is required",
		})
		return
	}
	items, err := h.Service.List(c.Request.Context(), product.ListRequest{
		Source: source, Policy: c.Query("policy"), Status: c.Query("status"),
		Principal: product.Principal{UserID: info.UserID, TeamID: info.TeamID},
	})
	if err != nil {
		if errors.Is(err, product.ErrSourceNotFound) {
			response.RespondWithError(c, http.StatusBadRequest, &response.ErrorResponse{
				Code: "invalid_source", ErrorDescription: "unknown product source",
			})
			return
		}
		var sourceErr *product.SourceError
		if errors.As(err, &sourceErr) {
			status := sourceErr.Status
			if status == 0 {
				status = http.StatusBadGateway
			}
			response.RespondWithError(c, status, &response.ErrorResponse{
				Code: sourceErr.Code, ErrorDescription: sourceErr.Error(),
			})
			return
		}
		response.RespondWithError(c, http.StatusBadGateway, &response.ErrorResponse{
			Code: "product_source_unavailable", ErrorDescription: err.Error(),
		})
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"data": items})
}
