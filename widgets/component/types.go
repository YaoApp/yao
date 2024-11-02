package component

// DSL the component DSL
type DSL struct {
	Bind      string   `json:"bind,omitempty"`
	HideLabel bool     `json:"hideLabel,omitempty"`
	Type      string   `json:"type,omitempty"`
	Compute   *Compute `json:"compute,omitempty"`
	Props     PropsDSL `json:"props,omitempty"`
}

// Actions the actions
type Actions []ActionDSL

// Instances the Instances
type Instances []InstanceDSL

// InstanceDSL the component instance DSL
type InstanceDSL struct {
	Name   string        `json:"name,omitempty"`
	Width  interface{}   `json:"width,omitempty"`
	Height interface{}   `json:"height,omitempty"`
	Fixed  bool          `json:"fixed,omitempty"` // for widget table
	Rows   []InstanceDSL `json:"rows,omitempty"`
}

// ActionsExport the export actions
type ActionsExport struct {
	Type    string  `json:"type,omitempty"`
	Xpath   string  `json:"xpath"`
	Actions Actions `json:"actions,omitempty"`
}

type aliasActionDSL ActionDSL

// ActionDSL the component action DSL
type ActionDSL struct {
	ID           string            `json:"id,omitempty"`
	Title        string            `json:"title,omitempty"`
	Width        int               `json:"width,omitempty"`
	Icon         string            `json:"icon,omitempty"`
	Style        string            `json:"style,omitempty"`
	Xpath        string            `json:"xpath,omitempty"`
	DivideLine   bool              `json:"divideLine,omitempty"`
	Hide         []string          `json:"hide,omitempty"` // Syntactic sugar ["add", "edit", "view"]
	ShowWhenAdd  bool              `json:"showWhenAdd,omitempty"`
	ShowWhenView bool              `json:"showWhenView,omitempty"`
	HideWhenEdit bool              `json:"hideWhenEdit,omitempty"`
	Props        PropsDSL          `json:"props,omitempty"`
	Confirm      *ConfirmActionDSL `json:"confirm,omitempty"`
	Action       ActionNodes       `json:"action,omitempty"`
	Disabled     *DisabledDSL      `json:"disabled,omitempty"`
}

// DisabledDSL the action disabled
type DisabledDSL struct {
	Field string      `json:"Field,omitempty"` //  Syntactic sugar -> bind
	Bind  string      `json:"bind,omitempty"`
	Eq    interface{} `json:"eq,omitempty"`    // string | array<string>  Syntactic sugar eq -> value
	Equal interface{} `json:"equal,omitempty"` // string | array<string>  Syntactic sugar equal -> value
	Value interface{} `json:"value,omitempty"` // string | array<string>
}

type aliasActionNodes []ActionNode

// ActionNodes the action nodes
type ActionNodes []ActionNode

// ActionNode the action node
type ActionNode map[string]interface{}

// ConfirmActionDSL action.confirm
type ConfirmActionDSL struct {
	Title string `json:"title,omitempty"`
	Desc  string `json:"desc,omitempty"`
}

// PropsDSL component props
type PropsDSL map[string]interface{}

// Compute process
type Compute struct {
	Process string `json:"process"`
	Args    []CArg `json:"args,omitempty"`
}

// computeAlias for JSON UnmarshalJSON
type computeAlias Compute

// CArg compute interface{}
type CArg struct {
	IsExp bool
	key   string
	value interface{}
}

// ComputeHanlder computeHanlder
type ComputeHanlder func(args ...interface{}) (interface{}, error)

// CloudPropsDSL the cloud props
type CloudPropsDSL struct {
	Xpath   string                 `json:"xpath,omitempty"`
	Type    string                 `json:"type,omitempty"`
	Name    string                 `json:"name,omitempty"`
	Process string                 `json:"process,omitempty"`
	Query   map[string]interface{} `json:"query,omitempty"`
	Props   map[string]interface{} `json:"props,omitempty"` // The original props
}
