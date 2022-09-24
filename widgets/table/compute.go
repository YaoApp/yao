package table

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
)

func (dsl *DSL) computeSearch(process *gou.Process, res map[string]interface{}, itemKey string) error {
	if data, has := res[itemKey]; has {
		if arr, ok := data.([]interface{}); ok {
			err := dsl.computeGet(process, arr)
			if err != nil {
				return err
			}
			res[itemKey] = arr
		}
	}
	return nil
}

func (dsl *DSL) computeGet(process *gou.Process, data []interface{}) error {

	messages := []string{}
	for idx, row := range data {
		switch row.(type) {
		case map[string]interface{}, maps.MapStr:
			rowMap := any.MapOf(row).MapStrAny
			err := dsl.computeFind(process, rowMap)
			if err != nil {
				messages = append(messages, err.Error())
			}
			data[idx] = rowMap
		}
	}

	if len(messages) > 0 {
		return fmt.Errorf("%s", strings.Join(messages, ";"))
	}
	return nil
}

func (dsl *DSL) computeFind(process *gou.Process, values map[string]interface{}) error {

	messages := []string{}
	for key := range values {
		err := dsl.computeOut(process, key, values)
		if err != nil {
			messages = append(messages, err.Error())
		}
	}

	if len(messages) > 0 {
		return fmt.Errorf("%s", strings.Join(messages, ";"))
	}

	return nil
}

func (dsl *DSL) computeSave(process *gou.Process, values map[string]interface{}) error {

	messages := []string{}
	for key := range values {
		err := dsl.computeIn(process, key, values)
		if err != nil {
			messages = append(messages, err.Error())
		}
	}

	if len(messages) > 0 {
		return fmt.Errorf("%s", strings.Join(messages, ";"))
	}

	return nil
}

func (dsl *DSL) computeIn(process *gou.Process, key string, values map[string]interface{}) error {
	if name, has := dsl.ComputesIn[key]; has {
		compute, err := gou.ProcessOf(name, key, values[key], values)
		if err != nil {
			log.Error("[table] %s compute-in -> %s %s %s", dsl.ID, name, key, err.Error())
			return fmt.Errorf("[table] %s compute-in -> %s %s %s", dsl.ID, name, key, err.Error())
		}

		res, err := compute.WithGlobal(process.Global).WithSID(process.Sid).Exec()
		if err != nil {
			log.Error("[table] %s compute-in -> %s %s %s", dsl.ID, name, key, err.Error())
			return fmt.Errorf("[table] %s compute-in -> %s %s %s", dsl.ID, name, key, err.Error())
		}
		values[key] = res
	}
	return nil
}

func (dsl *DSL) computeOut(process *gou.Process, key string, values map[string]interface{}) error {
	if name, has := dsl.ComputesOut[key]; has {
		compute, err := gou.ProcessOf(name, key, values[key], values)
		if err != nil {
			log.Error("[table] %s compute-out -> %s %s %s", dsl.ID, name, key, err.Error())
			return fmt.Errorf("[table] %s compute-out -> %s %s %s", dsl.ID, name, key, err.Error())
		}

		res, err := compute.WithGlobal(process.Global).WithSID(process.Sid).Exec()
		if err != nil {
			log.Error("[table] %s compute-out -> %s %s %s", dsl.ID, name, key, err.Error())
			return fmt.Errorf("[table] %s compute-out -> %s %s %s", dsl.ID, name, key, err.Error())
		}
		values[key] = res
	}
	return nil
}
