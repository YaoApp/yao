package computer

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/xun/dbal"
)

// Cloud task lifecycle.
const (
	TaskStatusQueued    = "queued"
	TaskStatusRunning   = "running"
	TaskStatusSucceeded = "succeeded"
	TaskStatusFailed    = "failed"

	TaskPhasePurchase  = "purchase"
	TaskPhaseProvision = "provision"
	TaskPhaseTai       = "tai"
	TaskPhaseReady     = "ready"

	CloudTaskModel = "__yao.cloud_tasks"

	// CloudTaskTimeout is the maximum time a worker may spend before failing a task.
	CloudTaskTimeout = 15 * time.Minute
)

var (
	// ErrTaskNotFound is returned when a task does not exist or is not owned by the caller.
	ErrTaskNotFound = errors.New("cloud task not found")
	// ErrTaskInstanceConflict means the dedupe key already has an instance; the
	// caller must not create a second purchase.
	ErrTaskInstanceConflict = errors.New("cloud task already has an instance")
	// ErrInvalidPurchaseRequest marks a caller validation failure.
	ErrInvalidPurchaseRequest = errors.New("invalid purchase request")
	// ErrCloudServerURL means Yao has no remotely reachable server URL.
	ErrCloudServerURL = errors.New("cloud-reachable Yao server URL is not configured")
)

// CloudTask is one durable purchase/provisioning task.
type CloudTask struct {
	ID            string
	Status        string
	Phase         string
	DedupeKey     string
	UserID        string
	TeamID        string
	BillingUserID string
	TaoName       string
	Resource      string
	Region        string
	Image         string
	BillingUnit   string
	PeriodMonths  int
	ComputerName  string
	Visibility    string
	Members       []string
	RequestParams map[string]any
	InstanceID    string
	ComputerID    string
	Error         string
	ErrorCode     string
	Attempts      int
	StartedAt     *time.Time
	FinishedAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// CloudTaskView is the public task response body.
type CloudTaskView struct {
	TaskID     string  `json:"task_id"`
	Status     string  `json:"status"`
	Phase      string  `json:"phase"`
	InstanceID *string `json:"instance_id"`
	ComputerID *string `json:"computer_id"`
	Error      *string `json:"error"`
}

func (t *CloudTask) View() CloudTaskView {
	out := CloudTaskView{TaskID: t.ID, Status: t.Status, Phase: t.Phase}
	if t.InstanceID != "" {
		out.InstanceID = &t.InstanceID
	}
	if t.ComputerID != "" {
		out.ComputerID = &t.ComputerID
	}
	if t.Error != "" {
		out.Error = &t.Error
	}
	return out
}

// CloudTaskStore persists cloud tasks.
type CloudTaskStore interface {
	Create(context.Context, *CloudTask) error
	Get(context.Context, string) (*CloudTask, error)
	GetByDedupe(context.Context, string) (*CloudTask, error)
	Update(context.Context, *CloudTask) error
	ListRunnable(context.Context, int) ([]*CloudTask, error)
	ResetRunning(context.Context) error
	Delete(context.Context, string) error
}

// ModelCloudTaskStore stores tasks in __yao.cloud_tasks.
type ModelCloudTaskStore struct{}

func CloudTaskModelStore() CloudTaskStore { return ModelCloudTaskStore{} }

func (ModelCloudTaskStore) model() (*model.Model, error) {
	if !model.Exists(CloudTaskModel) {
		return nil, fmt.Errorf("cloud task model is not loaded")
	}
	m := model.Select(CloudTaskModel)
	if m == nil {
		return nil, fmt.Errorf("cloud task model is not loaded")
	}
	return m, nil
}

func (ModelCloudTaskStore) Create(ctx context.Context, task *CloudTask) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m, err := (ModelCloudTaskStore{}).model()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
	}
	task.UpdatedAt = now
	return m.Insert(
		[]string{"id", "status", "phase", "dedupe_key", "user_id", "team_id", "billing_user_id",
			"tao_name", "resource", "region", "image", "billing_unit", "period_months", "computer_name", "visibility",
			"members", "request_params", "instance_id", "computer_id", "error", "error_code", "attempts", "created_at", "updated_at"},
		[][]interface{}{{
			task.ID, task.Status, task.Phase, task.DedupeKey, task.UserID, task.TeamID, task.BillingUserID,
			task.TaoName, task.Resource, task.Region, task.Image, task.BillingUnit, task.PeriodMonths, task.ComputerName, task.Visibility,
			task.Members, task.RequestParams, task.InstanceID, task.ComputerID, task.Error, task.ErrorCode, task.Attempts, task.CreatedAt, task.UpdatedAt,
		}},
	)
}

