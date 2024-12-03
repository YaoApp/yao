package login

import (
	"time"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/helper"
	"golang.org/x/crypto/bcrypt"
)

var loginTypes = map[string]string{
	"email":  "email",
	"mobile": "mobile",
}

// Export process

func exportProcess() {
	process.Register("yao.login.admin", processLoginAdmin)
}

// processLoginAdmin yao.admin.login 用户登录
func processLoginAdmin(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	payload := process.ArgsMap(0).Dot()
	log.With(log.F{"payload": payload}).Debug("processLoginAdmin")

	id := any.Of(payload.Get("captcha.id")).CString()
	value := any.Of(payload.Get("captcha.code")).CString()
	if id == "" {
		exception.New("Please enter the captcha ID", 400).Ctx(maps.Map{"id": id, "code": value}).Throw()
	}

	if value == "" {
		exception.New("Please enter the captcha code", 400).Ctx(maps.Map{"id": id, "code": value}).Throw()
	}

	if !helper.CaptchaValidate(id, value) {
		log.With(log.F{"id": id, "code": value}).Debug("ProcessLogin")
		exception.New("Captcha error", 401).Ctx(maps.Map{"id": id, "code": value}).Throw()
		return nil
	}

	sid := session.ID()
	if csid, ok := payload["sid"].(string); ok {
		sid = csid
	}

	email := any.Of(payload.Get("email")).CString()
	mobile := any.Of(payload.Get("mobile")).CString()
	password := any.Of(payload.Get("password")).CString()
	if email != "" {
		return auth("email", email, password, sid)
	} else if mobile != "" {
		return auth("mobile", mobile, password, sid)
	}

	exception.New("Parameter error", 400).Ctx(payload).Throw()
	return nil
}

func auth(field string, value string, password string, sid string) maps.Map {
	column, has := loginTypes[field]
	if !has {
		exception.New("Login type (%s) not supported", 400, field).Throw()
	}

	user := model.Select("admin.user")
	rows, err := user.Get(model.QueryParam{
		Select: []interface{}{"id", "password", "name", "type", "email", "mobile", "extra", "status"},
		Limit:  1,
		Wheres: []model.QueryWhere{
			{Column: column, Value: value},
			{Column: "status", Value: "enabled"},
		},
	})

	if err != nil {
		exception.New("Database query error", 500, field).Throw()
	}

	if len(rows) == 0 {
		exception.New("User not found (%s)", 404, value).Throw()
	}

	row := rows[0]
	passwordHash := row.Get("password").(string)
	row.Del("password")

	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		exception.New("Login password error (%v)", 403, value).Throw()
	}

	expiresAt := time.Now().Unix() + 3600*8

	// token := MakeToken(row, expiresAt)
	id := any.Of(row.Get("id")).CInt()
	token := helper.JwtMake(id, map[string]interface{}{}, map[string]interface{}{
		"expires_at": expiresAt,
		"sid":        sid,
		"issuer":     "yao",
	})
	log.Debug("[login] auth sid=%s", sid)
	session.Global().Expire(time.Duration(token.ExpiresAt)*time.Second).ID(sid).Set("user_id", id)
	session.Global().Expire(time.Duration(token.ExpiresAt)*time.Second).ID(sid).Set("user", row)
	session.Global().Expire(time.Duration(token.ExpiresAt)*time.Second).ID(sid).Set("issuer", "yao")

	studio := map[string]interface{}{}
	if config.Conf.Mode == "development" {

		studioToken := helper.JwtMake(id, map[string]interface{}{}, map[string]interface{}{
			"expires_at": expiresAt,
			"sid":        sid,
			"issuer":     "yao",
		}, []byte(config.Conf.Studio.Secret))

		studio["port"] = config.Conf.Studio.Port
		studio["token"] = studioToken.Token
		studio["expires_at"] = studioToken.ExpiresAt
	}

	// Get user menus
	menus := process.New("yao.app.menu").WithSID(sid).Run()
	return maps.Map{
		"expires_at": token.ExpiresAt,
		"token":      token.Token,
		"user":       row,
		"menus":      menus,
		"studio":     studio,
	}
}
