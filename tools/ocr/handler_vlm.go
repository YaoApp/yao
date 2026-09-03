package ocr

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	agentCtx "github.com/yaoapp/yao/agent/context"
	agentLLM "github.com/yaoapp/yao/agent/llm"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
)

// VLMHandler implements ProviderHandler for VLM-OCR models via LLM connectors.
type VLMHandler struct {
	ConnectorID string
	AuthInfo    *oauthTypes.AuthorizedInfo
}

// vlmSupportedTypes — VLM models handle all types through prompt engineering.
var vlmSupportedTypes = map[string]bool{
	"general":         true,
	"table":           true,
	"handwriting":     true,
	"invoice":         true,
	"receipt":         true,
	"id_card":         true,
	"bank_card":       true,
	"license":         true,
	"vehicle_license": true,
	"passport":        true,
	"license_plate":   true,
	"document":        true,
}

func (h *VLMHandler) SupportedTypes() map[string]bool {
	return vlmSupportedTypes
}

func (h *VLMHandler) Recognize(ctx context.Context, req *OCRRequest) (*OCRResponse, error) {
	if req.MimeType == "application/pdf" {
		return nil, fmt.Errorf("VLM-OCR does not support PDF input; use Baidu, Azure, or PaddleOCR")
	}
	conn, caps, err := agentLLM.ResolveConnector(h.ConnectorID, h.AuthInfo)
	if err != nil {
		return nil, fmt.Errorf("resolve OCR connector %q: %w", h.ConnectorID, err)
	}

	// Build system prompt for OCR task, append user custom prompt if present
	systemPrompt := buildOCRSystemPrompt(req.Type, req.OutputFormat, req.Mode, req.Language)
	if req.Prompt != "" {
		systemPrompt += "\n\n" + req.Prompt
	}

	// Encode image as base64 data URI for vision API
	dataURI := "data:" + req.MimeType + ";base64," + base64.StdEncoding.EncodeToString(req.Source)

	userText := "请识别图片中的所有文字内容。"

	messages := []agentCtx.Message{
		{Role: "system", Content: systemPrompt},
		{
			Role: "user",
			Content: []agentCtx.ContentPart{
				{Type: agentCtx.ContentImageURL, ImageURL: &agentCtx.ImageURL{URL: dataURI}},
				{Type: agentCtx.ContentText, Text: userText},
			},
		},
	}

	opts := &agentCtx.CompletionOptions{Capabilities: caps}
	instance, err := agentLLM.New(conn, opts)
	if err != nil {
		return nil, fmt.Errorf("create LLM instance: %w", err)
	}

	chatID := agentCtx.GenChatID()
	agCtx := agentCtx.New(ctx, h.AuthInfo, chatID)
	defer agCtx.Release()

	resp, err := instance.Post(agCtx, messages, opts)
	if err != nil {
		return nil, fmt.Errorf("VLM-OCR call: %w", err)
	}

	text := extractVLMText(resp.Content)
	ocrResp := &OCRResponse{
		Text:  text,
		Pages: 1,
		Metadata: map[string]string{
			"model": resp.Model,
		},
	}

	// Best-effort JSON parse when output_format=json
	if req.OutputFormat == "json" {
		parseVLMJSON(text, ocrResp)
	}

	return ocrResp, nil
}

// parseVLMJSON attempts to parse the VLM text response as JSON and populate Fields/Blocks.
// LLM output may be wrapped in ```json code fences; this is stripped before parsing.
// On parse failure the response is left as-is with text only.
func parseVLMJSON(text string, resp *OCRResponse) {
	cleaned := stripCodeFence(text)
	var parsed map[string]interface{}
	if json.Unmarshal([]byte(cleaned), &parsed) != nil {
		return
	}

	if fields, ok := parsed["fields"].(map[string]interface{}); ok && len(fields) > 0 {
		resp.Fields = fields
	}
	if t, ok := parsed["text"].(string); ok && t != "" {
		resp.Text = t
	}
	if blocks, ok := parsed["blocks"].([]interface{}); ok && len(blocks) > 0 {
		for _, b := range blocks {
			if bm, ok := b.(map[string]interface{}); ok {
				block := OCRBlock{}
				if t, ok := bm["text"].(string); ok {
					block.Text = t
				}
				if c, ok := bm["confidence"].(float64); ok {
					block.Confidence = c
				}
				if p, ok := bm["page"].(float64); ok {
					block.Page = int(p)
				}
				resp.Blocks = append(resp.Blocks, block)
			}
		}
	}
}

// stripCodeFence removes ```json ... ``` code fences from LLM output.
func stripCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		// Remove opening fence (```json or ```)
		if idx := strings.Index(s, "\n"); idx != -1 {
			s = s[idx+1:]
		}
		// Remove closing fence
		if idx := strings.LastIndex(s, "```"); idx != -1 {
			s = s[:idx]
		}
		s = strings.TrimSpace(s)
	}
	return s
}

// extractVLMText extracts text content from the LLM response.
func extractVLMText(content interface{}) string {
	if s, ok := content.(string); ok {
		return s
	}
	if parts, ok := content.([]agentCtx.ContentPart); ok {
		for _, p := range parts {
			if p.Type == agentCtx.ContentText {
				return p.Text
			}
		}
	}
	return fmt.Sprintf("%v", content)
}
