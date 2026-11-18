package user

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func installTaoUsageTestProvider(t *testing.T, issuer string) {
	t.Helper()
	old := providers["tao"]
	providers["tao"] = &Provider{ID: "tao", Issuer: issuer}
	t.Cleanup(func() { providers["tao"] = old })
}

func TestFetchTaoKeyBalance(t *testing.T) {
	t.Run("available balance", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "/v1/key", r.URL.Path)
			require.Equal(t, "Bearer sk-tao-test", r.Header.Get("Authorization"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"object":"key","valid":true,"balance":4052,"balance_available":true}`))
		}))
		defer server.Close()
		installTaoUsageTestProvider(t, server.URL)

		balance, available, err := fetchTaoKeyBalance(context.Background(), "sk-tao-test")
		require.NoError(t, err)
		require.True(t, available)
		require.NotNil(t, balance)
		require.Equal(t, int64(4052), *balance)
	})

	t.Run("balance unavailable is not zero", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"object":"key","valid":true,"balance_available":false}`))
		}))
		defer server.Close()
		installTaoUsageTestProvider(t, server.URL)

		balance, available, err := fetchTaoKeyBalance(context.Background(), "sk-tao-test")
		require.NoError(t, err)
		require.False(t, available)
		require.Nil(t, balance)
	})

	t.Run("non-200 is an error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		}))
		defer server.Close()
		installTaoUsageTestProvider(t, server.URL)

		_, available, err := fetchTaoKeyBalance(context.Background(), "sk-tao-bad")
		require.Error(t, err)
		require.False(t, available)
	})

	t.Run("malformed response is an error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`not-json`))
		}))
		defer server.Close()
		installTaoUsageTestProvider(t, server.URL)

		_, available, err := fetchTaoKeyBalance(context.Background(), "sk-tao-test")
		require.Error(t, err)
		require.False(t, available)
	})
}
