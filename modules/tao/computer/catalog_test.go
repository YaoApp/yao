package computer_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yaoapp/yao/e2e/taocloud"
	taocomputer "github.com/yaoapp/yao/modules/tao/computer"
)

type fakeProducts struct {
	items []taocomputer.Product
}

func (f *fakeProducts) UpsertProduct(_ context.Context, p taocomputer.Product) error {
	f.items = append(f.items, p)
	return nil
}
func (f *fakeProducts) DisableMissing(context.Context, string, map[string]struct{}) error { return nil }

func TestSyncCatalog(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	products := &fakeProducts{}
	svc := taocomputer.NewService(taocomputer.NewClient(srv.URL()), &fakeComputerService{}).WithProducts(products)
	n, err := svc.SyncCatalog(context.Background(), testTAOKey)
	if err != nil {
		t.Fatalf("SyncCatalog: %v", err)
	}
	if n != 2 || len(products.items) != 2 {
		t.Fatalf("synced=%d items=%d", n, len(products.items))
	}
	var flex *taocomputer.Product
	for i := range products.items {
		if products.items[i].ResourceID == "flex-2c2g" {
			flex = &products.items[i]
		}
	}
	if flex == nil || flex.ServiceID != "computer" || flex.Unit != "HOUR" || flex.PriceCredits == 0 || flex.Prices["MONTH"] == 0 {
		t.Fatalf("unexpected products: %+v", products.items)
	}
	if flex.Name != "Flex 2C2G" || flex.Description != "Flex 2C2G mock resource" {
		t.Fatalf("display fields: %+v", flex)
	}
}

func TestSyncCatalogRequiresWriter(t *testing.T) {
	svc := taocomputer.NewService(nil, &fakeComputerService{})
	if _, err := svc.SyncCatalog(context.Background(), testTAOKey); err == nil {
		t.Fatal("expected config error")
	}
}

type failingProducts struct{}

func (failingProducts) UpsertProduct(context.Context, taocomputer.Product) error {
	return context.Canceled
}
func (failingProducts) DisableMissing(context.Context, string, map[string]struct{}) error { return nil }

func TestSyncCatalogError(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	svc := taocomputer.NewService(taocomputer.NewClient(srv.URL()), &fakeComputerService{}).WithProducts(failingProducts{})
	if _, err := svc.SyncCatalog(context.Background(), testTAOKey); err == nil {
		t.Fatal("expected product error")
	}
}

type skipProducts struct{ count int }

func (s *skipProducts) UpsertProduct(context.Context, taocomputer.Product) error {
	s.count++
	return nil
}
func (s *skipProducts) DisableMissing(context.Context, string, map[string]struct{}) error { return nil }

func TestSyncCatalogSkipsOtherServices(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	products := &skipProducts{}
	svc := taocomputer.NewService(taocomputer.NewClient(srv.URL()), &fakeComputerService{}).WithProducts(products)
	if _, err := svc.SyncCatalog(context.Background(), testTAOKey); err != nil {
		t.Fatal(err)
	}
	if products.count != 2 {
		t.Fatalf("count=%d", products.count)
	}
}

func TestSyncCatalogCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	svc := taocomputer.NewService(taocomputer.NewClient("http://example.test"), &fakeComputerService{}).WithProducts(&skipProducts{})
	if _, err := svc.SyncCatalog(ctx, testTAOKey); err == nil {
		t.Fatal("expected context error")
	}
}

func TestSyncCatalogSkipsNonComputerService(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(taocomputer.Catalog{Services: []taocomputer.CatalogService{{ID: "image", Kind: "resource", Resources: []taocomputer.CatalogResource{{ID: "x"}}}}})
	}))
	defer srv.Close()
	products := &skipProducts{}
	svc := taocomputer.NewService(taocomputer.NewClient(srv.URL), &fakeComputerService{}).WithProducts(products)
	n, err := svc.SyncCatalog(context.Background(), testTAOKey)
	if err != nil || n != 0 || products.count != 0 {
		t.Fatalf("n=%d count=%d err=%v", n, products.count, err)
	}
}

func TestSyncCatalogSkipsDisabledResource(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(taocomputer.Catalog{Services: []taocomputer.CatalogService{{ID: "computer", Kind: "resource", Resources: []taocomputer.CatalogResource{{ID: "x", Enabled: false, BillingUnits: []string{"HOUR"}, Prices: map[string]int{"HOUR": 1}}}}}})
	}))
	defer srv.Close()
	products := &skipProducts{}
	svc := taocomputer.NewService(taocomputer.NewClient(srv.URL), &fakeComputerService{}).WithProducts(products)
	n, err := svc.SyncCatalog(context.Background(), testTAOKey)
	if err != nil || n != 0 || products.count != 0 {
		t.Fatalf("n=%d count=%d err=%v", n, products.count, err)
	}
}

