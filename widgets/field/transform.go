package field

import (
	"fmt"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/any"
)

// Transforms opend transform
var Transforms = map[string]*Transform{}

// OpenTransform open the transform from source
func OpenTransform(data []byte, name string) (*Transform, error) {
	trans := Transform{}
	err := jsoniter.Unmarshal(data, &trans)
	if err != nil {
		return nil, err
	}
	Transforms[name] = &trans
	return Transforms[name], nil
}

// IsNotFound check if the given error is not found
func IsNotFound(err error) bool {
	return strings.Contains(err.Error(), "does not found")
}

// Filter transform to filter
func (t *Transform) Filter(typeName string, data map[string]interface{}) (*FilterDSL, error) {

	if alias, has := t.Aliases[typeName]; has {
		typeName = alias
	}

	field, has := t.Fields[typeName]
	if !has {
		return nil, fmt.Errorf("%s does not found", typeName)
	}

	if field.Filter == nil {
		return nil, fmt.Errorf("%s.filter does not found", typeName)
	}

	return t.filter(field.Filter, data)
}

// Table transform to table
func (t *Transform) Table(typeName string, data map[string]interface{}) (*ColumnDSL, error) {

	if alias, has := t.Aliases[typeName]; has {
		typeName = alias
	}

	field, has := t.Fields[typeName]
	if !has {
		return nil, fmt.Errorf("%s does not found", typeName)
	}

	if field.Table == nil {
		return nil, fmt.Errorf("%s.table does not found", typeName)
	}

	return t.column(field.Table, data)
}

// Form transform to form
func (t *Transform) Form(typeName string, data map[string]interface{}) (*ColumnDSL, error) {

	if alias, has := t.Aliases[typeName]; has {
		typeName = alias
	}

	field, has := t.Fields[typeName]
	if !has {
		return nil, fmt.Errorf("%s does not found", typeName)
	}

	if field.Form == nil {
		return nil, fmt.Errorf("%s.form does not found", typeName)
	}

	return t.column(field.Form, data)
}

// trans transform to form/table
func (t *Transform) column(column *ColumnDSL, data map[string]interface{}) (*ColumnDSL, error) {
	if _, has := data["variables"]; !has {
		variables := any.Of(t.Variables).Map().MapStrAny.Dot()
		for k, v := range variables {
			data[k] = v
		}
	}

	new := column.Clone()
	return new.Replace(data)
}

// trans transform to filter
func (t *Transform) filter(filter *FilterDSL, data map[string]interface{}) (*FilterDSL, error) {
	if _, has := data["variables"]; !has {
		variables := any.Of(t.Variables).Map().MapStrAny.Dot()
		for k, v := range variables {
			data[k] = v
		}
	}
	new := filter.Clone()
	return new.Replace(data)
}
