package standard

import (
	"encoding/json"
	"fmt"
	"strings"

	kunlog "github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"

	"github.com/yaoapp/yao/agent/robot/logger"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
)

var log = logger.New("exec")

type execLogger struct {
	robot  *robottypes.Robot
	execID string
}

func newExecLogger(robot *robottypes.Robot, execID string) *execLogger {
	return &execLogger{robot: robot, execID: execID}
}

func (l *execLogger) robotID() string {
	if l.robot != nil {
		return l.robot.MemberID
	}
	return ""
}

func (l *execLogger) connector() string {
	if l.robot != nil {
		return l.robot.LanguageModel
	}
	return ""
}

func (l *execLogger) workspace() string {
	if l.robot != nil {
		return l.robot.Workspace
	}
	return ""
}

// ---------------------------------------------------------------------------
// P2: Task Overview
// ---------------------------------------------------------------------------

func (l *execLogger) logTaskOverview(tasks []robottypes.Task) {
	if config.IsDevelopment() {
		l.devTaskOverview(tasks)
	}
	kunlog.With(kunlog.F{
		"robot_id":       l.robotID(),
		"execution_id":   l.execID,
		"phase":          "tasks",
		"task_count":     len(tasks),
		"language_model": l.connector(),
		"workspace":      l.workspace(),
	}).Info("P2 task overview: %d tasks generated", len(tasks))
}

func (l *execLogger) devTaskOverview(tasks []robottypes.Task) {
	w := logger.Gray
	h := logger.BoldCyan
	v := logger.White
	r := logger.Reset

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n%s%s%s\n", h, strings.Repeat("═", 60), r))
	sb.WriteString(fmt.Sprintf("%s  TASK OVERVIEW%s\n", h, r))
	sb.WriteString(fmt.Sprintf("%s%s%s\n", h, strings.Repeat("─", 60), r))
	sb.WriteString(fmt.Sprintf("%s  Robot:     %s%s%s\n", w, v, l.robotID(), r))
	sb.WriteString(fmt.Sprintf("%s  Exec:      %s%s%s\n", w, v, l.execID, r))
	if l.connector() != "" {
		sb.WriteString(fmt.Sprintf("%s  Model:     %s%s%s\n", w, v, l.connector(), r))
	}
	if l.workspace() != "" {
		sb.WriteString(fmt.Sprintf("%s  Workspace: %s%s%s\n", w, v, l.workspace(), r))
	}
	sb.WriteString(fmt.Sprintf("%s%s%s\n", w, strings.Repeat("─", 60), r))
	for i, t := range tasks {
		desc := t.Description
		if desc == "" && len(t.Messages) > 0 {
			if s, ok := t.Messages[0].GetContentAsString(); ok {
				desc = s
			}
		}
		desc = truncate(desc, 72)
		sb.WriteString(fmt.Sprintf("%s  #%d %s%s%s [%s:%s]\n", w, i+1, v, t.ID, r, t.ExecutorType, t.ExecutorID))
		sb.WriteString(fmt.Sprintf("%s     %s%s\n", w, desc, r))
	}
	sb.WriteString(fmt.Sprintf("%s%s%s\n", w, strings.Repeat("─", 60), r))
	sb.WriteString(fmt.Sprintf("%s  Total: %s%d tasks%s\n", w, v, len(tasks), r))
	sb.WriteString(fmt.Sprintf("%s%s%s\n", h, strings.Repeat("═", 60), r))

	logger.Raw(sb.String())
}

// ---------------------------------------------------------------------------
// P3: Task Input
// ---------------------------------------------------------------------------

func (l *execLogger) logTaskInput(task *robottypes.Task, prompt string) {
	if config.IsDevelopment() {
		l.devTaskInput(task, prompt)
	}
	kunlog.With(kunlog.F{
		"robot_id":       l.robotID(),
		"execution_id":   l.execID,
		"task_id":        task.ID,
		"executor_type":  string(task.ExecutorType),
		"executor_id":    task.ExecutorID,
		"prompt_len":     len(prompt),
		"language_model": l.connector(),
	}).Info("Task input: %s [%s]", task.ID, task.ExecutorID)
}

func (l *execLogger) devTaskInput(task *robottypes.Task, prompt string) {
	w := logger.Gray
	v := logger.White
	r := logger.Reset

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s  ▶ Task %s%s%s [%s:%s]  Prompt: %d chars%s\n",
		w, v, task.ID, w, task.ExecutorType, task.ExecutorID, len(prompt), r))

	logger.Raw(sb.String())
}

