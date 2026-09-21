package ocr

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResolveTaoOCRType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Internal aliases → Tao native names
		{"general", "general_basic"},
		{"id_card", "idcard"},
		{"bank_card", "bankcard"},

		// Tao native names pass through
		{"general_basic", "general_basic"},
		{"accurate_basic", "accurate_basic"},
		{"table", "table"},
		{"handwriting", "handwriting"},
		{"idcard", "idcard"},
		{"bankcard", "bankcard"},

		// Unknown → default general_basic
		{"", "general_basic"},
		{"invoice", "general_basic"},
	}
	for _, tt := range tests {
		got := resolveTaoOCRType(tt.input)
		if got != tt.want {
			t.Errorf("resolveTaoOCRType(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTaoHandler_SupportedTypes(t *testing.T) {
	h := &TaoHandler{}
	types := h.SupportedTypes()
	// Tao native types
	for _, want := range []string{"general_basic", "accurate_basic", "table", "handwriting", "idcard", "bankcard"} {
		if !types[want] {
			t.Errorf("expected Tao native type %q in supported types", want)
		}
	}
	// Internal aliases
	for _, want := range []string{"general", "id_card", "bank_card"} {
		if !types[want] {
			t.Errorf("expected internal alias %q in supported types", want)
		}
	}
	if types["invoice"] {
		t.Error("invoice should not be in tao supported types")
	}
}

func TestTaoHandler_MissingCredentials(t *testing.T) {
	h := &TaoHandler{BaseURL: "", APIKey: ""}
	_, err := h.Recognize(context.Background(), &OCRRequest{Source: []byte("test")})
	if err == nil {
		t.Error("expected error for missing credentials")
	}
}

func TestTaoHandler_Recognize_Success(t *testing.T) {
	SetTaoOCRTypes(fallbackTaoTypes)
	t.Cleanup(InvalidateTaoOCRTypes)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/ocr/general_basic" {
			t.Errorf("expected /v1/ocr/general_basic, got %s", r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-key" {
			t.Errorf("expected Bearer test-key, got %s", auth)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if _, hasModel := body["model"]; hasModel {
			t.Error("body should not contain model field; Tao uses URL path for type")
		}
		if _, hasImage := body["image"]; !hasImage {
			t.Error("body should contain image field")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"words_result": []interface{}{
				map[string]interface{}{
					"words": "Hello World",
					"location": map[string]interface{}{
						"left": 10.0, "top": 20.0, "width": 100.0, "height": 30.0,
					},
					"probability": map[string]interface{}{"average": 0.95},
				},
				map[string]interface{}{
					"words": "Second line",
				},
			},
			"pages_number": 1.0,
		})
	}))
	defer srv.Close()

	h := &TaoHandler{BaseURL: srv.URL, APIKey: "test-key"}
	resp, err := h.Recognize(context.Background(), &OCRRequest{
		Source:   []byte("fake-image"),
		MimeType: "image/jpeg",
		Type:     "general",
		Mode:     "standard",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Text != "Hello World\nSecond line" {
		t.Errorf("text = %q, want %q", resp.Text, "Hello World\nSecond line")
	}
	if len(resp.Blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(resp.Blocks))
	}
	if resp.Blocks[0].Confidence != 0.95 {
		t.Errorf("block[0] confidence = %v, want 0.95", resp.Blocks[0].Confidence)
	}
	if len(resp.Blocks[0].BBox) != 4 {
		t.Fatalf("block[0] bbox len = %d, want 4", len(resp.Blocks[0].BBox))
	}
	if resp.Blocks[0].BBox[2] != 110.0 {
		t.Errorf("block[0] bbox[2] = %v, want 110.0 (left+width)", resp.Blocks[0].BBox[2])
	}
}

