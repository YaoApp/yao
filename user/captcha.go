package user

import (
	"errors"
	"image/color"

	"github.com/mojocn/base64Captcha"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/xlog"
)

var captchaStore = base64Captcha.DefaultMemStore

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
		Length:     4,
		Lang:       "zh",
		Background: "#FFFFFF",
	}
}

// MakeCaptcha 制作验证码
func MakeCaptcha(option CaptchaOption) (string, string) {

	if option.Width == 0 {
		option.Width = 240
	}

	if option.Height == 0 {
		option.Width = 80
	}

	if option.Length == 0 {
		option.Length = 4
	}

	if option.Lang == "" {
		option.Lang = "zh"
	}

	var driver base64Captcha.Driver
	switch option.Type {
	case "audio":
		driver = base64Captcha.NewDriverAudio(option.Length, option.Lang)
		break
	case "math":
		background := background(option.Background)
		driver = base64Captcha.NewDriverMath(
			option.Height, option.Width, 3,
			base64Captcha.OptionShowHollowLine, background,
			base64Captcha.DefaultEmbeddedFonts, []string{},
		)
		break
	default:
		driver = base64Captcha.NewDriverDigit(
			option.Height, option.Width, 5,
			0.7, 80,
		)
		break
	}

	c := base64Captcha.NewCaptcha(driver, captchaStore)
	id, content, err := c.Generate()
	if err != nil {
		exception.New("生成验证码出错 %s", 500, err).Throw()
	}

	// 打印日志
	if config.Conf.Mode == "debug" {
		xlog.Println("图形/音频验证码:", captchaStore.Get(id, false))
	}

	return id, content
}

// ValidateCaptcha 校验验证码
func ValidateCaptcha(id string, value string) bool {
	return captchaStore.Verify(id, value, true)
}

func background(s string) *color.RGBA {
	if s == "" {
		s = "#555555"
	}
	bg, err := parseHexColorFast(s)
	if err != nil {
		exception.New("背景色格式错误 %s", 400, s).Throw()
	}
	return &bg
}

func parseHexColorFast(s string) (c color.RGBA, err error) {
	c.A = 0xff

	if s[0] != '#' {
		return c, errors.New("invalid format")
	}

	hexToByte := func(b byte) byte {
		switch {
		case b >= '0' && b <= '9':
			return b - '0'
		case b >= 'a' && b <= 'f':
			return b - 'a' + 10
		case b >= 'A' && b <= 'F':
			return b - 'A' + 10
		}
		err = errors.New("invalid format")
		return 0
	}

	switch len(s) {
	case 7:
		c.R = hexToByte(s[1])<<4 + hexToByte(s[2])
		c.G = hexToByte(s[3])<<4 + hexToByte(s[4])
		c.B = hexToByte(s[5])<<4 + hexToByte(s[6])
	case 4:
		c.R = hexToByte(s[1]) * 17
		c.G = hexToByte(s[2]) * 17
		c.B = hexToByte(s[3]) * 17
	default:
		err = errors.New("invalid format")
	}
	return
}
