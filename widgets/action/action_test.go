package action

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
)

func TestNewProcess(t *testing.T) {
	defaults := testDefaults()
	test := NewProcess().
		Merge(defaults["yao.unit.Test1"]).
		SetHandler(testHandler)

	assert.Equal(t, "yao.unit.Test1", test.Name)
	assert.Equal(t, "bearer-jwt", test.Guard)
	assert.Equal(t, "yao.unit.T1", test.Process)
	assert.Equal(t, []interface{}{nil, nil, nil}, test.Default)
}

func TestProcessOf(t *testing.T) {
	defaults := testDefaults()
	test := NewProcess()
	new := ProcessOf(test).Merge(defaults["yao.unit.Test2"]).SetHandler(testHandler)
	assert.Equal(t, "yao.unit.Test2", new.Name)
	assert.Equal(t, "bearer-jwt", new.Guard)
	assert.Equal(t, "yao.unit.T2", new.Process)
	assert.Equal(t, []interface{}{nil, nil}, new.Default)

	new = ProcessOf(nil).Merge(defaults["yao.unit.Test3"]).SetHandler(testHandler)
	assert.Equal(t, "yao.unit.Test3", new.Name)
	assert.Equal(t, "bearer-jwt", new.Guard)
	assert.Equal(t, "yao.unit.T3", new.Process)
	assert.Equal(t, []interface{}{nil}, new.Default)
}

func testData() map[string]*Process {
	defaults := testDefaults()
	return map[string]*Process{
		"T0": NewProcess().Merge(defaults["yao.unit.Test1"]).SetHandler(testHandler),
		"T1": NewProcess().Merge(defaults["yao.unit.Test2"]).SetHandler(testHandler),
		"T2": NewProcess().Merge(defaults["yao.unit.Test3"]).SetHandler(testHandler),
		"T3": NewProcess().Merge(defaults["yao.unit.Test4"]).SetHandler(testHandler),
		"T4": NewProcess().Merge(defaults["yao.unit.Test5"]).SetHandler(testHandler),
	}
}

func testHandler(p *Process, process *process.Process) (interface{}, error) {
	args := p.Args(process)
	return args, nil
}

func testDefaults() map[string]*Process {
	return map[string]*Process{

		"yao.unit.Test1": {
			Name:    "yao.unit.Test1",
			Guard:   "bearer-jwt",
			Process: "yao.unit.T1",
			Default: []interface{}{nil, nil, nil},
		},

		"yao.unit.Test2": {
			Name:    "yao.unit.Test2",
			Guard:   "bearer-jwt",
			Process: "yao.unit.T2",
			Default: []interface{}{nil, nil},
		},

		"yao.unit.Test3": {
			Name:    "yao.unit.Test3",
			Guard:   "bearer-jwt",
			Process: "yao.unit.T3",
			Default: []interface{}{nil},
		},

		"yao.unit.Test4": {
			Name:    "yao.unit.Test4",
			Guard:   "bearer-jwt",
			Process: "yao.unit.T4",
			Default: []interface{}{},
		},
	}
}
