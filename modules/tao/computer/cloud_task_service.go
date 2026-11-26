package computer

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/yaoapp/kun/log"
	modulecomputer "github.com/yaoapp/yao/modules/computer"
	computertypes "github.com/yaoapp/yao/modules/computer/types"
)

// WithCloudTasks installs the durable task store.
func (s *Service) WithCloudTasks(store CloudTaskStore) *Service {
	s.cloudTasks = store
	return s
}

// WithTaskKeyResolver installs the resolver used by the worker to obtain the
// billing user's Tao key without persisting the secret in a task.
func (s *Service) WithTaskKeyResolver(resolver KeyResolver) *Service {
	s.taskKeys = resolver
	return s
}

// EnqueuePurchase creates or reuses the durable task for this purchase intent.
func (s *Service) EnqueuePurchase(ctx context.Context, userID, teamID, billingUserID string, req PurchaseRequest) (*CloudTask, error) {
	if s.cloudTasks == nil {
		return nil, fmt.Errorf("cloud task store is not configured")
	}
	if s.computers == nil {
		return nil, fmt.Errorf("computer service is not configured")
	}
	if err := validateCloudServerURL(s.serverURL); err != nil {
		return nil, err
	}
	if err := validatePurchaseIntent(req); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPurchaseRequest, err)
	}
	dedupe := cloudDedupeKey(userID, teamID, req)
	task := &CloudTask{
		ID: newTaskID(), Status: TaskStatusQueued, Phase: TaskPhasePurchase,
		DedupeKey: dedupe, UserID: userID, TeamID: teamID, BillingUserID: billingUserID,
		TaoName: req.Name, Resource: req.Resource, Region: req.Region, Image: stringParam(req.RequestParams, "image"),
		BillingUnit: req.BillingUnit, PeriodMonths: req.PeriodMonths, ComputerName: computerName(req),
		Visibility: req.Computer.Visibility, Members: append([]string(nil), req.Members...),
		RequestParams: cloneParams(req.RequestParams), CreatedAt: time.Now().UTC(),
	}
	if task.Visibility == "" {
		task.Visibility = computertypes.VisibilityPrivate
	}
	createErr := s.cloudTasks.Create(ctx, task)
	if createErr == nil {
		log.Info("[cloud-task] enqueue task_id=%s dedupe_key=%s user_id=%s", task.ID, task.DedupeKey, task.UserID)
		return task, nil
	}
	existing, getErr := s.cloudTasks.GetByDedupe(ctx, dedupe)
	if getErr != nil {
		return nil, fmt.Errorf("create cloud task: %w", createErr)
	}
	log.Info("[cloud-task] reuse task_id=%s dedupe_key=%s user_id=%s", existing.ID, dedupe, userID)
	return s.reuseTask(ctx, existing)
}

func (s *Service) reuseTask(ctx context.Context, task *CloudTask) (*CloudTask, error) {
	if task.Status == TaskStatusSucceeded {
		if task.ComputerID != "" && s.computers != nil {
			if _, err := s.computers.Get(ctx, task.ComputerID); err != nil {
				// Only a real "not found" releases the key. A transient DB error
				// must not create a second Tao purchase.
				if !errors.Is(err, modulecomputer.ErrNotFound) {
					return nil, err
				}
				if delErr := s.cloudTasks.Delete(ctx, task.ID); delErr != nil {
					return nil, delErr
				}
				return s.createFreshReusedTask(ctx, task)
			}
		}
		return task, nil
	}
	if task.Status == TaskStatusFailed {
		if task.InstanceID != "" {
			return task, ErrTaskInstanceConflict
		}
		task.Status = TaskStatusQueued
		task.Phase = TaskPhasePurchase
		task.Error = ""
		task.ErrorCode = ""
		task.FinishedAt = nil
		task.StartedAt = nil
		task.Attempts = 0
		if err := s.cloudTasks.Update(ctx, task); err != nil {
			return nil, err
		}
		return task, nil
	}
	return task, nil
}