func (ModelCloudTaskStore) Get(ctx context.Context, id string) (*CloudTask, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m, err := (ModelCloudTaskStore{}).model()
	if err != nil {
		return nil, err
	}
	row, err := m.Find(id, model.QueryParam{})
	if err != nil {
		return nil, ErrTaskNotFound
	}
	return cloudTaskFromRow(row), nil
}

func (ModelCloudTaskStore) GetByDedupe(ctx context.Context, dedupeKey string) (*CloudTask, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m, err := (ModelCloudTaskStore{}).model()
	if err != nil {
		return nil, err
	}
	rows, err := m.Get(model.QueryParam{Wheres: []model.QueryWhere{{Column: "dedupe_key", Value: dedupeKey}}})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrTaskNotFound
	}
	return cloudTaskFromRow(rows[0]), nil
}

func (ModelCloudTaskStore) Update(ctx context.Context, task *CloudTask) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m, err := (ModelCloudTaskStore{}).model()
	if err != nil {
		return err
	}
	task.UpdatedAt = time.Now().UTC()
	var startedAt, finishedAt interface{} = dbal.Raw("NULL"), dbal.Raw("NULL")
	if task.StartedAt != nil {
		startedAt = *task.StartedAt
	}
	if task.FinishedAt != nil {
		finishedAt = *task.FinishedAt
	}
	return m.Update(task.ID, map[string]interface{}{
		"status": task.Status, "phase": task.Phase, "instance_id": task.InstanceID, "computer_id": task.ComputerID,
		"error": task.Error, "error_code": task.ErrorCode, "attempts": task.Attempts,
		"started_at": startedAt, "finished_at": finishedAt, "updated_at": task.UpdatedAt,
	})
}

func (ModelCloudTaskStore) ListRunnable(ctx context.Context, limit int) ([]*CloudTask, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m, err := (ModelCloudTaskStore{}).model()
	if err != nil {
		return nil, err
	}
	rows, err := m.Get(model.QueryParam{
		Wheres: []model.QueryWhere{{Column: "status", OP: "in", Value: []string{TaskStatusQueued, TaskStatusRunning}}},
		Orders: []model.QueryOrder{{Column: "created_at", Option: "asc"}},
	})
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	out := make([]*CloudTask, 0, len(rows))
	for _, row := range rows {
		out = append(out, cloudTaskFromRow(row))
	}
	return out, nil
}

func (ModelCloudTaskStore) ResetRunning(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m, err := (ModelCloudTaskStore{}).model()
	if err != nil {
		return err
	}
	rows, err := m.Get(model.QueryParam{Wheres: []model.QueryWhere{{Column: "status", Value: TaskStatusRunning}}})
	if err != nil {
		return err
	}
	for _, row := range rows {
		task := cloudTaskFromRow(row)
		task.Status = TaskStatusQueued
		task.Error = ""
		task.ErrorCode = ""
		task.StartedAt = nil
		task.FinishedAt = nil
		task.Attempts = 0
		if err := (ModelCloudTaskStore{}).Update(ctx, task); err != nil {
			return err
		}
	}
	return nil
}

func (ModelCloudTaskStore) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m, err := (ModelCloudTaskStore{}).model()
	if err != nil {
		return err
	}
	return m.Delete(id)
}

