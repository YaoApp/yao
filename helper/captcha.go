package helper

import (
	"bytes"
	"encoding/base64"
	"time"

	"github.com/dchest/captcha"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
)

var store = captcha.NewMemoryStore(1024, 10*time.Minute)

func init() {
	captcha.SetCustomStore(store)
}

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
			exception.New("make audio captcha error: %s", 500, err).Throw()
		}
		content := "data:audio/mp3;base64," + base64.StdEncoding.EncodeToString(buff.Bytes())
		log.Debug("ID:%s Audio Captcha:%s", id, toString(store.Get(id, false)))
		return id, content

	default:
		err := captcha.WriteImage(buff, id, option.Width, option.Height)
		if err != nil {
			exception.New("make image captcha error: %s", 500, err).Throw()
		}

		content := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buff.Bytes())
		log.Debug("ID:%s Image Captcha:%s", id, toString(store.Get(id, false)))
		return id, content
	}

}

// CaptchaValidate Validate the captcha
func CaptchaValidate(id string, code string) bool {
	return captcha.VerifyString(id, code)
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

func toString(digits []byte) string {
	var buf bytes.Buffer
	for _, d := range digits {
		buf.WriteByte(d + '0')
	}
	return buf.String()
}
