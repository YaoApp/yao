package ocr

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// PaddleOCRHandler implements ProviderHandler for PaddleOCR / PaddleX Serving.
type PaddleOCRHandler struct {
	BaseURL string
	APIKey  string // optional authentication key
}

var paddleSupportedTypes = map[string]bool{
	"general":  true,
	"table":    true,
	"document": true,
}

func (h *PaddleOCRHandler) SupportedTypes() map[string]bool {
	return paddleSupportedTypes
}

func (h *PaddleOCRHandler) Recognize(ctx context.Context, req *OCRRequest) (*OCRResponse, error) {
	if h.BaseURL == "" {
		return nil, fmt.Errorf("PaddleOCR base_url is not configured")
	}

	pipeline := "PP-OCRv5"
	if p, ok := req.Extra["pipeline"].(string); ok && p != "" {
		pipeline = p
	}

	// Use PP-StructureV3 for table/document types or markdown output
	if req.Type == "table" || req.Type == "document" || req.OutputFormat == "markdown" {
		pipeline = "PP-StructureV3"
	}

	body := map[string]interface{}{
		"pipeline": pipeline,
	}
	b64 := base64.StdEncoding.EncodeToString(req.Source)
	if req.MimeType == "application/pdf" {
		body["pdf"] = b64
		body["fileType"] = 0
	} else {
		body["image"] = b64
		body["fileType"] = 1
	}

	// Pass through extra parameters
	for k, v := range req.Extra {
		if k != "pipeline" {
			body[k] = v
		}
	}

	reqBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := strings.TrimRight(h.BaseURL, "/") + "/ocr"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if h.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+h.APIKey)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("PaddleOCR request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("PaddleOCR returned HTTP %d: %s", httpResp.StatusCode, string(respBody))
	}

	return parsePaddleResponse(respBody, req.OutputFormat)
}

// parsePaddleResponse parses PaddleOCR / PaddleX Serving JSON response.
func parsePaddleResponse(data []byte, outputFormat string) (*OCRResponse, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse PaddleOCR response: %w", err)
	}

	resp := &OCRResponse{Pages: 1}

	// PaddleX returns results in "result" field
	result, _ := raw["result"].(map[string]interface{})
	if result == nil {
		// Fallback: entire response is the result
		result = raw
	}

	// Extract text blocks from rec_texts array
	recTexts, _ := result["rec_texts"].([]interface{})
	recScores, _ := result["rec_scores"].([]interface{})
	dtBoxes, _ := result["dt_polys"].([]interface{})

	var textParts []string
	for i, t := range recTexts {
		text, _ := t.(string)
		if text == "" {
			continue
		}
		textParts = append(textParts, text)

		block := OCRBlock{Text: text, Page: 1}
		if i < len(recScores) {
			if score, ok := recScores[i].(float64); ok {
				block.Confidence = score
			}
		}
		if i < len(dtBoxes) {
			block.BBox = extractPaddleBBox(dtBoxes[i])
		}
		resp.Blocks = append(resp.Blocks, block)
	}

	resp.Text = strings.Join(textParts, "\n")

	// If PP-StructureV3 returned markdown directly
	if md, ok := result["markdown"].(string); ok && md != "" {
		resp.Text = md
	}

	return resp, nil
}

// extractPaddleBBox converts PaddleOCR polygon points to normalized bbox.
// PaddleOCR returns dt_polys as [[x1,y1],[x2,y2],[x3,y3],[x4,y4]].
// We take the bounding rectangle and return [x_min, y_min, x_max, y_max].
// Note: coordinates are in pixel space; normalization requires image dimensions
// which are not available here, so we return raw pixel coordinates.
func extractPaddleBBox(poly interface{}) []float64 {
	points, ok := poly.([]interface{})
	if !ok || len(points) < 4 {
		return nil
	}

	var minX, minY, maxX, maxY float64
	for i, p := range points {
		pt, ok := p.([]interface{})
		if !ok || len(pt) < 2 {
			continue
		}
		x, _ := pt[0].(float64)
		y, _ := pt[1].(float64)
		if i == 0 {
			minX, minY, maxX, maxY = x, y, x, y
		} else {
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x > maxX {
				maxX = x
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	return []float64{minX, minY, maxX, maxY}
}
