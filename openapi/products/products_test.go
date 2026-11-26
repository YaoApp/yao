package products

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/modules/product"
)

type fakeProvider struct {
	source string
	items  []product.Product
	err    error
}

func (f fakeProvider) Source() string { return f.source }
func (f fakeProvider) ListProducts(context.Context, product.ListRequest) ([]product.Product, error) {
	return f.items, f.err
}

func productContext(path string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, path, nil)
	c.Set("__user_id", "user-1")
	c.Set("__team_id", "team-1")
	return c, rec
}

func TestListRequiresSource(t *testing.T) {
	c, rec := productContext("/v1/products")
	(&Handler{Service: product.NewService()}).List(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestListUnknownSource(t *testing.T) {
	c, rec := productContext("/v1/products?source=unknown")
	(&Handler{Service: product.NewService()}).List(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestListProviderError(t *testing.T) {
	svc := product.NewService()
	svc.Register(fakeProvider{source: "tao", err: &product.SourceError{Status: http.StatusForbidden, Code: "tao_key_missing", Description: "key missing"}})
	c, rec := productContext("/v1/products?source=tao")
	(&Handler{Service: svc}).List(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestListSuccess(t *testing.T) {
	svc := product.NewService()
	svc.Register(fakeProvider{source: "tao", items: []product.Product{{ID: "p1", Source: "tao", Policy: "paid", Status: "enabled"}}})
	c, rec := productContext("/v1/products?source=tao&policy=paid&status=enabled")
	(&Handler{Service: svc}).List(c)
	if rec.Code != http.StatusOK || rec.Body.Len() == 0 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestListUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/products?source=tao", nil)
	(&Handler{Service: product.NewService()}).List(c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestListGenericProviderError(t *testing.T) {
	svc := product.NewService()
	svc.Register(fakeProvider{source: "tao", err: context.Canceled})
	c, rec := productContext("/v1/products?source=tao")
	(&Handler{Service: svc}).List(c)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
