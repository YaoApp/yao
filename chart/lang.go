package chart

// Lang for applying a language pack
func (chart *Chart) Lang(trans func(widget string, inst string, value *string) bool) {
	inst := chart.Flow.Name
	widget := "chart"

	trans(widget, inst, &chart.Name)
	trans(widget, inst, &chart.Label)
	trans(widget, inst, &chart.Description)
	trans(widget, inst, &chart.Page.Primary)
	transMap(widget, inst, chart.Page.Layout, trans)
	chart.Output = transAny(widget, inst, chart.Output, trans)

	// Filters
	for name, filter := range chart.Filters {
		new := name
		trans(widget, inst, &new)
		trans(widget, inst, &filter.Label)
		delete(chart.Filters, name)

		// Props
		transMap(widget, inst, filter.Input.Props, trans)
		chart.Filters[new] = filter
	}

}

func transAny(widget string, inst string, input interface{}, trans func(widget string, inst string, value *string) bool) interface{} {
	switch input.(type) {
	case []interface{}:
		values := input.([]interface{})
		transArr(widget, inst, values, trans)
		input = values
		break

	case map[string]interface{}:
		values := input.(map[string]interface{})
		for name, value := range values {
			new := name
			newValue := value

			switch value.(type) {
			case string:
				val := value.(string)
				trans(widget, inst, &val)
				newValue = val
				break

			case []interface{}:
				vals := value.([]interface{})
				transArr(widget, inst, vals, trans)
				newValue = vals
				break

			case map[string]interface{}:
				vals := value.(map[string]interface{})
				transMap(widget, inst, vals, trans)
				newValue = vals
				break
			}

			trans(widget, inst, &new)
			delete(values, name)
			values[new] = newValue
		}
		input = values
		break
	}

	return input
}

func transMap(widget string, inst string, values map[string]interface{}, trans func(widget string, inst string, value *string) bool) {
	for key, value := range values {

		switch value.(type) {

		case string:
			v := value.(string)
			trans(widget, inst, &v)
			values[key] = v
			break

		case []interface{}:
			v := value.([]interface{})
			transArr(widget, inst, v, trans)
			values[key] = v
			break

		case map[string]interface{}:
			v := value.(map[string]interface{})
			transMap(widget, inst, v, trans)
			values[key] = v
			break
		}

	}
}

func transArr(widget string, inst string, values []interface{}, trans func(widget string, inst string, value *string) bool) {
	for key, value := range values {
		switch value.(type) {

		case string:
			v := value.(string)
			trans(widget, inst, &v)
			values[key] = v
			break

		case []interface{}:
			v := value.([]interface{})
			transArr(widget, inst, v, trans)
			values[key] = v
			break

		case map[string]interface{}:
			v := value.(map[string]interface{})
			transMap(widget, inst, v, trans)
			values[key] = v
			break
		}

	}
}
