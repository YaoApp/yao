package audio

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/mediaprobe"
	"github.com/yaoapp/gou/process"
	agentLLM "github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/openai"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/tools/preprocess"
	"github.com/yaoapp/yao/tools/resolve"
)

const maxAudioBytes = 25 << 20 // 25MB Whisper limit

// TranscribeHandler is the tools.audio_transcribe process handler.
func TranscribeHandler(proc *process.Process) interface{} {
	src := proc.ArgsString(0)
	if src == "" {
		if paths, ok := pathsFromArg(proc, 0); ok && len(paths) > 0 {
			return transcribeFromPaths(proc, paths)
		}
		return map[string]interface{}{"error": "audio_path is required: provide a file path or URL"}
	}
	language := proc.ArgsString(1)
	provider := proc.ArgsString(2)

	authInfo := authorized.ProcessAuthInfo(proc)
	if authInfo == nil {
		return map[string]interface{}{"error": "unauthorized: no auth info in request"}
	}

	connectorRole := "use::audio"
	if provider != "" {
		connectorRole = provider
	}
	conn, _, err := agentLLM.ResolveConnector(connectorRole, authInfo)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("resolve STT connector: %v", err)}
	}

	if paths, ok := pathsFromArg(proc, 0); ok && len(paths) > 1 {
		return transcribeMultiplePaths(proc.Context, paths, language, conn)
	}

	if paths, ok := extractPathArray(src); ok && len(paths) > 1 {
		return transcribeMultiplePaths(proc.Context, paths, language, conn)
	}

	localPath, cleanup, err := resolve.ToLocalPath(proc.Context, src)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("resolve audio: %v", err)}
	}
	defer cleanup()

	info, err := os.Stat(localPath)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("stat audio: %v", err)}
	}

	probeInfo, _ := mediaprobe.ProbeFile(localPath)
	needsSplit := false
	if info.Size() > maxAudioBytes {
		needsSplit = true
	}
	if probeInfo != nil && probeInfo.Duration > 600 {
		needsSplit = true
	}

	if needsSplit {
		chunks, splitCleanup, err := preprocess.SplitAudio(localPath, 600)
		if err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("audio split failed: %v (file: %s, size: %d bytes)", err, src, info.Size())}
		}
		defer splitCleanup()
		return transcribeChunks(proc.Context, chunks, language, conn)
	}

	return transcribeSingleFile(proc.Context, localPath, language, conn)
}

func transcribeFromPaths(proc *process.Process, paths []string) interface{} {
	language := proc.ArgsString(1)
	provider := proc.ArgsString(2)

	authInfo := authorized.ProcessAuthInfo(proc)
	if authInfo == nil {
		return map[string]interface{}{"error": "unauthorized: no auth info in request"}
	}

	connectorRole := "use::audio"
	if provider != "" {
		connectorRole = provider
	}
	conn, _, err := agentLLM.ResolveConnector(connectorRole, authInfo)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("resolve STT connector: %v", err)}
	}

	if len(paths) == 1 {
		localPath, cleanup, err := resolve.ToLocalPath(proc.Context, paths[0])
		if err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("resolve audio: %v", err)}
		}
		defer cleanup()
		return transcribeSingleFile(proc.Context, localPath, language, conn)
	}

	return transcribeMultiplePaths(proc.Context, paths, language, conn)
}

func transcribeSingleFile(ctx context.Context, path, language string, conn connector.Connector) map[string]interface{} {
	options := map[string]interface{}{
		"model": connectorModel(conn),
	}
	if language != "" {
		options["language"] = language
	}

	ai, err := openai.New(conn.ID())
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("create OpenAI client: %v", err)}
	}

	resp, ex := ai.AudioTranscriptionsFile(path, options)
	if ex != nil {
		return map[string]interface{}{"error": fmt.Sprintf("transcription failed: %s", ex.Message)}
	}

	if respMap, ok := resp.(map[string]interface{}); ok {
		text, _ := respMap["text"].(string)
		result := map[string]interface{}{
			"text":  text,
			"model": connectorModel(conn),
		}
		if text == "" {
			result["warning"] = "transcription returned empty text (audio may be silent, corrupted, or in unsupported format)"
		}
		return result
	}
	return map[string]interface{}{"error": "unexpected response format from STT API"}
}

func transcribeChunks(ctx context.Context, chunks []string, language string, conn connector.Connector) map[string]interface{} {
	var texts []string
	var errors []string
	var cleanups []func()

	defer func() {
		for _, c := range cleanups {
			c()
		}
	}()

	for i, chunk := range chunks {
		localPath, cleanup, err := resolve.ToLocalPath(ctx, chunk)
		if err != nil {
			errors = append(errors, fmt.Sprintf("chunk %d: resolve: %v", i, err))
			continue
		}
		cleanups = append(cleanups, cleanup)

		result := transcribeSingleFile(ctx, localPath, language, conn)
		if errMsg, ok := result["error"].(string); ok {
			errors = append(errors, fmt.Sprintf("chunk %d: %s", i, errMsg))
			continue
		}
		if text, ok := result["text"].(string); ok && text != "" {
			texts = append(texts, text)
		}
	}

	if len(texts) == 0 && len(errors) > 0 {
		return map[string]interface{}{
			"error": fmt.Sprintf("all %d chunks failed: %s", len(errors), strings.Join(errors, "; ")),
		}
	}

	result := map[string]interface{}{
		"text":   strings.Join(texts, "\n"),
		"model":  connectorModel(conn),
		"chunks": len(chunks),
	}
	if len(errors) > 0 {
		result["warning"] = fmt.Sprintf("%d/%d chunks failed: %s", len(errors), len(chunks), strings.Join(errors, "; "))
	}
	return result
}

func transcribeMultiplePaths(ctx context.Context, paths []string, language string, conn connector.Connector) map[string]interface{} {
	return transcribeChunks(ctx, paths, language, conn)
}

func extractPathArray(src string) ([]string, bool) {
	if !strings.Contains(src, "|") {
		return nil, false
	}

	parts := strings.Split(src, "|")
	var paths []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			paths = append(paths, p)
		}
	}
	if len(paths) <= 1 {
		return nil, false
	}
	return paths, true
}

func pathsFromArg(proc *process.Process, index int) ([]string, bool) {
	if len(proc.Args) <= index || proc.Args[index] == nil {
		return nil, false
	}

	switch v := proc.Args[index].(type) {
	case []string:
		if len(v) == 0 {
			return nil, false
		}
		return v, true
	case []interface{}:
		var paths []string
		for _, item := range v {
			s, ok := item.(string)
			if !ok || s == "" {
				continue
			}
			paths = append(paths, s)
		}
		if len(paths) == 0 {
			return nil, false
		}
		return paths, true
	default:
		return nil, false
	}
}

func connectorModel(conn connector.Connector) string {
	settings := conn.Setting()
	if settings == nil {
		return "whisper-1"
	}
	if model, ok := settings["model"].(string); ok && model != "" {
		return model
	}
	return "whisper-1"
}
