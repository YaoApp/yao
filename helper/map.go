package helper

import "github.com/yaoapp/kun/maps"

// MapValues 返回映射表的数值
func MapValues(record map[string]interface{}) []interface{} {
	values := []interface{}{}
	for _, value := range record {
		values = append(values, value)
	}
	return values
}

// MapKeys 返回映射表的键
func MapKeys(record map[string]interface{}) []string {
	keys := []string{}
	for key := range record {
		keys = append(keys, key)
	}
	return keys
}

// MapGet xiang.helper.MapGet 返回映射表给定键的值
func MapGet(record map[string]interface{}, key string) interface{} {
	data := maps.MapOf(record).Dot()
	return data.Get(key)
}

// MapSet xiang.helper.MapSet 设定数值并返回新映射表
func MapSet(record map[string]interface{}, key string, value interface{}) map[string]interface{} {
	record[key] = value
	return record
}

// MapDel xiang.helper.MapDel 删除数值并返回新映射表
func MapDel(record map[string]interface{}, key string) map[string]interface{} {
	delete(record, key)
	return record
}

// MapMultiDel xiang.helper.MapMultiDel 删除数值并返回新映射表
func MapMultiDel(record map[string]interface{}, keys ...string) map[string]interface{} {
	for _, key := range keys {
		delete(record, key)
	}
	return record
}
