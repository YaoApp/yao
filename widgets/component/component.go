package component

import (
	jsoniter "github.com/json-iterator/go"
)

// process
// yao.component.TagView
// yao.component.TagEdit
// yao.component.ImageView
// yao.component.UploadEdit

// Export processes
func Export() error {
	exportProcess()
	return nil
}

// MarshalJSON  Custom JSON parse
func (dsl DSL) MarshalJSON() ([]byte, error) {
	return jsoniter.Marshal(dsl.Map())
}

// Map cast to map[string]interface{}
func (dsl DSL) Map() map[string]interface{} {
	res := map[string]interface{}{
		"type":  dsl.Type,
		"props": map[string]interface{}(dsl.Props),
	}

	if dsl.HideLabel {
		res["hideLabel"] = true
	}

	if dsl.Bind != "" {
		res["bind"] = dsl.Bind
	}
	return res
}

// Clone Component
func (dsl *DSL) Clone() *DSL {
	new := DSL{
		Bind:    dsl.Bind,
		Type:    dsl.Type,
		Compute: dsl.Compute,
		Props:   PropsDSL{},
	}
	if dsl.Props != nil {
		for key, val := range dsl.Props {
			new.Props[key] = val
		}
	}
	return &new
}
