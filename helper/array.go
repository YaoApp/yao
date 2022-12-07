package helper

import (
	"fmt"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
)

// ArrayPluckValue ArrayPluck 参数
type ArrayPluckValue struct {
	Key   string                   `json:"key"`
	Value string                   `json:"value"`
	Items []map[string]interface{} `json:"items"`
}

// ArrayTreeOption Array转树形结构参数表
type ArrayTreeOption struct {
	Key      string      `json:"id"`       // 主键名称, 默认为 id
	Empty    interface{} `json:"empty"`    // Top节点 parent 数值, 默认为 0
	Parent   string      `json:"parent"`   // 父节点字段名称, 默认为 parent
	Children string      `json:"children"` // 子节点字段名称, 默认为 children
}

// ArrayColumn 返回多条数据记录，指定字段数值。
func ArrayColumn(records []map[string]interface{}, name string) []interface{} {
	values := []interface{}{}
	for _, record := range records {
		values = append(values, record[name])
	}
	return values
}

// ArrayKeep 仅保留指定键名的数据
func ArrayKeep(records []map[string]interface{}, keeps []string) []map[string]interface{} {
	values := []map[string]interface{}{}
	for _, record := range records {
		value := map[string]interface{}{}
		for _, keep := range keeps {
			value[keep] = record[keep]
		}
		values = append(values, value)
	}
	return values
}

// ArraySplit 将多条数记录集合，分解为一个 columns:[]string 和 values: [][]interface{}
func ArraySplit(records []map[string]interface{}) ([]string, [][]interface{}) {
	columns := []string{}
	values := [][]interface{}{}
	if len(records) == 0 {
		return columns, values
	}
	for column := range records[0] {
		columns = append(columns, column)
	}

	for _, record := range records {
		value := []interface{}{}
		for _, key := range columns {
			value = append(value, record[key])
		}
		values = append(values, value)
	}
	return columns, values
}

// ArrayPluck 将多个数据记录集合，合并为一个数据记录集合
//
//		columns: ["城市", "行业", "计费"]
//		pluck: {
//			"行业":{"key":"city", "value":"数量", "items":[{"city":"北京", "数量":32},{"city":"上海", "数量":20}]},
//			"计费":{"key":"city", "value":"计费种类", "items":[{"city":"北京", "计费种类":6},{"city":"西安", "计费种类":3}]},
//	 }
//
// return: [
//
//	{"城市":"北京", "行业":32, "计费":6},
//	{"城市":"上海", "行业":20, "计费":null},
//	{"城市":"西安", "行业":null, "计费":6}
//
// ]
func ArrayPluck(columns []string, pluck map[string]interface{}) []map[string]interface{} {
	if len(columns) < 2 {
		exception.New("ArrayPluck 参数错误, 应至少包含两列。", 400).Ctx(columns).Throw()
	}

	primary := columns[0]
	data := map[string]map[string]interface{}{}

	// 解析数据
	for name, value := range pluck { // name="行业", value={"key":"city", "value":"数量", "items":[{"city":"北京", "数量":32},{"city":"上海", "数量":20}]},
		arg := OfArrayPluckValue(value)
		for _, item := range arg.Items { // item = [{"city":"北京", "数量":32},{"city":"上海", "数量":20}]
			if v, has := item[arg.Key]; has { // arg.Key = "city"
				key := fmt.Sprintf("%#v", v) // key = `"北京"`
				val := item[arg.Value]       // arg.Value = "数量",  val = 32
				if _, has := data[key]; !has {
					data[key] = map[string]interface{}{} // {`"北京"`: {}}
					data[key][primary] = v               // {`"北京"`: {"城市":"北京"}}
				}
				data[key][name] = val // {`"北京"`: {"城市":"北京", "行业":32}}
			}
		}
	}

	// 空值处理
	res := []map[string]interface{}{}
	for key := range data { // key = `"北京"`
		for name := range pluck { // name = "行业"
			if _, has := data[key][name]; !has {
				data[key][name] = nil
			}
		}
		res = append(res, data[key])
	}

	return res
}

