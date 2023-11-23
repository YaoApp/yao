package azure

import (
	"fmt"
	"net/url"

	"github.com/yaoapp/yao/sui/core"
)

// Azure is the struct for the azure sui
type Azure struct {
	url url.URL
	*core.DSL
}

// new create a new azure sui
func new() (*Azure, error) {
	return nil, fmt.Errorf("Azure does not support yet")
}

// New create a new azure sui
func New(dsl *core.DSL) (*Azure, error) {

	// if dsl.Storage.Option == nil {
	// 	return nil, fmt.Errorf("option.host is required")
	// }

	// if dsl.Storage.Option["host"] == nil {
	// 	return nil, fmt.Errorf("option.host is required")
	// }

	// host, ok := dsl.Storage.Option["host"].(string)
	// if !ok {
	// 	return nil, fmt.Errorf("option.host %s is not a valid string", host)
	// }

	// u, err := url.Parse(host)
	// if err != nil {
	// 	return nil, fmt.Errorf("option.host %s is not a valid url", host)
	// }

	return new()
}

// GetTemplates get the templates
func (azure *Azure) GetTemplates() ([]core.ITemplate, error) {
	return nil, nil
}

// GetTemplate get the template
func (azure *Azure) GetTemplate(name string) (core.ITemplate, error) {
	return nil, nil
}

// UploadTemplate upload the template
func (azure *Azure) UploadTemplate(src string, dst string) (core.ITemplate, error) {
	return nil, nil
}
