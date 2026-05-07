package standard

import (
	"context"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/agent"
	agentcontext "github.com/yaoapp/yao/agent/context"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
	"github.com/yaoapp/yao/llmprovider"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/setting"
	"github.com/yaoapp/yao/tai/volume"
	taiworkspace "github.com/yaoapp/yao/tai/workspace"
)

// ============================================================================
// extractLLMContent — pure unit tests (no external dependencies)
// ============================================================================

func TestExtractLLMContent(t *testing.T) {
	t.Run("string content", func(t *testing.T) {
		resp := &agentcontext.CompletionResponse{Content: "  hello world  "}
		assert.Equal(t, "hello world", extractLLMContent(resp))
	})

	t.Run("nil response", func(t *testing.T) {
		assert.Equal(t, "", extractLLMContent(nil))
	})

	t.Run("non-string content", func(t *testing.T) {
		resp := &agentcontext.CompletionResponse{Content: []interface{}{"a", "b"}}
		assert.Equal(t, "", extractLLMContent(resp))
	})

	t.Run("empty string content", func(t *testing.T) {
		resp := &agentcontext.CompletionResponse{Content: "   "}
		assert.Equal(t, "", extractLLMContent(resp))
	})

	t.Run("multiline content trimmed", func(t *testing.T) {
		resp := &agentcontext.CompletionResponse{Content: "\n  summary line\n"}
		assert.Equal(t, "summary line", extractLLMContent(resp))
	})

	t.Run("int content", func(t *testing.T) {
		resp := &agentcontext.CompletionResponse{Content: 42}
		assert.Equal(t, "", extractLLMContent(resp))
	})
}

// ============================================================================
// extractAndVerifyFiles — unit tests with local volume-backed FS
// ============================================================================

func newTestWorkspaceFS(t *testing.T) taiworkspace.FS {
	t.Helper()
	dir := t.TempDir()
	vol := volume.NewLocal(dir)
	t.Cleanup(func() { vol.Close() })
	wfs := taiworkspace.New(vol, "ws-test")
	t.Cleanup(func() { wfs.Close() })
	return wfs
}

