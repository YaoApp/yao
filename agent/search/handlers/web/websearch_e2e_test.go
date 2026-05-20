//go:build e2e

package web_test

// Serper E2E tests temporarily disabled.

// func TestSerperWebSearch_E2E(t *testing.T) {
// 	testprepare.PrepareE2E(t)
//
// 	apiKey := os.Getenv("SERPER_API_KEY")
// 	if apiKey == "" {
// 		t.Fatal("SERPER_API_KEY is required for E2E web search test")
// 	}
//
// 	payload, err := json.Marshal(map[string]interface{}{
// 		"q":   "Yao App Engine low-code platform",
// 		"num": 5,
// 	})
// 	require.NoError(t, err)
//
// 	req, err := http.NewRequest("POST", "https://google.serper.dev/search", bytes.NewReader(payload))
// 	require.NoError(t, err)
// 	req.Header.Set("X-API-KEY", apiKey)
// 	req.Header.Set("Content-Type", "application/json")
//
// 	client := &http.Client{Timeout: 30 * time.Second}
// 	resp, err := client.Do(req)
// 	require.NoError(t, err)
// 	defer resp.Body.Close()
//
// 	body, err := io.ReadAll(resp.Body)
// 	require.NoError(t, err)
// 	require.Equal(t, http.StatusOK, resp.StatusCode, "Serper API returned non-200: %s", string(body))
//
// 	var result struct {
// 		Organic []struct {
// 			Title   string `json:"title"`
// 			Link    string `json:"link"`
// 			Snippet string `json:"snippet"`
// 		} `json:"organic"`
// 	}
// 	require.NoError(t, json.Unmarshal(body, &result))
// 	assert.NotEmpty(t, result.Organic, "Should have organic search results")
//
// 	for _, item := range result.Organic {
// 		assert.NotEmpty(t, item.Title)
// 		assert.NotEmpty(t, item.Link)
// 		t.Logf("  %s - %s", item.Title, item.Link)
// 	}
// 	t.Logf("Serper search returned %d organic results", len(result.Organic))
// }

// func TestSerperWebSearch_WithLimit_E2E(t *testing.T) {
// 	testprepare.PrepareE2E(t)
//
// 	apiKey := os.Getenv("SERPER_API_KEY")
// 	if apiKey == "" {
// 		t.Fatal("SERPER_API_KEY is required for E2E web search test")
// 	}
//
// 	payload, err := json.Marshal(map[string]interface{}{
// 		"q":   "Go programming language",
// 		"num": 3,
// 	})
// 	require.NoError(t, err)
//
// 	req, err := http.NewRequest("POST", "https://google.serper.dev/search", bytes.NewReader(payload))
// 	require.NoError(t, err)
// 	req.Header.Set("X-API-KEY", apiKey)
// 	req.Header.Set("Content-Type", "application/json")
//
// 	client := &http.Client{Timeout: 30 * time.Second}
// 	resp, err := client.Do(req)
// 	require.NoError(t, err)
// 	defer resp.Body.Close()
//
// 	body, err := io.ReadAll(resp.Body)
// 	require.NoError(t, err)
// 	require.Equal(t, http.StatusOK, resp.StatusCode)
//
// 	var result struct {
// 		Organic []struct {
// 			Title string `json:"title"`
// 			Link  string `json:"link"`
// 		} `json:"organic"`
// 	}
// 	require.NoError(t, json.Unmarshal(body, &result))
// 	assert.NotEmpty(t, result.Organic)
// 	assert.LessOrEqual(t, len(result.Organic), 3, "Should respect limit")
// 	t.Logf("Limited serper search returned %d results", len(result.Organic))
// }
