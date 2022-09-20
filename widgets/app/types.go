package app

// DSL the app DSL
type DSL struct {
	Name        string      `json:"name,omitempty"`
	Short       string      `json:"short,omitempty"`
	Version     string      `json:"version,omitempty"`
	Description string      `json:"description,omitempty"`
	Theme       string      `json:"theme,omitempty"`
	Menu        MenuDSL     `json:"menu,omitempty"`
	Optional    OptionalDSL `json:"optional,omitempty"`
}

// MenuDSL the menu DSL
type MenuDSL struct {
	Process string   `json:"process,omitempty"`
	Args    []string `json:"args,omitempty"`
}

// OptionalDSL the Optional DSL
type OptionalDSL struct {
	HideNotification bool   `json:"hideNotification,omitempty"`
	HideSetting      bool   `json:"hideSetting,omitempty"`
	Setting          string `json:"setting,omitempty"` // custom setting process
}
