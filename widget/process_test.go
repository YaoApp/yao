package widget

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestProcessSave(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	iform := preare(t)[1]

	assert.Panics(t, func() {
		process.New("widget.Save", "dyform", "feedback/new.form.yao", map[string]interface{}{}).Run()
	})

	assert.NotPanics(t, func() {
		process.New("widget.Save", "iform", "feedback/new.form.yao", map[string]interface{}{"columns": []interface{}{}}).Run()
	})

	defer iform.Remove("feedback/new.form.yao")

	instance, ok := iform.Instances.Load("feedback.new")
	if !ok {
		t.Fatal("feedback instance not found")
	}
	assert.Equal(t, "feedback.new", instance.(*Instance).id)
}

func TestProcessRemove(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	iform := preare(t)[1]

	assert.Panics(t, func() {
		process.New("widget.Remove", "dyform", "feedback/new.form.yao").Run()
	})

	assert.NotPanics(t, func() {
		process.New("widget.Remove", "iform", "feedback/new.form.yao").Run()
	})

	defer iform.Remove("feedback/new.form.yao")

	_, ok := iform.Instances.Load("feedback.new")
	assert.False(t, ok)
}
