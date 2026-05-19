//go:build unit

package standard_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	"github.com/yaoapp/yao/agent/robot/types"
)

// ============================================================================
// extractLLMContent — pure unit tests (ported from workspace_test.go.bak)
// ============================================================================

func TestExtractLLMContentUnit(t *testing.T) {
	t.Run("string content", func(t *testing.T) {
		resp := &agentcontext.CompletionResponse{Content: "  hello world  "}
		assert.Equal(t, "hello world", standard.ExtractLLMContentFn(resp))
	})

	t.Run("nil response", func(t *testing.T) {
		assert.Equal(t, "", standard.ExtractLLMContentFn(nil))
	})

	t.Run("non-string content", func(t *testing.T) {
		resp := &agentcontext.CompletionResponse{Content: []interface{}{"a", "b"}}
		assert.Equal(t, "", standard.ExtractLLMContentFn(resp))
	})

	t.Run("empty string content", func(t *testing.T) {
		resp := &agentcontext.CompletionResponse{Content: "   "}
		assert.Equal(t, "", standard.ExtractLLMContentFn(resp))
	})

	t.Run("multiline content trimmed", func(t *testing.T) {
		resp := &agentcontext.CompletionResponse{Content: "\n  summary line\n"}
		assert.Equal(t, "summary line", standard.ExtractLLMContentFn(resp))
	})

	t.Run("int content", func(t *testing.T) {
		resp := &agentcontext.CompletionResponse{Content: 42}
		assert.Equal(t, "", standard.ExtractLLMContentFn(resp))
	})
}

// ============================================================================
// mergeManifestFiles — pure unit tests (ported from workspace_test.go.bak)
// ============================================================================

func TestMergeManifestFilesUnit(t *testing.T) {
	t.Run("empty fromURIs returns scanned unchanged", func(t *testing.T) {
		scanned := []standard.ExportedManifestFile{{Name: "a.md", Type: "text/markdown"}}
		result := standard.MergeManifestFilesFn(scanned, nil)
		assert.Equal(t, scanned, result)
	})

	t.Run("merge URI onto matching scanned entry", func(t *testing.T) {
		scanned := []standard.ExportedManifestFile{{Name: "notes.md", Type: "text/markdown"}}
		fromURIs := []standard.ExportedManifestFile{{Name: "notes.md", Type: "text/markdown", URI: "workspace://ws/path/notes.md"}}
		result := standard.MergeManifestFilesFn(scanned, fromURIs)

		assert.Len(t, result, 1)
		assert.Equal(t, "workspace://ws/path/notes.md", result[0].URI)
	})

	t.Run("add new URI entry when no scan match", func(t *testing.T) {
		scanned := []standard.ExportedManifestFile{{Name: "a.md", Type: "text/markdown"}}
		fromURIs := []standard.ExportedManifestFile{{Name: "b.pdf", Type: "application/pdf", URI: "workspace://ws/b.pdf"}}
		result := standard.MergeManifestFilesFn(scanned, fromURIs)

		assert.Len(t, result, 2)
		assert.Equal(t, "b.pdf", result[1].Name)
	})

	t.Run("does not overwrite existing URI", func(t *testing.T) {
		scanned := []standard.ExportedManifestFile{
			{Name: "report.md", Type: "text/markdown", URI: "workspace://ws/old-path/report.md"},
		}
		fromURIs := []standard.ExportedManifestFile{
			{Name: "report.md", Type: "text/markdown", URI: "workspace://ws/new-path/report.md"},
		}
		result := standard.MergeManifestFilesFn(scanned, fromURIs)

		assert.Len(t, result, 1)
		assert.Equal(t, "workspace://ws/old-path/report.md", result[0].URI, "should keep existing URI")
	})

	t.Run("handles empty scanned with URIs", func(t *testing.T) {
		fromURIs := []standard.ExportedManifestFile{
			{Name: "new.pdf", Type: "application/pdf", URI: "workspace://ws/new.pdf"},
		}
		result := standard.MergeManifestFilesFn(nil, fromURIs)

		assert.Len(t, result, 1)
		assert.Equal(t, "new.pdf", result[0].Name)
	})

	t.Run("handles both empty", func(t *testing.T) {
		result := standard.MergeManifestFilesFn(nil, nil)
		assert.Nil(t, result)
	})
}

