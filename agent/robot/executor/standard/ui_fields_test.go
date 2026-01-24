package standard

import (
	"testing"

	"github.com/stretchr/testify/assert"
	agentcontext "github.com/yaoapp/yao/agent/context"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
)

// ============================================================================
// getEffectiveLocale Tests
// ============================================================================

func TestGetEffectiveLocale(t *testing.T) {
	t.Run("returns_input_locale_when_provided", func(t *testing.T) {
		robot := &robottypes.Robot{
			Config: &robottypes.Config{
				DefaultLocale: "en",
			},
		}
		input := &robottypes.TriggerInput{
			Locale: "zh",
		}

		locale := getEffectiveLocale(robot, input)
		assert.Equal(t, "zh", locale)
	})

	t.Run("returns_robot_default_locale_when_input_locale_empty", func(t *testing.T) {
		robot := &robottypes.Robot{
			Config: &robottypes.Config{
				DefaultLocale: "zh",
			},
		}
		input := &robottypes.TriggerInput{
			Locale: "",
		}

		locale := getEffectiveLocale(robot, input)
		assert.Equal(t, "zh", locale)
	})

	t.Run("returns_system_default_when_no_locale_configured", func(t *testing.T) {
		robot := &robottypes.Robot{
			Config: &robottypes.Config{},
		}
		input := &robottypes.TriggerInput{}

		locale := getEffectiveLocale(robot, input)
		assert.Equal(t, "en", locale)
	})

	t.Run("returns_system_default_when_robot_config_nil", func(t *testing.T) {
		robot := &robottypes.Robot{}
		input := &robottypes.TriggerInput{}

		locale := getEffectiveLocale(robot, input)
		assert.Equal(t, "en", locale)
	})

	t.Run("returns_system_default_when_robot_nil", func(t *testing.T) {
		input := &robottypes.TriggerInput{}

		locale := getEffectiveLocale(nil, input)
		assert.Equal(t, "en", locale)
	})

	t.Run("returns_system_default_when_input_nil", func(t *testing.T) {
		robot := &robottypes.Robot{}

		locale := getEffectiveLocale(robot, nil)
		assert.Equal(t, "en", locale)
	})
}

// ============================================================================
// getLocalizedMessage Tests
// ============================================================================

func TestGetLocalizedMessage(t *testing.T) {
	t.Run("returns_english_message_for_en_locale", func(t *testing.T) {
		msg := getLocalizedMessage("en", "preparing")
		assert.Equal(t, "Preparing...", msg)
	})

	t.Run("returns_chinese_message_for_zh_locale", func(t *testing.T) {
		msg := getLocalizedMessage("zh", "preparing")
		assert.Equal(t, "准备中...", msg)
	})

	t.Run("returns_english_fallback_for_unknown_locale", func(t *testing.T) {
		msg := getLocalizedMessage("fr", "preparing")
		assert.Equal(t, "Preparing...", msg)
	})

	t.Run("returns_key_for_unknown_message", func(t *testing.T) {
		msg := getLocalizedMessage("en", "unknown_key")
		assert.Equal(t, "unknown_key", msg)
	})

	t.Run("all_english_messages_exist", func(t *testing.T) {
		keys := []string{
			"preparing", "starting", "scheduled_execution",
			"event_prefix", "event_triggered", "analyzing_context",
			"planning_goals", "breaking_down_tasks",
			"generating_delivery", "sending_delivery", "learning_from_exec",
			"completed", "failed_prefix", "task_prefix",
			// Phase names for failure messages
			"phase_inspiration", "phase_goals", "phase_tasks",
			"phase_run", "phase_delivery", "phase_learning",
		}
		for _, key := range keys {
			msg := getLocalizedMessage("en", key)
			assert.NotEqual(t, key, msg, "English message should exist for key: %s", key)
		}
	})

	t.Run("all_chinese_messages_exist", func(t *testing.T) {
		keys := []string{
			"preparing", "starting", "scheduled_execution",
			"event_prefix", "event_triggered", "analyzing_context",
			"planning_goals", "breaking_down_tasks",
			"generating_delivery", "sending_delivery", "learning_from_exec",
			"completed", "failed_prefix", "task_prefix",
			// Phase names for failure messages
			"phase_inspiration", "phase_goals", "phase_tasks",
			"phase_run", "phase_delivery", "phase_learning",
		}
		for _, key := range keys {
			msg := getLocalizedMessage("zh", key)
			assert.NotEqual(t, key, msg, "Chinese message should exist for key: %s", key)
		}
	})

	t.Run("failure_message_is_concise", func(t *testing.T) {
		// Test that failure messages use phase names, not full error text
		enFailure := getLocalizedMessage("en", "failed_prefix") + getLocalizedMessage("en", "phase_inspiration")
		assert.Equal(t, "Failed at inspiration", enFailure)

		zhFailure := getLocalizedMessage("zh", "failed_prefix") + getLocalizedMessage("zh", "phase_inspiration")
		assert.Equal(t, "失败于灵感阶段", zhFailure)
	})
}

