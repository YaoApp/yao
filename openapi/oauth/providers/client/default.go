package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// DefaultClient provides a default implementation of ClientProvider
type DefaultClient struct {
	prefix string
	cache  store.Store
	store  store.Store
}

// DefaultClientOptions provides options for the DefaultClient
type DefaultClientOptions struct {
	Prefix string
	Cache  store.Store
	Store  store.Store
}

// NewDefaultClient creates a new DefaultClient
func NewDefaultClient(options *DefaultClientOptions) (*DefaultClient, error) {
	if options == nil {
		return nil, types.ErrInvalidConfiguration
	}

	if options.Store == nil {
		return nil, types.ErrStoreMissing
	}

	if options.Prefix == "" {
		options.Prefix = "__yao:"
	}

	return &DefaultClient{
		prefix: options.Prefix,
		cache:  options.Cache,
		store:  options.Store,
	}, nil
}

// Key generation methods

func (c *DefaultClient) clientKey(clientID string) string {
	return fmt.Sprintf("%soauth:client:%s", c.prefix, clientID)
}

func (c *DefaultClient) clientListKey() string {
	return fmt.Sprintf("%soauth:clients", c.prefix)
}

// GetClientByID retrieves client information using a client ID
func (c *DefaultClient) GetClientByID(ctx context.Context, clientID string) (*types.ClientInfo, error) {
	// Try cache first if available
	if c.cache != nil {
		if cached, ok := c.cache.Get(c.clientKey(clientID)); ok {
			if clientInfo, ok := cached.(*types.ClientInfo); ok {
				return clientInfo, nil
			}
		}
	}

	// Fallback to store
	key := c.clientKey(clientID)
	data, ok := c.store.Get(key)
	if !ok {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidClient,
			ErrorDescription: "Client not found",
		}
	}

	var clientInfo *types.ClientInfo

	// Handle different data types returned by different stores
	switch v := data.(type) {
	case *types.ClientInfo:
		// Direct object (from cache)
		clientInfo = v
	case map[string]interface{}:
		// Map with JSON field names (standard format)
		jsonData, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal map data: %w", err)
		}
		clientInfo = &types.ClientInfo{}
		if err := json.Unmarshal(jsonData, clientInfo); err != nil {
			return nil, fmt.Errorf("failed to unmarshal client data: %w", err)
		}
	case []byte:
		// Byte data (for backward compatibility)
		clientInfo = &types.ClientInfo{}
		if err := json.Unmarshal(v, clientInfo); err != nil {
			return nil, fmt.Errorf("failed to unmarshal client data: %w", err)
		}
	case string:
		// String data (for backward compatibility)
		clientInfo = &types.ClientInfo{}
		if err := json.Unmarshal([]byte(v), clientInfo); err != nil {
			return nil, fmt.Errorf("failed to unmarshal client data: %w", err)
		}
	default:
		// Try JSON marshaling as fallback for unknown types
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal data to JSON: %w", err)
		}
		clientInfo = &types.ClientInfo{}
		if err := json.Unmarshal(jsonData, clientInfo); err != nil {
			return nil, fmt.Errorf("failed to unmarshal client data: %w", err)
		}
	}

	// Cache the result if cache is available
	if c.cache != nil {
		c.cache.Set(c.clientKey(clientID), clientInfo, 5*time.Minute) // Cache for 5 minutes
	}

	return clientInfo, nil
}

// GetClientByCredentials retrieves and validates client using client credentials
func (c *DefaultClient) GetClientByCredentials(ctx context.Context, clientID string, clientSecret string) (*types.ClientInfo, error) {
	clientInfo, err := c.GetClientByID(ctx, clientID)
	if err != nil {
		return nil, err
	}

	// For public clients, no secret validation required
	if clientInfo.ClientType == types.ClientTypePublic {
		return clientInfo, nil
	}

	// For confidential clients, validate secret
	if clientInfo.ClientSecret != clientSecret {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidClient,
			ErrorDescription: "Invalid client credentials",
		}
	}

	return clientInfo, nil
}