func TestExtractAndVerifyFiles(t *testing.T) {
	t.Run("valid URI with existing file", func(t *testing.T) {
		wfs := newTestWorkspaceFS(t)
		require.NoError(t, wfs.MkdirAll("robots/r1/exec1/task-001", 0755))
		require.NoError(t, wfs.WriteFile("robots/r1/exec1/task-001/notes.md", []byte("hello"), 0644))

		r := &Runner{
			wsFS:    wfs,
			execDir: "robots/r1/exec1",
		}
		output := "I wrote the file to workspace://ws-test/robots/r1/exec1/task-001/notes.md for you."
		files := r.extractAndVerifyFiles(output)

		require.Len(t, files, 1)
		assert.Equal(t, "notes.md", files[0].Name)
		assert.Equal(t, "text/markdown", files[0].Type)
		assert.Equal(t, "workspace://ws-test/robots/r1/exec1/task-001/notes.md", files[0].URI)
	})

	t.Run("URI with non-existent file filtered out", func(t *testing.T) {
		wfs := newTestWorkspaceFS(t)

		r := &Runner{
			wsFS:    wfs,
			execDir: "robots/r1/exec1",
		}
		output := "See workspace://ws-test/robots/r1/exec1/task-001/missing.pdf"
		files := r.extractAndVerifyFiles(output)

		assert.Empty(t, files)
	})

	t.Run("URI with trailing backtick excluded by regex", func(t *testing.T) {
		wfs := newTestWorkspaceFS(t)
		require.NoError(t, wfs.MkdirAll("robots/r1/exec1/task-001", 0755))
		require.NoError(t, wfs.WriteFile("robots/r1/exec1/task-001/data.json", []byte("{}"), 0644))

		r := &Runner{
			wsFS:    wfs,
			execDir: "robots/r1/exec1",
		}
		// Backtick-wrapped URI — the regex excludes the backtick from the captured path
		output := "`workspace://ws-test/robots/r1/exec1/task-001/data.json`"
		files := r.extractAndVerifyFiles(output)

		require.Len(t, files, 1)
		assert.Equal(t, "data.json", files[0].Name)
		assert.Equal(t, "application/json", files[0].Type)
	})

	t.Run("duplicate URIs deduplicated", func(t *testing.T) {
		wfs := newTestWorkspaceFS(t)
		require.NoError(t, wfs.MkdirAll("robots/r1/exec1/task-001", 0755))
		require.NoError(t, wfs.WriteFile("robots/r1/exec1/task-001/report.html", []byte("<h1>hi</h1>"), 0644))

		r := &Runner{
			wsFS:    wfs,
			execDir: "robots/r1/exec1",
		}
		output := "workspace://ws-test/robots/r1/exec1/task-001/report.html and again workspace://ws-test/robots/r1/exec1/task-001/report.html"
		files := r.extractAndVerifyFiles(output)

		require.Len(t, files, 1)
		assert.Equal(t, "report.html", files[0].Name)
	})

	t.Run("empty output returns nil", func(t *testing.T) {
		wfs := newTestWorkspaceFS(t)
		r := &Runner{wsFS: wfs, execDir: "robots/r1/exec1"}
		assert.Nil(t, r.extractAndVerifyFiles(""))
	})

	t.Run("nil wsFS returns nil", func(t *testing.T) {
		r := &Runner{wsFS: nil, execDir: "robots/r1/exec1"}
		assert.Nil(t, r.extractAndVerifyFiles("some text with workspace://ws-test/foo/bar"))
	})

	t.Run("URI from different workspace filtered", func(t *testing.T) {
		wfs := newTestWorkspaceFS(t)
		r := &Runner{wsFS: wfs, execDir: "robots/r1/exec1"}
		output := "See workspace://other-ws/robots/r1/exec1/task-001/file.txt"
		files := r.extractAndVerifyFiles(output)
		assert.Empty(t, files)
	})

	t.Run("directory URI filtered out", func(t *testing.T) {
		wfs := newTestWorkspaceFS(t)
		require.NoError(t, wfs.MkdirAll("robots/r1/exec1/task-001", 0755))

		r := &Runner{wsFS: wfs, execDir: "robots/r1/exec1"}
		output := "workspace://ws-test/robots/r1/exec1/task-001"
		files := r.extractAndVerifyFiles(output)
		assert.Empty(t, files)
	})

	t.Run("multiple valid files extracted", func(t *testing.T) {
		wfs := newTestWorkspaceFS(t)
		require.NoError(t, wfs.MkdirAll("robots/r1/exec1/task-002", 0755))
		require.NoError(t, wfs.WriteFile("robots/r1/exec1/task-002/slides.html", []byte("<html>"), 0644))
		require.NoError(t, wfs.WriteFile("robots/r1/exec1/task-002/slides.pdf", []byte("%PDF"), 0644))

		r := &Runner{wsFS: wfs, execDir: "robots/r1/exec1"}
		output := "Generated workspace://ws-test/robots/r1/exec1/task-002/slides.html and exported workspace://ws-test/robots/r1/exec1/task-002/slides.pdf"
		files := r.extractAndVerifyFiles(output)

		require.Len(t, files, 2)
		names := []string{files[0].Name, files[1].Name}
		assert.Contains(t, names, "slides.html")
		assert.Contains(t, names, "slides.pdf")
	})
}

// ============================================================================
// mergeManifestFiles — pure unit tests
// ============================================================================

func TestMergeManifestFiles(t *testing.T) {
	t.Run("empty fromURIs returns scanned unchanged", func(t *testing.T) {
		scanned := []ManifestFile{{Name: "a.md", Type: "text/markdown"}}
		result := mergeManifestFiles(scanned, nil)
		assert.Equal(t, scanned, result)
	})

	t.Run("merge URI onto matching scanned entry", func(t *testing.T) {
		scanned := []ManifestFile{{Name: "notes.md", Type: "text/markdown"}}
		fromURIs := []ManifestFile{{Name: "notes.md", Type: "text/markdown", URI: "workspace://ws/path/notes.md"}}
		result := mergeManifestFiles(scanned, fromURIs)

		require.Len(t, result, 1)
		assert.Equal(t, "workspace://ws/path/notes.md", result[0].URI)
	})

	t.Run("add new URI entry when no scan match", func(t *testing.T) {
		scanned := []ManifestFile{{Name: "a.md", Type: "text/markdown"}}
		fromURIs := []ManifestFile{{Name: "b.pdf", Type: "application/pdf", URI: "workspace://ws/b.pdf"}}
		result := mergeManifestFiles(scanned, fromURIs)

		require.Len(t, result, 2)
		assert.Equal(t, "b.pdf", result[1].Name)
	})
}

// ============================================================================
// llmSummarize — integration test (real LLM call)
// ============================================================================

