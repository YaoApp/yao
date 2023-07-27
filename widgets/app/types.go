package app

// DSL the app DSL
type DSL struct {
	Name        string      `json:"name,omitempty"`
	Short       string      `json:"short,omitempty"`
	Version     string      `json:"version,omitempty"`
	Description string      `json:"description,omitempty"`
	Theme       string      `json:"theme,omitempty"`
	Lang        string      `json:"lang,omitempty"`
	Sid         string      `json:"sid,omitempty"`
	Logo        string      `json:"logo,omitempty"`
	Favicon     string      `json:"favicon,omitempty"`
	Menu        MenuDSL     `json:"menu,omitempty"`
	AdminRoot   string      `json:"adminRoot,omitempty"`
	Optional    OptionalDSL `json:"optional,omitempty"`
	Token       OptionalDSL `json:"token,omitempty"`
	Setting     string      `json:"setting,omitempty"` // custom setting process
	Setup       string      `json:"setup,omitempty"`   // setup process
}

// MenuDSL the menu DSL
type MenuDSL struct {
	Process string        `json:"process,omitempty"`
	Args    []interface{} `json:"args,omitempty"`
}

// OptionalDSL the Optional DSL
type OptionalDSL map[string]interface{}

// CFUN cloud function
type CFUN struct {
	Method string        `json:"method"`
	Args   []interface{} `json:"args,omitempty"`
}