// CreateClient creates a new OAuth client and returns the client information
func (c *DefaultClient) CreateClient(ctx context.Context, clientInfo *types.ClientInfo) (*types.ClientInfo, error) {
	// Check for nil client info
	if clientInfo == nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidRequest,
			ErrorDescription: "Client information is required",
		}
	}

	// Validate required fields
	if clientInfo.ClientID == "" {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidRequest,
			ErrorDescription: "Client ID is required",
		}
	}

	// Check if client already exists
	existing, err := c.GetClientByID(ctx, clientInfo.ClientID)
	if err == nil && existing != nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidClient,
			ErrorDescription: "Client already exists",
		}
	}

	// Set timestamps
	now := time.Now()
	clientInfo.CreatedAt = now
	clientInfo.UpdatedAt = now

	// Set defaults
	if clientInfo.ClientType == "" {
		clientInfo.ClientType = types.ClientTypeConfidential
	}

	// Validate client
	validationResult, err := c.ValidateClient(ctx, clientInfo)
	if err != nil {
		return nil, err
	}
	if !validationResult.Valid {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidRequest,
			ErrorDescription: strings.Join(validationResult.Errors, "; "),
		}
	}

	// Save client data
	if err := c.saveClient(ctx, clientInfo); err != nil {
		return nil, err
	}

	// Add to client list
	if err := c.addToClientList(ctx, clientInfo.ClientID); err != nil {
		return nil, err
	}

	// Cache the client if cache is available
	if c.cache != nil {
		c.cache.Set(c.clientKey(clientInfo.ClientID), clientInfo, 5*time.Minute)
	}

	return clientInfo, nil
}

// UpdateClient updates an existing OAuth client configuration
func (c *DefaultClient) UpdateClient(ctx context.Context, clientID string, clientInfo *types.ClientInfo) (*types.ClientInfo, error) {
	// Check for nil client info
	if clientInfo == nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidRequest,
			ErrorDescription: "Client information is required",
		}
	}

	// Check if client exists
	existing, err := c.GetClientByID(ctx, clientID)
	if err != nil {
		return nil, err
	}

	// Update client ID if provided
	if clientInfo.ClientID != "" && clientInfo.ClientID != clientID {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidRequest,
			ErrorDescription: "Cannot change client ID",
		}
	}

	// Set client ID and preserve creation time
	clientInfo.ClientID = clientID
	clientInfo.CreatedAt = existing.CreatedAt
	clientInfo.UpdatedAt = time.Now()

	// Validate client
	validationResult, err := c.ValidateClient(ctx, clientInfo)
	if err != nil {
		return nil, err
	}
	if !validationResult.Valid {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidRequest,
			ErrorDescription: strings.Join(validationResult.Errors, "; "),
		}
	}

	// Save updated client data
	if err := c.saveClient(ctx, clientInfo); err != nil {
		return nil, err
	}

	// Update cache if available
	if c.cache != nil {
		c.cache.Set(c.clientKey(clientID), clientInfo, 5*time.Minute)
	}

	return clientInfo, nil
}

// DeleteClient removes an OAuth client from the system
func (c *DefaultClient) DeleteClient(ctx context.Context, clientID string) error {
	// Check if client exists
	_, err := c.GetClientByID(ctx, clientID)
	if err != nil {
		return err
	}

	// Remove from client list
	if err := c.removeFromClientList(ctx, clientID); err != nil {
		return err
	}

	// Delete client data
	key := c.clientKey(clientID)
	if err := c.store.Del(key); err != nil {
		return fmt.Errorf("failed to delete client: %w", err)
	}

	// Clear cache if available
	if c.cache != nil {
		c.cache.Del(c.clientKey(clientID))
	}

	return nil
}

