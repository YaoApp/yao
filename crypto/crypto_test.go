package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
)

func TestMD4(t *testing.T) {
	args := []interface{}{"MD4", "123456"}
	res := gou.NewProcess("yao.crypto.hash", args...).Run()
	assert.Equal(t, "585028aa0f794af812ee3be8804eb14a", res)
}

func TestMD5(t *testing.T) {
	args := []interface{}{"MD5", "123456"}
	res := gou.NewProcess("yao.crypto.hash", args...).Run()
	assert.Equal(t, "e10adc3949ba59abbe56e057f20f883e", res)
}

func TestSHA1(t *testing.T) {
	args := []interface{}{"SHA1", "123456"}
	res := gou.NewProcess("yao.crypto.hash", args...).Run()
	assert.Equal(t, "7c4a8d09ca3762af61e59520943dc26494f8941b", res)
}

func TestSHA256(t *testing.T) {
	args := []interface{}{"SHA256", "123456"}
	res := gou.NewProcess("yao.crypto.hash", args...).Run()
	assert.Equal(t, "8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92", res)
}
