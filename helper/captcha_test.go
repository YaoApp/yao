package helper

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
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
	assert.True(t, CaptchaValidate(id, toString(store.Get(id, false))))

	id, content = CaptchaMake(CaptchaOption{
		Type:   "math",
		Width:  240,
		Height: 80,
		Length: 4,
		Lang:   "zh",
	})
	assert.IsType(t, "string", id)
	assert.IsType(t, "string", content)
	assert.True(t, CaptchaValidate(id, toString(store.Get(id, false))))

	id, content = CaptchaMake(CaptchaOption{
		Type:   "digit",
		Width:  240,
		Height: 80,
		Length: 4,
		Lang:   "zh",
	})
	assert.IsType(t, "string", id)
	assert.IsType(t, "string", content)
	assert.True(t, CaptchaValidate(id, toString(store.Get(id, false))))
}

func TestProcessCaptcha(t *testing.T) {
	args := url.Values{}
	args.Add("type", "math")
	args.Add("lang", "zh")
	p := process.New("xiang.helper.Captcha", args)
	res := p.Run().(maps.Map)
	assert.IsType(t, "string", res.Get("id"))
	assert.IsType(t, "string", res.Get("content"))

	value := toString(store.Get(res.Get("id").(string), false))
	p = process.New("xiang.helper.CaptchaValidate", res.Get("id"), value)
	assert.True(t, p.Run().(bool))
	assert.Panics(t, func() {
		process.New("xiang.helper.CaptchaValidate", res.Get("id"), "xxx").Run()
	})

	assert.Panics(t, func() {
		process.New("xiang.helper.CaptchaValidate", res.Get("id"), "").Run()
	})
}
