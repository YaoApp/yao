package computer

import (
	"context"
	"fmt"
)

// Product is one product upsert from Tao Catalog.
type Product struct {
	ServiceID      string
	ResourceID     string
	Name           string
	Description    string
	Unit           string
	BillingUnits   []string
	Prices         map[string]int
	Images         []string
	PriceCredits   int
	AvailableSlots int
}

// ProductWriter persists catalog entries as Yao products.
type ProductWriter interface {
	UpsertProduct(context.Context, Product) error
	DisableMissing(context.Context, string, map[string]struct{}) error
}

// WithProducts sets the product writer used by SyncCatalog.
func (s *Service) WithProducts(writer ProductWriter) *Service {
	s.products = writer
	return s
}

// SyncCatalog reads Tao /v1/catalog and upserts computer resources.
func (s *Service) SyncCatalog(ctx context.Context, taoKey string) (int, error) {
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}
	if s.client == nil || s.products == nil {
		return 0, fmt.Errorf("catalog sync is not configured")
	}
	catalog, err := s.client.GetCatalog(ctx, taoKey)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, service := range catalog.Services {
		if service.ID != "computer" || service.Kind != "resource" {
			continue
		}
		keep := map[string]struct{}{}
		for _, resource := range service.Resources {
			if !resource.Enabled || len(resource.BillingUnits) == 0 {
				continue
			}
			unit := resource.BillingUnits[0]
			if _, ok := resource.Prices[BillingHour]; ok {
				unit = BillingHour
			}
			keep[resource.ID] = struct{}{}
			if err := s.products.UpsertProduct(ctx, Product{
				ServiceID: service.ID, ResourceID: resource.ID,
				Unit: unit, BillingUnits: resource.BillingUnits, Prices: resource.Prices,
				Images: resource.Images, PriceCredits: resource.Prices[unit],
				AvailableSlots: resource.AvailableSlots,
			}); err != nil {
				return count, err
			}
			count++
		}
		if err := s.products.DisableMissing(ctx, service.ID, keep); err != nil {
			return count, err
		}
	}
	return count, nil
}
