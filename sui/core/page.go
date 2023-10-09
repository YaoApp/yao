package core

import jsoniter "github.com/json-iterator/go"

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
