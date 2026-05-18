package claude

import (
	"context"
	"io"

	"github.com/yaoapp/yao/agent/output/message"

	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	infra "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/tai/workspace"
)

// ExportPlatform wraps the unexported platform interface for black-box testing.
type ExportPlatform interface {
	OS() string
	Shell() string
	HomeEnv(workDir string) map[string]string
	EnvPromptNote() string
	PathJoin(parts ...string) string
	RootDir() string
	ShellCmd(script string) []string
	KillCmd(pattern string) []string
	KillSessionCmd(sessionName string) []string
	ListDirCmd(dir string) []string
	ConfigDir() string
	XauthoritySetup(workDir string) string
	BuildScript(input ExportScriptInput) (script string, stdin []byte)
}

// ExportScriptInput mirrors scriptInput for black-box testing.
type ExportScriptInput struct {
	Args         []string
	SystemPrompt string
	InputJSONL   string
	WorkDir      string
	PromptFile   string
}

func toInternal(in ExportScriptInput) scriptInput {
	return scriptInput{
		args:         in.Args,
		systemPrompt: in.SystemPrompt,
		inputJSONL:   in.InputJSONL,
		workDir:      in.WorkDir,
		promptFile:   in.PromptFile,
	}
}

type exportWrapper struct {
	p platform
}

func (w *exportWrapper) OS() string                         { return w.p.OS() }
func (w *exportWrapper) Shell() string                      { return w.p.Shell() }
func (w *exportWrapper) HomeEnv(d string) map[string]string { return w.p.HomeEnv(d) }
func (w *exportWrapper) EnvPromptNote() string              { return w.p.EnvPromptNote() }
func (w *exportWrapper) PathJoin(parts ...string) string    { return w.p.PathJoin(parts...) }
func (w *exportWrapper) RootDir() string                    { return w.p.RootDir() }
func (w *exportWrapper) ShellCmd(s string) []string         { return w.p.ShellCmd(s) }
func (w *exportWrapper) KillCmd(p string) []string          { return w.p.KillCmd(p) }
func (w *exportWrapper) KillSessionCmd(s string) []string   { return w.p.KillSessionCmd(s) }
func (w *exportWrapper) ListDirCmd(d string) []string       { return w.p.ListDirCmd(d) }
func (w *exportWrapper) ConfigDir() string                  { return w.p.ConfigDir() }
func (w *exportWrapper) XauthoritySetup(d string) string    { return w.p.XauthoritySetup(d) }
func (w *exportWrapper) BuildScript(in ExportScriptInput) (string, []byte) {
	return w.p.BuildScript(toInternal(in))
}

// ExportNewPosixPlatform creates a Linux posix platform for testing.
func ExportNewPosixPlatform(osName, workDir, shell, tempDir string) ExportPlatform {
	base := posixBase{os: osName, workDir: workDir, shell: shell, tempDir: tempDir}
	return &exportWrapper{p: &linuxPlatform{posixBase: base}}
}

// ExportNewDarwinPlatform creates a macOS platform for testing.
func ExportNewDarwinPlatform(workDir, shell, tempDir string) ExportPlatform {
	base := posixBase{os: "darwin", workDir: workDir, shell: shell, tempDir: tempDir}
	return &exportWrapper{p: &darwinPlatform{posixBase: base}}
}

// ExportNewLinuxPlatform creates a Linux platform for testing.
func ExportNewLinuxPlatform(workDir, shell, tempDir string, hasDisplay bool, sysHome string) ExportPlatform {
	base := posixBase{os: "linux", workDir: workDir, shell: shell, tempDir: tempDir}
	return &exportWrapper{p: &linuxPlatform{posixBase: base, hasDisplay: hasDisplay, sysHome: sysHome}}
}

// ExportNewWindowsPlatform creates a Windows platform for testing.
func ExportNewWindowsPlatform(workDir, shell, tempDir string) ExportPlatform {
	return &exportWrapper{p: newWindowsPlatform(workDir, shell, tempDir)}
}

// --- command.go pure function exports ---

// ExportHashUserID exposes hashUserID.
var ExportHashUserID = hashUserID

// ExportChatIDToSessionUUID exposes chatIDToSessionUUID.
var ExportChatIDToSessionUUID = chatIDToSessionUUID

// ExportSanitizeSessionName exposes sanitizeSessionName.
var ExportSanitizeSessionName = sanitizeSessionName

// ExportBuildMCPConfig exposes buildMCPConfig.
var ExportBuildMCPConfig = buildMCPConfig

