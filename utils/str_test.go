package utils_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	_ "github.com/yaoapp/yao/helper"
)

func TestProcessStrJoin(t *testing.T) {
	testPrepare()
	res := process.New("utils.str.Join", []interface{}{"FOO", 20, "BAR"}, ",").Run().(string)
	assert.Equal(t, "FOO,20,BAR", res)
}

func TestProcessStrJoinPath(t *testing.T) {
	testPrepare()
	res := process.New("utils.str.JoinPath", "data", 20, "app").Run().(string)
	shouldBe := fmt.Sprintf("data%s20%sapp", string(os.PathSeparator), string(os.PathSeparator))
	assert.Equal(t, shouldBe, res)
}

func TestProcessUUID(t *testing.T) {
	testPrepare()
	res := process.New("utils.str.UUID").Run().(string)
	_, err := uuid.Parse(res)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 36, len(res))
}

func TestProcessStrHex(t *testing.T) {
	testPrepare()
	res, err := process.New("utils.str.Hex", []byte{0x0, 0x1}).Exec()
	assert.Nil(t, err)
	assert.Equal(t, "0001", res)

	res, err = process.New("utils.str.Hex", string([]byte{0x0, 0x1})).Exec()
	assert.Nil(t, err)
	assert.Equal(t, "0001", res)

	res, err = process.New("utils.str.Hex", 1024).Exec()
	assert.Nil(t, err)
	assert.Nil(t, res)
}

func TestProcessPinyin(t *testing.T) {
	testPrepare()

	// Test default settings (no tone, space separator)
	res := process.New("utils.str.Pinyin", "你好世界").Run().(string)
	assert.Equal(t, "ni hao shi jie", res)

	// Test with tone enabled (boolean true)
	config := map[string]interface{}{
		"tone": true,
	}
	res = process.New("utils.str.Pinyin", "你好世界", config).Run().(string)
	assert.Equal(t, "nǐ hǎo shì jiè", res)

	// Test with tone as string "mark"
	config = map[string]interface{}{
		"tone": "mark",
	}
	res = process.New("utils.str.Pinyin", "你好世界", config).Run().(string)
	assert.Equal(t, "nǐ hǎo shì jiè", res)

	// Test with tone as string "number"
	config = map[string]interface{}{
		"tone": "number",
	}
	res = process.New("utils.str.Pinyin", "你好世界", config).Run().(string)
	assert.Equal(t, "ni3 hao3 shi4 jie4", res)

	// Test with tone enabled (boolean true) and custom separator
	config = map[string]interface{}{
		"tone":      true,
		"separator": "-",
	}
	res = process.New("utils.str.Pinyin", "你好世界", config).Run().(string)
	assert.Equal(t, "nǐ-hǎo-shì-jiè", res)

	// Test with tone "number" and custom separator
	config = map[string]interface{}{
		"tone":      "number",
		"separator": "-",
	}
	res = process.New("utils.str.Pinyin", "你好世界", config).Run().(string)
	assert.Equal(t, "ni3-hao3-shi4-jie4", res)

	// Test with heteronym enabled
	config = map[string]interface{}{
		"heteronym": true,
	}
	res = process.New("utils.str.Pinyin", "中国", config).Run().(string)
	assert.Contains(t, res, "zhong")
	assert.Contains(t, res, "guo")

	// Test with heteronym and tone enabled
	config = map[string]interface{}{
		"heteronym": true,
		"tone":      true,
	}
	res = process.New("utils.str.Pinyin", "中国", config).Run().(string)
	assert.Contains(t, res, "zhōng")
	assert.Contains(t, res, "guó")

	// Test with heteronym and tone number
	config = map[string]interface{}{
		"heteronym": true,
		"tone":      "number",
	}
	res = process.New("utils.str.Pinyin", "中国", config).Run().(string)
	assert.Contains(t, res, "zhong1")
	assert.NotContains(t, res, "zho1ng")
	assert.Contains(t, res, "guo2")

	// Test with only custom separator
	config = map[string]interface{}{
		"separator": "_",
	}
	res = process.New("utils.str.Pinyin", "你好世界", config).Run().(string)
	assert.Equal(t, "ni_hao_shi_jie", res)

	// Test with empty string
	res = process.New("utils.str.Pinyin", "").Run().(string)
	assert.Equal(t, "", res)

	// Test with Chinese characters only
	res = process.New("utils.str.Pinyin", "你好").Run().(string)
	assert.Equal(t, "ni hao", res)

	// Test with multiple words and spaces
	res = process.New("utils.str.Pinyin", "中国 北京").Run().(string)
	assert.Equal(t, "zhong guo bei jing", res)

	// Test with multiple consecutive spaces
	res = process.New("utils.str.Pinyin", "你好  世界").Run().(string)
	assert.Equal(t, "ni hao shi jie", res)

	// Test with leading and trailing spaces
	res = process.New("utils.str.Pinyin", " 你好世界 ").Run().(string)
	assert.Equal(t, "ni hao shi jie", res)

	// Test with mixed Chinese and English
	res = process.New("utils.str.Pinyin", "Hello你好World世界").Run().(string)
	assert.Equal(t, "ni hao shi jie", res)

	// Test with numbers and punctuation
	res = process.New("utils.str.Pinyin", "你好2023！世界。").Run().(string)
	assert.Equal(t, "ni hao shi jie", res)

	// Test with multi-character separator
	config = map[string]interface{}{
		"separator": "==",
	}
	res = process.New("utils.str.Pinyin", "你好世界", config).Run().(string)
	assert.Equal(t, "ni==hao==shi==jie", res)

	// Test with empty separator
	config = map[string]interface{}{
		"separator": "",
	}
	res = process.New("utils.str.Pinyin", "你好世界", config).Run().(string)
	assert.Equal(t, "nihaoshijie", res)

	// Test with special characters as separator
	config = map[string]interface{}{
		"separator": "★",
	}
	res = process.New("utils.str.Pinyin", "你好世界", config).Run().(string)
	assert.Equal(t, "ni★hao★shi★jie", res)

	// Test with multiple words and tone
	// Create a fresh config map to avoid reference issues
	toneConfig := map[string]interface{}{
		"tone": true,
	}
	res = process.New("utils.str.Pinyin", "你好美丽的世界", toneConfig).Run().(string)
	assert.Equal(t, "nǐ hǎo měi lì de shì jiè", res)
}