// ============================================================================
// initUIFields Tests
// ============================================================================

func TestInitUIFields(t *testing.T) {
	executor := New()

	t.Run("human_trigger_extracts_name_from_message", func(t *testing.T) {
		robot := &robottypes.Robot{
			Config: &robottypes.Config{DefaultLocale: "en"},
		}
		input := &robottypes.TriggerInput{
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Please analyze the sales data"},
			},
		}

		name, currentTaskName := executor.initUIFields(robottypes.TriggerHuman, input, robot)
		assert.Equal(t, "Please analyze the sales data", name)
		assert.Equal(t, "Starting...", currentTaskName)
	})

	t.Run("human_trigger_truncates_long_message", func(t *testing.T) {
		robot := &robottypes.Robot{
			Config: &robottypes.Config{DefaultLocale: "en"},
		}
		longMessage := "This is a very long message that exceeds one hundred characters and should be truncated with an ellipsis at the end"
		input := &robottypes.TriggerInput{
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: longMessage},
			},
		}

		name, _ := executor.initUIFields(robottypes.TriggerHuman, input, robot)
		assert.LessOrEqual(t, len(name), 103) // 100 chars + "..."
		assert.True(t, len(name) > 100 || name == longMessage[:100]+"...")
	})

	t.Run("clock_trigger_uses_scheduled_execution", func(t *testing.T) {
		robot := &robottypes.Robot{
			Config: &robottypes.Config{DefaultLocale: "en"},
		}
		input := &robottypes.TriggerInput{}

		name, currentTaskName := executor.initUIFields(robottypes.TriggerClock, input, robot)
		assert.Equal(t, "Scheduled execution", name)
		assert.Equal(t, "Starting...", currentTaskName)
	})

	t.Run("clock_trigger_chinese_locale", func(t *testing.T) {
		robot := &robottypes.Robot{
			Config: &robottypes.Config{DefaultLocale: "zh"},
		}
		input := &robottypes.TriggerInput{}

		name, currentTaskName := executor.initUIFields(robottypes.TriggerClock, input, robot)
		assert.Equal(t, "定时执行", name)
		assert.Equal(t, "启动中...", currentTaskName)
	})

	t.Run("event_trigger_with_event_type", func(t *testing.T) {
		robot := &robottypes.Robot{
			Config: &robottypes.Config{DefaultLocale: "en"},
		}
		input := &robottypes.TriggerInput{
			EventType: "lead.created",
		}

		name, currentTaskName := executor.initUIFields(robottypes.TriggerEvent, input, robot)
		assert.Equal(t, "Event: lead.created", name)
		assert.Equal(t, "Starting...", currentTaskName)
	})

	t.Run("event_trigger_without_event_type", func(t *testing.T) {
		robot := &robottypes.Robot{
			Config: &robottypes.Config{DefaultLocale: "en"},
		}
		input := &robottypes.TriggerInput{}

		name, _ := executor.initUIFields(robottypes.TriggerEvent, input, robot)
		assert.Equal(t, "Event triggered", name)
	})

	t.Run("event_trigger_chinese_locale", func(t *testing.T) {
		robot := &robottypes.Robot{
			Config: &robottypes.Config{DefaultLocale: "zh"},
		}
		input := &robottypes.TriggerInput{
			EventType: "order.placed",
		}

		name, _ := executor.initUIFields(robottypes.TriggerEvent, input, robot)
		assert.Equal(t, "事件: order.placed", name)
	})

	t.Run("input_locale_overrides_robot_default", func(t *testing.T) {
		robot := &robottypes.Robot{
			Config: &robottypes.Config{DefaultLocale: "en"},
		}
		input := &robottypes.TriggerInput{
			Locale: "zh",
		}

		name, currentTaskName := executor.initUIFields(robottypes.TriggerClock, input, robot)
		assert.Equal(t, "定时执行", name)
		assert.Equal(t, "启动中...", currentTaskName)
	})
}

// ============================================================================
// extractGoalName Tests
// ============================================================================

