package component

// DSL the component DSL
type DSL struct {
	Type  string   `json:"type,omitempty"`
	In    string   `json:"in,omitempty"`
	Out   string   `json:"out,omitempty"`
	Props PropsDSL `json:"props,omitempty"`
}

// Actions the actions
type Actions []ActionDSL

// Instances the Instances
type Instances []InstanceDSL

// InstanceDSL the component instance DSL
type InstanceDSL struct {
	Name   string      `json:"name,omitempty"`
	Width  interface{} `json:"width,omitempty"`
	Height interface{} `json:"height,omitempty"`
}

// ActionDSL the component action DSL
type ActionDSL struct {
	Title   string               `json:"title,omitempty"`
	Width   int                  `json:"width,omitempty"`
	Icon    string               `json:"icon,omitempty"`
	Style   string               `json:"style,omitempty"`
	Props   PropsDSL             `json:"props,omitempty"`
	Confirm *ConfirmActionDSL    `json:"confirm,omitempty"`
	Action  map[string]ParamsDSL `json:"action,omitempty"`
}

// ConfirmActionDSL action.confirm
type ConfirmActionDSL struct {
	Title string `json:"title,omitempty"`
	Desc  string `json:"desc,omitempty"`
}

// PropsDSL component props
type PropsDSL map[string]interface{}

// ParamsDSL action params
type ParamsDSL map[string]interface{}

// CloudPropsDSL the cloud props
type CloudPropsDSL struct {
	Xpath   string                 `json:"xpath,omitempty"`
	Name    string                 `json:"name,omitempty"`
	Process string                 `json:"process,omitempty"`
	Query   map[string]interface{} `json:"query,omitempty"`
}
