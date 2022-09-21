package utils

import "github.com/yaoapp/gou"

func init() {
	gou.AliasProcess("xiang.helper.Captcha", "yao.utils.Captcha")
	gou.AliasProcess("xiang.helper.CaptchaValidate", "yao.utils.CaptchaValidate")
}
