package standard

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	kunlog "github.com/yaoapp/kun/log"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/llm"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	taiworkspace "github.com/yaoapp/yao/tai/workspace"
	"github.com/yaoapp/yao/workspace"
	"gopkg.in/yaml.v3"
)

//go:embed prompts/workspace.yml
var workspacePromptYAML []byte

type workspacePrompts struct {
	Workspace    string `yaml:"workspace"`
	Context      string `yaml:"context"`
	Instructions string `yaml:"instructions"`
}

var wsPromptTpls struct {
	Workspace    *template.Template
	Context      *template.Template
	Instructions *template.Template
}

func init() {
	var p workspacePrompts
	if err := yaml.Unmarshal(workspacePromptYAML, &p); err != nil {
		return
	}
	wsPromptTpls.Workspace, _ = template.New("ws").Parse(p.Workspace)
	wsPromptTpls.Context, _ = template.New("ctx").Parse(p.Context)
	wsPromptTpls.Instructions, _ = template.New("inst").Parse(p.Instructions)
}

// Manifest is the shared context hub for an execution, written as manifest.json.
type Manifest struct {
	ExecID  string         `json:"exec_id"`
	RobotID string         `json:"robot_id"`
	Goals   string         `json:"goals"`
	Tasks   []ManifestTask `json:"tasks"`
}

// ManifestTask represents a single task entry in the manifest.
type ManifestTask struct {
	ID           string         `json:"id"`
	Order        int            `json:"order"`
	Description  string         `json:"description"`
	Executor     string         `json:"executor"`
	ExecutorType string         `json:"executor_type"`
	Status       string         `json:"status"`
	Summary      string         `json:"summary,omitempty"`
	KeyOutputs   []string       `json:"key_outputs,omitempty"`
	Files        []ManifestFile `json:"files,omitempty"`
	Error        string         `json:"error,omitempty"`
}

// ManifestFile represents a produced artifact.
type ManifestFile struct {
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
	Desc string `json:"desc,omitempty"`
	URI  string `json:"uri,omitempty"`
}

// ensureRobotWorkspace guarantees a workspace FS exists for the robot.
// If robot.Workspace is empty, it derives a deterministic ID and auto-creates.
// It also writes back robot.Workspace so subsequent callers get the correct ID.
func ensureRobotWorkspace(ctx *robottypes.Context, robot *robottypes.Robot) (taiworkspace.FS, error) {
	wsm := workspace.M()
	if wsm == nil {
		return nil, fmt.Errorf("workspace manager not available")
	}

	wsID := robot.Workspace
	if wsID == "" {
		nodes := wsm.Nodes()
		nodeID := ""
		for _, n := range nodes {
			if n.Online {
				nodeID = n.Name
				break
			}
		}
		if nodeID == "" {
			return nil, fmt.Errorf("no available node for workspace")
		}
		wsID = workspace.DefaultWorkspaceID(robot.TeamID, nodeID)

		if _, err := wsm.Get(ctx, wsID); err != nil {
			if _, err := wsm.Create(ctx, workspace.CreateOptions{
				ID:    wsID,
				Name:  "Robot Workspace",
				Owner: robot.TeamID,
				Node:  nodeID,
			}); err != nil {
				return nil, fmt.Errorf("workspace create failed: %w", err)
			}
		}
		robot.Workspace = wsID
	}

	return wsm.FS(ctx, wsID)
}

// initManifest creates the initial manifest.json with goals and pending task list.
// Uses slice index (not P2's order field) so manifests always have sequential numbering.
func (r *Runner) initManifest(exec *robottypes.Execution) {
	if r.wsFS == nil {
		return
	}

	goalsContent := ""
	if exec.Goals != nil {
		goalsContent = exec.Goals.Content
	}

	m := &Manifest{
		ExecID:  exec.ID,
		RobotID: r.robot.MemberID,
		Goals:   goalsContent,
		Tasks:   make([]ManifestTask, 0, len(exec.Tasks)),
	}

	for i, t := range exec.Tasks {
		m.Tasks = append(m.Tasks, ManifestTask{
			ID:           t.ID,
			Order:        i,
			Description:  t.Description,
			Executor:     t.ExecutorID,
			ExecutorType: string(t.ExecutorType),
			Status:       string(t.Status),
		})
	}

	r.writeManifest(m)
}

