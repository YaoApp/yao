package login

// DSL the login DSL
type DSL struct {
	ID              string               `json:"id,omitempty"`
	Name            string               `json:"name,omitempty"`
	Action          ActionDSL            `json:"action,omitempty"`
	Layout          LayoutDSL            `json:"layout,omitempty"`
	ThirdPartyLogin []ThirdPartyLoginDSL `json:"thirdPartyLogin,omitempty"`
}

// ActionDSL the login action DSL
type ActionDSL struct {
	Process string        `json:"process,omitempty"`
	Args    []interface{} `json:"args,omitempty"`
}

// LayoutDSL the login page layoutDSL
type LayoutDSL struct {
	Entry   string `json:"entry,omitempty"`
	Captcha string `json:"captcha,omitempty"`
	Cover   string `json:"cover,omitempty"`
	Slogan  string `json:"slogan,omitempty"`
	Site    string `json:"site,omitempty"`
}

// ThirdPartyLoginDSL the thirdparty login url
type ThirdPartyLoginDSL struct {
	Title string `json:"title,omitempty"`
	Href  string `json:"href,omitempty"`
	Icon  string `json:"icon,omitempty"`
	Blank bool   `json:"blank,omitempty"`
}
