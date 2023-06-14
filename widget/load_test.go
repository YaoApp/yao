package widget

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestLoad(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	Load(config.Conf)
	check(t)
}

func check(t *testing.T) {
	assert.NotNil(t, Widgets["dyform"])
	assert.NotNil(t, api.APIs["__yao.widget.dyform"])
	assert.NotNil(t, process.Handlers["widgets.dyform.find"])
	assert.NotNil(t, process.Handlers["widgets.dyform.delete"])
	assert.NotNil(t, process.Handlers["widgets.dyform.cancel"])
	assert.NotNil(t, process.Handlers["widgets.dyform.save"])
	assert.NotNil(t, process.Handlers["widgets.dyform.setting"])
}