// ValidateClient validates client information and configuration
func (c *DefaultClient) ValidateClient(ctx context.Context, clientInfo *types.ClientInfo) (*types.ValidationResult, error) {
	result := &types.ValidationResult{Valid: true}

	// Validate client ID
	if clientInfo.ClientID == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "Client ID is required")
	}

	// Validate client type
	if clientInfo.ClientType != types.ClientTypeConfidential &&
		clientInfo.ClientType != types.ClientTypePublic &&
		clientInfo.ClientType != types.ClientTypeCredentialed {
		result.Valid = false
		result.Errors = append(result.Errors, "Invalid client type")
	}

	// Validate client secret for confidential clients
	if clientInfo.ClientType == types.ClientTypeConfidential && clientInfo.ClientSecret == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "Client secret is required for confidential clients")
	}

	// Validate redirect URIs
	if len(clientInfo.RedirectURIs) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "At least one redirect URI is required")
	}

	// Validate grant types
	if len(clientInfo.GrantTypes) == 0 {
		clientInfo.GrantTypes = []string{types.GrantTypeAuthorizationCode}
	}

	// Validate response types
	if len(clientInfo.ResponseTypes) == 0 {
		clientInfo.ResponseTypes = []string{types.ResponseTypeCode}
	}

	return result, nil
}

// ListClients retrieves a list of clients with optional filtering
func (c *DefaultClient) ListClients(ctx context.Context, filters map[string]interface{}, limit int, offset int) ([]*types.ClientInfo, int, error) {
	// Get client list
	clientIDs, err := c.getClientList(ctx)
	if err != nil {
		return nil, 0, err
	}

	var clients []*types.ClientInfo

	// Load all clients
	for _, clientID := range clientIDs {
		client, err := c.GetClientByID(ctx, clientID)
		if err != nil {
			continue // Skip invalid clients
		}

		// Apply filters
		if c.matchesFilters(client, filters) {
			clients = append(clients, client)
		}
	}

	total := len(clients)

	// Apply pagination
	if offset > 0 {
		if offset >= len(clients) {
			return []*types.ClientInfo{}, total, nil
		}
		clients = clients[offset:]
	}

	if limit > 0 && len(clients) > limit {
		clients = clients[:limit]
	}

	return clients, total, nil
}

// ValidateRedirectURI validates if a redirect URI is registered for the client
func (c *DefaultClient) ValidateRedirectURI(ctx context.Context, clientID string, redirectURI string) (*types.ValidationResult, error) {
	client, err := c.GetClientByID(ctx, clientID)
	if err != nil {
		return nil, err
	}

	result := &types.ValidationResult{Valid: false}

	for _, uri := range client.RedirectURIs {
		if uri == redirectURI {
			result.Valid = true
			break
		}
	}

	if !result.Valid {
		result.Errors = append(result.Errors, "Redirect URI not registered for this client")
	}

	return result, nil
}

// ValidateScope validates if the client is authorized to request specific scopes
func (c *DefaultClient) ValidateScope(ctx context.Context, clientID string, scopes []string) (*types.ValidationResult, error) {
	client, err := c.GetClientByID(ctx, clientID)
	if err != nil {
		return nil, err
	}

	result := &types.ValidationResult{Valid: true}

	// If client has no scope restrictions, allow all scopes
	if client.Scope == "" {
		return result, nil
	}

	// Parse client allowed scopes
	allowedScopes := strings.Fields(client.Scope)
	allowedScopeMap := make(map[string]bool)
	for _, scope := range allowedScopes {
		allowedScopeMap[scope] = true
	}

	// Check each requested scope
	for _, scope := range scopes {
		if !allowedScopeMap[scope] {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Scope '%s' not allowed for this client", scope))
		}
	}

	return result, nil
}

// IsClientActive checks if a client is active and can be used for authentication
func (c *DefaultClient) IsClientActive(ctx context.Context, clientID string) (bool, error) {
	client, err := c.GetClientByID(ctx, clientID)
	if err != nil {
		return false, err
	}

	// For now, all existing clients are considered active
	// This can be extended to check additional status fields
	return client != nil, nil
}

// Helper methods