func TestTaoHandler_Recognize_IDCard(t *testing.T) {
	SetTaoOCRTypes(fallbackTaoTypes)
	t.Cleanup(InvalidateTaoOCRTypes)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ocr/idcard" {
			t.Errorf("expected /v1/ocr/idcard, got %s", r.URL.Path)
		}
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["id_card_side"] != "front" {
			t.Errorf("expected id_card_side=front, got %v", body["id_card_side"])
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"words_result": []interface{}{
				map[string]interface{}{"words": "张三"},
			},
		})
	}))
	defer srv.Close()

	h := &TaoHandler{BaseURL: srv.URL, APIKey: "test-key"}
	resp, err := h.Recognize(context.Background(), &OCRRequest{
		Source: []byte("fake-image"),
		Type:   "id_card",
		Extra:  map[string]interface{}{"id_card_side": "front"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Text != "张三" {
		t.Errorf("text = %q, want %q", resp.Text, "张三")
	}
}

func TestTaoHandler_Recognize_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	h := &TaoHandler{BaseURL: srv.URL, APIKey: "test-key"}
	_, err := h.Recognize(context.Background(), &OCRRequest{Source: []byte("test")})
	if err == nil {
		t.Error("expected error for server error")
	}
}

func TestTaoHandler_Recognize_APIError_String(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "invalid image format",
		})
	}))
	defer srv.Close()

	h := &TaoHandler{BaseURL: srv.URL, APIKey: "test-key"}
	_, err := h.Recognize(context.Background(), &OCRRequest{Source: []byte("test")})
	if err == nil {
		t.Error("expected error for API error response")
	}
}

func TestTaoHandler_Recognize_APIError_Object(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"message": "quota exceeded",
				"code":    429,
			},
		})
	}))
	defer srv.Close()

	h := &TaoHandler{BaseURL: srv.URL, APIKey: "test-key"}
	_, err := h.Recognize(context.Background(), &OCRRequest{Source: []byte("test")})
	if err == nil {
		t.Error("expected error for API error object response")
	}
}

func TestTaoHandler_Recognize_TextOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"text": "Direct text response",
		})
	}))
	defer srv.Close()

	h := &TaoHandler{BaseURL: srv.URL, APIKey: "test-key"}
	resp, err := h.Recognize(context.Background(), &OCRRequest{Source: []byte("test")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Text != "Direct text response" {
		t.Errorf("text = %q, want %q", resp.Text, "Direct text response")
	}
}

func TestFetchTaoOCRTypes_Success(t *testing.T) {
	InvalidateTaoOCRTypes()
	t.Cleanup(InvalidateTaoOCRTypes)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ocr" || r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"types": []string{"general_basic", "accurate_basic", "table", "handwriting", "idcard", "bankcard", "new_type"},
		})
	}))
	defer srv.Close()

	types := FetchTaoOCRTypes(srv.URL, "test-key")
	if types == nil {
		t.Fatal("expected non-nil types from fetch")
	}
	if !types["new_type"] {
		t.Error("expected dynamically fetched 'new_type' in cache")
	}

	// resolveTaoOCRType should now recognize the new type
	got := resolveTaoOCRType("new_type")
	if got != "new_type" {
		t.Errorf("resolveTaoOCRType(new_type) = %q, want new_type", got)
	}
}

func TestFetchTaoOCRTypes_Fallback(t *testing.T) {
	InvalidateTaoOCRTypes()
	t.Cleanup(InvalidateTaoOCRTypes)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	types := FetchTaoOCRTypes(srv.URL, "test-key")
	if types != nil {
		t.Error("expected nil return when API returns 404")
	}

	// Should fall back to hardcoded
	got := resolveTaoOCRType("general")
	if got != "general_basic" {
		t.Errorf("resolveTaoOCRType(general) = %q, want general_basic (fallback)", got)
	}
}

func TestSetTaoOCRTypes(t *testing.T) {
	InvalidateTaoOCRTypes()
	t.Cleanup(InvalidateTaoOCRTypes)

	custom := map[string]bool{"custom_type": true, "general_basic": true}
	SetTaoOCRTypes(custom)

	got := resolveTaoOCRType("custom_type")
	if got != "custom_type" {
		t.Errorf("resolveTaoOCRType(custom_type) = %q, want custom_type", got)
	}

	h := &TaoHandler{}
	supported := h.SupportedTypes()
	if !supported["custom_type"] {
		t.Error("SupportedTypes should include dynamically set custom_type")
	}
	if !supported["general"] {
		t.Error("SupportedTypes should still include internal alias 'general'")
	}
}

