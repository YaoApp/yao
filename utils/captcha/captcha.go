package captcha

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/dchest/captcha"
	"github.com/yaoapp/kun/log"
)

var store = captcha.NewMemoryStore(1024, 10*time.Minute)

func init() {
	captcha.SetCustomStore(store)
}

// Option 验证码配置
type Option struct {
	Type       string
	Height     int
	Width      int
	Length     int
	Lang       string
	Background string
}

// NewOption 创建验证码配置
func NewOption() Option {
	return Option{
		Width:      240,
		Height:     80,
		Length:     6,
		Lang:       "zh",
		Background: "#FFFFFF",
	}
}

// Generate 制作验证码
func Generate(option Option) (string, string) {
	if option.Width == 0 {
		option.Width = 240
	}

	if option.Height == 0 {
		option.Width = 80
	}

	if option.Length == 0 {
		option.Length = 6
	}

	if option.Lang == "" {
		option.Lang = "zh"
	}

	id := captcha.NewLen(option.Length)
	var data []byte
	var buff = bytes.NewBuffer(data)
	switch option.Type {

	case "audio":
		err := captcha.WriteAudio(buff, id, option.Lang)
		if err != nil {
			log.Error("make audio captcha error: %s", err)
			return "", ""
		}
		content := "data:audio/mp3;base64," + base64.StdEncoding.EncodeToString(buff.Bytes())
		log.Debug("ID:%s Audio Captcha:%s", id, toString(store.Get(id, false)))
		return id, content

	default:
		err := captcha.WriteImage(buff, id, option.Width, option.Height)
		if err != nil {
			log.Error("make image captcha error: %s", err)
			return "", ""
		}

		content := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buff.Bytes())
		log.Debug("ID:%s Image Captcha:%s", id, toString(store.Get(id, false)))
		return id, content
	}
}

// Validate validates the captcha (image/audio)
func Validate(id string, code string) bool {
	return captcha.VerifyString(id, code)
}

// Get retrieves the captcha answer for testing purposes
// Returns empty string if captcha ID not found or expired
func Get(id string) string {
	digits := store.Get(id, false)
	if digits == nil {
		return ""
	}
	return toString(digits)
}

// ValidateCloudflare validates a Cloudflare Turnstile token
// This function makes an HTTP request to Cloudflare's verification endpoint
//
// For testing, use Cloudflare's official test sitekeys:
// https://developers.cloudflare.com/turnstile/troubleshooting/testing/
func ValidateCloudflare(token, secret string) bool {
	if token == "" || secret == "" {
		return false
	}

	// Cloudflare Turnstile verification endpoint
	verifyURL := "https://challenges.cloudflare.com/turnstile/v0/siteverify"

	// Prepare request body
	requestBody := map[string]string{
		"secret":   secret,
		"response": token,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		log.Error("Failed to marshal Turnstile request: %v", err)
		return false
	}

	// Make HTTP POST request
	resp, err := http.Post(verifyURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Error("Failed to verify Turnstile token: %v", err)
		return false
	}
	defer resp.Body.Close()

	// Parse response
	var result struct {
		Success    bool     `json:"success"`
		ErrorCodes []string `json:"error-codes,omitempty"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		log.Error("Failed to parse Turnstile response: %v", err)
		return false
	}

	if !result.Success && len(result.ErrorCodes) > 0 {
		log.Warn("Turnstile verification failed: %v", result.ErrorCodes)
	}

	return result.Success
}

func toString(digits []byte) string {
	var buf bytes.Buffer
	for _, d := range digits {
		buf.WriteByte(d + '0')
	}
	return buf.String()
}