type disableFailProducts struct{ skipProducts }

func (disableFailProducts) UpsertProduct(context.Context, taocomputer.Product) error { return nil }
func (disableFailProducts) DisableMissing(context.Context, string, map[string]struct{}) error {
	return context.Canceled
}

func TestSyncCatalogDisableError(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	svc := taocomputer.NewService(taocomputer.NewClient(srv.URL()), &fakeComputerService{}).WithProducts(disableFailProducts{})
	if _, err := svc.SyncCatalog(context.Background(), testTAOKey); err == nil {
		t.Fatal("expected disable error")
	}
}

func TestSyncCatalogServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	svc := taocomputer.NewService(taocomputer.NewClient(srv.URL), &fakeComputerService{}).WithProducts(&skipProducts{})
	if _, err := svc.SyncCatalog(context.Background(), testTAOKey); err == nil {
		t.Fatal("expected server error")
	}
}

func TestListCloudProductsLiveAndRegionIdentity(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(taocomputer.Catalog{Services: []taocomputer.CatalogService{{
			ID: "computer", Kind: "resource",
			Resources: []taocomputer.CatalogResource{
				{ID: "cvm-2c2g", Name: "CVM 2C2G", Description: "CVM 2C2G description", Region: "gz-cn", CPU: 2, MemoryMB: 2048, DiskGB: 50,
					BillingUnits: []string{"HOUR", "MONTH"}, Prices: map[string]int{"HOUR": 480, "MONTH": 286800},
					DurationTiers: []taocomputer.DurationTier{{Months: "1-2", Discount: 0.87}},
					Images:        []string{"tao-base-1.0"}, Enabled: true, AvailableSlots: 100},
				{ID: "cvm-2c2g", Region: "bj-cn", CPU: 2, MemoryMB: 2048, DiskGB: 50,
					BillingUnits: []string{"HOUR"}, Prices: map[string]int{"HOUR": 480},
					Images: []string{"tao-base-1.0"}, Enabled: true, AvailableSlots: 3},
				{ID: "disabled", Region: "gz-cn", Enabled: false, BillingUnits: []string{"HOUR"}},
			},
		}}})
	}))
	defer srv.Close()
	svc := taocomputer.NewService(taocomputer.NewClient(srv.URL), &fakeComputerService{})
	products, err := svc.ListCloudProducts(context.Background(), testTAOKey)
	if err != nil {
		t.Fatal(err)
	}
	if len(products) != 2 {
		t.Fatalf("products=%+v", products)
	}
	if products[0].ID == products[1].ID {
		t.Fatalf("region must participate in live id: %+v", products)
	}
	if products[0].Resource != "cvm-2c2g" || products[0].Source != "tao" || products[0].Policy != "paid" ||
		products[0].Health != "healthy" || products[0].Status != "enabled" {
		t.Fatalf("mapping=%+v", products[0])
	}
	if len(products[0].Images) != 1 || products[0].Prices["HOUR"] != 480 || products[0].Spec["disk_gb"] != 50 {
		t.Fatalf("live fields=%+v", products[0])
	}
	if products[0].Name != "CVM 2C2G" || products[0].Description != "CVM 2C2G description" {
		t.Fatalf("display fields=%+v", products[0])
	}
	if products[1].Name != "cvm-2c2g" || products[1].Description != "" {
		t.Fatalf("fallback display fields=%+v", products[1])
	}
}

func TestListCloudProductsServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	svc := taocomputer.NewService(taocomputer.NewClient(srv.URL), &fakeComputerService{})
	if _, err := svc.ListCloudProducts(context.Background(), testTAOKey); err == nil {
		t.Fatal("expected catalog error")
	}
}

func TestListCloudProductsEmptyAndClientErrors(t *testing.T) {
	if _, err := taocomputer.NewService(nil, &fakeComputerService{}).ListCloudProducts(context.Background(), testTAOKey); err == nil {
		t.Fatal("nil client should fail")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	svc := taocomputer.NewService(taocomputer.NewClient("http://example.test"), &fakeComputerService{})
	if _, err := svc.ListCloudProducts(ctx, testTAOKey); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled err=%v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(taocomputer.Catalog{Services: []taocomputer.CatalogService{{ID: "other", Kind: "resource"}, {ID: "computer", Kind: "other"}}})
	}))
	defer srv.Close()
	svc = taocomputer.NewService(taocomputer.NewClient(srv.URL), &fakeComputerService{})
	products, err := svc.ListCloudProducts(context.Background(), testTAOKey)
	if err != nil || len(products) != 0 {
		t.Fatalf("empty catalog=%+v err=%v", products, err)
	}
}
