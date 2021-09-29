package user

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/xiang/config"
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

	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		exception.New("登录密码错误", 403, value).Throw()
	}

	expiresAt := time.Now().Unix() + 3600
	token := MakeToken(row, expiresAt)
	row.Del("password")
	return maps.Map{
		"expires_at": expiresAt,
		"token":      token,
		"user":       row,
	}
}

// MakeToken  生成 JWT Token
func MakeToken(row maps.Map, ExpiresAt int64) string {
	claims := &JwtClaims{
		ID:   int(row.Get("id").(int64)),
		Type: row.Get("type").(string),
		Name: row.Get("name").(string),
		StandardClaims: jwt.StandardClaims{
			Subject:   fmt.Sprintf("%d", row.Get("id")),
			ExpiresAt: ExpiresAt,
			Issuer:    fmt.Sprintf("%d", row.Get("id")),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(config.Conf.JWT.Secret))
	if err != nil {
		exception.New("生成登录口令失败 %s", 500, err).Throw()
	}

	return tokenString
}
