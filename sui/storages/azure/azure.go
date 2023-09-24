package azure

import (
	"fmt"
	"net/url"

	"github.com/yaoapp/yao/sui/core"
)

// Remote is the struct for the azure sui
type Remote struct{ url url.URL }

// new create a new azure sui
func new(host url.URL) (*Remote, error) {
	return &Remote{url: host}, nil
}

// New create a new azure sui
func New(dsl *core.DSL) (*Remote, error) {

	if dsl.Storage.Option == nil {
		return nil, fmt.Errorf("option.host is required")
	}

	if dsl.Storage.Option["host"] == nil {
		return nil, fmt.Errorf("option.host is required")
	}

	host, ok := dsl.Storage.Option["host"].(string)
	if !ok {
		return nil, fmt.Errorf("option.host %s is not a valid string", host)
	}

	u, err := url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("option.host %s is not a valid url", host)
	}

	return new(*u)
}

// GetTemplates get the templates
func (azure *Remote) GetTemplates() ([]core.ITemplate, error) {
	return nil, nil
}

// GetTemplate get the template
func (azure *Remote) GetTemplate(name string) (core.ITemplate, error) {
	return nil, nil
}

// UploadTemplate upload the template
func (azure *Remote) UploadTemplate(src string, dst string) (core.ITemplate, error) {
	return nil, nil
}
