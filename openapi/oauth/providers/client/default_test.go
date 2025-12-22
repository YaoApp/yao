package client

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/gou/store/lru"
	"github.com/yaoapp/gou/store/xun"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/test"
)

// Store configuration for parameterized tests
type StoreConfig struct {
	Name    string
	GetFunc func(*testing.T) store.Store
}

// Test helpers
func getMongoStore(t *testing.T) store.Store {
	// Skip test if MongoDB is not available
	host := os.Getenv("MONGO_TEST_HOST")
	if host == "" {
		t.Skip("MongoDB not available - set MONGO_TEST_HOST environment variable")
	}

	// Create MongoDB store using connector
	mongoConnector, err := connector.New("mongo", "oauth_test", []byte(`{
		"name": "OAuth Test MongoDB",
		"type": "mongo",
		"options": {
			"db": "oauth_test",
			"hosts": [{
				"host": "`+host+`",
				"port": "`+os.Getenv("MONGO_TEST_PORT")+`",
				"user": "`+os.Getenv("MONGO_TEST_USER")+`",
				"pass": "`+os.Getenv("MONGO_TEST_PASS")+`"
			}]
		}
	}`))
	require.NoError(t, err)

	mongoStore, err := store.New(mongoConnector, nil)
	require.NoError(t, err)

	return mongoStore
}

func getXunStore(t *testing.T) store.Store {
	// Use test.Prepare to initialize the environment (database, etc.)
	test.Prepare(t, config.Conf)

	// Create xun store using default database connection
	xunStore, err := xun.New(xun.Option{
		Table:     "__yao_oauth_client_test",
		Connector: "default",
		CacheSize: 1024,
	})
	require.NoError(t, err)

	// Clean up on test completion
	t.Cleanup(func() {
		xunStore.Clear()
		xunStore.Close()
		test.Clean()
	})

	return xunStore
}

func getLRUCache(t *testing.T) store.Store {
	cache, err := lru.New(1000)
	require.NoError(t, err)
	return cache
}

// Get all available store configurations
func getStoreConfigs() []StoreConfig {
	return []StoreConfig{
		{Name: "MongoDB", GetFunc: getMongoStore},
		{Name: "Xun", GetFunc: getXunStore},
	}
}

