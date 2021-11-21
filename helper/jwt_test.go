package helper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
)

func TestJwt(t *testing.T) {
	data := map[string]interface{}{"hello": "world", "id": 1}
	token := JwtMake(1, data, 1, "Unit Test", "Test", "UnitTest")
	tokenString := token["token"].(string)
	res := JwtValidate(tokenString)
	assert.Equal(t, float64(1), res["id"])
	assert.Equal(t, "world", res["hello"])
	time.Sleep(2 * time.Second)
	assert.Panics(t, func() { JwtValidate(tokenString) })
}

func TestProcessJwt(t *testing.T) {
	data := map[string]interface{}{"hello": "world", "id": 1}
	args := []interface{}{1, data, 1, "Unit Test", "Test", "UnitTest"}
	process := gou.NewProcess("xiang.helper.JwtMake", args...)
	token := process.Run().(map[string]interface{})
	tokenString := token["token"].(string)
	res := gou.NewProcess("xiang.helper.JwtValidate", tokenString).Run().(map[string]interface{})
	assert.Equal(t, float64(1), res["id"])
	assert.Equal(t, "world", res["hello"])
	time.Sleep(2 * time.Second)
	assert.Panics(t, func() { gou.NewProcess("xiang.helper.JwtValidate", tokenString).Run() })
}
