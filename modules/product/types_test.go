package product

import (
	"context"
	"errors"
	"testing"
)

type fakeProvider struct {
	source string
	items  []Product
	err    error
}

func (f fakeProvider) Source() string { return f.source }
func (f fakeProvider) ListProducts(context.Context, ListRequest) ([]Product, error) {
	return f.items, f.err
}

func TestServiceRegistrationAndFiltering(t *testing.T) {
	svc := NewService()
	svc.Register(fakeProvider{source: "tao", items: []Product{
		{ID: "1", Source: "tao", Policy: "paid", Status: "enabled"},
		{ID: "2", Source: "tao", Policy: "free", Status: "enabled"},
		{ID: "3", Source: "tao", Policy: "paid", Status: "disabled"},
	}})
	if len(svc.Providers()) != 1 {
		t.Fatalf("providers=%v", svc.Providers())
	}
	items, err := svc.List(context.Background(), ListRequest{Source: "tao", Policy: "paid", Status: "enabled"})
	if err != nil || len(items) != 1 || items[0].ID != "1" {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	if _, err := svc.List(context.Background(), ListRequest{Source: "missing"}); !errors.Is(err, ErrSourceNotFound) {
		t.Fatalf("missing source err=%v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := svc.List(ctx, ListRequest{Source: "tao"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel err=%v", err)
	}
}

func TestSourceError(t *testing.T) {
	err := &SourceError{Status: 403, Code: "forbidden", Description: "denied"}
	if err.Error() != "denied" {
		t.Fatalf("error text=%q", err.Error())
	}
	if err.Status != 403 || err.Code != "forbidden" {
		t.Fatalf("err=%+v", err)
	}
}

func TestServiceNilAndProviderError(t *testing.T) {
	var nilSvc *Service
	if nilSvc.Providers() != nil {
		t.Fatal("nil service should have no providers")
	}
	svc := NewService()
	svc.Register(nil)
	svc.Register(fakeProvider{})
	if len(svc.Providers()) != 0 {
		t.Fatalf("empty provider registered: %v", svc.Providers())
	}
	wantErr := errors.New("provider down")
	svc.Register(fakeProvider{source: "x", err: wantErr})
	if _, err := svc.List(context.Background(), ListRequest{Source: "x"}); !errors.Is(err, wantErr) {
		t.Fatalf("provider err=%v", err)
	}
}

func TestSourceErrorFallbacks(t *testing.T) {
	if got := (&SourceError{Code: "c", Err: errors.New("underlying")}).Error(); got != "underlying" {
		t.Fatalf("error=%q", got)
	}
	if got := (&SourceError{Code: "c"}).Error(); got != "c" {
		t.Fatalf("error=%q", got)
	}
	if got := (*SourceError)(nil).Error(); got != "" {
		t.Fatalf("nil error=%q", got)
	}
}