func (s *Service) createFreshReusedTask(ctx context.Context, old *CloudTask) (*CloudTask, error) {
	task := *old
	task.ID = newTaskID()
	task.Status = TaskStatusQueued
	task.Phase = TaskPhasePurchase
	task.Error = ""
	task.ErrorCode = ""
	task.InstanceID = ""
	task.ComputerID = ""
	task.Attempts = 0
	task.CreatedAt = time.Now().UTC()
	task.FinishedAt = nil
	task.StartedAt = nil
	if err := s.cloudTasks.Create(ctx, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// GetTask returns a task only when it belongs to the current user.
func (s *Service) GetTask(ctx context.Context, userID, taskID string) (*CloudTask, error) {
	if s.cloudTasks == nil {
		return nil, fmt.Errorf("cloud task store is not configured")
	}
	task, err := s.cloudTasks.Get(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task.UserID != userID {
		return nil, ErrTaskNotFound
	}
	return task, nil
}

// StartCloudWorker recovers interrupted tasks and starts the processing loop.
func (s *Service) StartCloudWorker(ctx context.Context) {
	if s.cloudTasks == nil {
		return
	}
	s.workerOnce.Do(func() {
		if err := s.cloudTasks.ResetRunning(ctx); err != nil {
			log.Error("[cloud-task] reset running tasks failed: %v", err)
		}
		workerCtx, cancel := context.WithCancel(ctx)
		s.workerStop = cancel
		go s.cloudWorkerLoop(workerCtx)
	})
}

// StopCloudWorker stops the processing loop.
func (s *Service) StopCloudWorker() {
	s.workerMu.Lock()
	defer s.workerMu.Unlock()
	if s.workerStop != nil {
		s.workerStop()
		s.workerStop = nil
	}
}

func (s *Service) cloudWorkerLoop(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.processRunnableTasks(ctx)
		}
	}
}

func (s *Service) processRunnableTasks(ctx context.Context) {
	tasks, err := s.cloudTasks.ListRunnable(ctx, 10)
	if err != nil {
		log.Error("[cloud-task] list runnable failed: %v", err)
		return
	}
	for _, task := range tasks {
		if _, loaded := s.processing.LoadOrStore(task.ID, struct{}{}); loaded {
			continue
		}
		go func(task *CloudTask) {
			defer s.processing.Delete(task.ID)
			s.processCloudTask(ctx, task)
		}(task)
	}
}

func (s *Service) processCloudTask(ctx context.Context, task *CloudTask) {
	if s.cloudTasks == nil || s.computers == nil {
		log.Error("[cloud-task] worker dependencies are not configured task_id=%s", task.ID)
		return
	}
	if task.Status == TaskStatusQueued {
		task.Status = TaskStatusRunning
		task.Attempts++
		now := time.Now().UTC()
		if task.StartedAt == nil {
			task.StartedAt = &now
		}
		if err := s.cloudTasks.Update(ctx, task); err != nil {
			log.Error("[cloud-task] claim/update %s failed: %v", task.ID, err)
			return
		}
	}
	// Computer online is the authoritative completion signal. Tao may still
	// report pending/running or a previous phase may not have been persisted;
	// do not leave the task stuck in provision in that case.
	if task.ComputerID != "" {
		if computer, getErr := s.computers.Get(ctx, task.ComputerID); getErr == nil && computer != nil && computer.Status == computertypes.StatusOnline {
			s.succeedTask(ctx, task)
			return
		}
	}
	if task.StartedAt != nil && time.Since(*task.StartedAt) > CloudTaskTimeout {
		s.failTask(ctx, task, task.Phase, fmt.Errorf("cloud task timed out after %s (phase=%s)", CloudTaskTimeout, task.Phase))
		return
	}
	if s.taskKeys == nil {
		s.failTask(ctx, task, task.Phase, errors.New("tao key resolver is not configured"))
		return
	}
	taoKey, err := s.taskKeys.ResolveTaoKey(ctx, task.BillingUserID)
	if err != nil {
		s.failTask(ctx, task, task.Phase, err)
		return
	}
	if err := validateCloudServerURL(s.serverURL); err != nil {
		s.failTask(ctx, task, task.Phase, err)
		return
	}
	req := taskPurchaseRequest(task)
	req.UserID, req.TeamID, req.IdempotencyKey = task.UserID, task.TeamID, task.DedupeKey

	if task.InstanceID == "" {
		if s.client == nil {
			s.failTask(ctx, task, TaskPhasePurchase, errors.New("tao computer client is not configured"))
			return
		}
		instance, err := s.createTaoInstance(ctx, taoKey, req)
		if err != nil {
			s.failTask(ctx, task, TaskPhasePurchase, err)
			return
		}
		task.InstanceID = instance.InstanceID
		task.Phase = TaskPhaseProvision
		if err := s.cloudTasks.Update(ctx, task); err != nil {
			log.Error("[cloud-task] claim/update %s failed: %v", task.ID, err)
			return
		}
		computer, err := s.attachComputer(ctx, instance, req)
		if err != nil {
			s.failTask(ctx, task, TaskPhaseProvision, err)
			return
		}
		task.ComputerID = computer.ID
		if err := s.cloudTasks.Update(ctx, task); err != nil {
			log.Error("[cloud-task] claim/update %s failed: %v", task.ID, err)
			return
		}
	}

	if task.InstanceID != "" && task.ComputerID == "" {
		if s.client == nil {
			s.failTask(ctx, task, TaskPhaseProvision, errors.New("tao computer client is not configured"))
			return
		}
		instance, err := s.client.Get(ctx, taoKey, task.InstanceID)
		if err != nil {
			s.failTask(ctx, task, TaskPhasePurchase, err)
			return
		}
		computer, err := s.attachComputer(ctx, instance, req)
		if err != nil {
			s.failTask(ctx, task, TaskPhaseProvision, err)
			return
		}
		task.ComputerID = computer.ID
		task.Phase = TaskPhaseProvision
		if err := s.cloudTasks.Update(ctx, task); err != nil {
			log.Error("[cloud-task] claim/update %s failed: %v", task.ID, err)
			return
		}
	}

	if task.Phase == TaskPhaseProvision {
		result, err := s.Get(ctx, taoKey, task.InstanceID)
		if err != nil {
			s.failTask(ctx, task, TaskPhaseProvision, err)
			return
		}
		if result.Instance != nil {
			switch result.Instance.Status {
			case "running":
				task.Phase = TaskPhaseTai
				if err := s.cloudTasks.Update(ctx, task); err != nil {
					log.Error("[cloud-task] persist phase=tai %s failed: %v", task.ID, err)
					return
				}
			case "error", "deleted":
				s.failTask(ctx, task, TaskPhaseProvision, fmt.Errorf("tao instance %s reached %s", task.InstanceID, result.Instance.Status))
				return
			default:
				return
			}
		}
	}

	if task.Phase == TaskPhaseTai {
		if task.ComputerID == "" {
			s.failTask(ctx, task, TaskPhaseTai, errors.New("computer id is empty"))
			return
		}
		computer, err := s.computers.Get(ctx, task.ComputerID)
		if err != nil {
			s.failTask(ctx, task, TaskPhaseTai, err)
			return
		}
		switch computer.Status {
		case computertypes.StatusOnline:
			s.succeedTask(ctx, task)
		case computertypes.StatusError, computertypes.StatusReleased:
			s.failTask(ctx, task, TaskPhaseTai, fmt.Errorf("computer %s reached %s", computer.ID, computer.Status))
		default:
			// The global cloud task timeout above covers the waiting phase.
		}
	}
}

func (s *Service) succeedTask(ctx context.Context, task *CloudTask) {
	task.Status = TaskStatusSucceeded
	task.Phase = TaskPhaseReady
	now := time.Now().UTC()
	task.FinishedAt = &now
	task.Error = ""
	task.ErrorCode = ""
	if err := s.cloudTasks.Update(ctx, task); err != nil {
		log.Error("[cloud-task] persist success %s failed: %v", task.ID, err)
		return
	}
	log.Info("[cloud-task] succeeded task_id=%s computer_id=%s", task.ID, task.ComputerID)
}

func (s *Service) failTask(ctx context.Context, task *CloudTask, phase string, err error) {
	task.Status = TaskStatusFailed
	task.Phase = phase
	task.Error = errorText(err)
	task.ErrorCode = taskErrorCode(err)
	now := time.Now().UTC()
	task.FinishedAt = &now
	if err := s.cloudTasks.Update(ctx, task); err != nil {
		log.Error("[cloud-task] persist failure %s failed: %v", task.ID, err)
	}
}

func validateCloudServerURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ErrCloudServerURL
	}
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" {
		return fmt.Errorf("%w: invalid %q", ErrCloudServerURL, raw)
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" || host == "0.0.0.0" || host == "::1" {
		return fmt.Errorf("%w: %s", ErrCloudServerURL, raw)
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return fmt.Errorf("%w: %s", ErrCloudServerURL, raw)
	}
	return nil
}