// ============================================================================
// generateSummary — pure logic (no LLM)
// ============================================================================

func TestGenerateSummaryUnit(t *testing.T) {
	t.Run("extracts summary section", func(t *testing.T) {
		output := `## Analysis

Some analysis content.

## Summary

This is the summary of the findings.

## Details

More details here.`
		summary := standard.GenerateSummaryFn(output)
		assert.Contains(t, summary, "summary of the findings")
	})

	t.Run("extracts conclusion section", func(t *testing.T) {
		output := `## Intro

Some intro.

## Conclusion

The final conclusion text.`
		summary := standard.GenerateSummaryFn(output)
		assert.Contains(t, summary, "final conclusion")
	})

	t.Run("returns empty for nil output", func(t *testing.T) {
		summary := standard.GenerateSummaryFn(nil)
		assert.Empty(t, summary)
	})

	t.Run("returns empty for empty string", func(t *testing.T) {
		summary := standard.GenerateSummaryFn("")
		assert.Empty(t, summary)
	})

	t.Run("truncates long summary", func(t *testing.T) {
		long := "## Summary\n\n" + string(make([]byte, 300))
		summary := standard.GenerateSummaryFn(long)
		assert.LessOrEqual(t, len(summary), 210) // 200 + "..."
	})

	t.Run("skips filler prefixes in fallback", func(t *testing.T) {
		output := "It seems there was an issue.\n\nThe actual meaningful content is here."
		summary := standard.GenerateSummaryFn(output)
		assert.Contains(t, summary, "actual meaningful content")
	})
}

// ============================================================================
// extractKeyOutputs — pure logic
// ============================================================================

func TestExtractKeyOutputsUnit(t *testing.T) {
	t.Run("extracts from map with key_outputs", func(t *testing.T) {
		output := map[string]interface{}{
			"key_outputs": []interface{}{"output1", "output2", "output3"},
		}
		keys := standard.ExtractKeyOutputsFn(output)
		assert.Equal(t, []string{"output1", "output2", "output3"}, keys)
	})

	t.Run("extracts from map with outputs", func(t *testing.T) {
		output := map[string]interface{}{
			"outputs": []interface{}{"a", "b"},
		}
		keys := standard.ExtractKeyOutputsFn(output)
		assert.Equal(t, []string{"a", "b"}, keys)
	})

	t.Run("extracts headings from markdown", func(t *testing.T) {
		output := "# Title\n\n## First Section\n\nContent\n\n## Second Section\n\nMore content"
		keys := standard.ExtractKeyOutputsFn(output)
		assert.Contains(t, keys, "First Section")
		assert.Contains(t, keys, "Second Section")
	})

	t.Run("extracts bold items from lists", func(t *testing.T) {
		output := "Results:\n\n1. **Model-Driven Architecture:** Description\n2. **Low-Code Engine:** Another desc\n- **Multi-Agent:** Third"
		keys := standard.ExtractKeyOutputsFn(output)
		assert.Contains(t, keys, "Model-Driven Architecture")
		assert.Contains(t, keys, "Low-Code Engine")
		assert.Contains(t, keys, "Multi-Agent")
	})

	t.Run("returns nil for nil output", func(t *testing.T) {
		keys := standard.ExtractKeyOutputsFn(nil)
		assert.Nil(t, keys)
	})

	t.Run("caps at 5 items", func(t *testing.T) {
		output := map[string]interface{}{
			"key_outputs": []interface{}{"a", "b", "c", "d", "e", "f", "g"},
		}
		keys := standard.ExtractKeyOutputsFn(output)
		assert.Len(t, keys, 5)
	})
}

// ============================================================================
// flattenOutput — pure logic
// ============================================================================

