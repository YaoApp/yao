package tao

import (
	"errors"
	"net/http"
	"testing"

	"github.com/yaoapp/yao/modules/product"
	taocomputer "github.com/yaoapp/yao/modules/tao/computer"
	"github.com/yaoapp/yao/openapi/user"
)

func TestTaoProductSourceError(t *testing.T) {
	err := taoSourceError(&taocomputer.APIError{Status: http.StatusServiceUnavailable})
	var sourceErr *product.SourceError
	if !errors.As(err, &sourceErr) || sourceErr.Status != http.StatusBadGateway {
		t.Fatalf("err=%+v", err)
	}
	billing := &user.TaoBillingError{Status: http.StatusForbidden, Code: "owner_key_missing", Description: "missing"}
	err = taoSourceError(billing)
	if !errors.As(err, &sourceErr) || sourceErr.Status != http.StatusForbidden || sourceErr.Code != "owner_key_missing" {
		t.Fatalf("billing err=%+v", err)
	}
}

func TestTaoProductTiers(t *testing.T) {
	got := toProductTiers([]taocomputer.DurationTier{{Months: "1-2", Discount: 0.87}})
	if len(got) != 1 || got[0].Months != "1-2" || got[0].Discount != 0.87 {
		t.Fatalf("tiers=%+v", got)
	}
}
