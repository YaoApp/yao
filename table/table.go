package table

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/kun/exception"
)

// Tables 已载入模型
var Tables = map[string]*Table{}

// Load 载入数据表格
func Load(source string, name string) *Table {
	var input io.Reader = nil
	if strings.HasPrefix(source, "file://") || strings.HasPrefix(source, "fs://") {
		filename := strings.TrimPrefix(source, "file://")
		filename = strings.TrimPrefix(filename, "fs://")
		file, err := os.Open(filename)
		if err != nil {
			exception.Err(err, 400).Throw()
		}
		defer file.Close()
		input = file
	} else {
		input = strings.NewReader(source)
	}

	table := Table{
		Source: source,
		Table:  name,
	}
	err := helper.UnmarshalFile(input, &table)
	if err != nil {
		exception.Err(err, 400).Throw()
	}

	table.loadColumns()
	table.loadFilters()
	table.loadAPIs()
	Tables[name] = &table
	return Tables[name]
}

// Select 读取已加载表格配置
func Select(name string) *Table {
	tab, has := Tables[name]
	if !has {
		exception.New(
			fmt.Sprintf("Table:%s; 尚未加载", name),
			400,
		).Throw()
	}
	return tab
}

// Reload 更新数据表格配置
func (tab *Table) Reload() *Table {
	tab = Load(tab.Source, tab.Name)
	return tab
}
