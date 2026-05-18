package opencode

import (
	"context"
	"io"
	"sync"

	"github.com/yaoapp/gou/connector"
	gouTypes "github.com/yaoapp/gou/types"
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/xun/dbal/schema"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
)

// --- Config exports ---

var BuildOpenCodeConfig = buildOpenCodeConfig
var BuildMCPConfig = buildMCPConfig
var BuildProviderConfig = buildProviderConfig

// --- Roles exports ---

var InjectRoleProviders = injectRoleProviders
var InjectRoleEnvVars = injectRoleEnvVars
var ResolvePrimaryConnector = resolvePrimaryConnector

// --- Command exports ---

var ChatIDToSessionID = chatIDToSessionID
var SanitizeSessionName = sanitizeSessionName
var LastUserText = lastUserText
var BuildStdinMessage = buildStdinMessage
var BuildSandboxEnvPrompt = buildSandboxEnvPrompt
var GetProviderPrefix = getProviderPrefix

// --- Platform exports ---

var ShellQuote = shellQuote
var ShellQuotePowerShell = shellQuotePowerShell
var VisionCopyCmd = visionCopyCmd

// ShellQuoteForPlatformExport wraps shellQuoteForPlatform for external tests.
func ShellQuoteForPlatformExport(p interface{ OS() string }, program string, args ...string) string {
	return shellQuoteForPlatform(p.(platform), program, args...)
}

// NewPosixBase creates a posixBase for testing.
func NewPosixBase(os, workDir, shell string) *posixBase {
	return &posixBase{os: os, workDir: workDir, shell: shell}
}

// PosixBase is an exported type alias for external tests.
type PosixBase = posixBase

// NewWindowsPlatformForTest wraps newWindowsPlatform for external tests.
var NewWindowsPlatformForTest = newWindowsPlatform

// --- Parse exports ---

var ExtractSummary = extractSummary
var Truncate = truncate

// NewStreamParserForTest creates a streamParser for external tests.
func NewStreamParserForTest(handler message.StreamFunc) *streamParser {
	return newStreamParser(handler)
}

// StreamParserParse calls parse on a streamParser.
func StreamParserParse(p *streamParser, ctx context.Context, stdout io.ReadCloser) error {
	return p.parse(ctx, stdout)
}

// StreamParserCompleted returns whether the parser completed.
func StreamParserCompleted(p *streamParser) bool {
	return p.completed
}

// StreamParser is an exported type alias.
type StreamParser = streamParser

// ChunkRecord stores a captured stream chunk for testing.
type ChunkRecord struct {
	EventType message.StreamChunkType
	Data      string
}

// CollectHandler creates a handler that appends chunks to a slice.
func CollectHandler(records *[]ChunkRecord, mu *sync.Mutex) message.StreamFunc {
	return func(chunkType message.StreamChunkType, data []byte) int {
		mu.Lock()
		defer mu.Unlock()
		*records = append(*records, ChunkRecord{EventType: chunkType, Data: string(data)})
		return 0
	}
}

// --- Fake connector for role tests ---

type fakeConn struct {
	id       string
	typ      int
	settings map[string]interface{}
}

func (f *fakeConn) Register(string, string, []byte) error { return nil }
func (f *fakeConn) Query() (query.Query, error)           { return nil, nil }
func (f *fakeConn) Schema() (schema.Schema, error)        { return nil, nil }
func (f *fakeConn) Close() error                          { return nil }
func (f *fakeConn) ID() string                            { return f.id }
func (f *fakeConn) Is(t int) bool                         { return f.typ == t }
func (f *fakeConn) Setting() map[string]interface{}       { return f.settings }
func (f *fakeConn) GetMetaInfo() gouTypes.MetaInfo        { return gouTypes.MetaInfo{} }

// NewFakeOpenAI creates a fake OpenAI connector for tests.
func NewFakeOpenAI(id, host, model, key string) connector.Connector {
	return &fakeConn{
		id:  id,
		typ: connector.OPENAI,
		settings: map[string]interface{}{
			"host":  host,
			"model": model,
			"key":   key,
		},
	}
}

// NewFakeAnthropic creates a fake Anthropic connector for tests.
func NewFakeAnthropic(id, host, model, key string) connector.Connector {
	return &fakeConn{
		id:  id,
		typ: connector.ANTHROPIC,
		settings: map[string]interface{}{
			"host":  host,
			"model": model,
			"key":   key,
		},
	}
}

// --- Type re-exports for convenience ---

type Message = agentContext.Message
type ContentPart = agentContext.ContentPart

// PrepareRequest re-export.
type PrepareRequest = types.PrepareRequest

// MCPServer re-export.
type MCPServer = types.MCPServer

// StreamRequest re-export.
type StreamRequest = types.StreamRequest

// SandboxConfig re-export.
type SandboxConfig = types.SandboxConfig
