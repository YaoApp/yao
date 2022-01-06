package table

// // UnmarshalJSON for json marshalJSON
// func (table *Table) UnmarshalJSON(data []byte) error {

// 	var v interface{}
// 	err := jsoniter.Unmarshal(data, &v)
// 	if err != nil {
// 		return err
// 	}

// 	// values := []interface{}{}
// 	// switch v.(type) {
// 	// case string: // "kind rollup 所有类型, city"
// 	// 	strarr := strings.Split(v.(string), ",")
// 	// 	for _, str := range strarr {
// 	// 		groups.PushString(str)
// 	// 	}
// 	// 	break
// 	// case []interface{}: // ["name", {"field":"foo"}, "id rollup 所有类型"]
// 	// 	values = v.([]interface{})
// 	// 	break
// 	// }

// 	// for _, value := range values {
// 	// 	switch value.(type) {
// 	// 	case string: // "kind rollup 所有类型"
// 	// 		groups.PushString(value.(string))
// 	// 		break
// 	// 	case map[string]interface{}: // {"field":"foo"}
// 	// 		groups.PushMap(value.(map[string]interface{}))
// 	// 		break
// 	// 	}
// 	// }

// 	return nil
// }

// // MarshalJSON for json marshalJSON
// func (table Table) MarshalJSON() ([]byte, error) {
// 	return jsoniter.Marshal(nil)
// }
