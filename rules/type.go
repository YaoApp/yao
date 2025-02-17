package rules

type Rule struct {
	ID           string `json:"id,omitempty"`
	Key          string `json:"-"`
	Name         string `json:"name,omitempty"`
	Icon         string `json:"icon,omitempty"`
	Path         string `json:"path,omitempty"`
	Visible_menu int    `json:"visible_menu,omitempty"`
	Children     []Rule `json:"children,omitempty"`
}

type DSL struct {
	*Rule
	file   string `json:"-"`
	source []byte `json:"-"`
}
