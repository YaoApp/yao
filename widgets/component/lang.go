package component

// Trans trans
func (column *DSL) Trans(widgetName string, inst string, trans func(widget string, inst string, value *string) bool) bool {
	return transMap(column.Props, widgetName, inst, trans)
}

func transMap(value map[string]interface{}, widgetName string, inst string, trans func(widget string, inst string, value *string) bool) bool {
	res := false
	for key, val := range value {

		switch val.(type) {
		case map[string]interface{}:
			if transMap(val.(map[string]interface{}), widgetName, inst, trans) {
				res = true
			}
			break

		case []interface{}:
			if transArr(val.([]interface{}), widgetName, inst, trans) {
				res = true
			}
			break

		case string:
			new := val.(string)
			if trans(widgetName, inst, &new) {
				val = new
				res = true
			}
			break
		}

		if trans(widgetName, inst, &key) {
			res = true
		}

		value[key] = val
	}

	return res
}

func transArr(value []interface{}, widgetName string, inst string, trans func(widget string, inst string, value *string) bool) bool {
	res := false
	for idx, val := range value {

		switch val.(type) {
		case map[string]interface{}:
			if transMap(val.(map[string]interface{}), widgetName, inst, trans) {
				res = true
			}
			break

		case []interface{}:
			if transArr(val.([]interface{}), widgetName, inst, trans) {
				res = true
			}
			break

		case string:
			new := val.(string)
			if trans(widgetName, inst, &new) {
				val = new
				res = true
			}
			break
		}

		value[idx] = val
	}

	return res
}
