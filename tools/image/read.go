package vision

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"

	goufs "github.com/yaoapp/gou/fs"
	"github.com/yaoapp/gou/process"
	agentCtx "github.com/yaoapp/yao/agent/context"
	agentLLM "github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/attachment"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	ws "github.com/yaoapp/yao/workspace"
)

//go:embed schema.json
var SchemaJSON []byte

// ImageReadResponse is the return type for image_read.
type ImageReadResponse struct {
	Content string `json:"content"`
	Model   string `json:"model"`
}

// ReadImage reads and analyzes an image using the vision model.
// It resolves the image from src, finds a vision-capable connector via llmprovider,
// and returns the model's text description.
func ReadImage(goCtx context.Context, src string, prompt string, maxSize int, authInfo *oauthTypes.AuthorizedInfo) (*ImageReadResponse, error) {
	if prompt == "" {
		prompt = "Please describe this image in detail."
	}
	if maxSize <= 0 {
		maxSize = 1080
	}

	imageURI, err := resolveImage(src, maxSize)
	if err != nil {
		return nil, fmt.Errorf("resolve image: %w", err)
	}

	conn, caps, err := agentLLM.ResolveConnector("use::vision", authInfo)
	if err != nil {
		return nil, fmt.Errorf("resolve vision connector: %w", err)
	}

	opts := &agentCtx.CompletionOptions{Capabilities: caps}
	instance, err := agentLLM.New(conn, opts)
	if err != nil {
		return nil, fmt.Errorf("create LLM instance: %w", err)
	}

	messages := []agentCtx.Message{{
		Role: "user",
		Content: []agentCtx.ContentPart{
			{Type: agentCtx.ContentImageURL, ImageURL: &agentCtx.ImageURL{URL: imageURI}},
			{Type: agentCtx.ContentText, Text: prompt},
		},
	}}

	chatID := agentCtx.GenChatID()
	ctx := agentCtx.New(goCtx, authInfo, chatID)
	defer ctx.Release()

	resp, err := instance.Post(ctx, messages, opts)
	if err != nil {
		return nil, fmt.Errorf("vision model call: %w", err)
	}

	return &ImageReadResponse{
		Content: extractTextContent(resp.Content),
		Model:   resp.Model,
	}, nil
}

// Handler is the tools.image_read process handler.
func Handler(proc *process.Process) interface{} {
	src := proc.ArgsString(0)
	if src == "" {
		return map[string]interface{}{"error": "image_path is required: provide a file path, URL, or URI"}
	}

	authInfo := authorized.ProcessAuthInfo(proc)
	if authInfo == nil {
		return map[string]interface{}{"error": "unauthorized: no auth info in request"}
	}

	resp, err := ReadImage(proc.Context, src,
		proc.ArgsString(1, "Please describe this image in detail."),
		proc.ArgsInt(2, 1080),
		authInfo,
	)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return resp
}

// resolveImage reads image bytes from any supported source, resizes, and returns a data URI.
func resolveImage(src string, maxSize int) (string, error) {
	raw, err := readBytes(src)
	if err != nil {
		return "", err
	}

	data, mime := resizeImage(raw, maxSize)

	if len(data) > 3<<20 {
		return "", fmt.Errorf("compressed image still exceeds 3MB, try a smaller max_size (current: %d)", maxSize)
	}

	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

// readBytes dispatches to the appropriate reader based on URI scheme.
func readBytes(src string) ([]byte, error) {
	switch {
	case strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://"):
		return httpGet(src)

	case strings.HasPrefix(src, "workspace://"):
		rest := strings.TrimPrefix(src, "workspace://")
		idx := strings.Index(rest, "/")
		if idx < 0 {
			return nil, fmt.Errorf("invalid workspace URI: %s", src)
		}
		return ws.M().ReadFile(context.Background(), rest[:idx], rest[idx+1:])

	case strings.HasPrefix(src, "attach://"):
		rest := strings.TrimPrefix(src, "attach://")
		idx := strings.Index(rest, "/")
		if idx < 0 {
			return nil, fmt.Errorf("invalid attach URI: %s", src)
		}
		mgr, ok := attachment.Managers[rest[:idx]]
		if !ok {
			return nil, fmt.Errorf("uploader not found: %s", rest[:idx])
		}
		return mgr.Read(context.Background(), rest[idx+1:])

	case strings.HasPrefix(src, "yao://"):
		dataFS, err := goufs.Get("data")
		if err != nil {
			return nil, err
		}
		return dataFS.ReadFile(strings.TrimPrefix(src, "yao://"))

	case strings.HasPrefix(src, "data:"):
		return decodeDataURI(src)

	default:
		return nil, fmt.Errorf("unsupported image source: %s", src)
	}
}

func httpGet(rawURL string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: %d", rawURL, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 20<<20))
}

// resizeImage decodes, resizes (longest edge <= maxSize), re-encodes as JPEG.
// Returns original bytes unchanged when already small enough or if decode fails.
func resizeImage(data []byte, maxSize int) ([]byte, string) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data, http.DetectContentType(data)
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= maxSize && h <= maxSize {
		return data, http.DetectContentType(data)
	}
	ratio := float64(maxSize) / float64(max(w, h))
	newW, newH := int(float64(w)*ratio), int(float64(h)*ratio)
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.BiLinear.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 85}); err != nil {
		return data, http.DetectContentType(data)
	}
	return buf.Bytes(), "image/jpeg"
}

func decodeDataURI(uri string) ([]byte, error) {
	parts := strings.SplitN(uri, ",", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid data URI format")
	}
	return base64.StdEncoding.DecodeString(parts[1])
}

func extractTextContent(content interface{}) string {
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
