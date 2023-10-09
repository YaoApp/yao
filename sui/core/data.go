package core

import (
	"strings"

	jsoniter "github.com/json-iterator/go"
)

// Data get the data
func (page *Page) Data(request *Request) (map[string]interface{}, map[string]interface{}, error) {

	setting := map[string]interface{}{
		"title": strings.ToUpper(page.Name),
	}

	if page.Codes.DATA.Code != "" {
		err := jsoniter.UnmarshalFromString(page.Codes.DATA.Code, &setting)
		if err != nil {
			return nil, nil, err
		}
	}
	return nil, setting, nil
}

// Render render for the html
func (page *Page) Render(html string, data map[string]interface{}, warnings []string) (string, error) {
	return html, nil
}