// ArrayUnique 数组排重
func ArrayUnique(columns []interface{}) []interface{} {
	res := []interface{}{}
	m := make(map[string]bool)
	for _, val := range columns {
		key := fmt.Sprintf("%v", val)
		if _, ok := m[key]; !ok {
			m[key] = true
			res = append(res, val)
		}
	}
	return res
}

// ArrayStringUnique 数组排重
func ArrayStringUnique(columns []string) []string {
	res := []string{}
	m := make(map[string]bool)
	for _, key := range columns {
		if _, ok := m[key]; !ok {
			m[key] = true
			res = append(res, key)
		}
	}
	return res
}

// OfArrayPluckValue Any 转 ArrayPluckValue
func OfArrayPluckValue(any interface{}) ArrayPluckValue {
	content, err := jsoniter.Marshal(any)
	if err != nil {
		exception.New("ArrayPluck 参数错误", 400).Ctx(err.Error()).Throw()
	}
	value := ArrayPluckValue{Items: []map[string]interface{}{}}
	err = jsoniter.Unmarshal(content, &value)
	if err != nil {
		exception.New("ArrayPluck 参数错误", 400).Ctx(err.Error()).Throw()
	}
	return value
}

// NewArrayTreeOption 创建配置
func NewArrayTreeOption(option map[string]interface{}) ArrayTreeOption {

	new := ArrayTreeOption{
		Empty:    0,
		Key:      "id",
		Parent:   "parent",
		Children: "children",
	}

	if v, ok := option["empty"]; ok {
		new.Empty = v
	}

	if v, ok := option["parent"].(string); ok {
		new.Parent = v
	}

	if v, ok := option["primary"].(string); ok {
		new.Key = v
	}

	if v, ok := option["children"].(string); ok {
		new.Children = v
	}
	return new
}

// ArrayTree []map[string]interface{} 转树形结构
func ArrayTree(records []map[string]interface{}, setting map[string]interface{}) []map[string]interface{} {
	opt := NewArrayTreeOption(setting)
	return opt.Tree(records)
}

// Tree Array 转换为 Tree
func (opt ArrayTreeOption) Tree(records []map[string]interface{}) []map[string]interface{} {

	mapping := map[string]map[string]interface{}{}
	for i := range records {
		if key, has := records[i][opt.Key]; has {
			primary := fmt.Sprintf("%v", key)
			mapping[primary] = map[string]interface{}{}
			mapping[primary][opt.Children] = []map[string]interface{}{}
			for k, v := range records[i] {
				mapping[primary][k] = v
			}
		}
	}

	// 向上归集
	for key, record := range mapping {
		parent := fmt.Sprintf("%v", record[opt.Parent])
		empty := fmt.Sprintf("%v", opt.Empty)
		if parent == empty { // 第一级
			continue
		}
		pKey := fmt.Sprintf("%v", parent)
		if _, has := mapping[pKey]; !has {
			continue
		}
		children, ok := mapping[pKey][opt.Children].([]map[string]interface{})
		if !ok {
			children = []map[string]interface{}{}
		}
		children = append(children, mapping[key])
		mapping[pKey][opt.Children] = children
	}

	res := []map[string]interface{}{}
	for i := range records {
		if key, has := records[i][opt.Key]; has {
			record := mapping[fmt.Sprintf("%v", key)]
			if pValue, has := record[opt.Parent]; has {
				parent := fmt.Sprintf("%v", pValue)
				empty := fmt.Sprintf("%v", opt.Empty)
				if parent == empty { // 父类为空
					res = append(res, record)
				} else if _, has := mapping[parent]; !has { // 或者父类为定义的
					res = append(res, record)
				}
			}
		}
	}
	return res
}

// ArrayMapSet []map[string]interface{} 设定数值
func ArrayMapSet(records []map[string]interface{}, key string, value interface{}) []map[string]interface{} {
	res := []map[string]interface{}{}
	for i := range records {
		record := records[i]
		record[key] = value
		res = append(res, record)
	}
	return res
}

// ArrayMapSetMapStr []map[string]interface{} 设定数值
func ArrayMapSetMapStr(records []maps.MapStr, key string, value interface{}) []maps.MapStr {
	res := []maps.MapStr{}
	for i := range records {
		record := records[i]
		record[key] = value
		res = append(res, record)
	}
	return res
}
