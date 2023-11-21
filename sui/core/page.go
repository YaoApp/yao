package core

import (
	"strings"

	jsoniter "github.com/json-iterator/go"
)

// Get get the base info
func (page *Page) Get() *Page {
	return page
}

// GetConfig get the config
func (page *Page) GetConfig() *PageConfig {
	if page.Config == nil && page.Codes.CONF.Code != "" {
		var config PageConfig
		err := jsoniter.Unmarshal([]byte(page.Codes.CONF.Code), &config)
		if err == nil {
			page.Config = &config
		}
	}
	return page.Config
}

// Data get the data
func (page *Page) Data(request *Request) (Data, map[string]interface{}, error) {

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

// Exec get the data
func (page *Page) Exec(request *Request) (Data, error) {

	if page.Codes.DATA.Code == "" {
		return map[string]interface{}{}, nil
	}

	data, err := request.ExecString(page.Codes.DATA.Code)
	if err != nil {
		return nil, err
	}

	return data, nil
}
