package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/maps"
)

func TestProcessEnv(t *testing.T) {
	err := process.New("xiang.helper.EnvSet", "XIANG_UNIT_TEST", "FOO").Run()
	assert.Nil(t, err)
	test := process.New("xiang.helper.EnvGet", "XIANG_UNIT_TEST").Run().(string)
	assert.Equal(t, "FOO", test)
}

func TestProcessEnvMulti(t *testing.T) {
	err := process.New("xiang.helper.EnvMultiSet", maps.Map{"XIANG_UNIT_TEST": "FOO", "XIANG_UNIT_TEST2": "BAR"}).Run()
	assert.Nil(t, err)
	test := process.New("xiang.helper.EnvMultiGet", "XIANG_UNIT_TEST", "XIANG_UNIT_TEST2").Run().(map[string]string)
	assert.Equal(t, "FOO", test["XIANG_UNIT_TEST"])
	assert.Equal(t, "BAR", test["XIANG_UNIT_TEST2"])
}
