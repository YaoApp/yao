package user

import (
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
)

func init() {
	gou.RegisterProcessHandler("xiang.user.Captcha", ProcessCaptcha)
	gou.RegisterProcessHandler("xiang.user.Login", ProcessLogin)
}

// ProcessLogin xiang.user.Login 用户登录
func ProcessLogin(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	payload := process.ArgsMap(0).Dot()
	log.With(log.F{"payload": payload}).Debug("ProcessLogin")

	id := any.Of(payload.Get("captcha.id")).CString()
	value := any.Of(payload.Get("captcha.code")).CString()
	if id == "" {
		exception.New("请输入验证码ID", 400).Ctx(maps.Map{"id": id, "code": value}).Throw()
	}
	if value == "" {
		exception.New("请输入验证码", 400).Ctx(maps.Map{"id": id, "code": value}).Throw()
	}
	if !ValidateCaptcha(id, value) {
		log.With(log.F{"id": id, "code": value}).Debug("ProcessLogin")
		exception.New("验证码不正确", 403).Ctx(maps.Map{"id": id, "code": value}).Throw()
		return nil
	}

	email := any.Of(payload.Get("email")).CString()
	mobile := any.Of(payload.Get("mobile")).CString()
	password := any.Of(payload.Get("password")).CString()
	if email != "" {
		return Auth("email", email, password)
	} else if mobile != "" {
		return Auth("mobile", mobile, password)
	}

	exception.New("参数错误", 400).Ctx(payload).Throw()
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
