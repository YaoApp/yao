package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
)

func TestIF(t *testing.T) {

	process.Register("xiang.unit.return", func(process *process.Process) interface{} {
		return process.Args
	})

	case1 := CaseParam{
		When: []Condition{
			{Left: "张三", OP: "=", Right: "李四", Compute: Computes["="]},
			{OR: true, Left: "李四", OP: "=", Right: "李四", Compute: Computes["="]},
		},
		Name:    "打印信息",
		Process: "xiang.unit.Return",
		Args:    []interface{}{"world"},
	}

	case2 := CaseParam{
		When: []Condition{
			{Left: "张三", OP: "=", Right: "张三", Compute: Computes["="]},
		},
		Name:    "打印信息",
		Process: "xiang.unit.Return",
		Args:    []interface{}{"foo"},
	}

	v := IF(case1, case2).([]interface{})
	assert.Equal(t, "world", v[0])
}

func TestProcessIF(t *testing.T) {

	process.Register("xiang.unit.return", func(process *process.Process) interface{} {
		return process.Args
	})

	args := []interface{}{
		map[string]interface{}{
			"when":    []map[string]interface{}{{"用户": "张三", "=": "李四"}, {"or": true, "用户": "李四", "=": "李四"}},
			"name":    "打印信息",
			"process": "xiang.unit.Return",
			"args":    []interface{}{"world"},
		},
		map[string]interface{}{
			"when":    []map[string]interface{}{{"用户": "张三", "=": "张三"}},
			"name":    "打印信息",
			"process": "xiang.unit.Return",
			"args":    []interface{}{"foo"},
		},
	}
	process := process.New("xiang.helper.IF", args...)
	res := process.Run().([]interface{})
	assert.Equal(t, "world", res[0])
}
