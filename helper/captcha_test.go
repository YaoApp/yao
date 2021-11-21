package helper

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/maps"
)

func TestCaptcha(t *testing.T) {
	id, content := CaptchaMake(CaptchaOption{
		Type:   "audio",
		Width:  240,
		Height: 80,
		Length: 4,
		Lang:   "zh",
	})
	assert.IsType(t, "string", id)
	assert.IsType(t, "string", content)
	captchaStore.Get(id, false)
	assert.True(t, CaptchaValidate(id, captchaStore.Get(id, false)))

	id, content = CaptchaMake(CaptchaOption{
		Type:   "math",
		Width:  240,
		Height: 80,
		Length: 4,
		Lang:   "zh",
	})
	assert.IsType(t, "string", id)
	assert.IsType(t, "string", content)
	captchaStore.Get(id, false)
	assert.True(t, CaptchaValidate(id, captchaStore.Get(id, false)))

	id, content = CaptchaMake(CaptchaOption{
		Type:   "digit",
		Width:  240,
		Height: 80,
		Length: 4,
		Lang:   "zh",
	})
	assert.IsType(t, "string", id)
	assert.IsType(t, "string", content)
	captchaStore.Get(id, false)
	assert.True(t, CaptchaValidate(id, captchaStore.Get(id, false)))
}

func TestProcessCaptcha(t *testing.T) {
	args := url.Values{}
	args.Add("type", "math")
	args.Add("lang", "zh")
	process := gou.NewProcess("xiang.helper.Captcha", args)
	res := process.Run().(maps.Map)
	assert.IsType(t, "string", res.Get("id"))
	assert.IsType(t, "string", res.Get("content"))

	value := captchaStore.Get(res.Get("id").(string), false)
	process = gou.NewProcess("xiang.helper.CaptchaValidate", res.Get("id"), value)
	assert.True(t, process.Run().(bool))
	assert.Panics(t, func() {
		gou.NewProcess("xiang.helper.CaptchaValidate", res.Get("id"), "xxx").Run()
	})

	assert.Panics(t, func() {
		gou.NewProcess("xiang.helper.CaptchaValidate", res.Get("id"), "").Run()
	})
}
