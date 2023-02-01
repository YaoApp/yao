package helper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
)

func TestJwt(t *testing.T) {
	data := map[string]interface{}{"hello": "world", "id": 1}
	option := map[string]interface{}{"subject": "Unit Test", "audience": "Test", "issuer": "UnitTest", "timeout": 1, "sid": ""}
	token := JwtMake(1, data, option)
	tokenString := token.Token
	res := JwtValidate(tokenString)
	assert.NotNil(t, res)
	assert.Equal(t, float64(1), res.Data["id"])
	assert.Equal(t, "world", res.Data["hello"])
	assert.Equal(t, "UnitTest", res.Issuer)
	time.Sleep(2 * time.Second)
	assert.Panics(t, func() { JwtValidate(tokenString) })
}

func TestProcessJwt(t *testing.T) {
	data := map[string]interface{}{"hello": "world", "id": 1}
	option := map[string]interface{}{"subject": "Unit Test", "audience": "Test", "issuer": "UnitTest", "timeout": 1, "sid": ""}
	args := []interface{}{1, data, option}
	p := process.New("xiang.helper.JwtMake", args...)
	token := p.Run().(JwtToken)
	tokenString := token.Token
	res := process.New("xiang.helper.JwtValidate", tokenString).Run().(*JwtClaims)
	assert.Equal(t, float64(1), res.Data["id"])
	assert.Equal(t, "world", res.Data["hello"])
	assert.Equal(t, "UnitTest", res.Issuer)
	time.Sleep(2 * time.Second)
	assert.Panics(t, func() { process.New("xiang.helper.JwtValidate", tokenString).Run() })
}
