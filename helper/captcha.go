package helper

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
	utilscaptcha "github.com/yaoapp/yao/utils/captcha"
)

// CaptchaOption 验证码配置
type CaptchaOption struct {
	Type       string
	Height     int
	Width      int
	Length     int
	Lang       string
	Background string
}

// NewCaptchaOption 创建验证码配置
func NewCaptchaOption() CaptchaOption {
	return CaptchaOption{
		Width:      240,
		Height:     80,
		Length:     6,
		Lang:       "zh",
		Background: "#FFFFFF",
	}
}

// CaptchaMake 制作验证码
func CaptchaMake(option CaptchaOption) (string, string) {
	// Convert to utils captcha option
	utilsOption := utilscaptcha.Option{
		Type:       option.Type,
		Height:     option.Height,
		Width:      option.Width,
		Length:     option.Length,
		Lang:       option.Lang,
		Background: option.Background,
	}
	return utilscaptcha.Generate(utilsOption)
}

// CaptchaValidate Validate the captcha (image/audio)
func CaptchaValidate(id string, code string) bool {
	return utilscaptcha.Validate(id, code)
}

// CaptchaGet retrieves the captcha answer for testing purposes
// Returns empty string if captcha ID not found or expired
func CaptchaGet(id string) string {
	return utilscaptcha.Get(id)
}

// CaptchaValidateCloudflare validates a Cloudflare Turnstile token
// This function makes an HTTP request to Cloudflare's verification endpoint
//
// For testing, use Cloudflare's official test sitekeys:
// https://developers.cloudflare.com/turnstile/troubleshooting/testing/
func CaptchaValidateCloudflare(token, secret string) bool {
	return utilscaptcha.ValidateCloudflare(token, secret)
}

// ProcessCaptchaValidate xiang.helper.CaptchaValidate image/audio captcha
func ProcessCaptchaValidate(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	id := process.ArgsString(0)
	code := process.ArgsString(1)
	if code == "" {
		exception.New("Please enter the captcha.", 400).Throw()
		return false
	}
	if !CaptchaValidate(id, code) {
		exception.New("Invalid captcha.", 400).Throw()
		return false
	}
	return true
}

// ProcessCaptcha xiang.helper.Captcha image/audio captcha
func ProcessCaptcha(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	option := CaptchaOption{
		Width:      any.Of(process.ArgsURLValue(0, "width", "240")).CInt(),
		Height:     any.Of(process.ArgsURLValue(0, "height", "80")).CInt(),
		Length:     any.Of(process.ArgsURLValue(0, "length", "6")).CInt(),
		Type:       process.ArgsURLValue(0, "type", "math"),
		Background: process.ArgsURLValue(0, "background", "#FFFFFF"),
		Lang:       process.ArgsURLValue(0, "lang", "zh"),
	}
	id, content := CaptchaMake(option)
	return maps.Map{
		"id":      id,
		"content": content,
	}
}
