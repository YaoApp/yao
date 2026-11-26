// Package product defines the product catalog contract independent of any
// upstream source (Tao, internal gateway, or future providers).
package product

import (
	"context"
	"errors"
	"sync"
)

// ErrSourceNotFound is returned when no provider is registered for a source.
var ErrSourceNotFound = errors.New("product source not found")

// SourceError is a provider failure that carries the HTTP status and stable
// error code the API should return. It keeps provider-specific errors out of
// the generic product service.
type SourceError struct {
	Status      int
	Code        string
	Description string
	Err         error
}

func (e *SourceError) Error() string {
	if e == nil {
		return ""
	}
	if e.Description != "" {
		return e.Description
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Code
}

func (e *SourceError) Unwrap() error { return e.Err }

// Principal is the authenticated caller context needed by a provider.
type Principal struct {
	UserID string
	TeamID string
}

// ListRequest is a normalized product list request.
type ListRequest struct {
	Source    string
	Policy    string
	Status    string
	Principal Principal
}

// DurationTier is a monthly duration discount tier.
type DurationTier struct {
	Months   string  `json:"months"`
	Discount float64 `json:"discount"`
}

// Product is the source-neutral product shape returned by /v1/products.
type Product struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	Scope          string         `json:"scope"`
	Source         string         `json:"source"`
	Platform       string         `json:"platform,omitempty"`
	Region         string         `json:"region,omitempty"`
	Resource       string         `json:"resource,omitempty"`
	Spec           map[string]any `json:"spec,omitempty"`
	Policy         string         `json:"policy"`
	TotalSlots     int            `json:"total_slots"`
	AvailableSlots int            `json:"available_slots"`
	UserQuota      int            `json:"user_quota,omitempty"`
	BillingUnits   []string       `json:"billing_units,omitempty"`
	Prices         map[string]int `json:"prices,omitempty"`
	DurationTiers  []DurationTier `json:"duration_tiers,omitempty"`
	Images         []string       `json:"images,omitempty"`
	PriceCredits   int            `json:"price_credits,omitempty"`
	Health         string         `json:"health"`
	Status         string         `json:"status"`
}

// Provider supplies products for one source.
type Provider interface {
	Source() string
	ListProducts(context.Context, ListRequest) ([]Product, error)
}

// Service registers product providers and dispatches list requests.
type Service struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

// NewService creates an empty product service.
func NewService() *Service { return &Service{providers: map[string]Provider{}} }

// Register registers one source provider. A later registration replaces the
// previous provider for the same source.
func (s *Service) Register(provider Provider) {
	if s == nil || provider == nil || provider.Source() == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.providers == nil {
		s.providers = map[string]Provider{}
	}
	s.providers[provider.Source()] = provider
}

// Providers returns the registered source names.
func (s *Service) Providers() []string {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.providers))
	for source := range s.providers {
		out = append(out, source)
	}
	return out
}

// List lists products from the requested source. It never falls back to a
// different source.
func (s *Service) List(ctx context.Context, req ListRequest) ([]Product, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	provider, ok := s.providers[req.Source]
	s.mu.RUnlock()
	if !ok {
		return nil, ErrSourceNotFound
	}
	items, err := provider.ListProducts(ctx, req)
	if err != nil {
		return nil, err
	}
	out := make([]Product, 0, len(items))
	for _, item := range items {
		if req.Policy != "" && item.Policy != req.Policy {
			continue
		}
		if req.Status != "" && item.Status != req.Status {
			continue
		}
		out = append(out, item)
	}
	return out, nil
}
