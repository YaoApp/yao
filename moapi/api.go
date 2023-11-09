package moapi

import "github.com/yaoapp/gou/api"

var dsl = []byte(`
{
	"name": "Moapi API",
	"description": "The API for Moapi",
	"version": "1.0.0",
	"guard": "bearer-jwt",
	"group": "__moapi/v1",
	"paths": [
		{
			"path": "/images/generations",
			"method": "POST",
			"process": "moapi.images.Generations",
			"in": ["$payload.model", "$payload.prompt", ":payload"],
			"out": { "status": 200, "type": "application/json" }
		}
	]
}
`)

func registerAPI() error {
	_, err := api.LoadSource("<moapi.v1>.yao", dsl, "moapi.v1")
	return err
}
