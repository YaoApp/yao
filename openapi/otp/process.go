package otp

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/openapi/utils"
)

func init() {
	process.RegisterGroup("otp", map[string]process.Handler{
		"create": processCreate,
		"verify": processVerify,
		"login":  processLogin,
		"revoke": processRevoke,
	})
}

// processCreate handles otp.Create(params).
// args[0]: map with team_id, member_id, user_id, expires_in, redirect, scope,
//
//	token_expires_in, consume.
func processCreate(p *process.Process) interface{} {
	p.ValidateArgNums(1)

	raw := p.ArgsMap(0)
	params := &GenerateParams{
		TeamID:   utils.ToString(raw["team_id"]),
		MemberID: utils.ToString(raw["member_id"]),
		UserID:   utils.ToString(raw["user_id"]),
		Redirect: utils.ToString(raw["redirect"]),
		Scope:    utils.ToString(raw["scope"]),
		Consume:  true,
	}
	if v, ok := raw["expires_in"]; ok {
		params.ExpiresIn = utils.ToInt(v)
	}
	if v, ok := raw["token_expires_in"]; ok {
		params.TokenExpiresIn = utils.ToInt(v)
	}
	if v, ok := raw["consume"]; ok {
		params.Consume = utils.ToBool(v)
	}

	code, err := OTP.Create(params)
	if err != nil {
		exception.New(err.Error(), 400).Throw()
	}
	return code
}

// processVerify handles otp.Verify(code).
// args[0]: code string.
func processVerify(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	code := p.ArgsString(0)

	payload, err := OTP.Verify(code)
	if err != nil {
		exception.New(err.Error(), 400).Throw()
	}
	return payload
}

// processLogin handles otp.Login(code, locale?).
// args[0]: code string; args[1]: locale string (optional).
func processLogin(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	code := p.ArgsString(0)

	locale := ""
	if p.NumOfArgs() > 1 {
		locale = p.ArgsString(1)
	}

	result, err := OTP.Login(code, locale)
	if err != nil {
		exception.New(err.Error(), 401).Throw()
	}
	return result
}

// processRevoke handles otp.Revoke(code).
// args[0]: code string.
func processRevoke(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	code := p.ArgsString(0)

	if err := OTP.Revoke(code); err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return nil
}
