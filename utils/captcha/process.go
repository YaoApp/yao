package captcha

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
)

// ProcessGenerate utils.captcha.Generate - Generate captcha
func ProcessGenerate(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	option := NewOption()

	// Parse options from process args
	if process.NumOfArgs() > 0 {
		optMap := process.ArgsMap(0, map[string]interface{}{})
		if width, ok := optMap["width"].(int); ok {
			option.Width = width
		}
		if height, ok := optMap["height"].(int); ok {
			option.Height = height
		}
		if length, ok := optMap["length"].(int); ok {
			option.Length = length
		}
		if captchaType, ok := optMap["type"].(string); ok {
			option.Type = captchaType
		}
		if lang, ok := optMap["lang"].(string); ok {
			option.Lang = lang
		}
		if bg, ok := optMap["background"].(string); ok {
			option.Background = bg
		}
	}

	id, content := Generate(option)
	return maps.Map{
		"id":      id,
		"content": content,
	}
}

// ProcessValidate utils.captcha.Verify - Validate captcha
func ProcessValidate(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	id := process.ArgsString(0)
	code := process.ArgsString(1)

	if code == "" {
		exception.New("Please enter the captcha.", 400).Throw()
		return false
	}

	if !Validate(id, code) {
		exception.New("Invalid captcha.", 400).Throw()
		return false
	}

	return true
}

// ProcessGet utils.captcha.Get - Get captcha code (for testing)
func ProcessGet(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	id := process.ArgsString(0)
	return Get(id)
}
