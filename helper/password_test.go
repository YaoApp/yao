package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
)

func TestPassword(t *testing.T) {
	assert.True(t, PasswordValidate("U123456p+", "$2a$04$TS/rWBs66jADjQl8fa.w..ivkNAjH8d4sI1OPGvEB9Leed6EpzIF2"))
	assert.Panics(t, func() {
		PasswordValidate("U123456p+", "123456")
	})
}

func TestProcessPassword(t *testing.T) {
	pwd := "U123456p+"
	hash := "$2a$04$TS/rWBs66jADjQl8fa.w..ivkNAjH8d4sI1OPGvEB9Leed6EpzIF2"
	args := []interface{}{pwd, hash}
	p := process.New("xiang.helper.PasswordValidate", args...)
	res := p.Run()
	assert.True(t, res.(bool))

	args = []interface{}{pwd, "123456"}
	p = process.New("xiang.helper.PasswordValidate", args...)
	assert.Panics(t, func() {
		p.Run()
	})
}