func taskPurchaseRequest(task *CloudTask) PurchaseRequest {
	return PurchaseRequest{
		Resource: task.Resource, Region: task.Region, Name: task.TaoName,
		BillingUnit: task.BillingUnit, PeriodMonths: task.PeriodMonths,
		RequestParams: cloneParams(task.RequestParams), Members: append([]string(nil), task.Members...),
		Computer: ComputerSpec{
			Name: task.ComputerName, Visibility: task.Visibility, Members: append([]string(nil), task.Members...),
		},
	}
}

func validatePurchaseIntent(req PurchaseRequest) error {
	if strings.TrimSpace(req.Resource) == "" || strings.TrimSpace(req.Region) == "" || strings.TrimSpace(req.Name) == "" {
		return errors.New("resource, region and name are required")
	}
	if len([]rune(req.Name)) > 50 {
		return errors.New("name must be 1-50 characters")
	}
	if req.BillingUnit != BillingHour && req.BillingUnit != BillingMonth {
		return errors.New("billing_unit must be HOUR or MONTH")
	}
	if req.BillingUnit == BillingMonth && req.PeriodMonths != 1 && req.PeriodMonths != 3 && req.PeriodMonths != 6 && req.PeriodMonths != 12 {
		return errors.New("period_months must be 1/3/6/12 for MONTH")
	}
	if req.BillingUnit == BillingHour && req.PeriodMonths != 0 {
		return errors.New("period_months is only for MONTH")
	}
	if stringParam(req.RequestParams, "image") == "" {
		return errors.New("request_params.image is required")
	}
	if req.Computer.Visibility == "" {
		req.Computer.Visibility = computertypes.VisibilityPrivate
	}
	if req.Computer.Visibility != computertypes.VisibilityPrivate && req.Computer.Visibility != computertypes.VisibilityMembers && req.Computer.Visibility != computertypes.VisibilityTeam {
		return errors.New("computer.visibility must be private, team or members")
	}
	if len(req.Members) > 0 && req.Computer.Visibility != computertypes.VisibilityMembers {
		return errors.New("members is only allowed when visibility is members")
	}
	return nil
}

func stringParam(params map[string]any, key string) string {
	if params == nil {
		return ""
	}
	value, _ := params[key].(string)
	return value
}

func cloneParams(params map[string]any) map[string]any {
	if params == nil {
		return nil
	}
	out := make(map[string]any, len(params))
	for key, value := range params {
		out[key] = value
	}
	return out
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func taskErrorCode(err error) string {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return fmt.Sprintf("tao_%d", apiErr.Status)
	}
	if errors.Is(err, ErrTaiStart) {
		return "tai_start_error"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	return "task_failed"
}
