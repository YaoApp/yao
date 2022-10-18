package component

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

// Map cast to map[string]interface{}
func (dsl DSL) Map() map[string]interface{} {
	return map[string]interface{}{
		"type":  dsl.Type,
		"in":    dsl.In,
		"out":   dsl.Out,
		"props": map[string]interface{}(dsl.Props),
	}
}

// Clone Component
func (dsl *DSL) Clone() *DSL {
	new := DSL{
		Type:  dsl.Type,
		In:    dsl.In,
		Out:   dsl.Out,
		Props: PropsDSL{},
	}
	if dsl.Props != nil {
		for key, val := range dsl.Props {
			new.Props[key] = val
		}
	}
	return &new
}
