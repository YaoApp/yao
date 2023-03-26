package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
)

func TestMD4(t *testing.T) {
	// Hash
	args := []interface{}{"MD4", "123456"}
	res := process.New("crypto.Hash", args...).Run()
	assert.Equal(t, "585028aa0f794af812ee3be8804eb14a", res)

	// HMac
	args = append(args, "123456")
	res = process.New("crypto.Hmac", args...).Run()
	assert.Equal(t, "356f45727db95d65843b2794474d741c", res)
}

func TestMD5(t *testing.T) {
	// Hash
	args := []interface{}{"MD5", "123456"}
	res := process.New("crypto.Hash", args...).Run()
	assert.Equal(t, "e10adc3949ba59abbe56e057f20f883e", res)

	// HMac
	args = append(args, "123456")
	res = process.New("crypto.Hmac", args...).Run()
	assert.Equal(t, "30ce71a73bdd908c3955a90e8f7429ef", res)
}

func TestSHA1(t *testing.T) {
	// Hash
	args := []interface{}{"SHA1", "123456"}
	res := process.New("crypto.Hash", args...).Run()
	assert.Equal(t, "7c4a8d09ca3762af61e59520943dc26494f8941b", res)

	// HMac
	args = append(args, "123456")
	res = process.New("crypto.Hmac", args...).Run()
	assert.Equal(t, "74b55b6ab2b8e438ac810435e369e3047b3951d0", res)
}

func TestSHA256(t *testing.T) {
	// Hash
	args := []interface{}{"SHA256", "123456"}
	res := process.New("crypto.Hash", args...).Run()
	assert.Equal(t, "8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92", res)

	// HMac
	args = append(args, "123456")
	res = process.New("crypto.Hmac", args...).Run()
	assert.Equal(t, "b8ad08a3a547e35829b821b75370301dd8c4b06bdd7771f9b541a75914068718", res)
}

func TestSHA1Base64(t *testing.T) {
	// Hash
	args := []interface{}{"SHA1", "123456"}
	res := process.New("crypto.Hash", args...).Run()
	assert.Equal(t, "7c4a8d09ca3762af61e59520943dc26494f8941b", res)

	// HMac
	args = append(args, "123456", "base64")
	res = process.New("crypto.Hmac", args...).Run()
	assert.Equal(t, "dLVbarK45DisgQQ142njBHs5UdA=", res)
}
