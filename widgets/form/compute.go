package form

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/log"
)

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
			log.Error("[form] %s compute-in -> %s %s %s", dsl.ID, name, key, err.Error())
			return fmt.Errorf("[form] %s compute-in -> %s %s %s", dsl.ID, name, key, err.Error())
		}

		res, err := compute.WithGlobal(process.Global).WithSID(process.Sid).Exec()
		if err != nil {
			log.Error("[form] %s compute-in -> %s %s %s", dsl.ID, name, key, err.Error())
			return fmt.Errorf("[form] %s compute-in -> %s %s %s", dsl.ID, name, key, err.Error())
		}
		values[key] = res
	}
	return nil
}

func (dsl *DSL) computeOut(process *gou.Process, key string, values map[string]interface{}) error {
	if name, has := dsl.ComputesOut[key]; has {
		compute, err := gou.ProcessOf(name, key, values[key], values)
		if err != nil {
			log.Error("[form] %s compute-out -> %s %s %s", dsl.ID, name, key, err.Error())
			return fmt.Errorf("[form] %s compute-out -> %s %s %s", dsl.ID, name, key, err.Error())
		}

		res, err := compute.WithGlobal(process.Global).WithSID(process.Sid).Exec()
		if err != nil {
			log.Error("[form] %s compute-out -> %s %s %s", dsl.ID, name, key, err.Error())
			return fmt.Errorf("[form] %s compute-out -> %s %s %s", dsl.ID, name, key, err.Error())
		}
		values[key] = res
	}
	return nil
}
