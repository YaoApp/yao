package component

import (
	"fmt"
	"regexp"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/query"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/utils"
)

var varRe = regexp.MustCompile(`\[\[\s*\$([A-Za-z0-9_\-]+)\s*\]\]`)

// QueryProp query prop
type QueryProp struct {
	Debug       bool                     `json:"debug,omitempty"`
	Engine      string                   `json:"engine"`
	From        string                   `json:"from"`
	LabelField  string                   `json:"labelField,omitempty"`
	ValueField  string                   `json:"valueField,omitempty"`
	IconField   string                   `json:"iconField,omitempty"`
	ColorField  string                   `json:"colorField,omitempty"`
	LabelFormat string                   `json:"labelFormat,omitempty"`
	ValueFormat string                   `json:"valueFormat,omitempty"`
	IconFormat  string                   `json:"iconFormat,omitempty"`
	ColorFormat string                   `json:"colorFormat,omitempty"`
	Wheres      []map[string]interface{} `json:"wheres,omitempty"`
	param       model.QueryParam
	dsl         map[string]interface{}
	props       map[string]interface{}
}

// Option select option
type Option struct {
	Label string      `json:"label"`
	Value interface{} `json:"value"`
	Icon  string      `json:"icon,omitempty"`
	Color string      `json:"color,omitempty"`
}

// Export process
func exportProcess() {
	process.Register("yao.component.getoptions", processGetOptions)
	process.Register("yao.component.selectoptions", processSelectOptions) // Deprecated
}

// processGetOptions get options
func processGetOptions(process *process.Process) interface{} {

	process.ValidateArgNums(2)
	params := process.ArgsMap(0, map[string]interface{}{})
	props := process.ArgsMap(1, map[string]interface{}{})

	// Paser props
	p, err := parseOptionsProps(params, props)
	if err != nil {
		exception.New(err.Error(), 400).Throw()
	}

	// Using the query DSL
	options := []Option{}
	if p.Engine != "" {
		engine, err := query.Select(p.Engine)
		if err != nil {
			exception.New(err.Error(), 400).Throw()
		}

		if p.Debug {
			fmt.Println("")
			fmt.Println("-- yao.Component.GetOptions Debug ----------------------")
			fmt.Println("Params: ")
			utils.Dump(params)
			fmt.Println("Engine: ", p.Engine)
			fmt.Println("QueryDSL: ")
			utils.Dump(p.dsl)
		}

		qb, err := engine.Load(p.dsl)
		if err != nil {
			exception.New(err.Error(), 400).Throw()
		}

		// Query the data
		data := qb.Get(params)
		if p.Debug {
			fmt.Println("Query Result: ")
			utils.Dump(data)
		}

		for _, row := range data {
			p.format(&options, row)
		}

		if p.Debug {
			fmt.Println("Options: ")
			utils.Dump(options)
			fmt.Println("-------------------------------------------------------")
		}

		return options
	}

	// Using the QueryParam
	if p.Debug {
		fmt.Println("")
		fmt.Println("-- yao.Component.GetOptions Debug ----------------------")
		fmt.Println("Params: ")
		utils.Dump(params)

		fmt.Println("QueryParam: ")
		utils.Dump(p.param)
	}

	// Query param
	m := model.Select(p.From)
	data, err := m.Get(p.param)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	if p.Debug {
		fmt.Println("Query Result: ")
		utils.Dump(data)
	}

	// Format the data
	for _, row := range data {
		p.format(&options, row)
	}

	if p.Debug {
		fmt.Println("Options: ")
		utils.Dump(options)
		fmt.Println("-------------------------------------------------------")
	}

	return options
}

// parseOptionsProps parse options props
func parseOptionsProps(params, props map[string]interface{}) (*QueryProp, error) {
	if props["query"] == nil {
		exception.New("props.query is required", 400).Throw()
	}

	// Read props
	if v, ok := props["query"].(map[string]interface{}); ok {
		props = v
	}

	raw, err := jsoniter.Marshal(props)
	if err != nil {
		return nil, err
	}

	qprops := QueryProp{}
	err = jsoniter.Unmarshal(raw, &qprops)
	if err != nil {
		return nil, err
	}

	qprops.props = props
	err = qprops.parse(params)
	if err != nil {
		return nil, err
	}
	return &qprops, nil
}

