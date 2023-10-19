package core

// Setting the struct for the DSL
func (sui *DSL) Setting() (*Setting, error) {
	return &Setting{
		ID:    sui.ID,
		Guard: sui.Guard,
		Option: map[string]interface{}{
			"disableCodeEditor": false,
		},
	}, nil
}
