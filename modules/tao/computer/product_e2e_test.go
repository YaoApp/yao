//go:build e2e

package computer_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/yaoapp/gou/model"
	taocomputer "github.com/yaoapp/yao/modules/tao/computer"
)

func TestE2EModelProductWriter(t *testing.T) {
	_ = testEnvironment(t)
	writer := taocomputer.ModelProductWriter{}
	m := model.Select("__yao.products")
	_ = m.Insert([]string{"id", "name", "description", "scope", "policy", "status", "metadata"}, [][]interface{}{{"other-product", "other", "other", "system", "auto", "enabled", map[string]interface{}{"tao_service_id": "image", "tao_resource_id": "x"}}})
	t.Cleanup(func() { _ = m.Delete("other-product") })
	p := taocomputer.Product{ServiceID: "computer", ResourceID: "flex-2c2g", Unit: "HOUR", PriceCredits: 10, AvailableSlots: 5}
	if err := writer.UpsertProduct(context.Background(), p); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	p.AvailableSlots = 4
	if err := writer.UpsertProduct(context.Background(), p); err != nil {
		t.Fatalf("upsert update: %v", err)
	}
	rows, err := m.Get(model.QueryParam{Wheres: []model.QueryWhere{{Column: "id", Value: taocomputer.ProductIDForTest("computer", "flex-2c2g")}}})
	if err != nil || len(rows) == 0 {
		t.Fatalf("product rows=%d err=%v", len(rows), err)
	}
	if rows[0]["name"] != "flex-2c2g" {
		t.Fatalf("product=%+v", rows[0])
	}
	if err := writer.DisableMissing(context.Background(), "computer", map[string]struct{}{}); err != nil {
		t.Fatalf("disable missing: %v", err)
	}
	rows, err = m.Get(model.QueryParam{Wheres: []model.QueryWhere{{Column: "id", Value: taocomputer.ProductIDForTest("computer", "flex-2c2g")}}})
	if err != nil || len(rows) == 0 || fmt.Sprint(rows[0]["status"]) != "disabled" {
		t.Fatalf("disabled rows=%v err=%v", rows, err)
	}
	_ = m.Delete(rows[0]["id"].(string))
}