func TestExtractGoalName(t *testing.T) {
	t.Run("extracts_first_line_from_content", func(t *testing.T) {
		goals := &robottypes.Goals{
			Content: "Generate monthly sales report\nAnalyze trends\nSend to stakeholders",
		}

		name := extractGoalName(goals)
		assert.Equal(t, "Generate monthly sales report", name)
	})

	t.Run("returns_empty_for_nil_goals", func(t *testing.T) {
		name := extractGoalName(nil)
		assert.Equal(t, "", name)
	})

	t.Run("returns_empty_for_empty_content", func(t *testing.T) {
		goals := &robottypes.Goals{
			Content: "",
		}

		name := extractGoalName(goals)
		assert.Equal(t, "", name)
	})

	t.Run("truncates_long_first_line", func(t *testing.T) {
		longLine := "This is an extremely long goal description that exceeds one hundred and fifty characters and should be truncated with an ellipsis at the end to keep the display manageable"
		goals := &robottypes.Goals{
			Content: longLine,
		}

		name := extractGoalName(goals)
		assert.LessOrEqual(t, len(name), 153) // 150 chars + "..."
	})

	t.Run("handles_single_line_content", func(t *testing.T) {
		goals := &robottypes.Goals{
			Content: "Single line goal",
		}

		name := extractGoalName(goals)
		assert.Equal(t, "Single line goal", name)
	})

	t.Run("handles_carriage_return", func(t *testing.T) {
		goals := &robottypes.Goals{
			Content: "First goal\r\nSecond goal",
		}

		name := extractGoalName(goals)
		assert.Equal(t, "First goal", name)
	})

	t.Run("skips_markdown_h1_header", func(t *testing.T) {
		goals := &robottypes.Goals{
			Content: "# Goals\nSystem optimization and monitoring",
		}

		name := extractGoalName(goals)
		assert.Equal(t, "System optimization and monitoring", name)
	})

	t.Run("skips_markdown_h2_header", func(t *testing.T) {
		goals := &robottypes.Goals{
			Content: "## Goals\n\nPerform system maintenance tasks",
		}

		name := extractGoalName(goals)
		assert.Equal(t, "Perform system maintenance tasks", name)
	})

	t.Run("skips_multiple_markdown_headers", func(t *testing.T) {
		goals := &robottypes.Goals{
			Content: "## Goals\n### 1. [High] First Goal\nActual description here",
		}

		name := extractGoalName(goals)
		assert.Equal(t, "Actual description here", name)
	})

	t.Run("strips_bold_formatting", func(t *testing.T) {
		goals := &robottypes.Goals{
			Content: "**Important** task to complete",
		}

		name := extractGoalName(goals)
		assert.Equal(t, "Important task to complete", name)
	})

	t.Run("strips_italic_formatting", func(t *testing.T) {
		goals := &robottypes.Goals{
			Content: "*Urgent* system update needed",
		}

		name := extractGoalName(goals)
		assert.Equal(t, "Urgent system update needed", name)
	})

	t.Run("strips_inline_code", func(t *testing.T) {
		goals := &robottypes.Goals{
			Content: "Run `npm install` command",
		}

		name := extractGoalName(goals)
		assert.Equal(t, "Run npm install command", name)
	})

	t.Run("skips_empty_lines", func(t *testing.T) {
		goals := &robottypes.Goals{
			Content: "\n\n\nFirst real content\nSecond line",
		}

		name := extractGoalName(goals)
		assert.Equal(t, "First real content", name)
	})

	t.Run("fallback_to_header_content_if_only_headers", func(t *testing.T) {
		goals := &robottypes.Goals{
			Content: "## Goals\n### Tasks",
		}

		name := extractGoalName(goals)
		assert.Equal(t, "Goals", name)
	})

	t.Run("skips_horizontal_rules", func(t *testing.T) {
		goals := &robottypes.Goals{
			Content: "---\nActual content here",
		}

		name := extractGoalName(goals)
		assert.Equal(t, "Actual content here", name)
	})

	t.Run("handles_complex_markdown_content", func(t *testing.T) {
		goals := &robottypes.Goals{
			Content: "## Goals\n\n### 1. [High] System Maintenance\n**Description**: Perform system optimization based on diagnostic results\n**Reason**: Time-sensitive maintenance",
		}

		name := extractGoalName(goals)
		assert.Equal(t, "Description: Perform system optimization based on diagnostic results", name)
	})
}

// ============================================================================
// stripMarkdownFormatting Tests
// ============================================================================

