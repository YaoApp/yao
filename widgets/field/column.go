package field

import (
	"fmt"
	"io"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/yao/widgets/component"
	"github.com/yaoapp/yao/widgets/expression"
	"golang.org/x/crypto/md4"
)

// UnmarshalJSON for json UnmarshalJSON
func (column *ColumnDSL) UnmarshalJSON(data []byte) error {
	var alias aliasColumnDSL
	err := jsoniter.Unmarshal(data, &alias)
	if err != nil {
		return err
	}

	*column = ColumnDSL(alias)
	column.ID, err = column.Hash()
	if err != nil {
		return err
	}

	return nil
}

// Hash hash value
func (column ColumnDSL) Hash() (string, error) {
	h := md4.New()
	origin := fmt.Sprintf("COLUMN::%#v", column.Map())
	// fmt.Println(origin)
	io.WriteString(h, origin)
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// Replace replace with data
func (column ColumnDSL) Replace(data map[string]interface{}) (*ColumnDSL, error) {
	new := column
	err := expression.Replace(&new.Key, data)
	if err != nil {
		return nil, err
	}

	err = expression.Replace(&new.Bind, data)
	if err != nil {
		return nil, err
	}

	if new.Edit != nil {
		err = expression.Replace(&new.Edit.Props, data)
		if err != nil {
			return nil, err
		}
	}

	if new.View != nil {
		err = expression.Replace(&new.View.Props, data)
		if err != nil {
			return nil, err
		}
	}

	new.ID, err = column.Hash()
	if err != nil {
		return nil, err
	}

	return &new, nil
}

// ViewBind get the bind field name of view
func (column ColumnDSL) ViewBind() string {
	if column.View != nil && column.View.Bind != "" {
		return column.View.Bind
	}
	return column.Bind
}

// EditBind get the bind field name of edit
func (column ColumnDSL) EditBind() string {
	if column.Edit != nil && column.Edit.Bind != "" {
		return column.Edit.Bind
	}
	return column.Bind
}

// Parse the column dsl, add the default value, and parse the backend only props
func (column ColumnDSL) Parse() {
	if column.View != nil {
		column.View.Parse()
	}
	if column.Edit != nil {
		column.Edit.Parse()
	}
}

// Clone column
func (column *ColumnDSL) Clone() *ColumnDSL {
	new := ColumnDSL{
		Key:  column.Key,
		Bind: column.Bind,
		Link: column.Link,
	}

	if column.View != nil {
		new.View = column.View.Clone()
	}

	if column.Edit != nil {
		new.Edit = column.Edit.Clone()
	}
	return &new
}

// Map cast to map[string]inteface{}
func (column ColumnDSL) Map() map[string]interface{} {
	res := map[string]interface{}{
		"id":   column.ID,
		"bind": column.Bind,
	}

	if column.HideLabel {
		res["hideLabel"] = true
	}

	if column.Data != nil {
		res["data"] = map[string]interface{}{"process": column.Data.Process, "query": column.Data.Query}
	}

	if column.Link != "" {
		res["link"] = column.Link
	}

	if column.View != nil {
		res["view"] = column.View.Map()
	}

	if column.Edit != nil {
		res["edit"] = column.Edit.Map()
	}
	return res
}

// CPropsMerge merge the Columns cloud props
func (columns Columns) CPropsMerge(cloudProps map[string]component.CloudPropsDSL, getXpath func(name string, kind string, column ColumnDSL) (xpath string)) error {

	for name, column := range columns {

		if column.Data != nil {
			xpath := getXpath(name, "data", column)
			cProps := map[string]component.CloudPropsDSL{}
			cprop := *column.Data
			cprop.Xpath = xpath
			cprop.Name = "data"
			cprop.Type = "data"
			fullname := fmt.Sprintf("%s.$data", xpath)
			cProps[fullname] = cprop
			mergeCProps(cloudProps, cProps)
		}

		if column.Edit != nil && column.Edit.Props != nil {
			xpath := getXpath(name, "edit", column)
			cProps, err := column.Edit.Props.CloudProps(xpath, column.Edit.Type)
			if err != nil {
				return err
			}

			mergeCProps(cloudProps, cProps)
		}

		if column.View != nil && column.View.Props != nil {
			xpath := getXpath(name, "view", column)
			cProps, err := column.View.Props.CloudProps(xpath, column.View.Type)
			if err != nil {
				return err
			}
			mergeCProps(cloudProps, cProps)
		}
	}

	return nil
}

func mergeCProps(cloudProps map[string]component.CloudPropsDSL, cProps map[string]component.CloudPropsDSL) {
	for k, v := range cProps {
		cloudProps[k] = v
	}
}
