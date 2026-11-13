package computer_test

import (
	"context"
	"encoding/json"
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
