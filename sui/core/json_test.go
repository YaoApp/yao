package core

import (
	"reflect"
	"testing"

	jsoniter "github.com/json-iterator/go"
)

func TestRequestSourceUnmarshalJSON(t *testing.T) {
	testCases := []struct {
		name       string
		jsonData   string
		expected   *RequestSource
		shouldFail bool
	}{
		{
			name: "Valid JSON with string query and headers",
			jsonData: `{
				"uid": "123",
				"mock": {
					"method": "GET",
					"query": {
						"q1": "value1",
						"q2": "value2"
					},
					"headers": {
						"header1": "value1",
						"header2": "value2"
					}
				}
			}`,
			expected: &RequestSource{
				UID: "123",
				Mock: &PageMock{
					Method: "GET",
					Query: map[string][]string{
						"q1": {"value1"},
						"q2": {"value2"},
					},
					Headers: map[string][]string{
						"header1": {"value1"},
						"header2": {"value2"},
					},
				},
			},
			shouldFail: false,
		},
		{
			name: "Valid JSON with array query and headers",
			jsonData: `{
				"uid": "456",
				"mock": {
					"method": "POST",
					"query": {
						"q1": ["value1", "value2"],
						"q2": ["value3"]
					},
					"headers": {
						"header1": ["value1", "value2"],
						"header2": "value3"
					}
				}
			}`,
			expected: &RequestSource{
				UID: "456",
				Mock: &PageMock{
					Method: "POST",
					Query: map[string][]string{
						"q1": {"value1", "value2"},
						"q2": {"value3"},
					},
					Headers: map[string][]string{
						"header1": {"value1", "value2"},
						"header2": {"value3"},
					},
				},
			},
			shouldFail: false,
		},
		{
			name: "Valid JSON with invalid query",
			jsonData: `{
				"uid": "789",
				"mock": {
					"method": "PUT",
					"query":"1203"
				}
			}`,
			expected:   nil,
			shouldFail: true,
		},
		// Add more test cases here to cover other scenarios.
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var requestSource RequestSource
			err := jsoniter.Unmarshal([]byte(testCase.jsonData), &requestSource)

			if testCase.shouldFail {
				if err == nil {
					t.Errorf("%s: Expected unmarshal to fail, but it succeeded", testCase.name)
				}
			} else {
				if err != nil {
					t.Errorf("Unmarshal failed: %v", err)
				}
				if !reflect.DeepEqual(requestSource, *testCase.expected) {
					t.Errorf("Unmarshaled result does not match expected result")
				}
			}
		})
	}
}

func TestPageConfigUnmarshalJSON(t *testing.T) {
	testCases := []struct {
		name       string
		jsonData   string
		expected   *PageConfig
		shouldFail bool
	}{
		{
			name: "Valid JSON with PageSetting and PageMock",
			jsonData: `{
				"title": "Page Title",
				"mock": {
					"method": "GET",
					"query": {
						"q1": "value1",
						"q2": "value2"
					},
					"headers": {
						"header1": "value1",
						"header2": "value2"
					}
				}
			}`,
			expected: &PageConfig{
				PageSetting: PageSetting{
					Title: "Page Title",
				},
				Mock: &PageMock{
					Method: "GET",
					Query: map[string][]string{
						"q1": {"value1"},
						"q2": {"value2"},
					},
					Headers: map[string][]string{
						"header1": {"value1"},
						"header2": {"value2"},
					},
				},
			},
			shouldFail: false,
		},
		{
			name: "Valid JSON with PageSetting only",
			jsonData: `{
				"title": "Page Title"
			}`,
			expected: &PageConfig{
				PageSetting: PageSetting{
					Title: "Page Title",
				},
				Mock: nil,
			},
			shouldFail: false,
		},
		{
			name: "Valid JSON with PageMock only",
			jsonData: `{
				"mock": {
					"method": "GET",
					"query": {
						"q1": "value1",
						"q2": "value2"
					}
				}
			}`,
			expected: &PageConfig{
				Mock: &PageMock{
					Method: "GET",
					Query: map[string][]string{
						"q1": {"value1"},
						"q2": {"value2"},
					},
				},
			},
			shouldFail: false,
		},
		{
			name: "Valid JSON with invalid query",
			jsonData: `{
				"title": "Page Title",
				"mock": {
					"method": "PUT",
					"query": "invalid query"
				}
			}`,
			expected:   nil,
			shouldFail: true,
		},
		// Add more test cases here to cover other scenarios.
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var pageConfig PageConfig
			err := jsoniter.Unmarshal([]byte(testCase.jsonData), &pageConfig)

			if testCase.shouldFail {
				if err == nil {
					t.Errorf("%s: Expected unmarshal to fail, but it succeeded", testCase.name)
				}
			} else {

				if err != nil {
					t.Errorf("Unmarshal failed: %v", err)
				}
				if !reflect.DeepEqual(pageConfig, *testCase.expected) {
					t.Errorf("Unmarshaled result does not match expected result")
				}
			}
		})
	}
}