func TestParseTaoOCRResponse_InvalidJSON(t *testing.T) {
	_, err := parseTaoOCRResponse([]byte("not json"), "general_basic")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseTaoOCRResponse_EmptyWordsSkipped(t *testing.T) {
	data, _ := json.Marshal(map[string]interface{}{
		"words_result": []interface{}{
			map[string]interface{}{"words": ""},
			map[string]interface{}{"words": "valid"},
		},
	})
	resp, err := parseTaoOCRResponse(data, "general_basic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Blocks) != 1 {
		t.Errorf("expected 1 block (empty words skipped), got %d", len(resp.Blocks))
	}
}

func TestParseTaoOCRResponse_TableEmpty(t *testing.T) {
	data, _ := json.Marshal(map[string]interface{}{
		"table_num": 0.0,
	})
	resp, err := parseTaoOCRResponse(data, "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Text != "未检出表格" {
		t.Errorf("expected '未检出表格', got %q", resp.Text)
	}
}

func TestParseTaoOCRResponse_TableWithData(t *testing.T) {
	data, _ := json.Marshal(map[string]interface{}{
		"table_num": 1.0,
		"tables_result": []interface{}{
			map[string]interface{}{
				"body": []interface{}{
					map[string]interface{}{"words": "Name"},
					map[string]interface{}{"words": "Age"},
				},
			},
		},
	})
	resp, err := parseTaoOCRResponse(data, "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Text == "未检出表格" {
		t.Error("expected table content, got '未检出表格'")
	}
	if resp.Text == "" {
		t.Error("expected non-empty table text")
	}
}

func TestTaoHandler_IDCard_DefaultSide(t *testing.T) {
	SetTaoOCRTypes(fallbackTaoTypes)
	t.Cleanup(InvalidateTaoOCRTypes)

	var gotSide string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		gotSide, _ = body["id_card_side"].(string)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"words_result": []interface{}{
				map[string]interface{}{"words": "张三"},
			},
		})
	}))
	defer srv.Close()

	h := &TaoHandler{BaseURL: srv.URL, APIKey: "test-key"}
	_, err := h.Recognize(context.Background(), &OCRRequest{
		Source: []byte("fake"),
		Type:   "id_card",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotSide != "front" {
		t.Errorf("expected default id_card_side='front', got %q", gotSide)
	}
}

func TestTaoHandler_429_NoRetry(t *testing.T) {
	SetTaoOCRTypes(fallbackTaoTypes)
	t.Cleanup(InvalidateTaoOCRTypes)

	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("rate limited"))
	}))
	defer srv.Close()

	h := &TaoHandler{BaseURL: srv.URL, APIKey: "test-key"}
	_, err := h.Recognize(context.Background(), &OCRRequest{
		Source: []byte("fake"),
		Type:   "general",
	})
	if err == nil {
		t.Fatal("expected error on 429, got nil")
	}
	if attempts != 1 {
		t.Errorf("expected exactly 1 attempt (no retry), got %d", attempts)
	}
}

func TestCheckCardStatus_OtherTypeCard(t *testing.T) {
	data, _ := json.Marshal(map[string]interface{}{
		"image_status":     "other_type_card",
		"words_result_num": 0.0,
	})
	_, err := parseTaoOCRResponse(data, "idcard")
	if err == nil {
		t.Fatal("expected error for other_type_card")
	}
	if !strings.Contains(err.Error(), "card content mismatch") {
		t.Errorf("expected 'card content mismatch' in error, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "do not retry") {
		t.Errorf("expected 'do not retry' in error, got: %s", err.Error())
	}
}

func TestCheckCardStatus_ErrorCode216630(t *testing.T) {
	data, _ := json.Marshal(map[string]interface{}{
		"error_code": 216630.0,
		"error_msg":  "recognize bank card error",
	})
	_, err := parseTaoOCRResponse(data, "bankcard")
	if err == nil {
		t.Fatal("expected error for error_code 216630")
	}
	if !strings.Contains(err.Error(), "216630") {
		t.Errorf("expected error code in message, got: %s", err.Error())
	}
}

func TestCheckCardStatus_Normal(t *testing.T) {
	data, _ := json.Marshal(map[string]interface{}{
		"image_status":     "normal",
		"words_result_num": 5.0,
		"words_result": []interface{}{
			map[string]interface{}{"words": "张三"},
		},
	})
	resp, err := parseTaoOCRResponse(data, "idcard")
	if err != nil {
		t.Fatalf("unexpected error for normal card: %v", err)
	}
	if resp.Text != "张三" {
		t.Errorf("expected '张三', got %q", resp.Text)
	}
}
