package core

import (
	"github.com/yaoapp/gou/application"
)

// Load load the dsl
func Load(file string, id string) (*DSL, error) {

	data, err := application.App.Read(file)
	if err != nil {
		return nil, err
	}

	dsl := DSL{ID: id}
	err = application.Parse(file, data, &dsl)
	if err != nil {
		return nil, err
	}

	return &dsl, nil
}
