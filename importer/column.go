package importer

import (
	"fmt"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

// MarshalJSON for json marshalJSON
func (column Column) MarshalJSON() ([]byte, error) {
	data := column.ToMap()
	return jsoniter.Marshal(data)
}

// UnmarshalJSON for json marshalJSON
func (column *Column) UnmarshalJSON(source []byte) error {
	var data = map[string]interface{}{}
	err := jsoniter.Unmarshal(source, &data)
	if err != nil {
		return err
	}

	new, err := ColumnOf(data)
	if err != nil {
		return err
	}

	*column = *new
	return nil
}

// ColumnOf 映射表转换为字段定义
func ColumnOf(data map[string]interface{}) (*Column, error) {
	var column = &Column{}

	err := column.setLabel(data)
	if err != nil {
		return nil, err
	}

	err = column.setName(data)
	if err != nil {
		return nil, err
	}

	err = column.setMatch(data)
	if err != nil {
		return nil, err
	}

	err = column.setRules(data)
	if err != nil {
		return nil, err
	}

	if primary, ok := data["primary"].(bool); ok {
		column.Primary = primary
	}

	if nullable, ok := data["nullable"].(bool); ok {
		column.Nullable = nullable
	}

	return column, nil
}

// ToMap 转换为映射表
func (column Column) ToMap() map[string]interface{} {

	data := map[string]interface{}{
		"name":  column.Field,
		"label": column.Label,
		"match": column.Match,
		"rules": column.Rules,
	}

	if column.Nullable {
		data["nullable"] = true
	}

	if column.Primary {
		data["primary"] = true
	}

	return data
}

// setRules 设置清洗规则
func (column *Column) setRules(data map[string]interface{}) error {
	rules, err := GetArrayString(data, "rules")
	if err != nil {
		return err
	}

	// 检查 process 是否存在

	column.Rules = rules
	return nil
}

// setLabel 读取并设置字段标签
func (column *Column) setLabel(data map[string]interface{}) error {
	label, err := GetString(data, "label", true)
	if err != nil {
		return err
	}
	column.Label = label
	return nil
}

// setMatch 读取并设置字段名称
func (column *Column) setMatch(data map[string]interface{}) error {
	match, err := GetArrayString(data, "match")
	if err != nil {
		return err
	}
	column.Match = match
	return nil
}

// setName 读取并设置字段名称
func (column *Column) setName(data map[string]interface{}) error {
	name, err := GetString(data, "name", true)
	if err != nil {
		return err
	}

	column.Field = name // 留存原始数值

	if strings.Contains(name, "[*]") { // Array
		namer := strings.Split(name, "[*]")
		name = namer[0]
		column.IsArray = true
		name = strings.Join(namer, "")
	}

	if strings.Contains(name, ".") { // Object
		namer := strings.Split(name, ".")
		name = namer[0]
		if len(namer) > 1 {
			column.IsObject = true
			column.Key = strings.Join(namer[1:], ".")
		}
	}

	column.Name = name
	return nil
}

// GetString 读取字符串格式
func GetString(data map[string]interface{}, key string, required bool) (string, error) {
	value, ok := data[key].(string)
	if !ok {
		if bytes, isok := data[key].([]byte); isok {
			ok = isok
			value = string(bytes)
		}
	}
	if !ok || (value == "" && required) {
		return "", ErrorF("the %s format is incorrect", key)
	}
	return value, nil
}

// GetArrayString 读取字符串数组
func GetArrayString(data map[string]interface{}, key string) ([]string, error) {
	value := []string{}

	if data[key] == nil {
		return value, nil
	}

	if v, ok := data[key].(string); ok {
		return []string{v}, nil
	}

	value, ok := data[key].([]string)
	if !ok {
		if anys, isok := data[key].([]interface{}); isok {
			ok = isok
			for _, any := range anys {
				value = append(value, fmt.Sprintf("%v", any))
			}
		}

	}

	if !ok {
		if anys, isok := data[key].([][]byte); isok {
			ok = isok
			for _, any := range anys {
				value = append(value, string(any))
			}
		}
	}

	if !ok {
		return nil, ErrorF("the %s format is incorrect", key)
	}
	return value, nil
}

// ErrorF 返回错误数据对象
func ErrorF(format string, data ...interface{}) error {
	values := []interface{}{}
	for _, value := range data {
		v, _ := jsoniter.Marshal(value)
		values = append(values, v)
	}
	return fmt.Errorf(format, values...)
}