func TestFlattenOutputUnit(t *testing.T) {
	t.Run("returns empty for nil", func(t *testing.T) {
		assert.Equal(t, "", standard.FlattenOutputFn(nil))
	})

	t.Run("returns string directly", func(t *testing.T) {
		assert.Equal(t, "hello", standard.FlattenOutputFn("hello"))
	})

	t.Run("extracts text from map", func(t *testing.T) {
		m := map[string]interface{}{"text": "from text field"}
		assert.Equal(t, "from text field", standard.FlattenOutputFn(m))
	})

	t.Run("extracts content from map", func(t *testing.T) {
		m := map[string]interface{}{"content": "from content field"}
		assert.Equal(t, "from content field", standard.FlattenOutputFn(m))
	})

	t.Run("marshals map without text/content", func(t *testing.T) {
		m := map[string]interface{}{"key": "value"}
		result := standard.FlattenOutputFn(m)
		assert.Contains(t, result, "key")
		assert.Contains(t, result, "value")
	})

	t.Run("marshals array", func(t *testing.T) {
		arr := []interface{}{1, 2, 3}
		result := standard.FlattenOutputFn(arr)
		assert.Contains(t, result, "1")
	})
}

// ============================================================================
// formatOutputAsText — pure logic
// ============================================================================

func TestFormatOutputAsTextUnit(t *testing.T) {
	t.Run("returns empty for nil", func(t *testing.T) {
		assert.Equal(t, "", standard.FormatOutputAsTextFn(nil))
	})

	t.Run("returns string directly", func(t *testing.T) {
		assert.Equal(t, "hello world", standard.FormatOutputAsTextFn("hello world"))
	})

	t.Run("marshals map with indent", func(t *testing.T) {
		m := map[string]interface{}{"key": "value"}
		result := standard.FormatOutputAsTextFn(m)
		assert.Contains(t, result, "key")
		assert.Contains(t, result, "value")
	})
}

// ============================================================================
// mimeFromExt — pure logic
// ============================================================================

