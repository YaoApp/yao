package user

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCaptcha(t *testing.T) {
	id, content := MakeCaptcha(CaptchaOption{
		Type:   "audio",
		Width:  240,
		Height: 80,
		Length: 4,
		Lang:   "zh",
	})
	assert.IsType(t, "string", id)
	assert.IsType(t, "string", content)
	captchaStore.Get(id, false)
	assert.True(t, ValidateCaptcha(id, captchaStore.Get(id, false)))

	id, content = MakeCaptcha(CaptchaOption{
		Type:   "math",
		Width:  240,
		Height: 80,
		Length: 4,
		Lang:   "zh",
	})
	assert.IsType(t, "string", id)
	assert.IsType(t, "string", content)
	captchaStore.Get(id, false)
	assert.True(t, ValidateCaptcha(id, captchaStore.Get(id, false)))

	id, content = MakeCaptcha(CaptchaOption{
		Type:   "digit",
		Width:  240,
		Height: 80,
		Length: 4,
		Lang:   "zh",
	})
	assert.IsType(t, "string", id)
	assert.IsType(t, "string", content)
	captchaStore.Get(id, false)
	assert.True(t, ValidateCaptcha(id, captchaStore.Get(id, false)))
}
