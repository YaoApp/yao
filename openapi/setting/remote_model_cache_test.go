package setting

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestFetchRemoteModels_CachesAfterFirstCall(t *testing.T) {
	var hits int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hits, 1)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "gpt-4o", "object": "model"},
			},
		})
	}))
	defer srv.Close()

	invalidateRemoteModelCache()

	models := fetchRemoteModels(srv.URL, "test-key")
	if len(models) == 0 {
		t.Fatal("expected models from first fetch, got none")
	}
	if atomic.LoadInt64(&hits) != 1 {
		t.Fatalf("expected 1 HTTP hit after first fetch, got %d", atomic.LoadInt64(&hits))
	}

	models2 := fetchRemoteModels(srv.URL, "test-key")
	if len(models2) == 0 {
		t.Fatal("expected models from cached fetch, got none")
	}
	if atomic.LoadInt64(&hits) != 1 {
		t.Fatalf("expected still 1 HTTP hit after second fetch (cache), got %d", atomic.LoadInt64(&hits))
	}
}

func TestFetchRemoteModels_InvalidateForcesRefetch(t *testing.T) {
	var hits int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hits, 1)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "gpt-4o", "object": "model"},
			},
		})
	}))
	defer srv.Close()

	invalidateRemoteModelCache()

	fetchRemoteModels(srv.URL, "test-key")
	if atomic.LoadInt64(&hits) != 1 {
		t.Fatalf("expected 1 HTTP hit, got %d", atomic.LoadInt64(&hits))
	}

	invalidateRemoteModelCache()

	fetchRemoteModels(srv.URL, "test-key")
	if atomic.LoadInt64(&hits) != 2 {
		t.Fatalf("expected 2 HTTP hits after invalidation, got %d", atomic.LoadInt64(&hits))
	}
}

func TestFetchRemoteModels_URLChangeForcesRefetch(t *testing.T) {
	var hits int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hits, 1)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "gpt-4o", "object": "model"},
			},
		})
	}))
	defer srv.Close()

	invalidateRemoteModelCache()

	fetchRemoteModels(srv.URL, "test-key")
	if atomic.LoadInt64(&hits) != 1 {
		t.Fatalf("expected 1 HTTP hit, got %d", atomic.LoadInt64(&hits))
	}

	fetchRemoteModels(srv.URL+"/other", "test-key")
	if atomic.LoadInt64(&hits) != 2 {
		t.Fatalf("expected 2 HTTP hits after URL change, got %d", atomic.LoadInt64(&hits))
	}
}
