package dsh

import (
	"context"
	"io"

	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	infra "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/tai/workspace"
)

// ExportPlatform wraps the unexported platform interface for black-box testing.
type ExportPlatform interface {
	OS() string
	HomeEnv(workDir string) map[string]string
	PathJoin(parts ...string) string
	ShellCmd(script string) []string
	KillCmd(pattern string) []string
	KillSessionCmd(sessionName string) []string
	GracefulKillSessionCmd(sessionName string) []string
	BuildScript(input ExportScriptInput) (script string, stdin []byte)
}

// ExportScriptInput mirrors scriptInput for black-box testing.
type ExportScriptInput struct {
	CordisYAML   string
	ConfigFile   string
	InputJSONRPC string
}

func toInternal(in ExportScriptInput) scriptInput {
	return scriptInput{
		cordisYAML:   in.CordisYAML,
		configFile:   in.ConfigFile,
		inputJSONRPC: in.InputJSONRPC,
	}
}

type exportWrapper struct {
	p platform
}

func (w *exportWrapper) OS() string                         { return w.p.OS() }
func (w *exportWrapper) HomeEnv(d string) map[string]string { return w.p.HomeEnv(d) }
func (w *exportWrapper) PathJoin(parts ...string) string    { return w.p.PathJoin(parts...) }
func (w *exportWrapper) ShellCmd(s string) []string         { return w.p.ShellCmd(s) }
func (w *exportWrapper) KillCmd(p string) []string          { return w.p.KillCmd(p) }
func (w *exportWrapper) KillSessionCmd(s string) []string   { return w.p.KillSessionCmd(s) }
func (w *exportWrapper) GracefulKillSessionCmd(s string) []string {
	return w.p.GracefulKillSessionCmd(s)
}
func (w *exportWrapper) BuildScript(in ExportScriptInput) (string, []byte) {
	return w.p.BuildScript(toInternal(in))
}

// ExportNewPosixPlatform creates a POSIX platform for testing.
func ExportNewPosixPlatform() ExportPlatform {
	return &exportWrapper{p: newPosixPlatform()}
}

// ExportNewWindowsPlatform creates a Windows platform for testing.
func ExportNewWindowsPlatform(shell string) ExportPlatform {
	return &exportWrapper{p: newWindowsPlatform(shell)}
}

// --- Pure function exports ---

// ExportExtractLastUserMessage exposes extractLastUserMessage.
var ExportExtractLastUserMessage = extractLastUserMessage

// ExportConnectorSetting exposes connectorSetting.
var ExportConnectorSetting = connectorSetting

// ExportBuildSystemPrompt exposes buildSystemPrompt.
func ExportBuildSystemPrompt(req *types.StreamRequest, workDir string) string {
	return buildSystemPrompt(req, workDir)
}

// ExportBuildEnv wraps buildEnv for black-box testing.
func ExportBuildEnv(req *types.StreamRequest, p ExportPlatform, workDir, apiKey, baseURL, systemPrompt string) map[string]string {
	return buildEnv(req, p.(*exportWrapper).p, workDir, apiKey, baseURL, systemPrompt)
}

// ExportExtractToolSummary exposes extractToolSummary.
var ExportExtractToolSummary = extractToolSummary

// ExportTruncateStr exposes truncateStr.
var ExportTruncateStr = truncateStr

// ExportInjectDSHSemanticType exposes injectDSHSemanticType.
var ExportInjectDSHSemanticType = injectDSHSemanticType

// ExportRenderCordisConfig exposes RenderCordisConfig (already public, alias for consistency).
var ExportRenderCordisConfig = RenderCordisConfig

// ExportBuildCancelRPC exposes session.buildCancelRPC for testing.
func ExportBuildCancelRPC(chatID string) []byte {
	s := &session{chatID: chatID}
	return s.buildCancelRPC()
}

// ExportBuildShutdownRPC exposes session.buildShutdownRPC for testing.
func ExportBuildShutdownRPC() []byte {
	s := &session{}
	return s.buildShutdownRPC()
}

// ExportIsContextErr exposes isContextErr for testing.
var ExportIsContextErr = isContextErr

// --- Stream parser export ---

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

// Completed returns whether the parser saw the idle signal.
func (p *ExportStreamParser) Completed() bool {
	return p.inner.completed
}

// --- FakeComputer for black-box tests ---

type FakeComputer struct {
	WorkDirVal string
	OSName     string
	ShellName  string
}

func NewFakeComputer(workDir string) *FakeComputer {
	return &FakeComputer{WorkDirVal: workDir, OSName: "linux", ShellName: "bash"}
}

func NewFakeWindowsComputer(workDir string) *FakeComputer {
	return &FakeComputer{WorkDirVal: workDir, OSName: "windows", ShellName: "pwsh"}
}

func (f *FakeComputer) GetWorkDir() string      { return f.WorkDirVal }
func (f *FakeComputer) BindWorkplace(string)    {}
func (f *FakeComputer) Workplace() workspace.FS { return nil }
func (f *FakeComputer) ComputerInfo() infra.ComputerInfo {
	return infra.ComputerInfo{System: infra.SystemInfo{OS: f.OSName, Shell: f.ShellName}}
}
func (f *FakeComputer) Exec(_ context.Context, _ []string, _ ...infra.ExecOption) (*infra.ExecResult, error) {
	return &infra.ExecResult{}, nil
}
func (f *FakeComputer) Stream(_ context.Context, _ []string, _ ...infra.ExecOption) (*infra.ExecStream, error) {
	return nil, nil
}
func (f *FakeComputer) VNC(_ context.Context) (string, error)                    { return "", nil }
func (f *FakeComputer) Proxy(_ context.Context, _ int, _ string) (string, error) { return "", nil }
func (f *FakeComputer) ListPorts(_ context.Context) ([]*infra.PortInfo, error) {
	return nil, nil
}
func (f *FakeComputer) ListProcesses(_ context.Context, _ ...infra.ListProcessesOption) ([]*infra.ProcessInfo, *infra.SystemLoad, error) {
	return nil, nil, nil
}
