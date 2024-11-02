package utils_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	_ "github.com/yaoapp/yao/helper"
	"github.com/yaoapp/yao/utils/datetime"
	"github.com/yaoapp/yao/utils/str"
	"github.com/yaoapp/yao/utils/tree"
)

func TestProcessNow(t *testing.T) {
	testPrepare()
	assert.LessOrEqual(t, time.Now().Unix(), process.New("utils.now.Timestamp").Run().(int64))
	assert.LessOrEqual(t, time.Now().UnixMilli(), process.New("utils.now.Timestampms").Run().(int64))
	assert.NotNil(t, process.New("utils.now.Date").Run())
	assert.NotNil(t, process.New("utils.now.Time").Run())
	assert.NotNil(t, process.New("utils.now.DateTime").Run())
}

func testPrepare() {

	// Tree
	process.Register("utils.tree.Flatten", tree.ProcessFlatten)

	process.Alias("xiang.helper.StrConcat", "utils.str.Concat")
	process.Alias("xiang.helper.HexToString", "utils.str.Hex")
	process.Register("utils.str.Join", str.ProcessJoin)
	process.Register("utils.str.JoinPath", str.ProcessJoinPath)
	process.Register("utils.str.UUID", str.ProcessUUID)

	process.Register("utils.now.Time", datetime.ProcessTime)
	process.Register("utils.now.Date", datetime.ProcessDate)
	process.Register("utils.now.DateTime", datetime.ProcessDateTime)
	process.Register("utils.now.Timestamp", datetime.ProcessTimestamp)
	process.Register("utils.now.Timestampms", datetime.ProcessTimestampms)
}