// ExportBuildMCPAllowedTools exposes buildMCPAllowedTools.
var ExportBuildMCPAllowedTools = buildMCPAllowedTools

// ExportBuildLastUserMessageJSONL exposes buildLastUserMessageJSONL.
var ExportBuildLastUserMessageJSONL = buildLastUserMessageJSONL

// ExportIsStandardAnthropicModel exposes isStandardAnthropicModel.
var ExportIsStandardAnthropicModel = isStandardAnthropicModel

// ExportBuildClaudeCodeCapabilities exposes buildClaudeCodeCapabilities.
var ExportBuildClaudeCodeCapabilities = buildClaudeCodeCapabilities

// ExportBuildSandboxEnvPrompt wraps buildSandboxEnvPrompt for black-box testing.
func ExportBuildSandboxEnvPrompt(p ExportPlatform, workDir string) string {
	return buildSandboxEnvPrompt(p.(*exportWrapper).p, workDir)
}

// ExportBuildEnv wraps buildEnv for black-box testing.
func ExportBuildEnv(req *types.StreamRequest, p ExportPlatform) map[string]string {
	return buildEnv(req, p.(*exportWrapper).p)
}

// ExportBuildArgs wraps buildArgs for black-box testing.
func ExportBuildArgs(req *types.StreamRequest, hasMCP bool, mcpToolPattern string, p ExportPlatform, isContinuation bool, assistantID, chatID string) []string {
	r := &Runner{hasMCP: hasMCP, mcpToolPattern: mcpToolPattern}
	return buildArgs(req, r, p.(*exportWrapper).p, isContinuation, assistantID, chatID)
}

// ExportBuildModelCapabilityPrompt wraps buildModelCapabilityPrompt.
var ExportBuildModelCapabilityPrompt = buildModelCapabilityPrompt

// ExportBuildSingleA2OConfig wraps buildSingleA2OConfig.
var ExportBuildSingleA2OConfig = buildSingleA2OConfig

// ExportResolveAllRoleConnectors wraps resolveAllRoleConnectors.
var ExportResolveAllRoleConnectors = resolveAllRoleConnectors

// ExportConnectorHost wraps connectorHost.
var ExportConnectorHost = connectorHost

// ExportConnectorProtocols wraps connectorProtocols.
var ExportConnectorProtocols = connectorProtocols

// ExportSupportsProtocol wraps supportsProtocol.
var ExportSupportsProtocol = supportsProtocol

// --- FakeComputer for black-box tests ---

type FakeComputer struct {
	WorkDirVal string
}

func NewFakeComputer(workDir string) *FakeComputer {
	return &FakeComputer{WorkDirVal: workDir}
}

func (f *FakeComputer) GetWorkDir() string      { return f.WorkDirVal }
func (f *FakeComputer) BindWorkplace(string)    {}
func (f *FakeComputer) Workplace() workspace.FS { return nil }
func (f *FakeComputer) ComputerInfo() infra.ComputerInfo {
	return infra.ComputerInfo{System: infra.SystemInfo{OS: "linux", Shell: "bash"}}
}
func (f *FakeComputer) Exec(_ context.Context, _ []string, _ ...infra.ExecOption) (*infra.ExecResult, error) {
	return &infra.ExecResult{}, nil
}
func (f *FakeComputer) Stream(_ context.Context, _ []string, _ ...infra.ExecOption) (*infra.ExecStream, error) {
	return nil, nil
}
func (f *FakeComputer) VNC(_ context.Context) (string, error)                    { return "", nil }
func (f *FakeComputer) Proxy(_ context.Context, _ int, _ string) (string, error) { return "", nil }

// --- Export default constants ---

const ExportDefaultA2OPort = defaultA2OPort

// ExportBuildLastUserJSONL wraps buildLastUserMessageJSONL for typed Message input.
func ExportBuildLastUserJSONL(msgs []agentContext.Message) string {
	return buildLastUserMessageJSONL(msgs)
}

// ExportExtractSummary exposes extractSummary for testing.
var ExportExtractSummary = extractSummary

// ExportTruncate exposes truncate for testing.
var ExportTruncate = truncate

// ExportNewStreamParser creates a streamParser for black-box testing.
func ExportNewStreamParser(handler message.StreamFunc) *ExportStreamParser {
	return &ExportStreamParser{inner: newStreamParser(handler)}
}

// ExportStreamParser wraps the internal streamParser.
type ExportStreamParser struct {
	inner *streamParser
}

// Parse runs the parser on the given stdout reader.
func (p *ExportStreamParser) Parse(ctx context.Context, stdout io.ReadCloser) error {
	return p.inner.parse(ctx, stdout)
}
