package core

import (
	jsoniter "github.com/json-iterator/go"
)

// UnmarshalJSON Custom JSON unmarshal function for PageMock
func (mock *PageMock) UnmarshalJSON(data []byte) error {

	if mock == nil {
		return nil
	}

	type Alias struct {
		Method  string                 `json:"method"`
		Params  map[string]string      `json:"params,omitempty"`
		Query   map[string]interface{} `json:"query,omitempty"`
		Headers map[string]interface{} `json:"headers,omitempty"`
		Body    interface{}            `json:"body,omitempty"`
	}

	aux := &Alias{}
	if err := jsoniter.Unmarshal(data, &aux); err != nil {
		return err
	}

	method := aux.Method
	if method == "" {
		method = "GET"
	}
	mock.Body = aux.Body
	mock.Method = method
	mock.Params = aux.Params
	mock.Query = convertRecordToMap(aux.Query)
	mock.Headers = convertRecordToMap(aux.Headers)
	return nil
}

// Helper function to convert TypeScript Record<string, string | string[]> to map[string][]string
func convertRecordToMap(record map[string]interface{}) map[string][]string {
	if record == nil {
		return nil
	}

	result := make(map[string][]string)
	for key, value := range record {
		switch v := value.(type) {
		case string:
			result[key] = []string{v}
		case []interface{}:
			strValues := make([]string, len(v))
			for i, item := range v {
				if str, ok := item.(string); ok {
					strValues[i] = str
				}
			}
			result[key] = strValues
		}
	}
	return result
}
