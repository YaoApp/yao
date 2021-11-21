package helper

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xiang/config"
)

// JwtClaims 用户Token
type JwtClaims struct {
	ID   int
	Data map[string]interface{}
	jwt.StandardClaims
}

// JwtValidate JWT 校验
func JwtValidate(tokenString string) map[string]interface{} {
	token, err := jwt.ParseWithClaims(tokenString, &JwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.Conf.JWT.Secret), nil
	})

	if err != nil {
		exception.New("令牌无效", 403).Ctx(err.Error()).Throw()
		return nil
	}

	if claims, ok := token.Claims.(*JwtClaims); ok && token.Valid {
		return claims.Data
	}

	exception.New("令牌无效", 403).Ctx(token.Claims).Throw()
	return nil
}

// JwtMake  生成 JWT
// subject options[0], audience options[1], issuer options[1]
func JwtMake(id int, data map[string]interface{}, timeout int64, options ...string) map[string]interface{} {
	now := time.Now().Unix()
	expiresAt := now + timeout
	uid := fmt.Sprintf("%d", id)
	subject := "User Token"
	audience := "Xiang Metadata Admin Panel"
	issuer := fmt.Sprintf("xiang:%d", id)
	length := len(options)
	if length > 0 {
		subject = options[0]
	}
	if length > 1 {
		audience = options[1]
	}
	if length > 2 {
		issuer = options[2]
	}
	claims := &JwtClaims{
		ID:   id,
		Data: data,
		StandardClaims: jwt.StandardClaims{
			Id:        uid,       // 唯一ID
			Subject:   subject,   // 主题
			Audience:  audience,  // 接收人
			ExpiresAt: expiresAt, // 过期时间
			NotBefore: now,       // 生效时间
			IssuedAt:  now,       // 签发时间
			Issuer:    issuer,    // 签发人
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(config.Conf.JWT.Secret))
	if err != nil {
		exception.New("生成令牌失败", 500).Ctx(err).Throw()
	}
	return map[string]interface{}{
		"token":      tokenString,
		"expires_at": expiresAt,
	}
}

// ProcessJwtMake xiang.helper.JwtMake 生成JWT
func ProcessJwtMake(process *gou.Process) interface{} {
	process.ValidateArgNums(3)
	id := process.ArgsInt(0)
	data := process.ArgsMap(1)
	timeout := int64(process.ArgsInt(2))
	args := []string{}
	for i := 3; i < len(process.Args); i++ {
		args = append(args, fmt.Sprintf("%v", process.Args[i]))
	}
	return JwtMake(id, data, timeout, args...)
}

// ProcessJwtValidate xiang.helper.JwtValidate 校验JWT
func ProcessJwtValidate(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	tokenString := process.ArgsString(0)
	return JwtValidate(tokenString)
}