// format format the option
func (q *QueryProp) format(options *[]Option, row map[string]interface{}) {
	label := row[q.LabelField]
	value := row[q.ValueField]
	option := Option{Label: fmt.Sprintf("%v", label), Value: value}
	if q.IconField != "" {
		option.Icon = fmt.Sprintf("%v", row[q.IconField])
	}

	if q.ColorField != "" {
		option.Color = fmt.Sprintf("%v", row[q.ColorField])
	}

	if q.LabelFormat != "" {
		option.Label = q.replaceString(q.LabelFormat, row)
	}

	if q.ValueFormat != "" {
		option.Value = q.replaceString(q.ValueFormat, row)
	}

	if q.IconField != "" && q.IconFormat != "" {
		option.Icon = q.replaceString(q.IconFormat, row)
	}

	if q.ColorField != "" && q.ColorFormat != "" {
		option.Color = q.replaceString(q.ColorFormat, row)
	}

	// Update the option
	*options = append(*options, option)
}

func (q *QueryProp) parse(query map[string]interface{}) error {
	if q.Wheres == nil {
		q.Wheres = []map[string]interface{}{}
	}

	if query == nil {
		query = map[string]interface{}{}
	}

	// Validate the query param required fields
	if q.Engine == "" {
		if q.From == "" {
			return fmt.Errorf("props.from is required")
		}
		if q.LabelField == "" {
			return fmt.Errorf("props.labelField is required")
		}
		if q.ValueField == "" {
			return fmt.Errorf("props.valueField is required")
		}
	}

	// Parse wheres
	wheres := []map[string]interface{}{}
	for _, where := range q.Wheres {
		if q.replaceWhere(where, query) {
			wheres = append(wheres, where)
		}
	}

	// Update the props
	props := map[string]interface{}{}
	for key, value := range q.props {
		props[key] = value
	}
	props["wheres"] = wheres

	// Return the query dsl, if the engine is set
	if q.Engine != "" {

		if q.LabelField == "" {
			q.LabelField = "label"
		}

		if q.ValueField == "" {
			q.ValueField = "value"
		}

		if q.IconField == "" {
			q.IconField = "icon"
		}

		q.dsl = props
		return nil
	}

	// Parse the query param from the props
	q.param = model.QueryParam{
		Model:  q.From,
		Select: []interface{}{q.LabelField, q.ValueField},
	}

	if q.IconField != "" {
		q.param.Select = append(q.param.Select, q.IconField)
	}

	if q.ColorField != "" {
		q.param.Select = append(q.param.Select, q.ColorField)
	}

	raw, err := jsoniter.Marshal(props)
	if err != nil {
		return err
	}

	err = jsoniter.Unmarshal(raw, &q.param)
	if err != nil {
		return err
	}

	return nil
}

func (q *QueryProp) replaceString(format string, data map[string]interface{}) string {
	if data == nil {
		return format
	}

	matches := varRe.FindAllStringSubmatch(format, -1)
	if len(matches) > 0 {
		for _, match := range matches {
			name := match[1]
			orignal := match[0]
			if val, ok := data[name]; ok {
				format = strings.ReplaceAll(format, orignal, fmt.Sprintf("%v", val))
			}
		}
	}

	return format
}

// Replace replace the query where condition
// return true if the where condition is effective, otherwise return false
func (q *QueryProp) replaceWhere(where map[string]interface{}, data map[string]interface{}) bool {
	if where == nil {
		return false
	}

	for key, value := range where {
		if v, ok := value.(string); ok {
			matches := varRe.FindAllStringSubmatch(v, -1)
			if len(matches) > 0 {
				orignal := matches[0][0]
				name := matches[0][1]
				if val, ok := data[name]; ok {

					// Check if the value is empty
					if val == nil || val == "" {
						return false
					}

					if q.Engine == "" {
						where[key] = val
						// Replace the value
						if v, ok := val.(string); ok {
							where[key] = strings.Replace(v, orignal, fmt.Sprintf("%v", val), 1)
						}
						return true
					}

					// Where in condition for the query dsl
					if where["in"] != nil {
						where[key] = val
						return true
					}

					// Replace the value
					where[key] = strings.Replace(v, orignal, fmt.Sprintf("?:%v", name), 1) // SQL injection protection
					return true
				}
				return false
			}
		}
	}

	return true
}

// Deprecated: please use processGetOptions instead
// This function may cause security issue, please use processGetOptions instead
// It will be removed when the v0.10.4 released
func processSelectOptions(process *process.Process) interface{} {
	message := "process yao.component.SelectOptions is deprecated, please use yao.component.GetOptions instead"
	exception.New(message, 400).Throw()
	return nil
}
