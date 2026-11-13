package computer

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"

	"github.com/yaoapp/gou/model"
)

// ModelProductWriter upserts Tao catalog resources into __yao.products.
type ModelProductWriter struct{}

// UpsertProduct writes one product row keyed by service/resource.
func (ModelProductWriter) UpsertProduct(ctx context.Context, p Product) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m := model.Select("__yao.products")
	if m == nil {
		return fmt.Errorf("product model is not loaded")
	}
	id := productID(p.ServiceID, p.ResourceID)
	rows, err := m.Get(model.QueryParam{Wheres: []model.QueryWhere{{Column: "id", Value: id}}})
	if err != nil {
		return err
	}
	metadata := map[string]interface{}{
		"tao_service_id": p.ServiceID, "tao_resource_id": p.ResourceID,
		"unit": p.Unit, "billing_units": p.BillingUnits,
		"prices": p.Prices, "images": p.Images,
	}
	if len(rows) > 0 {
		return m.Update(id, map[string]interface{}{
			"name": p.ResourceID, "available_slots": p.AvailableSlots,
			"price_credits": p.PriceCredits, "status": "enabled", "metadata": metadata,
		})
	}
	return m.Insert(
		[]string{"id", "name", "description", "scope", "policy", "total_slots", "available_slots", "price_credits", "status", "metadata"},
		[][]interface{}{{id, p.ResourceID, "Tao catalog resource " + p.ResourceID, "system", "auto", p.AvailableSlots, p.AvailableSlots, p.PriceCredits, "enabled", metadata}},
	)
}

// ProductIDForTest exposes the deterministic product ID for e2e checks.
func ProductIDForTest(serviceID, resourceID string) string { return productID(serviceID, resourceID) }

// productID is the deterministic logical unique key for (service, resource).
func productID(serviceID, resourceID string) string {
	sum := sha1.Sum([]byte(serviceID + ":" + resourceID))
	return "tao-" + hex.EncodeToString(sum[:])[:27]
}

var _ ProductWriter = ModelProductWriter{}

// DisableMissing disables Tao products that are no longer in the catalog.
func (ModelProductWriter) DisableMissing(ctx context.Context, serviceID string, keep map[string]struct{}) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m := model.Select("__yao.products")
	if m == nil {
		return fmt.Errorf("product model is not loaded")
	}
	rows, err := m.Get(model.QueryParam{})
	if err != nil {
		return err
	}
	for _, row := range rows {
		metadata, ok := row["metadata"].(map[string]interface{})
		if !ok || fmt.Sprint(metadata["tao_service_id"]) != serviceID {
			continue
		}
		resourceID := fmt.Sprint(metadata["tao_resource_id"])
		if _, ok := keep[resourceID]; ok {
			continue
		}
		if id, ok := row["id"].(string); ok {
			if err := m.Update(id, map[string]interface{}{"status": "disabled"}); err != nil {
				return err
			}
		}
	}
	return nil
}
