package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
)

func TestSetupAPIs(t *testing.T) {
	assignFlow := Select("assign")
	assignFlow.SetupAPIs()

	api := gou.SelectAPI("xiang.workflow.assign")
	assert.Equal(t, "xiang.workflow.assign", api.Name)
	assert.Equal(t, 9, len(api.HTTP.Paths))
}
