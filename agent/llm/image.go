package llm

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"time"

	"github.com/yaoapp/gou/connector"
	gouhttp "github.com/yaoapp/gou/http"
	goullm "github.com/yaoapp/gou/llm"
)

// ImageGenResponse holds the result of an image generation call.
// Image is always base64 encoded; if the provider returns a URL, it is downloaded and converted.
type ImageGenResponse struct {
	Image  string `json:"image"`  // base64 encoded image data
	Format string `json:"format"` // image format, e.g. "png", "jpeg"
}

// GenerateImage calls the /images/generations endpoint through the connector.
// options may include: size, n, quality, style, model, etc.
func GenerateImage(conn connector.Connector, prompt string, options map[string]interface{}) (*ImageGenResponse, error) {
	host, key, authMode := resolveConnSettings(conn)
	if host == "" {
		return nil, fmt.Errorf("no host found in connector settings")
	}
	if key == "" {
		return nil, fmt.Errorf("API key is not set")
	}

	if options == nil {
		options = map[string]interface{}{}
	}
	options["prompt"] = prompt
	if _, ok := options["model"]; !ok {
		if lc, ok := conn.(goullm.LLMConnector); ok {
			if m := lc.GetModel(); m != "" {
				options["model"] = m
			}
		}
	}

	url := connector.BuildAPIURL(host, "/images/generations")
	req := gouhttp.New(url)
	req.SetHeader("Content-Type", "application/json")
	setImageAuthHeaders(req, authMode, key)

	resp := req.Post(options)
	if resp.Status != 200 {
		errMsg := extractAPIError(resp.Data)
		return nil, fmt.Errorf("image generation failed (status %d, url %s): %s", resp.Status, url, errMsg)
	}

	return extractImageFromResponse(resp.Data)
}

func resolveConnSettings(conn connector.Connector) (host, key string, authMode goullm.AuthMode) {
	authMode = goullm.AuthBearer
	if lc, ok := conn.(goullm.LLMConnector); ok {
		host = lc.GetURL()
		key = lc.GetKey()
		authMode = lc.GetAuthMode()
	}
	if host == "" || key == "" {
		setting := conn.Setting()
		if host == "" {
			host, _ = setting["host"].(string)
		}
		if key == "" {
			key, _ = setting["key"].(string)
		}
	}
	return
}

func setImageAuthHeaders(req *gouhttp.Request, authMode goullm.AuthMode, key string) {
	switch authMode {
	case goullm.AuthAPIKey:
		req.SetHeader("api-key", key)
	case goullm.AuthXAPIKey:
		req.SetHeader("x-api-key", key)
	default:
		req.SetHeader("Authorization", fmt.Sprintf("Bearer %s", key))
	}
}

// EditImage calls the image editing endpoint through the connector.
// editFormat determines the API protocol: "multipart" for form-data POST /images/edits (OpenAI),
// "json" for JSON POST /images/generations with image field (Seedream).
// Empty editFormat defaults to "multipart" (the standard OpenAI protocol).
func EditImage(conn connector.Connector, imageInput string, prompt string, options map[string]interface{}, editFormat string) (*ImageGenResponse, error) {
	host, key, authMode := resolveConnSettings(conn)
	if host == "" {
		return nil, fmt.Errorf("no host found in connector settings")
	}
	if key == "" {
		return nil, fmt.Errorf("API key is not set")
	}

	model := ""
	if lc, ok := conn.(goullm.LLMConnector); ok {
		model = lc.GetModel()
	}
	if m, ok := options["model"].(string); ok && m != "" {
		model = m
	}
	size, _ := options["size"].(string)
	if size == "" {
		size = "1024x1024"
	}

	if editFormat == "json" {
		return editImageJSON(host, key, authMode, imageInput, prompt, model, size)
	}
	return editImageMultipart(host, key, authMode, imageInput, prompt, model, size)
}

