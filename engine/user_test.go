package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/user"
)

func TestUserAuth(t *testing.T) {

	res := user.Auth("email", "xiang@iqka.com", "A123456p+")
	assert.True(t, res.Has("user"))
	assert.True(t, res.Has("token"))
	assert.True(t, res.Has("expires_at"))
	assert.Panics(t, func() {
		user.Auth("email", "xiang@iqka.com", "A123456p+22")
	})

	res = user.Auth("mobile", "13900001111", "U123456p+")
	assert.True(t, res.Has("user"))
	assert.True(t, res.Has("token"))
	assert.True(t, res.Has("expires_at"))

	assert.Panics(t, func() {
		user.Auth("email", "1390000111", "A123456p+22")
	})

}
