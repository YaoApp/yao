package tao

import (
	"context"
	"errors"

	"github.com/yaoapp/yao/modules/product"
	taocomputer "github.com/yaoapp/yao/modules/tao/computer"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/user"
)

// taoProductProvider adapts Tao's live catalog to the generic product service.
// The generic /v1/products API does not know that this source is Tao.
type taoProductProvider struct {
	service *taocomputer.Service
	keys    taocomputer.KeyResolver
}

func newTaoProductProvider(service *taocomputer.Service, keys taocomputer.KeyResolver) product.Provider {
	return &taoProductProvider{service: service, keys: keys}
}

func (p *taoProductProvider) Source() string { return "tao" }

func (p *taoProductProvider) ListProducts(ctx context.Context, req product.ListRequest) ([]product.Product, error) {
	info := &oauthtypes.AuthorizedInfo{UserID: req.Principal.UserID, TeamID: req.Principal.TeamID}
	identity, err := user.ResolveTaoBillingIdentity(ctx, info)
	if err != nil {
		return nil, taoSourceError(err)
	}
	key, err := p.keys.ResolveTaoKey(ctx, identity.UserID)
	if err != nil {
		return nil, taoSourceError(err)
	}
	items, err := p.service.ListCloudProducts(ctx, key)
	if err != nil {
		return nil, taoSourceError(err)
	}
	out := make([]product.Product, 0, len(items))
	for _, item := range items {
		out = append(out, product.Product{
			ID: item.ID, Name: item.Name, Description: item.Description,
			Scope: item.Scope, Source: item.Source, Region: item.Region, Resource: item.Resource,
			Spec: item.Spec, Policy: item.Policy, TotalSlots: item.TotalSlots,
			AvailableSlots: item.AvailableSlots, BillingUnits: item.BillingUnits,
			Prices: item.Prices, DurationTiers: toProductTiers(item.DurationTiers),
			Images: item.Images, Health: item.Health, Status: item.Status,
		})
	}
	return out, nil
}

func toProductTiers(tiers []taocomputer.DurationTier) []product.DurationTier {
	out := make([]product.DurationTier, 0, len(tiers))
	for _, tier := range tiers {
		out = append(out, product.DurationTier{Months: tier.Months, Discount: tier.Discount})
	}
	return out
}

func taoSourceError(err error) error {
	var billingErr *user.TaoBillingError
	if errors.As(err, &billingErr) {
		return &product.SourceError{Status: billingErr.Status, Code: billingErr.Code, Description: billingErr.Description, Err: billingErr}
	}
	var apiErr *taocomputer.APIError
	if errors.As(err, &apiErr) {
		return &product.SourceError{Status: 502, Code: "tao_unavailable", Description: "Tao catalog is unavailable", Err: apiErr}
	}
	return &product.SourceError{Status: 502, Code: "tao_unavailable", Description: "Tao catalog is unavailable", Err: err}
}
