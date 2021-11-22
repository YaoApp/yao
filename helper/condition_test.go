package helper

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
)

func TestCondition(t *testing.T) {
	data := []byte(`{ "用户不存在": "张三", "=": "李四" }`)
	cond := Condition{}
	err := jsoniter.Unmarshal(data, &cond)
	assert.Nil(t, err)
	assert.Equal(t, "张三", cond.Left)
	assert.Equal(t, "李四", cond.Right)
	assert.Equal(t, "=", cond.OP)
	assert.Equal(t, "用户不存在", cond.Comment)
	assert.Equal(t, false, cond.OR)
	assert.False(t, cond.Exec())

	data = []byte(`{ "用户不存在":"张三", "is":"null" }`)
	cond = Condition{}
	err = jsoniter.Unmarshal(data, &cond)
	assert.Nil(t, err)
	assert.Equal(t, "张三", cond.Left)
	assert.Equal(t, "null", cond.Right)
	assert.Equal(t, "is", cond.OP)
	assert.Equal(t, "用户不存在", cond.Comment)
	assert.Equal(t, false, cond.OR)
	assert.False(t, cond.Exec())

	data = []byte(`{ "left":"李四", "right":"李四", "op":"=", "or":true }`)
	cond = Condition{}
	err = jsoniter.Unmarshal(data, &cond)
	assert.Nil(t, err)
	assert.Equal(t, "李四", cond.Left)
	assert.Equal(t, "李四", cond.Right)
	assert.Equal(t, "=", cond.OP)
	assert.Equal(t, true, cond.OR)
	assert.True(t, cond.Exec())

	data, err = jsoniter.Marshal(cond)
	assert.Nil(t, err)
	str := string(data)
	assert.Contains(t, str, `"=":"李四"`)
	assert.Contains(t, str, `"or":true`)
	assert.Contains(t, str, `"left":"李四"`)
}