func cloudTaskFromRow(row map[string]interface{}) *CloudTask {
	task := &CloudTask{
		ID: stringValue(row, "id"), Status: stringValue(row, "status"), Phase: stringValue(row, "phase"),
		DedupeKey: stringValue(row, "dedupe_key"), UserID: stringValue(row, "user_id"), TeamID: stringValue(row, "team_id"),
		BillingUserID: stringValue(row, "billing_user_id"), TaoName: stringValue(row, "tao_name"), Resource: stringValue(row, "resource"),
		Region: stringValue(row, "region"), Image: stringValue(row, "image"), BillingUnit: stringValue(row, "billing_unit"),
		ComputerName: stringValue(row, "computer_name"), Visibility: stringValue(row, "visibility"),
		InstanceID: stringValue(row, "instance_id"), ComputerID: stringValue(row, "computer_id"),
		Error: stringValue(row, "error"), ErrorCode: stringValue(row, "error_code"),
	}
	task.PeriodMonths, _ = intValue(row["period_months"])
	task.Attempts, _ = intValue(row["attempts"])
	task.StartedAt = timeValue(row["started_at"])
	task.FinishedAt = timeValue(row["finished_at"])
	task.CreatedAt = timeValueOrZero(row["created_at"])
	task.UpdatedAt = timeValueOrZero(row["updated_at"])
	_ = decodeJSONValue(row["members"], &task.Members)
	_ = decodeJSONValue(row["request_params"], &task.RequestParams)
	return task
}

func timeValueOrZero(value interface{}) time.Time {
	if t := timeValue(value); t != nil {
		return *t
	}
	return time.Time{}
}

func intValue(value interface{}) (int, bool) {
	if value == nil {
		return 0, false
	}
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case uint64:
		return int(v), true
	case string:
		var n int
		_, err := fmt.Sscan(v, &n)
		return n, err == nil
	default:
		return 0, false
	}
}

func decodeJSONValue(value interface{}, target interface{}) bool {
	if value == nil {
		return false
	}
	var data []byte
	switch v := value.(type) {
	case string:
		if v == "" {
			return false
		}
		data = []byte(v)
	case []byte:
		data = v
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return false
		}
		data = raw
	}
	return json.Unmarshal(data, target) == nil
}

func newTaskID() string {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return fmt.Sprintf("task_%d", time.Now().UnixNano())
	}
	return "task_" + strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw))
}

// cloudDedupeKey normalizes the purchase intent. Disk size and team context are
// part of the key so distinct disks/teams cannot collide.
func cloudDedupeKey(userID, teamID string, req PurchaseRequest) string {
	members := append([]string(nil), req.Members...)
	sort.Strings(members)
	members = uniqueStrings(members)
	disk := ""
	if req.RequestParams != nil {
		if value, ok := req.RequestParams["disk_gb"]; ok && value != nil {
			disk = normalizeNumber(value)
		}
	}
	image, _ := req.RequestParams["image"].(string)
	computerName := req.Computer.Name
	if computerName == "" {
		computerName = req.Name
	}
	visibility := req.Computer.Visibility
	if visibility == "" {
		visibility = "private"
	}
	parts := []string{
		"v1", userID, teamID, req.Resource, req.Region, image,
		disk, req.BillingUnit, fmt.Sprint(req.PeriodMonths), computerName, visibility,
		strings.Join(members, ","),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return fmt.Sprintf("%x", sum[:])
}

func stringValue(row map[string]interface{}, key string) string {
	if row[key] == nil {
		return ""
	}
	if value, ok := row[key].(string); ok {
		return value
	}
	return fmt.Sprint(row[key])
}

func timeValue(value interface{}) *time.Time {
	switch v := value.(type) {
	case time.Time:
		return &v
	case *time.Time:
		return v
	case string:
		if v == "" {
			return nil
		}
		for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05"} {
			if t, err := time.Parse(layout, v); err == nil {
				return &t
			}
		}
	}
	return nil
}

func uniqueStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	out := values[:1]
	for _, value := range values[1:] {
		if value != out[len(out)-1] {
			out = append(out, value)
		}
	}
	return out
}

func normalizeNumber(value interface{}) string {
	switch v := value.(type) {
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%.0f", v)
	case float32:
		return fmt.Sprintf("%.0f", v)
	case uint64:
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprint(value)
	}
}