// editImageMultipart sends multipart/form-data POST to /images/edits (OpenAI style).
func editImageMultipart(host, key string, authMode goullm.AuthMode, imageInput, prompt, model, size string) (*ImageGenResponse, error) {
	imageBytes, err := resolveImageBytes(imageInput)
	if err != nil {
		return nil, fmt.Errorf("resolve image: %w", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	mimeType := http.DetectContentType(imageBytes)
	if !strings.HasPrefix(mimeType, "image/") {
		mimeType = "image/png"
	}
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="image"; filename="image"`)
	h.Set("Content-Type", mimeType)
	part, err := writer.CreatePart(h)
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(imageBytes); err != nil {
		return nil, fmt.Errorf("write image data: %w", err)
	}

	writer.WriteField("prompt", prompt)
	if model != "" {
		writer.WriteField("model", model)
	}
	if size != "" {
		writer.WriteField("size", size)
	}
	writer.Close()

	url := connector.BuildAPIURL(host, "/images/edits")
	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	setHTTPAuthHeader(req, authMode, key)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("image edit failed (status %d, url %s): %s", resp.StatusCode, url, string(respBody))
	}

	var data interface{}
	if err := json.Unmarshal(respBody, &data); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return extractImageFromResponse(data)
}

// editImageJSON sends JSON POST to /images/generations with image field (Seedream style).
func editImageJSON(host, key string, authMode goullm.AuthMode, imageInput, prompt, model, size string) (*ImageGenResponse, error) {
	payload := map[string]interface{}{
		"prompt": prompt,
		"image":  imageInput,
		"size":   size,
	}
	if model != "" {
		payload["model"] = model
	}

	url := connector.BuildAPIURL(host, "/images/generations")
	req := gouhttp.New(url)
	req.SetHeader("Content-Type", "application/json")
	setImageAuthHeaders(req, authMode, key)

	resp := req.Post(payload)
	if resp.Status != 200 {
		errMsg := extractAPIError(resp.Data)
		return nil, fmt.Errorf("image edit failed (status %d, url %s): %s", resp.Status, url, errMsg)
	}
	return extractImageFromResponse(resp.Data)
}

// resolveImageBytes converts an image input (data URI, URL, or raw base64) into raw bytes.
func resolveImageBytes(input string) ([]byte, error) {
	if strings.HasPrefix(input, "data:") {
		idx := strings.Index(input, ",")
		if idx < 0 {
			return nil, fmt.Errorf("invalid data URI: no comma separator")
		}
		return base64.StdEncoding.DecodeString(input[idx+1:])
	}

	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Get(input)
		if err != nil {
			return nil, fmt.Errorf("download image: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("download returned status %d", resp.StatusCode)
		}
		return io.ReadAll(resp.Body)
	}

	return base64.StdEncoding.DecodeString(input)
}

// setHTTPAuthHeader sets auth headers on a stdlib http.Request (for multipart path).
func setHTTPAuthHeader(req *http.Request, authMode goullm.AuthMode, key string) {
	switch authMode {
	case goullm.AuthAPIKey:
		req.Header.Set("api-key", key)
	case goullm.AuthXAPIKey:
		req.Header.Set("x-api-key", key)
	default:
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))
	}
}

func extractImageFromResponse(data interface{}) (*ImageGenResponse, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal response: %w", err)
	}

	var parsed struct {
		Data []struct {
			B64JSON *string `json:"b64_json"`
			URL     *string `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	if len(parsed.Data) == 0 {
		return nil, fmt.Errorf("provider returned empty data array, no image was generated")
	}

	item := parsed.Data[0]

	if item.B64JSON != nil && *item.B64JSON != "" {
		return &ImageGenResponse{Image: *item.B64JSON, Format: "png"}, nil
	}

	if item.URL != nil && *item.URL != "" {
		b64, format, err := downloadImageAsBase64(*item.URL)
		if err != nil {
			return nil, fmt.Errorf("provider returned url but download failed: %w", err)
		}
		return &ImageGenResponse{Image: b64, Format: format}, nil
	}

	return nil, fmt.Errorf("provider returned data but neither b64_json nor url field is present, the model may not support image generation")
}

func downloadImageAsBase64(imageURL string) (b64 string, format string, err error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(imageURL)
	if err != nil {
		return "", "", fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("read body: %w", err)
	}
	if len(body) == 0 {
		return "", "", fmt.Errorf("downloaded image is empty")
	}

	format = "png"
	ct := resp.Header.Get("Content-Type")
	switch {
	case strings.Contains(ct, "jpeg") || strings.Contains(ct, "jpg"):
		format = "jpeg"
	case strings.Contains(ct, "webp"):
		format = "webp"
	case strings.Contains(ct, "gif"):
		format = "gif"
	default:
		if strings.Contains(imageURL, ".jpeg") || strings.Contains(imageURL, ".jpg") {
			format = "jpeg"
		} else if strings.Contains(imageURL, ".webp") {
			format = "webp"
		}
	}

	b64 = base64.StdEncoding.EncodeToString(body)
	return b64, format, nil
}

func extractAPIError(data interface{}) string {
	raw, err := json.Marshal(data)
	if err != nil {
		return fmt.Sprintf("%v", data)
	}

	var parsed struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &parsed); err == nil && parsed.Error.Message != "" {
		return parsed.Error.Message
	}
	return string(raw)
}
