package helper

import (
	"fmt"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/exception"
)

// ArrayPluckValue ArrayPluck 参数
type ArrayPluckValue struct {
	Key   string                   `json:"key"`
	Value string                   `json:"value"`
	Items []map[string]interface{} `json:"items"`
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
// 	columns: ["城市", "行业", "计费"]
// 	pluck: {
// 		"行业":{"key":"city", "value":"数量", "items":[{"city":"北京", "数量":32},{"city":"上海", "数量":20}]},
// 		"计费":{"key":"city", "value":"计费种类", "items":[{"city":"北京", "计费种类":6},{"city":"西安", "计费种类":3}]},
//  }
// return: [
// 		{"城市":"北京", "行业":32, "计费":6},
// 		{"城市":"上海", "行业":20, "计费":null},
// 		{"城市":"西安", "行业":null, "计费":6}
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
