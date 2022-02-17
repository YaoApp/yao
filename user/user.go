package user

import (
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/helper"
	"golang.org/x/crypto/bcrypt"
)

// JwtClaims JWT claims
type JwtClaims struct {
	ID   int
	Type string
	Name string
	jwt.StandardClaims
}

var loginTypes = map[string]string{
	"email":  "email",
	"mobile": "mobile",
}

// Auth 用户身份鉴权
func Auth(field string, value string, password string) maps.Map {
	column, has := loginTypes[field]
	if !has {
		exception.New("登录方式(%s)尚未支持", 400, field).Throw()
	}

	user := gou.Select("xiang.user")
	rows, err := user.Get(gou.QueryParam{
		Select: []interface{}{"id", "password", "name", "type", "email", "mobile", "extra"},
		Limit:  1,
		Wheres: []gou.QueryWhere{
			{Column: column, Value: value},
			{Column: "status", Value: "enabled"},
		},
	})

	if err != nil {
		exception.New("数据库查询错误", 500, field).Throw()
	}

	if len(rows) == 0 {
		exception.New("用户不存在(%s)", 404, value).Throw()
	}

	row := rows[0]
	passwordHash := row.Get("password").(string)
	row.Del("password")

	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		exception.New("登录密码错误", 403, value).Throw()
	}

	expiresAt := time.Now().Unix() + 3600

	// token := MakeToken(row, expiresAt)
	sid := session.ID()
	id := any.Of(row.Get("id")).CInt()
	token := helper.JwtMake(id, map[string]interface{}{}, map[string]interface{}{
		"expires_at": expiresAt,
		"sid":        sid,
		"issuer":     "xiang",
	})
	session.Global().Expire(time.Duration(token.ExpiresAt)*time.Second).ID(sid).Set("user_id", id)
	session.Global().ID(sid).Set("user", row)
	session.Global().ID(sid).Set("issuer", "xiang")

	// 读取菜单
	menus := gou.NewProcess("flows.xiang.menu").Run()
	return maps.Map{
		"expires_at": token.ExpiresAt,
		"token":      token.Token,
		"user":       row,
		"menus":      menus,
	}
}
