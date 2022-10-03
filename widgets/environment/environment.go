package environment

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

// ReadFile open file and replace the content $ENV.XXX
func ReadFile(file string, defaults ...map[string]interface{}) ([]byte, error) {
	content := map[string]interface{}{}
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	err = jsoniter.Unmarshal(bytes, &content)
	if err != nil {
		return nil, err
	}

	defaultValues := map[string]interface{}{}
	if len(defaults) > 0 {
		defaultValues = defaults[0]
	}

	Replace(content, defaultValues)
	return jsoniter.Marshal(content)
}

// Replace replace content
func Replace(content map[string]interface{}, defaults map[string]interface{}) {
	replaceMap(content, defaults)
}

func replaceAny(value interface{}, defaults map[string]interface{}) interface{} {
	switch value.(type) {
	case string:
		return replaceStr(value.(string), defaults)

	case map[string]interface{}:
		return replaceMap(value.(map[string]interface{}), defaults)

	case []interface{}:
		return replaceArr(value.([]interface{}), defaults)
	}
	return value
}

func replaceStr(value string, defaults map[string]interface{}) interface{} {

	if strings.HasPrefix(value, "\\$$ENV.") {
		return fmt.Sprintf("$ENV.%s", strings.TrimLeft(value, "\\$$ENV."))
	}

	if !strings.HasPrefix(value, "$ENV.") {
		return value
	}

	name := strings.TrimLeft(value, "$ENV.")
	v := os.Getenv(name)
	if v != "" {
		return v
	}

	return defaults[name]
}

func replaceMap(values map[string]interface{}, defaults map[string]interface{}) map[string]interface{} {
	for key, value := range values {
		values[key] = replaceAny(value, defaults)
	}
	return values
}

func replaceArr(values []interface{}, defaults map[string]interface{}) []interface{} {
	for key, value := range values {
		values[key] = replaceAny(value, defaults)
	}
	return values
}