func (c *DefaultClient) saveClient(ctx context.Context, clientInfo *types.ClientInfo) error {
	key := c.clientKey(clientInfo.ClientID)

	// Convert to map[string]interface{} using JSON serialization to ensure consistent field names
	jsonData, err := json.Marshal(clientInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal client data: %w", err)
	}

	var clientMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &clientMap); err != nil {
		return fmt.Errorf("failed to unmarshal to map: %w", err)
	}

	// Store the map - this ensures JSON field names are used consistently
	if err := c.store.Set(key, clientMap, 0); err != nil {
		return fmt.Errorf("failed to save client: %w", err)
	}

	return nil
}

func (c *DefaultClient) getClientList(ctx context.Context) ([]string, error) {
	// Try cache first if available
	if c.cache != nil {
		if cached, ok := c.cache.Get(c.clientListKey()); ok {
			if clientIDs, ok := cached.([]string); ok {
				return clientIDs, nil
			}
		}
	}

	// Fallback to store
	key := c.clientListKey()
	data, ok := c.store.Get(key)
	if !ok {
		return []string{}, nil
	}

	var clientIDs []string

	// Handle different data types returned by different stores
	switch v := data.(type) {
	case []string:
		// Direct slice (from stores that preserve slice types)
		clientIDs = v
	case []interface{}:
		// Interface slice (from stores that decode to interface slices)
		for _, item := range v {
			if str, ok := item.(string); ok {
				clientIDs = append(clientIDs, str)
			}
		}
	case []byte:
		// Byte data (for backward compatibility)
		if err := json.Unmarshal(v, &clientIDs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal client list: %w", err)
		}
	case string:
		// String data (for backward compatibility)
		if err := json.Unmarshal([]byte(v), &clientIDs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal client list: %w", err)
		}
	default:
		// Handle MongoDB primitive types and other BSON types
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal data to JSON: %w", err)
		}
		if err := json.Unmarshal(jsonData, &clientIDs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal client list: %w", err)
		}
	}

	// Cache the result if cache is available
	if c.cache != nil {
		c.cache.Set(c.clientListKey(), clientIDs, 5*time.Minute)
	}

	return clientIDs, nil
}

func (c *DefaultClient) saveClientList(ctx context.Context, clientIDs []string) error {
	key := c.clientListKey()

	// Store the slice directly - this should work consistently across stores
	if err := c.store.Set(key, clientIDs, 0); err != nil {
		return fmt.Errorf("failed to save client list: %w", err)
	}

	// Update cache if available
	if c.cache != nil {
		c.cache.Set(c.clientListKey(), clientIDs, 5*time.Minute)
	}

	return nil
}

func (c *DefaultClient) addToClientList(ctx context.Context, clientID string) error {
	clientIDs, err := c.getClientList(ctx)
	if err != nil {
		return err
	}

	// Check if already exists
	for _, id := range clientIDs {
		if id == clientID {
			return nil // Already exists
		}
	}

	clientIDs = append(clientIDs, clientID)

	// Clear cache first to ensure consistency
	if c.cache != nil {
		c.cache.Del(c.clientListKey())
	}

	return c.saveClientList(ctx, clientIDs)
}

func (c *DefaultClient) removeFromClientList(ctx context.Context, clientID string) error {
	clientIDs, err := c.getClientList(ctx)
	if err != nil {
		return err
	}

	// Remove client ID
	var newClientIDs []string
	for _, id := range clientIDs {
		if id != clientID {
			newClientIDs = append(newClientIDs, id)
		}
	}

	// Clear cache first to ensure consistency
	if c.cache != nil {
		c.cache.Del(c.clientListKey())
	}

	return c.saveClientList(ctx, newClientIDs)
}

func (c *DefaultClient) matchesFilters(client *types.ClientInfo, filters map[string]interface{}) bool {
	if filters == nil {
		return true
	}

	for key, value := range filters {
		switch key {
		case "client_type":
			if client.ClientType != value.(string) {
				return false
			}
		case "client_name":
			if client.ClientName != value.(string) {
				return false
			}
		case "application_type":
			if client.ApplicationType != value.(string) {
				return false
			}
		}
	}

	return true
}
