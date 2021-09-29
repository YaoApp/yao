package user

import (
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/maps"
)

func init() {
	// 注册处理器
	gou.RegisterProcessHandler("xiang.user.Captcha", ProcessCaptcha)
}

// ProcessLogin xiang.user.Login 用户登录
func ProcessLogin(process *gou.Process) interface{} {
	return nil
}

// ProcessCaptcha xiang.user.Captcha 验证码
func ProcessCaptcha(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	option := CaptchaOption{
		Width:      any.Of(process.ArgsURLValue(0, "width", "240")).CInt(),
		Height:     any.Of(process.ArgsURLValue(0, "height", "80")).CInt(),
		Length:     any.Of(process.ArgsURLValue(0, "height", "4")).CInt(),
		Type:       process.ArgsURLValue(0, "type", "math"),
		Background: process.ArgsURLValue(0, "background", "#FFFFFF"),
		Lang:       process.ArgsURLValue(0, "lang", "zh"),
	}
	id, content := MakeCaptcha(option)
	return maps.Map{
		"id":      id,
		"content": content,
	}
}

// ProcessToken xiang.user.Token 使用 Key & Secret 换取 Token
func ProcessToken(process *gou.Process) interface{} {
	return nil
}

// ProcessTokenRefresh xiang.user.TokenRefresh  刷新Token
func ProcessTokenRefresh(process *gou.Process) interface{} {
	return nil
}

// ProcessInfo xiang.user.Info 读取当前用户资料
func ProcessInfo(process *gou.Process) interface{} {
	return nil
}
