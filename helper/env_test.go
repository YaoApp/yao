package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
)

func TestProcessEnv(t *testing.T) {
	err := gou.NewProcess("xiang.helper.EnvSet", "XIANG_UNIT_TEST", "FOO").Run()
	assert.Nil(t, err)
	test := gou.NewProcess("xiang.helper.EnvGet", "XIANG_UNIT_TEST").Run().(string)
	assert.Equal(t, "FOO", test)
}
