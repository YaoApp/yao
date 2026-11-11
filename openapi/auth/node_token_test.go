package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

type fakeNodeTokenService struct {
	refreshErr error
	rotateErr  error
	revokeErr  error
	info       *oauth.NodeInfo
	infoErr    error
}

func (f *fakeNodeTokenService) RefreshNodeToken(ctx context.Context, raw string) (*oauth.NodeToken, error) {
	if f.refreshErr != nil {
		return nil, f.refreshErr
	}
	return &oauth.NodeToken{Token: "new-token", ExpiresAt: time.Unix(200, 0)}, nil
}
func (f *fakeNodeTokenService) RotateNodeToken(ctx context.Context, computerID string) (*oauth.NodeToken, error) {
	if f.rotateErr != nil {
		return nil, f.rotateErr
	}
	return &oauth.NodeToken{Token: "rotate-token", ExpiresAt: time.Unix(200, 0)}, nil
}
func (f *fakeNodeTokenService) RevokeNodeToken(ctx context.Context, jti string) error {
	return f.revokeErr
}
func (f *fakeNodeTokenService) GetNodeTokenInfo(ctx context.Context, jti string) (*oauth.NodeInfo, error) {
	return f.info, f.infoErr
}

type fakeAuthorizer struct{ err error }

func (f fakeAuthorizer) CanManage(ctx context.Context, info *types.AuthorizedInfo, computerID string) error {
	return f.err
}

func testContext(method, target, body string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(method, target, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return c, rec
}

func TestNodeTokenHandlerRefresh(t *testing.T) {
	h := &NodeTokenHandler{Service: &fakeNodeTokenService{}, Authorizer: fakeAuthorizer{}}
	c, rec := testContext(http.MethodPost, "/auth/node-tokens/refresh", "")
	c.Request.Header.Set("Authorization", "Bearer old-token")
	h.Refresh(c)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "new-token")

	c, rec = testContext(http.MethodPost, "/auth/node-tokens/refresh", `{"node_token":"body-token"}`)
	h.Refresh(c)
	require.Equal(t, http.StatusOK, rec.Code)

	c, rec = testContext(http.MethodPost, "/auth/node-tokens/refresh", "")
	h.Refresh(c)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestNodeTokenHandlerRotate(t *testing.T) {
	h := &NodeTokenHandler{Service: &fakeNodeTokenService{}, Authorizer: fakeAuthorizer{}}
	c, rec := testContext(http.MethodPost, "/auth/node-tokens/rotate", `{"computer_id":"cmp_1"}`)
	h.Rotate(c)
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	c, rec = testContext(http.MethodPost, "/auth/node-tokens/rotate", `{}`)
	c.Set("__user_id", "user_1")
	h.Rotate(c)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	h.Authorizer = fakeAuthorizer{err: ErrComputerForbidden}
	c, rec = testContext(http.MethodPost, "/auth/node-tokens/rotate", `{"computer_id":"cmp_1"}`)
	c.Set("__user_id", "user_1")
	h.Rotate(c)
	require.Equal(t, http.StatusForbidden, rec.Code)

	h.Authorizer = fakeAuthorizer{err: ErrComputerNotFound}
	c, rec = testContext(http.MethodPost, "/auth/node-tokens/rotate", `{"computer_id":"cmp_1"}`)
	c.Set("__user_id", "user_1")
	h.Rotate(c)
	require.Equal(t, http.StatusNotFound, rec.Code)

	h.Authorizer = fakeAuthorizer{}
	c, rec = testContext(http.MethodPost, "/auth/node-tokens/rotate", `{"computer_id":"cmp_1"}`)
	c.Set("__user_id", "user_1")
	h.Rotate(c)
	require.Equal(t, http.StatusOK, rec.Code)

	h.Service = &fakeNodeTokenService{rotateErr: oauth.ErrNodeTokenConflict}
	c, rec = testContext(http.MethodPost, "/auth/node-tokens/rotate", `{"computer_id":"cmp_1"}`)
	c.Set("__user_id", "user_1")
	h.Rotate(c)
	require.Equal(t, http.StatusConflict, rec.Code)
}

func TestNodeTokenHandlerRevoke(t *testing.T) {
	h := &NodeTokenHandler{Service: &fakeNodeTokenService{}, Authorizer: fakeAuthorizer{}}
	c, rec := testContext(http.MethodDelete, "/auth/node-tokens/nt_1", "")
	h.Revoke(c)
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	c, rec = testContext(http.MethodDelete, "/auth/node-tokens/nt_1", "")
	c.Set("__user_id", "user_1")
	c.Params = gin.Params{{Key: "jti", Value: "nt_1"}}
	h.Service = &fakeNodeTokenService{infoErr: oauth.ErrNodeTokenMissing}
	h.Revoke(c)
	require.Equal(t, http.StatusNoContent, rec.Code)

	h.Service = &fakeNodeTokenService{info: &oauth.NodeInfo{ComputerID: "cmp_1"}}
	h.Authorizer = fakeAuthorizer{err: ErrComputerForbidden}
	c, rec = testContext(http.MethodDelete, "/auth/node-tokens/nt_1", "")
	c.Set("__user_id", "user_1")
	c.Params = gin.Params{{Key: "jti", Value: "nt_1"}}
	h.Revoke(c)
	require.Equal(t, http.StatusForbidden, rec.Code)

	h.Authorizer = fakeAuthorizer{}
	c, rec = testContext(http.MethodDelete, "/auth/node-tokens/nt_1", "")
	c.Set("__user_id", "user_1")
	c.Params = gin.Params{{Key: "jti", Value: "nt_1"}}
	h.Revoke(c)
	require.Equal(t, http.StatusNoContent, rec.Code)

	h.Service = &fakeNodeTokenService{info: &oauth.NodeInfo{ComputerID: "cmp_1"}, revokeErr: errors.New("boom")}
	c, rec = testContext(http.MethodDelete, "/auth/node-tokens/nt_1", "")
	c.Set("__user_id", "user_1")
	c.Params = gin.Params{{Key: "jti", Value: "nt_1"}}
	h.Revoke(c)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}