func createTestClient(clientID string) *types.ClientInfo {
	return &types.ClientInfo{
		ClientID:      clientID,
		ClientSecret:  "secret-" + clientID,
		ClientName:    "Test Client " + clientID,
		ClientType:    types.ClientTypeConfidential,
		RedirectURIs:  []string{"https://example.com/callback"},
		GrantTypes:    []string{types.GrantTypeAuthorizationCode},
		ResponseTypes: []string{types.ResponseTypeCode},
		Scope:         "openid profile email",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func TestNewDefaultClient(t *testing.T) {
	storeConfigs := getStoreConfigs()

	for _, config := range storeConfigs {
		t.Run(config.Name, func(t *testing.T) {
			t.Run("valid options", func(t *testing.T) {
				store := config.GetFunc(t)
				cache := getLRUCache(t)

				client, err := NewDefaultClient(&DefaultClientOptions{
					Prefix: "test:",
					Store:  store,
					Cache:  cache,
				})

				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.Equal(t, "test:", client.prefix)
				assert.Equal(t, store, client.store)
				assert.Equal(t, cache, client.cache)
			})

			t.Run("nil options", func(t *testing.T) {
				client, err := NewDefaultClient(nil)

				assert.Error(t, err)
				assert.Nil(t, client)
				assert.Equal(t, types.ErrInvalidConfiguration, err)
			})

			t.Run("nil store", func(t *testing.T) {
				client, err := NewDefaultClient(&DefaultClientOptions{
					Prefix: "test:",
					Store:  nil,
				})

				assert.Error(t, err)
				assert.Nil(t, client)
				assert.Equal(t, types.ErrStoreMissing, err)
			})

			t.Run("empty prefix uses default", func(t *testing.T) {
				store := config.GetFunc(t)

				client, err := NewDefaultClient(&DefaultClientOptions{
					Store: store,
				})

				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.Equal(t, "__yao:", client.prefix)
			})

			t.Run("without cache", func(t *testing.T) {
				store := config.GetFunc(t)

				client, err := NewDefaultClient(&DefaultClientOptions{
					Prefix: "test:",
					Store:  store,
				})

				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.Nil(t, client.cache)
			})
		})
	}
}

func TestKeyGeneration(t *testing.T) {
	storeConfigs := getStoreConfigs()

	for _, config := range storeConfigs {
		t.Run(config.Name, func(t *testing.T) {
			store := config.GetFunc(t)
			client, err := NewDefaultClient(&DefaultClientOptions{
				Prefix: "test:",
				Store:  store,
			})
			require.NoError(t, err)

			t.Run("client key", func(t *testing.T) {
				key := client.clientKey("test-client")
				expected := "test:oauth:client:test-client"
				assert.Equal(t, expected, key)
			})

			t.Run("client list key", func(t *testing.T) {
				key := client.clientListKey()
				expected := "test:oauth:clients"
				assert.Equal(t, expected, key)
			})
		})
	}
}

func TestCreateClient(t *testing.T) {
	storeConfigs := getStoreConfigs()

	for _, config := range storeConfigs {
		t.Run(config.Name, func(t *testing.T) {
			ctx := context.Background()

			t.Run("create client without cache", func(t *testing.T) {
				store := config.GetFunc(t)
				client, err := NewDefaultClient(&DefaultClientOptions{
					Prefix: "test1:",
					Store:  store,
				})
				require.NoError(t, err)

				// Clean up first
				client.store.Clear()

				testClient := createTestClient("test-client-1")

				created, err := client.CreateClient(ctx, testClient)
				assert.NoError(t, err)
				assert.NotNil(t, created)
				assert.Equal(t, testClient.ClientID, created.ClientID)
				assert.Equal(t, testClient.ClientSecret, created.ClientSecret)
				assert.NotZero(t, created.CreatedAt)
				assert.NotZero(t, created.UpdatedAt)

				// Verify client is in store
				retrieved, err := client.GetClientByID(ctx, testClient.ClientID)
				assert.NoError(t, err)
				assert.Equal(t, testClient.ClientID, retrieved.ClientID)
			})

			t.Run("create client with cache", func(t *testing.T) {
				store := config.GetFunc(t)
				cache := getLRUCache(t)
				client, err := NewDefaultClient(&DefaultClientOptions{
					Prefix: "test2:",
					Store:  store,
					Cache:  cache,
				})
				require.NoError(t, err)

				// Clean up first
				client.store.Clear()
				client.cache.Clear()

				testClient := createTestClient("test-client-2")

				created, err := client.CreateClient(ctx, testClient)
				assert.NoError(t, err)
				assert.NotNil(t, created)

				// Verify client is cached
				key := client.clientKey(testClient.ClientID)
				cached, ok := client.cache.Get(key)
				assert.True(t, ok)
				assert.NotNil(t, cached)
			})

			t.Run("create client with empty ID", func(t *testing.T) {
				store := config.GetFunc(t)
				client, err := NewDefaultClient(&DefaultClientOptions{
					Prefix: "test3:",
					Store:  store,
				})
				require.NoError(t, err)

				testClient := createTestClient("")

				created, err := client.CreateClient(ctx, testClient)
				assert.Error(t, err)
				assert.Nil(t, created)
				assert.Contains(t, err.Error(), "Client ID is required")
			})
		})
	}
}

func TestGetClientByID(t *testing.T) {
	storeConfigs := getStoreConfigs()

	for _, config := range storeConfigs {
		t.Run(config.Name, func(t *testing.T) {
			ctx := context.Background()

			t.Run("get client without cache", func(t *testing.T) {
				store := config.GetFunc(t)
				client, err := NewDefaultClient(&DefaultClientOptions{
					Prefix: "test4:",
					Store:  store,
				})
				require.NoError(t, err)

				// Clean up first
				client.store.Clear()

				testClient := createTestClient("test-client-4")

				// Create client first
				_, err = client.CreateClient(ctx, testClient)
				require.NoError(t, err)

				// Get client
				retrieved, err := client.GetClientByID(ctx, testClient.ClientID)
				assert.NoError(t, err)
				assert.NotNil(t, retrieved)
				assert.Equal(t, testClient.ClientID, retrieved.ClientID)
				assert.Equal(t, testClient.ClientSecret, retrieved.ClientSecret)
			})

			t.Run("get client with cache hit", func(t *testing.T) {
				store := config.GetFunc(t)
				cache := getLRUCache(t)
				client, err := NewDefaultClient(&DefaultClientOptions{
					Prefix: "test5:",
					Store:  store,
					Cache:  cache,
				})
				require.NoError(t, err)

				// Clean up first
				client.store.Clear()
				client.cache.Clear()

				testClient := createTestClient("test-client-5")

				// Create client first
				_, err = client.CreateClient(ctx, testClient)
				require.NoError(t, err)

				// Get client (should hit cache)
				retrieved, err := client.GetClientByID(ctx, testClient.ClientID)
				assert.NoError(t, err)
				assert.NotNil(t, retrieved)
				assert.Equal(t, testClient.ClientID, retrieved.ClientID)
			})

			t.Run("get non-existent client", func(t *testing.T) {
				store := config.GetFunc(t)
				client, err := NewDefaultClient(&DefaultClientOptions{
					Prefix: "test6:",
					Store:  store,
				})
				require.NoError(t, err)

				retrieved, err := client.GetClientByID(ctx, "non-existent")
				assert.Error(t, err)
				assert.Nil(t, retrieved)
				assert.Contains(t, err.Error(), "Client not found")
			})
		})
	}
}

func TestDeleteClient(t *testing.T) {
	storeConfigs := getStoreConfigs()

	for _, config := range storeConfigs {
		t.Run(config.Name, func(t *testing.T) {
			ctx := context.Background()

			t.Run("delete client with cache", func(t *testing.T) {
				store := config.GetFunc(t)
				cache := getLRUCache(t)
				client, err := NewDefaultClient(&DefaultClientOptions{
					Prefix: "test7:",
					Store:  store,
					Cache:  cache,
				})
				require.NoError(t, err)

				// Clean up first
				client.store.Clear()
				client.cache.Clear()

				testClient := createTestClient("test-client-7")

				// Create client first
				_, err = client.CreateClient(ctx, testClient)
				require.NoError(t, err)

				// Verify client is cached
				key := client.clientKey(testClient.ClientID)
				_, ok := client.cache.Get(key)
				assert.True(t, ok)

				// Delete client
				err = client.DeleteClient(ctx, testClient.ClientID)
				assert.NoError(t, err)

				// Verify cache is cleared
				_, ok = client.cache.Get(key)
				assert.False(t, ok)
			})

			t.Run("delete non-existent client", func(t *testing.T) {
				store := config.GetFunc(t)
				client, err := NewDefaultClient(&DefaultClientOptions{
					Prefix: "test8:",
					Store:  store,
				})
				require.NoError(t, err)

				err = client.DeleteClient(ctx, "non-existent")
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "Client not found")
			})
		})
	}
}

func TestValidateClient(t *testing.T) {
	storeConfigs := getStoreConfigs()

	for _, config := range storeConfigs {
		t.Run(config.Name, func(t *testing.T) {
			store := config.GetFunc(t)
			client, err := NewDefaultClient(&DefaultClientOptions{
				Prefix: "test9:",
				Store:  store,
			})
			require.NoError(t, err)

			ctx := context.Background()

			t.Run("valid client", func(t *testing.T) {
				testClient := createTestClient("test-client-9")

				result, err := client.ValidateClient(ctx, testClient)
				assert.NoError(t, err)
				assert.True(t, result.Valid)
				assert.Empty(t, result.Errors)
			})

			t.Run("client without ID", func(t *testing.T) {
				testClient := createTestClient("")

				result, err := client.ValidateClient(ctx, testClient)
				assert.NoError(t, err)
				assert.False(t, result.Valid)
				assert.Contains(t, result.Errors, "Client ID is required")
			})

			t.Run("client with invalid type", func(t *testing.T) {
				testClient := createTestClient("test-client-10")
				testClient.ClientType = "invalid"

				result, err := client.ValidateClient(ctx, testClient)
				assert.NoError(t, err)
				assert.False(t, result.Valid)
				assert.Contains(t, result.Errors, "Invalid client type")
			})
		})
	}
}

func TestCacheConsistency(t *testing.T) {
	storeConfigs := getStoreConfigs()

	for _, config := range storeConfigs {
		t.Run(config.Name, func(t *testing.T) {
			store := config.GetFunc(t)
			cache := getLRUCache(t)
			ctx := context.Background()

			client, err := NewDefaultClient(&DefaultClientOptions{
				Prefix: "test10:",
				Store:  store,
				Cache:  cache,
			})
			require.NoError(t, err)

			// Clean up first
			client.store.Clear()
			client.cache.Clear()

			testClient := createTestClient("test-client-10")

			// Create client
			_, err = client.CreateClient(ctx, testClient)
			require.NoError(t, err)

			// Verify cache is updated
			key := client.clientKey(testClient.ClientID)
			cached, ok := client.cache.Get(key)
			assert.True(t, ok)
			assert.NotNil(t, cached)

			// Update client
			updateData := &types.ClientInfo{
				ClientID:      testClient.ClientID,
				ClientSecret:  "updated-secret",
				ClientName:    "Updated Client",
				ClientType:    types.ClientTypeConfidential,
				RedirectURIs:  []string{"https://updated.com/callback"},
				GrantTypes:    []string{types.GrantTypeAuthorizationCode},
				ResponseTypes: []string{types.ResponseTypeCode},
				Scope:         "openid profile",
			}

			_, err = client.UpdateClient(ctx, testClient.ClientID, updateData)
			require.NoError(t, err)

			// Verify cache is updated
			cached, ok = client.cache.Get(key)
			assert.True(t, ok)
			cachedClient := cached.(*types.ClientInfo)
			assert.Equal(t, "Updated Client", cachedClient.ClientName)
			assert.Equal(t, "updated-secret", cachedClient.ClientSecret)

			// Delete client
			err = client.DeleteClient(ctx, testClient.ClientID)
			require.NoError(t, err)

			// Verify cache is cleared
			_, ok = client.cache.Get(key)
			assert.False(t, ok)
		})
	}
}
