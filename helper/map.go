package helper

// MapValues 返回映射的数值
func MapValues(record map[string]interface{}) []interface{} {
	values := []interface{}{}
	for _, value := range record {
		values = append(values, value)
	}
	return values
}

// MapKeys 返回映射的键
func MapKeys(record map[string]interface{}) []string {
	keys := []string{}
	for key := range record {
		keys = append(keys, key)
	}
	return keys
}