// ---------------------------------------------------------------------------
// P3: Task Output
// ---------------------------------------------------------------------------

func (l *execLogger) logTaskOutput(task *robottypes.Task, result *robottypes.TaskResult) {
	if config.IsDevelopment() {
		l.devTaskOutput(task, result)
	}

	fields := kunlog.F{
		"robot_id":       l.robotID(),
		"execution_id":   l.execID,
		"task_id":        result.TaskID,
		"success":        result.Success,
		"duration_ms":    result.Duration,
		"language_model": l.connector(),
	}
	if result.Output != nil {
		fields["output_type"] = fmt.Sprintf("%T", result.Output)
		fields["output_len"] = outputLen(result.Output)
	}
	if result.Error != "" {
		fields["error"] = result.Error
	}
	if result.Success {
		kunlog.With(fields).Info("Task completed: %s (%dms)", result.TaskID, result.Duration)
	} else {
		kunlog.With(fields).Warn("Task failed: %s (%dms) %s", result.TaskID, result.Duration, result.Error)
	}
}

func (l *execLogger) devTaskOutput(task *robottypes.Task, result *robottypes.TaskResult) {
	w := logger.Gray
	v := logger.White
	g := logger.BoldGreen
	rd := logger.BoldRed
	r := logger.Reset

	var sb strings.Builder
	if result.Success {
		sb.WriteString(fmt.Sprintf("%s  ✓ %s%s%s completed %s(%dms)%s\n",
			g, v, result.TaskID, g, w, result.Duration, r))
		out := outputSummary(result.Output)
		if len(out) > 120 {
			out = out[:120] + "..."
		}
		sb.WriteString(fmt.Sprintf("%s    Output: %s%s%s\n", w, v, out, r))
	} else {
		sb.WriteString(fmt.Sprintf("%s  ✗ %s%s%s failed %s(%dms)%s\n",
			rd, v, result.TaskID, rd, w, result.Duration, r))
		sb.WriteString(fmt.Sprintf("%s    Error:  %s%s%s\n", w, logger.Red, result.Error, r))
	}

	logger.Raw(sb.String())
}

// ---------------------------------------------------------------------------
// Agent Call
// ---------------------------------------------------------------------------

func (l *execLogger) logAgentCall(agentID string, result *CallResult) {
	if result == nil {
		return
	}
	if config.IsDevelopment() {
		l.devAgentCall(agentID, result)
	}

	fields := kunlog.F{
		"robot_id":       l.robotID(),
		"execution_id":   l.execID,
		"agent_id":       agentID,
		"content_len":    len(result.Content),
		"language_model": l.connector(),
	}
	if result.Next != nil {
		fields["next_type"] = fmt.Sprintf("%T", result.Next)
		fields["next_len"] = outputLen(result.Next)
	}
	kunlog.With(fields).Info("Agent call: %s (content=%d, next=%T)", agentID, len(result.Content), result.Next)
}

func (l *execLogger) devAgentCall(agentID string, result *CallResult) {
	w := logger.Gray
	v := logger.White
	c := logger.Cyan
	r := logger.Reset

	nextInfo := "—"
	if result.Next != nil {
		nextInfo = fmt.Sprintf("%T (len=%d)", result.Next, outputLen(result.Next))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s  → Agent(%s%s%s) Content: %s%d%s chars  Next: %s%s%s\n",
		c, v, agentID, c, v, len(result.Content), w, v, nextInfo, r))

	logger.Raw(sb.String())
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (l *execLogger) prefix() string {
	if l.connector() != "" {
		return fmt.Sprintf("[robot:%s|exec:%s|model:%s]", l.robotID(), l.execID, l.connector())
	}
	return fmt.Sprintf("[robot:%s|exec:%s]", l.robotID(), l.execID)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func indentText(s string, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

func outputSummary(v interface{}) string {
	if v == nil {
		return "<nil>"
	}
	switch val := v.(type) {
	case string:
		if len(val) > 500 {
			return fmt.Sprintf("string(len=%d) %s...", len(val), val[:500])
		}
		return fmt.Sprintf("string(len=%d) %s", len(val), val)
	case map[string]interface{}:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		return fmt.Sprintf("map{%s}", strings.Join(keys, ", "))
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%T(marshal-error)", v)
		}
		s := string(raw)
		if len(s) > 500 {
			return fmt.Sprintf("%T(len=%d) %s...", v, len(s), s[:500])
		}
		return fmt.Sprintf("%T %s", v, s)
	}
}

func outputLen(v interface{}) int {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case string:
		return len(val)
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return 0
		}
		return len(raw)
	}
}
