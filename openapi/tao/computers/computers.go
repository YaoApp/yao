// Package computers exposes the Tao cloud purchase HTTP API.
package computers

import (
	"errors"
	"net/http"
	"path"
	"strconv"

	"github.com/gin-gonic/gin"
	taocomputer "github.com/yaoapp/yao/modules/tao/computer"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

// Handler implements /v1/tao/computers.
type Handler struct {
	Service *taocomputer.Service
	Keys    taocomputer.KeyResolver
}

// Attach registers Tao computer purchase routes.
func Attach(group *gin.RouterGroup, oauthSvc oauthtypes.OAuth, service *taocomputer.Service, keys taocomputer.KeyResolver) {
	h := &Handler{Service: service, Keys: keys}
	group.POST("", oauthSvc.Guard, h.Purchase)
	group.GET("", oauthSvc.Guard, h.List)
	group.GET("/:instance_id", oauthSvc.Guard, h.Get)
	group.POST("/:instance_id/start", oauthSvc.Guard, h.Action)
	group.POST("/:instance_id/stop", oauthSvc.Guard, h.Action)
	group.POST("/:instance_id/reboot", oauthSvc.Guard, h.Action)
	group.DELETE("/:instance_id", oauthSvc.Guard, h.Delete)
}

type purchaseRequest struct {
	Resource      string                   `json:"resource"`
	Region        string                   `json:"region"`
	Name          string                   `json:"name"`
	BillingUnit   string                   `json:"billing_unit"`
	PeriodMonths  int                      `json:"period_months"`
	RequestParams map[string]any           `json:"request_params"`
	SSH           *taocomputer.SSHRequest  `json:"ssh"`
	Metadata      map[string]string        `json:"metadata"`
	Computer      taocomputer.ComputerSpec `json:"computer"`
}

func (h *Handler) Purchase(c *gin.Context) {
	info := authorized.GetInfo(c)
	if info == nil || info.UserID == "" {
		forbidden(c)
		return
	}
	var req purchaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	if req.SSH != nil || req.Computer.SSHKeyRef != "" {
		badRequest(c, errors.New("ssh key is managed by the server"))
		return
	}
	key, err := h.resolveKey(c, info)
	if err != nil {
		forbidden(c)
		return
	}
	result, err := h.Service.Purchase(c.Request.Context(), key, taocomputer.PurchaseRequest{
		Resource: req.Resource, Region: req.Region, Name: req.Name,
		BillingUnit: req.BillingUnit, PeriodMonths: req.PeriodMonths,
		RequestParams: req.RequestParams, Metadata: req.Metadata,
		IdempotencyKey: c.GetHeader("Idempotency-Key"),
		Computer:       req.Computer, UserID: info.UserID, TeamID: info.TeamID,
	})
	if err != nil {
		apiError(c, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusAccepted, purchaseResponse(result, false))
}

// Action sends a lifecycle action to a Tao host.
func (h *Handler) Action(c *gin.Context) {
	info := authorized.GetInfo(c)
	if info == nil || info.UserID == "" {
		forbidden(c)
		return
	}
	key, err := h.resolveKey(c, info)
	if err != nil {
		forbidden(c)
		return
	}
	action := path.Base(c.FullPath())
	if action == "." || action == "/" {
		action = path.Base(c.Request.URL.Path)
	}
	if action == "" {
		badRequest(c, errors.New("action is required"))
		return
	}
	result, err := h.Service.Action(c.Request.Context(), key, c.Param("instance_id"), action)
	if err != nil {
		apiError(c, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusAccepted, purchaseResponse(result, false))
}

// Delete deletes a Tao host and releases the Computer.
func (h *Handler) Delete(c *gin.Context) {
	info := authorized.GetInfo(c)
	if info == nil || info.UserID == "" {
		forbidden(c)
		return
	}
	key, err := h.resolveKey(c, info)
	if err != nil {
		forbidden(c)
		return
	}
	result, err := h.Service.Delete(c.Request.Context(), key, c.Param("instance_id"))
	if err != nil {
		apiError(c, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusAccepted, purchaseResponse(result, false))
}

func (h *Handler) Get(c *gin.Context) {
	info := authorized.GetInfo(c)
	if info == nil || info.UserID == "" {
		forbidden(c)
		return
	}
	key, err := h.resolveKey(c, info)
	if err != nil {
		forbidden(c)
		return
	}
	result, err := h.Service.Get(c.Request.Context(), key, c.Param("instance_id"))
	if err != nil {
		apiError(c, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, purchaseResponse(result, true))
}

func (h *Handler) List(c *gin.Context) {
	info := authorized.GetInfo(c)
	if info == nil || info.UserID == "" {
		forbidden(c)
		return
	}
	key, err := h.resolveKey(c, info)
	if err != nil {
		forbidden(c)
		return
	}
	items, err := h.Service.List(c.Request.Context(), key, taocomputer.ListOptions{
		Status: c.Query("status"), Region: c.Query("region"),
		Page: queryInt(c, "page"), Size: queryInt(c, "size"),
	})
	if err != nil {
		apiError(c, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"object": "list", "data": items, "total": len(items)})
}

func (h *Handler) resolveKey(c *gin.Context, info *oauthtypes.AuthorizedInfo) (string, error) {
	if h.Keys == nil {
		return "", errors.New("tao key resolver is not configured")
	}
	return h.Keys.ResolveTaoKey(c.Request.Context(), info.UserID)
}

func purchaseResponse(result *taocomputer.PurchaseResult, detail bool) gin.H {
	out := gin.H{}
	if result.Instance != nil {
		out["instance_id"] = result.Instance.InstanceID
		out["status"] = result.Instance.Status
		out["raw_status"] = result.Instance.RawStatus
		out["region"] = result.Instance.Region
		out["resource"] = result.Instance.Resource
		out["billing_unit"] = result.Instance.BillingUnit
		out["period_months"] = result.Instance.PeriodMonths
		out["price_credits"] = result.Instance.PriceCredits
		out["price_total_credits"] = result.Instance.PriceTotalCredits
		out["expires_at"] = result.Instance.ExpiresAt
		out["ssh"] = result.Instance.SSH
		out["metadata"] = result.Instance.Metadata
		out["created_at"] = result.Instance.CreatedAt
		if detail {
			out["public_ip"] = result.Instance.PublicIP
			out["private_ip"] = result.Instance.PrivateIP
			out["last_error"] = result.Instance.LastError
			out["updated_at"] = result.Instance.UpdatedAt
		}
	}
	if result.Computer != nil {
		out["computer_id"] = result.Computer.ID
		out["computer_status"] = result.Computer.Status
	}
	return out
}

func queryInt(c *gin.Context, key string) int {
	value, _ := strconv.Atoi(c.Query(key))
	return value
}

func badRequest(c *gin.Context, err error) {
	response.RespondWithError(c, http.StatusBadRequest, &response.ErrorResponse{Code: response.ErrInvalidRequest.Code, ErrorDescription: err.Error()})
}

func forbidden(c *gin.Context) {
	response.RespondWithError(c, http.StatusForbidden, &response.ErrorResponse{Code: "forbidden", ErrorDescription: "access denied"})
}

func apiError(c *gin.Context, err error) {
	if errors.Is(err, taocomputer.ErrInstanceDeleted) {
		response.RespondWithError(c, http.StatusNotFound, &response.ErrorResponse{Code: "not_found", ErrorDescription: "Tao instance deleted"})
		return
	}
	if errors.Is(err, taocomputer.ErrInvalidAction) {
		response.RespondWithError(c, http.StatusBadRequest, &response.ErrorResponse{Code: "invalid_action", ErrorDescription: err.Error()})
		return
	}
	if errors.Is(err, taocomputer.ErrTaiStart) {
		response.RespondWithError(c, http.StatusBadGateway, &response.ErrorResponse{Code: "tai_error", ErrorDescription: "Tai 启动失败"})
		return
	}
	var apiErr *taocomputer.APIError
	if errors.As(err, &apiErr) {
		status := http.StatusBadGateway
		switch apiErr.Status {
		case http.StatusUnauthorized, http.StatusForbidden:
			status = http.StatusForbidden
		case http.StatusNotFound:
			status = http.StatusNotFound
		case http.StatusPaymentRequired:
			status = http.StatusPaymentRequired
		case http.StatusConflict:
			status = http.StatusConflict
		case http.StatusTooManyRequests:
			status = http.StatusTooManyRequests
		case http.StatusUnprocessableEntity, http.StatusBadRequest:
			status = http.StatusBadRequest
		}
		response.RespondWithError(c, status, &response.ErrorResponse{Code: "tao_error", ErrorDescription: apiErr.Error()})
		return
	}
	response.RespondWithError(c, http.StatusInternalServerError, &response.ErrorResponse{Code: "server_error", ErrorDescription: err.Error()})
}
