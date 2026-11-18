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

func TestFetchTaoAccountBalanceWithToken(t *testing.T) {
	t.Run("account balance", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "/api/v1/users/me/balance", r.URL.Path)
			require.Equal(t, "Bearer tao-access", r.Header.Get("Authorization"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"balance":1000,"total_income":1000,"total_expense":0}`))
		}))
		defer server.Close()
		installTaoUsageTestProvider(t, server.URL)

		balance, err := fetchTaoAccountBalanceWithToken(context.Background(), "tao-access")
		require.NoError(t, err)
		require.Equal(t, int64(1000), balance)
	})

	t.Run("missing balance", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{}`))
		}))
		defer server.Close()
		installTaoUsageTestProvider(t, server.URL)

		_, err := fetchTaoAccountBalanceWithToken(context.Background(), "tao-access")
		require.Error(t, err)
	})

	t.Run("non-200", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "forbidden", http.StatusForbidden)
		}))
		defer server.Close()
		installTaoUsageTestProvider(t, server.URL)

		_, err := fetchTaoAccountBalanceWithToken(context.Background(), "tao-access")
		require.Error(t, err)
	})

	t.Run("malformed response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`not-json`))
		}))
		defer server.Close()
		installTaoUsageTestProvider(t, server.URL)

		_, err := fetchTaoAccountBalanceWithToken(context.Background(), "tao-access")
		require.Error(t, err)
	})
}
