package otp

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
)

// ProcessGenerate utils.otp.Generate - Generate OTP code
func ProcessGenerate(process *process.Process) interface{} {
	process.ValidateArgNums(1)

	option := NewOption()

	// Parse options from process args
	if process.NumOfArgs() > 0 {
		optMap := process.ArgsMap(0, map[string]interface{}{})
		if length, ok := optMap["length"].(int); ok {
			option.Length = length
		}
		if expiration, ok := optMap["expiration"].(int); ok {
			option.Expiration = expiration
		}
		if codeType, ok := optMap["type"].(string); ok {
			option.Type = codeType
		}
	}

	id, code := Generate(option)

	return maps.Map{
		"id":   id,
		"code": code,
	}
}

// ProcessValidate utils.otp.Validate - Validate OTP code
func ProcessValidate(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	id := process.ArgsString(0)
	code := process.ArgsString(1)

	// Default clear to true
	clear := true
	if process.NumOfArgs() > 2 {
		clear = process.ArgsBool(2)
	}

	if code == "" {
		exception.New("OTP code is required", 400).Throw()
		return false
	}

	if !Validate(id, code, clear) {
		exception.New("Invalid or expired OTP code", 400).Throw()
		return false
	}

	return true
}

// ProcessGet utils.otp.Get - Get OTP code (for testing)
func ProcessGet(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	id := process.ArgsString(0)
	return Get(id)
}

// ProcessDelete utils.otp.Delete - Delete OTP code
func ProcessDelete(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	id := process.ArgsString(0)
	Delete(id)
	return nil
}