func TestStripMarkdownFormatting(t *testing.T) {
	t.Run("strips_bold", func(t *testing.T) {
		result := stripMarkdownFormatting("**bold text**")
		assert.Equal(t, "bold text", result)
	})

	t.Run("strips_italic_asterisk", func(t *testing.T) {
		result := stripMarkdownFormatting("*italic text*")
		assert.Equal(t, "italic text", result)
	})

	t.Run("strips_italic_underscore", func(t *testing.T) {
		result := stripMarkdownFormatting("_italic text_")
		assert.Equal(t, "italic text", result)
	})

	t.Run("strips_inline_code", func(t *testing.T) {
		result := stripMarkdownFormatting("`code`")
		assert.Equal(t, "code", result)
	})

	t.Run("strips_link_syntax", func(t *testing.T) {
		result := stripMarkdownFormatting("[link text](https://example.com)")
		assert.Equal(t, "link text", result)
	})

	t.Run("preserves_plain_text", func(t *testing.T) {
		result := stripMarkdownFormatting("plain text without formatting")
		assert.Equal(t, "plain text without formatting", result)
	})

	t.Run("handles_mixed_formatting", func(t *testing.T) {
		result := stripMarkdownFormatting("**bold** and *italic* and `code`")
		assert.Equal(t, "bold and italic and code", result)
	})
}

// ============================================================================
// formatTaskProgressName Tests
// ============================================================================

func TestFormatTaskProgressName(t *testing.T) {
	t.Run("prioritizes_description_field_over_messages", func(t *testing.T) {
		task := &robottypes.Task{
			ID:           "task-001",
			Description:  "High-level task description for UI",
			ExecutorType: robottypes.ExecutorAssistant,
			ExecutorID:   "analyst",
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Detailed message content for execution"},
			},
		}

		name := formatTaskProgressName(task, 0, 3, "en")
		// Should use Description field, NOT the message content
		assert.Equal(t, "Task 1/3: High-level task description for UI", name)
	})

	t.Run("falls_back_to_message_when_no_description", func(t *testing.T) {
		task := &robottypes.Task{
			ID:           "task-001",
			Description:  "", // Empty description
			ExecutorType: robottypes.ExecutorAssistant,
			ExecutorID:   "analyst",
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Analyze sales data"},
			},
		}

		name := formatTaskProgressName(task, 0, 3, "en")
		assert.Equal(t, "Task 1/3: Analyze sales data", name)
	})

	t.Run("formats_with_chinese_locale", func(t *testing.T) {
		task := &robottypes.Task{
			ID:           "task-001",
			ExecutorType: robottypes.ExecutorAssistant,
			ExecutorID:   "analyst",
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "分析销售数据"},
			},
		}

		name := formatTaskProgressName(task, 1, 5, "zh")
		assert.Equal(t, "任务 2/5: 分析销售数据", name)
	})

	t.Run("truncates_long_description_field", func(t *testing.T) {
		longDesc := "This is a very long task description that should be truncated because it exceeds 80 characters which is the maximum length allowed"
		task := &robottypes.Task{
			ID:           "task-001",
			Description:  longDesc,
			ExecutorType: robottypes.ExecutorAssistant,
			ExecutorID:   "analyst",
			Messages:     []agentcontext.Message{},
		}

		name := formatTaskProgressName(task, 0, 1, "en")
		// Should be "Task 1/1: " (11 chars) + truncated content (83 chars max with "...")
		assert.Contains(t, name, "...")
		assert.LessOrEqual(t, len(name), 100)
	})

	t.Run("truncates_long_message_content", func(t *testing.T) {
		longContent := "This is a very long message content that should be truncated because it exceeds 80 characters which is the maximum length allowed"
		task := &robottypes.Task{
			ID:           "task-001",
			Description:  "", // No description, will use message
			ExecutorType: robottypes.ExecutorAssistant,
			ExecutorID:   "analyst",
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: longContent},
			},
		}

		name := formatTaskProgressName(task, 0, 1, "en")
		// Should be "Task 1/1: " (11 chars) + truncated content (83 chars max with "...")
		assert.Contains(t, name, "...")
		assert.LessOrEqual(t, len(name), 100)
	})

	t.Run("fallback_to_executor_info_when_no_messages", func(t *testing.T) {
		task := &robottypes.Task{
			ID:           "task-001",
			ExecutorType: robottypes.ExecutorMCP,
			ExecutorID:   "calculator",
			Messages:     []agentcontext.Message{},
		}

		name := formatTaskProgressName(task, 2, 4, "en")
		assert.Equal(t, "Task 3/4: mcp:calculator", name)
	})
}
