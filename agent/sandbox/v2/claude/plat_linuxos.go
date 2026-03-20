package claude

import "fmt"

// linuxPlatform overrides Linux container-specific behavior.
type linuxPlatform struct {
	posixBase
	hasDisplay bool   // true when the container has a DISPLAY env var (Desktop/VNC)
	sysHome    string // original system HOME (e.g., /root) before redirection
}

// XauthoritySetup handles X11 authentication cookie for Desktop containers.
//
// Desktop containers (VNC/noVNC): X Server creates .Xauthority under the
// system home (e.g., /root), but we redirect HOME to the workspace. GUI
// tools look for $HOME/.Xauthority, so we must copy the real cookie.
//
// Headless containers: no X Server, nothing to do.
func (p *linuxPlatform) XauthoritySetup(workDir string) string {
	if !p.hasDisplay || p.sysHome == "" {
		return ""
	}
	src := p.sysHome + "/.Xauthority"
	dst := workDir + "/.Xauthority"
	return fmt.Sprintf("[ -f %q ] && cp %q %q 2>/dev/null\n", src, src, dst)
}

func (p *linuxPlatform) EnvPromptNote() string {
	if p.hasDisplay {
		return `
- **Desktop Environment**: You have access to a Linux desktop via VNC (GUI applications, browsers, etc.)
- **Important**: When you launch GUI applications, do NOT close them unless explicitly asked`
	}
	return ""
}

func (p *linuxPlatform) BuildScript(in scriptInput) (string, []byte) {
	return p.buildBashScript(in, p.XauthoritySetup(in.workDir)), nil
}