// writeManifest serializes and writes manifest.json.
func (r *Runner) writeManifest(m *Manifest) {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		kunlog.Warn("[robot-workspace] marshal manifest: %v", err)
		return
	}
	p := path.Join(r.execDir, "manifest.json")
	if err := r.wsFS.WriteFile(p, data, 0644); err != nil {
		kunlog.Warn("[robot-workspace] write manifest: %v", err)
	}
}

// readManifest reads and parses manifest.json from workspace.
func (r *Runner) readManifest() (*Manifest, error) {
	if r.wsFS == nil {
		return nil, fmt.Errorf("wsFS not available")
	}
	data, err := r.wsFS.ReadFile(path.Join(r.execDir, "manifest.json"))
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// writeTaskOutput writes the three task files and updates manifest after task completion.
func (r *Runner) writeTaskOutput(task *robottypes.Task, result *robottypes.TaskResult, promptSnapshot string) {
	if r.wsFS == nil {
		return
	}

	taskID := task.ID

	// Write task-NNN.input.md (prompt snapshot for debug)
	if promptSnapshot != "" {
		inputPath := path.Join(r.execDir, taskID+".input.md")
		if err := r.wsFS.WriteFile(inputPath, []byte(promptSnapshot), 0644); err != nil {
			kunlog.Warn("[robot-workspace] write %s.input.md: %v", taskID, err)
		}
	}

	// Write task-NNN.output.md (full output)
	outputText := formatOutputAsText(result.Output)
	outputPath := path.Join(r.execDir, taskID+".output.md")
	if err := r.wsFS.WriteFile(outputPath, []byte(outputText), 0644); err != nil {
		kunlog.Warn("[robot-workspace] write %s.output.md: %v", taskID, err)
	}

	// Write task-NNN.json (metadata)
	meta := map[string]interface{}{
		"id":            taskID,
		"executor":      task.ExecutorID,
		"executor_type": string(task.ExecutorType),
		"status":        string(task.Status),
		"duration_ms":   result.Duration,
		"success":       result.Success,
	}
	if result.Error != "" {
		meta["error"] = result.Error
	}
	metaJSON, _ := json.MarshalIndent(meta, "", "  ")
	metaPath := path.Join(r.execDir, taskID+".json")
	if err := r.wsFS.WriteFile(metaPath, metaJSON, 0644); err != nil {
		kunlog.Warn("[robot-workspace] write %s.json: %v", taskID, err)
	}

	r.updateManifestForTask(task, result)
}

// updateManifestForTask reads manifest, updates the matching task entry, and writes back.
func (r *Runner) updateManifestForTask(task *robottypes.Task, result *robottypes.TaskResult) {
	m, err := r.readManifest()
	if err != nil {
		kunlog.Warn("[robot-workspace] read manifest for update: %v", err)
		return
	}

	// Scan files first so LLM summary can reference them
	files := r.scanTaskArtifacts(task.ID)
	outputURIs := r.extractAndVerifyFiles(result.Output)
	files = mergeManifestFiles(files, outputURIs)

	for i := range m.Tasks {
		if m.Tasks[i].ID == task.ID {
			if result.Success {
				m.Tasks[i].Status = "completed"
				summary := r.llmSummarize(task, result.Output, files)
				if summary == "" {
					summary = generateSummary(result.Output)
				}
				m.Tasks[i].Summary = summary
				m.Tasks[i].KeyOutputs = extractKeyOutputs(result.Output)
			} else {
				m.Tasks[i].Status = "failed"
				m.Tasks[i].Error = result.Error
				if result.Error != "" {
					m.Tasks[i].Summary = "Failed: " + result.Error
				}
			}
			m.Tasks[i].Files = files
			break
		}
	}

	r.writeManifest(m)
}

// scanTaskArtifacts scans the task-id/ directory for produced files.
func (r *Runner) scanTaskArtifacts(taskID string) []ManifestFile {
	if r.wsFS == nil {
		return nil
	}

	dirPath := path.Join(r.execDir, taskID)
	entries, err := r.wsFS.ReadDir(dirPath)
	if err != nil {
		return nil
	}

	var files []ManifestFile
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		files = append(files, ManifestFile{
			Name: e.Name(),
			Type: mimeFromExt(filepath.Ext(e.Name())),
		})
	}
	return files
}