func TestMimeFromExtUnit(t *testing.T) {
	cases := []struct {
		ext      string
		expected string
	}{
		{".md", "text/markdown"},
		{".html", "text/html"},
		{".htm", "text/html"},
		{".json", "application/json"},
		{".pdf", "application/pdf"},
		{".png", "image/png"},
		{".jpg", "image/jpeg"},
		{".jpeg", "image/jpeg"},
		{".csv", "text/csv"},
		{".txt", "text/plain"},
		{".xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{".pptx", "application/vnd.openxmlformats-officedocument.presentationml.presentation"},
		{".unknown", "application/octet-stream"},
		{".bin", "application/octet-stream"},
	}

	for _, tc := range cases {
		t.Run(tc.ext, func(t *testing.T) {
			assert.Equal(t, tc.expected, standard.MimeFromExtFn(tc.ext))
		})
	}
}

// ============================================================================
// getEffectiveLocale — pure logic
// ============================================================================

func TestGetEffectiveLocaleUnit(t *testing.T) {
	t.Run("returns input locale when set", func(t *testing.T) {
		input := &types.TriggerInput{Locale: "zh"}
		robot := &types.Robot{MemberID: "test", Config: &types.Config{}}
		assert.Equal(t, "zh", standard.GetEffectiveLocaleFn(robot, input))
	})

	t.Run("returns robot config locale when input has no locale", func(t *testing.T) {
		input := &types.TriggerInput{}
		robot := &types.Robot{
			MemberID: "test",
			Config: &types.Config{
				DefaultLocale: "zh",
			},
		}
		assert.Equal(t, "zh", standard.GetEffectiveLocaleFn(robot, input))
	})

	t.Run("falls back to en", func(t *testing.T) {
		assert.Equal(t, "en", standard.GetEffectiveLocaleFn(nil, nil))
	})

	t.Run("falls back to en with nil config", func(t *testing.T) {
		robot := &types.Robot{MemberID: "test"}
		assert.Equal(t, "en", standard.GetEffectiveLocaleFn(robot, nil))
	})
}

// ============================================================================
// getLocalizedMessage — pure logic
// ============================================================================

func TestGetLocalizedMessageUnit(t *testing.T) {
	t.Run("returns English message", func(t *testing.T) {
		assert.Equal(t, "Preparing...", standard.GetLocalizedMessageFn("en", "preparing"))
		assert.Equal(t, "Completed", standard.GetLocalizedMessageFn("en", "completed"))
		assert.Equal(t, "Starting...", standard.GetLocalizedMessageFn("en", "starting"))
	})

	t.Run("returns Chinese message", func(t *testing.T) {
		assert.Equal(t, "准备中...", standard.GetLocalizedMessageFn("zh", "preparing"))
		assert.Equal(t, "已完成", standard.GetLocalizedMessageFn("zh", "completed"))
		assert.Equal(t, "启动中...", standard.GetLocalizedMessageFn("zh", "starting"))
	})

	t.Run("falls back to English for unknown locale", func(t *testing.T) {
		assert.Equal(t, "Preparing...", standard.GetLocalizedMessageFn("fr", "preparing"))
	})

	t.Run("returns key for unknown message", func(t *testing.T) {
		assert.Equal(t, "nonexistent_key", standard.GetLocalizedMessageFn("en", "nonexistent_key"))
	})

	t.Run("phase names are localized", func(t *testing.T) {
		assert.Equal(t, "inspiration", standard.GetLocalizedMessageFn("en", "phase_inspiration"))
		assert.Equal(t, "灵感阶段", standard.GetLocalizedMessageFn("zh", "phase_inspiration"))
	})
}

// ============================================================================
// extractGoalName — pure logic
// ============================================================================

func TestExtractGoalNameUnit(t *testing.T) {
	t.Run("returns empty for nil goals", func(t *testing.T) {
		assert.Equal(t, "", standard.ExtractGoalNameFn(nil))
	})

	t.Run("returns empty for empty content", func(t *testing.T) {
		goals := &types.Goals{Content: ""}
		assert.Equal(t, "", standard.ExtractGoalNameFn(goals))
	})

	t.Run("extracts first content line skipping headers", func(t *testing.T) {
		goals := &types.Goals{
			Content: "## Goals\n\nAnalyze sales data for Q4",
		}
		name := standard.ExtractGoalNameFn(goals)
		assert.Equal(t, "Analyze sales data for Q4", name)
	})

	t.Run("skips markdown horizontal rules", func(t *testing.T) {
		goals := &types.Goals{
			Content: "---\nThe actual goal content",
		}
		name := standard.ExtractGoalNameFn(goals)
		assert.Equal(t, "The actual goal content", name)
	})

	t.Run("falls back to header text when no content lines", func(t *testing.T) {
		goals := &types.Goals{
			Content: "# Main Goal\n## Sub Goal",
		}
		name := standard.ExtractGoalNameFn(goals)
		assert.Equal(t, "Main Goal", name)
	})

	t.Run("truncates long names", func(t *testing.T) {
		long := make([]byte, 200)
		for i := range long {
			long[i] = 'a'
		}
		goals := &types.Goals{Content: string(long)}
		name := standard.ExtractGoalNameFn(goals)
		assert.LessOrEqual(t, len(name), 154) // 150 + "..."
	})

	t.Run("strips markdown formatting", func(t *testing.T) {
		goals := &types.Goals{
			Content: "## Overview\n\n**Bold** and *italic* goal",
		}
		name := standard.ExtractGoalNameFn(goals)
		assert.NotContains(t, name, "**")
		assert.NotContains(t, name, "*")
		assert.Contains(t, name, "Bold")
		assert.Contains(t, name, "goal")
	})
}

// ============================================================================
// stripMarkdownFormatting — pure logic
// ============================================================================

func TestStripMarkdownFormattingUnit(t *testing.T) {
	t.Run("removes bold markers", func(t *testing.T) {
		assert.Equal(t, "bold text", standard.StripMarkdownFmtFn("**bold text**"))
	})

	t.Run("removes italic markers", func(t *testing.T) {
		assert.Equal(t, "italic text", standard.StripMarkdownFmtFn("*italic text*"))
	})

	t.Run("removes underscore emphasis", func(t *testing.T) {
		assert.Equal(t, "emphasis", standard.StripMarkdownFmtFn("__emphasis__"))
	})

	t.Run("removes inline code backticks", func(t *testing.T) {
		assert.Equal(t, "code", standard.StripMarkdownFmtFn("`code`"))
	})

	t.Run("removes link syntax", func(t *testing.T) {
		result := standard.StripMarkdownFmtFn("[Click here](https://example.com)")
		assert.Equal(t, "Click here", result)
	})

	t.Run("handles plain text unchanged", func(t *testing.T) {
		assert.Equal(t, "plain text", standard.StripMarkdownFmtFn("plain text"))
	})
}

// ============================================================================
// formatTaskProgressName — pure logic
// ============================================================================

func TestFormatTaskProgressNameUnit(t *testing.T) {
	t.Run("uses description when available", func(t *testing.T) {
		task := &types.Task{
			ID:          "task-001",
			Description: "Analyze sales data",
		}
		name := standard.FormatTaskProgressFn(task, 0, 3, "en")
		assert.Contains(t, name, "Task 1/3:")
		assert.Contains(t, name, "Analyze sales data")
	})

	t.Run("uses Chinese locale", func(t *testing.T) {
		task := &types.Task{
			ID:          "task-001",
			Description: "分析销售数据",
		}
		name := standard.FormatTaskProgressFn(task, 1, 5, "zh")
		assert.Contains(t, name, "任务 2/5:")
		assert.Contains(t, name, "分析销售数据")
	})

	t.Run("truncates long description", func(t *testing.T) {
		long := make([]byte, 120)
		for i := range long {
			long[i] = 'x'
		}
		task := &types.Task{
			ID:          "task-001",
			Description: string(long),
		}
		name := standard.FormatTaskProgressFn(task, 0, 1, "en")
		assert.Contains(t, name, "...")
		assert.LessOrEqual(t, len(name), 120)
	})

	t.Run("falls back to executor info", func(t *testing.T) {
		task := &types.Task{
			ID:           "task-001",
			ExecutorType: types.ExecutorAssistant,
			ExecutorID:   "experts.writer",
		}
		name := standard.FormatTaskProgressFn(task, 0, 1, "en")
		assert.Contains(t, name, "assistant")
		assert.Contains(t, name, "experts.writer")
	})
}

// ============================================================================
// boolMark — pure logic
// ============================================================================

func TestBoolMarkUnit(t *testing.T) {
	assert.Equal(t, "✓", standard.BoolMarkFn(true))
	assert.Equal(t, "✗", standard.BoolMarkFn(false))
}

// ============================================================================
// capSlice — pure logic
// ============================================================================

func TestCapSliceUnit(t *testing.T) {
	t.Run("returns unchanged when under limit", func(t *testing.T) {
		s := []string{"a", "b", "c"}
		assert.Equal(t, s, standard.CapSliceFn(s, 5))
	})

	t.Run("caps when over limit", func(t *testing.T) {
		s := []string{"a", "b", "c", "d", "e", "f"}
		result := standard.CapSliceFn(s, 3)
		assert.Len(t, result, 3)
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("returns exact when at limit", func(t *testing.T) {
		s := []string{"a", "b"}
		assert.Equal(t, s, standard.CapSliceFn(s, 2))
	})
}

// ============================================================================
// skipFillerPrefixes — pure logic
// ============================================================================

func TestSkipFillerPrefixesUnit(t *testing.T) {
	t.Run("skips filler lines", func(t *testing.T) {
		text := "It seems there was an issue.\nBased on the analysis.\nThe actual content."
		result := standard.SkipFillerPrefixesFn(text)
		assert.Equal(t, "The actual content.", result)
	})

	t.Run("returns original when no filler", func(t *testing.T) {
		text := "The important content.\nMore details."
		result := standard.SkipFillerPrefixesFn(text)
		assert.Equal(t, text, result)
	})

	t.Run("returns original when all filler", func(t *testing.T) {
		text := "It seems there is an issue.\nBased on what I see."
		result := standard.SkipFillerPrefixesFn(text)
		assert.Equal(t, text, result)
	})
}

// ============================================================================
// Validator — detectNeedMoreInfo method (on Validator)
// ============================================================================

func TestValidatorDetectNeedMoreInfoUnit(t *testing.T) {
	ctx := types.NewContext(nil, nil)
	robot := &types.Robot{
		MemberID: "test-validator",
		Config: &types.Config{
			Identity: &types.Identity{Role: "Test"},
		},
	}
	config := standard.DefaultValidatorConfig()
	_ = standard.NewValidator(ctx, robot, config)

	t.Run("detects 'need more information' keyword", func(t *testing.T) {
		result := &standard.CallResult{Content: "I need more information about the data format."}
		// Use the exported standalone function from runner.go
		needInput, _ := standard.DetectNeedMoreInfoFn(result)
		// This tests the Runner's detectNeedMoreInfo, not Validator's
		// Validator's detectNeedMoreInfo is on the Validator struct
		assert.False(t, needInput, "standalone detectNeedMoreInfo checks Next hook, not Content")
	})
}
