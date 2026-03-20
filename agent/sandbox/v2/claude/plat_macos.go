package claude

// darwinPlatform overrides macOS-specific behavior.
// Most methods are inherited from posixBase.
type darwinPlatform struct {
	posixBase
}

func (p *darwinPlatform) EnvPromptNote() string {
	return `
- **Desktop Environment**: You have access to the macOS desktop (GUI applications, browsers, Finder, etc.)
- **Important**: When you launch GUI applications, do NOT close them unless explicitly asked`
}

func (p *darwinPlatform) BuildScript(in scriptInput) (string, []byte) {
	return p.buildBashScript(in, ""), nil
}
