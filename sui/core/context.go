package core

// NewBuildContext create a new build context
func NewBuildContext(global *GlobalBuildContext) *BuildContext {
	return &BuildContext{
		components:    map[string]string{},
		sequence:      1,
		scripts:       []ScriptNode{},
		scriptUnique:  map[string]bool{},
		styles:        []StyleNode{},
		styleUnique:   map[string]bool{},
		jitComponents: map[string]bool{},
		global:        global,
		warnings:      []string{},
		visited:       map[string]int{},
		stack:         []string{},
	}
}

// NewTranslateContext create a new translate context
func NewTranslateContext() *TranslateContext {
	return &TranslateContext{
		sequence:     1,
		translations: []Translation{},
	}
}

// NewGlobalBuildContext create a new global build context
func NewGlobalBuildContext() *GlobalBuildContext {
	return &GlobalBuildContext{
		jitComponents: map[string]bool{},
	}
}

// GetJitComponents get the just in time components
func (ctx *BuildContext) GetJitComponents() []string {
	if ctx.jitComponents == nil {
		return []string{}
	}
	jitComponents := []string{}
	for name := range ctx.jitComponents {
		jitComponents = append(jitComponents, name)
	}
	return jitComponents
}

// GetComponents get the components
func (ctx *BuildContext) GetComponents() []string {
	if ctx.components == nil {
		return []string{}
	}
	components := []string{}
	for _, name := range ctx.components {
		components = append(components, name)
	}
	return components
}

// GetTranslations get the translations
func (ctx *BuildContext) GetTranslations() []Translation {
	if ctx.translations == nil {
		return []Translation{}
	}
	return ctx.translations
}

// GetJitComponents get the just in time components
func (globalCtx *GlobalBuildContext) GetJitComponents() []string {
	if globalCtx.jitComponents == nil {
		return []string{}
	}

	jitComponents := []string{}
	for name := range globalCtx.jitComponents {
		jitComponents = append(jitComponents, name)
	}
	return jitComponents
}

func (ctx *BuildContext) addJitComponent(name string) {
	name = dataTokens.ReplaceAllString(name, "*")
	name = propTokens.ReplaceAllString(name, "*")
	ctx.jitComponents[name] = true
	if ctx.global != nil {
		ctx.global.jitComponents[name] = true
	}
}

func (ctx *BuildContext) isJitComponent(name string) bool {
	hasStmt := dataTokens.MatchString(name)
	hasProp := propTokens.MatchString(name)
	return hasStmt || hasProp
}
