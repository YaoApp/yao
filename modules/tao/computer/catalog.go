package computer

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
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
				Name: resourceDisplayName(resource), Description: strings.TrimSpace(resource.Description),
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

// CloudProduct is the Tao-specific live product shape consumed by the Tao product provider.
type CloudProduct struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	Scope          string         `json:"scope"`
	Source         string         `json:"source"`
	Region         string         `json:"region"`
	Resource       string         `json:"resource"`
	Spec           map[string]any `json:"spec"`
	Policy         string         `json:"policy"`
	TotalSlots     int            `json:"total_slots"`
	AvailableSlots int            `json:"available_slots"`
	BillingUnits   []string       `json:"billing_units"`
	Prices         map[string]int `json:"prices"`
	DurationTiers  []DurationTier `json:"duration_tiers,omitempty"`
	Images         []string       `json:"images"`
	Health         string         `json:"health"`
	Status         string         `json:"status"`
}

// ListCloudProducts reads the live Tao catalog and maps enabled computer
// resources to the CUI Product contract. It deliberately does not read or
// write __yao.products.
func (s *Service) ListCloudProducts(ctx context.Context, taoKey string) ([]CloudProduct, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if s.client == nil {
		return nil, fmt.Errorf("tao computer service is not configured")
	}
	catalog, err := s.client.GetCatalog(ctx, taoKey)
	if err != nil {
		return nil, err
	}
	out := []CloudProduct{}
	for _, service := range catalog.Services {
		if service.ID != "computer" || service.Kind != "resource" {
			continue
		}
		for _, resource := range service.Resources {
			if !resource.Enabled {
				continue
			}
			out = append(out, CloudProduct{
				ID:   cloudProductID(service.ID, resource.ID, resource.Region),
				Name: resourceDisplayName(resource), Description: strings.TrimSpace(resource.Description),
				Scope: "system", Source: "tao", Region: resource.Region, Resource: resource.ID,
				Spec: map[string]any{
					"cpu_cores": resource.CPU, "memory_mb": resource.MemoryMB,
					"disk_gb": resource.DiskGB, "bandwidth_mbps": resource.BandwidthMbps,
					"images": resource.Images,
				},
				Policy: "paid", TotalSlots: resource.AvailableSlots, AvailableSlots: resource.AvailableSlots,
				BillingUnits: resource.BillingUnits, Prices: resource.Prices,
				DurationTiers: resource.DurationTiers, Images: resource.Images,
				Health: "healthy", Status: "enabled",
			})
		}
	}
	return out, nil
}

// resourceDisplayName returns the configured product name, with the opaque
// resource id as a backward-compatible fallback for older Tao builds.
func resourceDisplayName(resource CatalogResource) string {
	if name := strings.TrimSpace(resource.Name); name != "" {
		return name
	}
	return resource.ID
}

func cloudProductID(serviceID, resourceID, region string) string {
	sum := sha1.Sum([]byte(serviceID + ":" + resourceID + ":" + region))
	return "tao-" + hex.EncodeToString(sum[:])[:27]
}