// llmSummarize uses a lightweight LLM to generate a concise task summary
// from the full task context (description, output, produced files).
// Returns empty string on any failure, allowing caller to fall back to static extraction.
func (r *Runner) llmSummarize(task *robottypes.Task, output interface{}, files []ManifestFile) string {
	text := flattenOutput(output)
	if text == "" {
		return ""
	}

	conn, _, err := llm.ResolveConnector("use::light", r.ctx.Auth)
	if err != nil {
		kunlog.Warn("[robot-workspace] llmSummarize resolve connector: %v", err)
		return ""
	}

	opts := llm.BuildCompletionOptions(conn, nil)
	instance, err := llm.New(conn, opts)
	if err != nil {
		kunlog.Warn("[robot-workspace] llmSummarize create LLM: %v", err)
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Task: " + task.Description + "\n\n")
	if len(files) > 0 {
		sb.WriteString("Produced files:\n")
		for _, f := range files {
			sb.WriteString("- " + f.Name + " (" + f.Type + ")\n")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("Output:\n")
	outputText := text
	if len(outputText) > 4000 {
		outputText = outputText[:4000]
	}
	sb.WriteString(outputText)

	messages := []agentcontext.Message{
		{Role: agentcontext.RoleSystem, Content: "Summarize the task execution result in 1-2 concise sentences. " +
			"Focus on what was actually produced or accomplished, not the process. " +
			"If files were produced, mention them. Reply in the same language as the task description."},
		{Role: agentcontext.RoleUser, Content: sb.String()},
	}

	agentCtx := agentcontext.New(r.ctx.Context, r.ctx.Auth, "")
	defer agentCtx.Release()

	resp, err := instance.Post(agentCtx, messages, opts)
	if err != nil {
		kunlog.Warn("[robot-workspace] llmSummarize Post: %v", err)
		return ""
	}

	return extractLLMContent(resp)
}

// extractLLMContent extracts the text content from a CompletionResponse.
func extractLLMContent(resp *agentcontext.CompletionResponse) string {
	if resp == nil {
		return ""
	}
	if s, ok := resp.Content.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

// workspaceURIRegex matches workspace://wsID/path patterns in markdown links and plain text.
// Excludes trailing backticks, quotes, and brackets that are markdown formatting artifacts.
var workspaceURIRegex = regexp.MustCompile("workspace://([^/\\s)]+)/([^\\s)\\]`\"']+)")

// extractAndVerifyFiles extracts workspace:// URIs from output and verifies each
// file exists via wsFS.Stat, eliminating false positives from regex artifacts.
func (r *Runner) extractAndVerifyFiles(output interface{}) []ManifestFile {
	text := flattenOutput(output)
	if text == "" || r.wsFS == nil {
		return nil
	}

	wsID, err := r.wsFS.GetID()
	if err != nil {
		return nil
	}
	prefix := "workspace://" + wsID + "/"

	matches := workspaceURIRegex.FindAllStringSubmatch(text, -1)
	seen := make(map[string]bool)
	var files []ManifestFile
	for _, m := range matches {
		uri := "workspace://" + m[1] + "/" + m[2]
		if seen[uri] {
			continue
		}
		seen[uri] = true

		if !strings.HasPrefix(uri, prefix) {
			continue
		}
		relPath := strings.TrimPrefix(uri, prefix)

		info, err := r.wsFS.Stat(relPath)
		if err != nil || info.IsDir() {
			continue
		}

		name := filepath.Base(relPath)
		files = append(files, ManifestFile{
			Name: name,
			Type: mimeFromExt(filepath.Ext(name)),
			URI:  uri,
		})
	}
	return files
}

// mergeManifestFiles deduplicates files from scanTaskArtifacts and extractWorkspaceURIs.
// If a URI-bearing entry has the same Name as a scan entry, the URI is merged onto the
// existing entry instead of creating a duplicate.
func mergeManifestFiles(scanned []ManifestFile, fromURIs []ManifestFile) []ManifestFile {
	if len(fromURIs) == 0 {
		return scanned
	}

	nameIndex := make(map[string]int, len(scanned))
	for i, f := range scanned {
		nameIndex[f.Name] = i
	}

	for _, uf := range fromURIs {
		if idx, exists := nameIndex[uf.Name]; exists {
			if scanned[idx].URI == "" {
				scanned[idx].URI = uf.URI
			}
		} else {
			nameIndex[uf.Name] = len(scanned)
			scanned = append(scanned, uf)
		}
	}
	return scanned
}

// --- Summary & key_outputs extraction ---

// llmPrefixPatterns are common LLM filler prefixes that carry no information.
var llmPrefixPatterns = []string{
	"It seems ", "It appears ", "Here is ", "Here's ",
	"Based on ", "I encountered ", "I wasn't able ",
	"I'm currently unable ", "Let me ", "I recommend ",
}

// generateSummary produces a concise summary of the task's actual output/result.
// It prioritizes conclusion/summary sections over the beginning of the output,
// because agent responses typically start with planning/thinking text.
func generateSummary(output interface{}) string {
	text := flattenOutput(output)
	if text == "" {
		return ""
	}

	const maxLen = 200

	// Try to find an explicit summary/conclusion section
	for _, heading := range []string{"## Summary", "## Conclusion", "## Result", "## 总结", "## 结论", "## 结果"} {
		if idx := strings.Index(text, heading); idx >= 0 {
			section := strings.TrimSpace(text[idx+len(heading):])
			section = strings.TrimPrefix(section, "\n")
			if nextH := strings.Index(section, "\n## "); nextH > 0 {
				section = section[:nextH]
			}
			section = strings.TrimSpace(section)
			if section != "" {
				if len(section) > maxLen {
					return section[:maxLen] + "..."
				}
				return section
			}
		}
	}

	// No explicit section — use last substantive paragraph as it's
	// more likely to contain the actual result than the beginning
	paragraphs := strings.Split(text, "\n\n")
	for i := len(paragraphs) - 1; i >= 0; i-- {
		p := strings.TrimSpace(paragraphs[i])
		if p == "" || len(p) < 10 {
			continue
		}
		isFiller := false
		for _, prefix := range llmPrefixPatterns {
			if strings.HasPrefix(p, prefix) {
				isFiller = true
				break
			}
		}
		if isFiller {
			continue
		}
		if len(p) > maxLen {
			return p[:maxLen] + "..."
		}
		return p
	}

	// Fallback: skip filler and take from the beginning
	text = skipFillerPrefixes(text)
	if len(text) > maxLen {
		return text[:maxLen] + "..."
	}
	return text
}

// skipFillerPrefixes skips past common LLM opening phrases to find substantive content.
func skipFillerPrefixes(text string) string {
	lines := strings.SplitN(text, "\n", 20)
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		isFiller := false
		for _, prefix := range llmPrefixPatterns {
			if strings.HasPrefix(trimmed, prefix) {
				isFiller = true
				break
			}
		}
		if !isFiller {
			return strings.Join(lines[i:], "\n")
		}
	}
	return text
}

// boldPatternRegex matches **bold text** in markdown
var boldPatternRegex = regexp.MustCompile(`\*\*([^*]+)\*\*`)

// extractKeyOutputs tries to extract structured key_outputs from the output.
func extractKeyOutputs(output interface{}) []string {
	if output == nil {
		return nil
	}

	// If output is a map with key_outputs/outputs/results, extract directly
	if m, ok := output.(map[string]interface{}); ok {
		for _, key := range []string{"key_outputs", "outputs", "results"} {
			if arr, ok := m[key].([]interface{}); ok {
				result := make([]string, 0, len(arr))
				for _, v := range arr {
					if s, ok := v.(string); ok {
						result = append(result, s)
					}
				}
				if len(result) > 0 {
					return capSlice(result, 5)
				}
			}
		}
	}

	text := flattenOutput(output)

	// Try ## headings
	if strings.Contains(text, "\n## ") {
		var headings []string
		for _, line := range strings.Split(text, "\n") {
			if strings.HasPrefix(line, "## ") {
				headings = append(headings, strings.TrimPrefix(line, "## "))
			}
		}
		if len(headings) > 0 {
			return capSlice(headings, 5)
		}
	}

	// Try **bold** items from numbered lists or bullet lists
	// e.g. "1. **Model-Driven Architecture:**" or "- **Low-Code Engine**"
	var boldItems []string
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.Contains(trimmed, "**") {
			continue
		}
		// Only extract from list-like lines
		if !(strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") ||
			(len(trimmed) > 2 && trimmed[0] >= '0' && trimmed[0] <= '9' && trimmed[1] == '.')) {
			continue
		}
		matches := boldPatternRegex.FindStringSubmatch(trimmed)
		if len(matches) >= 2 {
			item := strings.TrimRight(matches[1], ":")
			if len(item) > 0 && len(item) < 80 {
				boldItems = append(boldItems, item)
			}
		}
	}
	if len(boldItems) > 0 {
		return capSlice(boldItems, 5)
	}

	return nil
}

func capSlice(s []string, max int) []string {
	if len(s) > max {
		return s[:max]
	}
	return s
}

// flattenOutput converts any output value to a plain text string.
func flattenOutput(output interface{}) string {
	if output == nil {
		return ""
	}
	switch v := output.(type) {
	case string:
		return v
	case map[string]interface{}:
		if text, ok := v["text"].(string); ok {
			return text
		}
		if content, ok := v["content"].(string); ok {
			return content
		}
		b, _ := json.Marshal(v)
		return string(b)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	}
}

// formatOutputAsText converts task output to markdown text for .output.md files.
func formatOutputAsText(output interface{}) string {
	if output == nil {
		return ""
	}
	switch v := output.(type) {
	case string:
		return v
	default:
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	}
}

// --- Prompt template rendering ---

// wsTemplateData holds template variables for the workspace section.
type wsTemplateData struct {
	ExecDir string
	Files   []wsFileEntry
	TaskDir string
}

type wsFileEntry struct {
	Path string
	Desc string
}

// ctxTemplateData holds template variables for the execution context section.
type ctxTemplateData struct {
	Goals          string
	Locale         string
	CompletedTasks []ctxTaskEntry
	FailedTasks    []ctxFailedEntry
	CurrentOrder   int
	CurrentID      string
	CurrentDesc    string
	CurrentType    string
	CurrentExec    string
	FailureWarning string
}

type ctxTaskEntry struct {
	Seq          int
	ID           string
	Description  string
	ExecutorType string
	Executor     string
	Summary      string
	KeyOutputs   string
	Files        string
	HasFiles     bool
}

type ctxFailedEntry struct {
	Seq         int
	ID          string
	Description string
	Error       string
}

// instTemplateData holds template variables for the instructions section.
type instTemplateData struct {
	TaskInstructions string
	ExpectedOutput   string
}

// buildWorkspacePrompt renders the full prompt for a task from manifest + template.
func (r *Runner) buildWorkspacePrompt(manifest *Manifest, taskIndex int, task *robottypes.Task, taskInstructions string) string {
	if manifest == nil || taskIndex >= len(manifest.Tasks) {
		return taskInstructions
	}

	var sb strings.Builder

	// Section 1: Workspace
	if wsPromptTpls.Workspace != nil {
		wd := wsTemplateData{
			ExecDir: r.execDir + "/",
			TaskDir: r.execDir + "/" + manifest.Tasks[taskIndex].ID + "/",
		}
		wd.Files = append(wd.Files, wsFileEntry{
			Path: "manifest.json",
			Desc: "Execution context: goals, completed task summaries, progress",
		})
		for i := 0; i < taskIndex; i++ {
			t := manifest.Tasks[i]
			if t.Status == "completed" {
				wd.Files = append(wd.Files, wsFileEntry{
					Path: t.ID + ".output.md",
					Desc: "Full output: " + t.Description,
				})
				for _, f := range t.Files {
					desc := "Artifact: " + f.Name
					if f.URI != "" {
						desc = "Artifact: " + f.Name + " (" + f.URI + ")"
					}
					filePath := t.ID + "/" + f.Name
					if f.URI != "" {
						filePath = f.URI
					}
					wd.Files = append(wd.Files, wsFileEntry{
						Path: filePath,
						Desc: desc,
					})
				}
			}
		}
		var buf bytes.Buffer
		if err := wsPromptTpls.Workspace.Execute(&buf, wd); err == nil {
			sb.WriteString(buf.String())
			sb.WriteString("\n\n")
		}
	}

	// Section 2: Execution Context
	if wsPromptTpls.Context != nil {
		ct := manifest.Tasks[taskIndex]
		cd := ctxTemplateData{
			Goals:        manifest.Goals,
			Locale:       r.locale,
			CurrentID:    ct.ID,
			CurrentOrder: taskIndex + 1,
			CurrentDesc:  ct.Description,
			CurrentType:  ct.ExecutorType,
			CurrentExec:  ct.Executor,
		}

		seq := 0
		failedCount := 0
		for i := 0; i < taskIndex; i++ {
			t := manifest.Tasks[i]
			seq++
			if t.Status == "completed" {
				entry := ctxTaskEntry{
					Seq:          seq,
					ID:           t.ID,
					Description:  t.Description,
					ExecutorType: t.ExecutorType,
					Executor:     t.Executor,
					Summary:      t.Summary,
					KeyOutputs:   strings.Join(t.KeyOutputs, ", "),
				}
				if len(t.Files) > 0 {
					entry.HasFiles = true
					names := make([]string, 0, len(t.Files))
					for _, f := range t.Files {
						names = append(names, f.Name)
					}
					entry.Files = strings.Join(names, ", ")
				}
				cd.CompletedTasks = append(cd.CompletedTasks, entry)
			} else if t.Status == "failed" {
				failedCount++
				errMsg := t.Error
				if errMsg == "" {
					errMsg = t.Summary
				}
				cd.FailedTasks = append(cd.FailedTasks, ctxFailedEntry{
					Seq:         seq,
					ID:          t.ID,
					Description: t.Description,
					Error:       errMsg,
				})
			}
		}

		// P8: failure cascade warning
		if failedCount > 0 && len(cd.CompletedTasks) == 0 {
			cd.FailureWarning = "WARNING: All previous tasks failed. You may lack necessary input data. Do your best with available information or report the limitation."
		} else if failedCount > 0 {
			cd.FailureWarning = fmt.Sprintf("Note: %d previous task(s) failed. Some expected input may be missing.", failedCount)
		}

		var buf bytes.Buffer
		if err := wsPromptTpls.Context.Execute(&buf, cd); err == nil {
			sb.WriteString(buf.String())
			sb.WriteString("\n\n")
		}
	}

	// Section 3: Task Instructions (enriched with expected_output from P2)
	if wsPromptTpls.Instructions != nil {
		instData := instTemplateData{TaskInstructions: taskInstructions}
		if task != nil && task.ExpectedOutput != "" {
			instData.ExpectedOutput = task.ExpectedOutput
		}
		var buf bytes.Buffer
		if err := wsPromptTpls.Instructions.Execute(&buf, instData); err == nil {
			sb.WriteString(buf.String())
		}
	} else {
		sb.WriteString("## Task Instructions\n\n")
		sb.WriteString(taskInstructions)
	}

	return sb.String()
}

func mimeFromExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".md":
		return "text/markdown"
	case ".html", ".htm":
		return "text/html"
	case ".json":
		return "application/json"
	case ".pdf":
		return "application/pdf"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".csv":
		return "text/csv"
	case ".txt":
		return "text/plain"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	default:
		return "application/octet-stream"
	}
}