func setupLLMProvider(t *testing.T) {
	t.Helper()

	if err := setting.Init(); err != nil {
		t.Skipf("setting.Init failed (store not available): %v", err)
	}
	if err := llmprovider.Init(); err != nil {
		t.Skipf("llmprovider.Init failed: %v", err)
	}
	if err := agent.SyncLLMDefaults(); err != nil {
		t.Skipf("SyncLLMDefaults failed: %v", err)
	}

	connIDs := connector.AIConnectors
	if len(connIDs) == 0 {
		t.Skip("no AI connectors available in test env")
	}

	t.Cleanup(func() {
		s, _ := store.Get("__yao.store")
		if s != nil {
			s.Del("llmprovider:*")
		}
		c, _ := store.Get("__yao.cache")
		if c != nil {
			c.Del("llmprovider:*")
		}
	})
}

func TestLLMSummarize(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test (requires real LLM)")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)
	setupLLMProvider(t)

	auth := &oauthtypes.AuthorizedInfo{UserID: "test-user", TeamID: "test-team"}
	ctx := robottypes.NewContext(context.Background(), auth)

	r := &Runner{
		ctx: ctx,
	}

	task := &robottypes.Task{
		ID:          "task-001",
		Description: "Research Yao Agents platform and compile structured notes",
	}

	output := `## Research Findings

Yao Agents is an AI-powered platform that enables developers to build intelligent agents.

### Key Features
- **Model-Driven Architecture**: Define data models in YAML/JSON
- **Low-Code Engine**: Visual workflow builder
- **Multi-Agent Orchestration**: Coordinate multiple AI agents

### Conclusion
Yao Agents provides a comprehensive toolkit for building production-ready AI applications with minimal boilerplate code.`

	files := []ManifestFile{
		{Name: "research-notes.md", Type: "text/markdown"},
	}

	summary := r.llmSummarize(task, output, files)

	t.Logf("LLM Summary: %s", summary)
	assert.NotEmpty(t, summary, "summary should not be empty")
	assert.Less(t, len(summary), 500, "summary should be concise (< 500 chars)")
}

// ============================================================================
// updateManifestForTask end-to-end — integration test
// ============================================================================

func TestUpdateManifestForTaskWithLLMSummary(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test (requires real LLM)")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)
	setupLLMProvider(t)

	wfs := newTestWorkspaceFS(t)
	auth := &oauthtypes.AuthorizedInfo{UserID: "test-user", TeamID: "test-team"}
	ctx := robottypes.NewContext(context.Background(), auth)

	execDir := "robots/r1/exec1"
	r := &Runner{
		ctx:     ctx,
		wsFS:    wfs,
		execDir: execDir,
		robot:   &robottypes.Robot{MemberID: "r1"},
	}

	task := robottypes.Task{
		ID:           "task-001",
		Description:  "Research Yao Agents platform",
		ExecutorID:   "yao.general",
		ExecutorType: robottypes.ExecutorAssistant,
		Status:       robottypes.TaskPending,
	}
	exec := &robottypes.Execution{
		ID:    "exec1",
		Tasks: []robottypes.Task{task},
	}

	r.initManifest(exec)

	// Verify manifest was created with pending status
	m, err := r.readManifest()
	require.NoError(t, err)
	require.Len(t, m.Tasks, 1)
	assert.Equal(t, "pending", m.Tasks[0].Status)

	// Write an artifact file that scanTaskArtifacts can discover
	require.NoError(t, wfs.MkdirAll(path.Join(execDir, "task-001"), 0755))
	require.NoError(t, wfs.WriteFile(path.Join(execDir, "task-001", "notes.md"), []byte("research content"), 0644))

	// Simulate task completion with output referencing the artifact
	wsID, _ := wfs.GetID()
	result := &robottypes.TaskResult{
		Success:  true,
		Duration: 5000,
		Output:   "Completed research. Notes saved to workspace://" + wsID + "/" + execDir + "/task-001/notes.md",
	}

	r.updateManifestForTask(&task, result)

	// Re-read and verify
	m, err = r.readManifest()
	require.NoError(t, err)
	require.Len(t, m.Tasks, 1)

	mt := m.Tasks[0]
	assert.Equal(t, "completed", mt.Status)
	assert.NotEmpty(t, mt.Summary, "summary should be generated (LLM or fallback)")
	t.Logf("Summary: %s", mt.Summary)

	// Files should include the scanned artifact, potentially with URI merged
	assert.NotEmpty(t, mt.Files, "files should be populated")
	found := false
	for _, f := range mt.Files {
		if f.Name == "notes.md" {
			found = true
			assert.Equal(t, "text/markdown", f.Type)
		}
	}
	assert.True(t, found, "notes.md should be in files list")
}
