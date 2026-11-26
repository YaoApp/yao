package computer_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yaoapp/yao/e2e/taocloud"
	modulecomputer "github.com/yaoapp/yao/modules/computer"
	computertypes "github.com/yaoapp/yao/modules/computer/types"
	taocomputer "github.com/yaoapp/yao/modules/tao/computer"
)

type fakeTaskStore struct {
	mu             sync.Mutex
	tasks          map[string]*taocomputer.CloudTask
	updateFailures int
	createErr      error
	deleteErr      error
	listErr        error
	resetErr       error
	getErr         error
}

func newFakeTaskStore() *fakeTaskStore {
	return &fakeTaskStore{tasks: map[string]*taocomputer.CloudTask{}}
}

func (s *fakeTaskStore) Create(_ context.Context, task *taocomputer.CloudTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.createErr != nil {
		return s.createErr
	}
	for _, old := range s.tasks {
		if old.DedupeKey == task.DedupeKey {
			return errors.New("duplicate dedupe key")
		}
	}
	copy := *task
	s.tasks[task.ID] = &copy
	return nil
}
func (s *fakeTaskStore) Get(_ context.Context, id string) (*taocomputer.CloudTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.getErr != nil {
		return nil, s.getErr
	}
	task, ok := s.tasks[id]
	if !ok {
		return nil, taocomputer.ErrTaskNotFound
	}
	copy := *task
	return &copy, nil
}
func (s *fakeTaskStore) GetByDedupe(_ context.Context, key string) (*taocomputer.CloudTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, task := range s.tasks {
		if task.DedupeKey == key {
			copy := *task
			return &copy, nil
		}
	}
	return nil, taocomputer.ErrTaskNotFound
}
func (s *fakeTaskStore) Update(_ context.Context, task *taocomputer.CloudTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.updateFailures > 0 {
		s.updateFailures--
		return errors.New("injected task update failure")
	}
	copy := *task
	s.tasks[task.ID] = &copy
	return nil
}
func (s *fakeTaskStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.deleteErr != nil {
		return s.deleteErr
	}
	delete(s.tasks, id)
	return nil
}
func (s *fakeTaskStore) ListRunnable(context.Context, int) ([]*taocomputer.CloudTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listErr != nil {
		return nil, s.listErr
	}
	out := make([]*taocomputer.CloudTask, 0, len(s.tasks))
	for _, task := range s.tasks {
		if task.Status == taocomputer.TaskStatusQueued || task.Status == taocomputer.TaskStatusRunning {
			copy := *task
			out = append(out, &copy)
		}
	}
	return out, nil
}
func (s *fakeTaskStore) ResetRunning(context.Context) error { return nil }

type fakeTaskKeys struct{ key string }

func (f fakeTaskKeys) ResolveTaoKey(context.Context, string) (string, error) {
	if f.key == "" {
		return "sk-tao-test", nil
	}
	return f.key, nil
}

func taskPurchaseRequestForTest(disk int, visibility string) taocomputer.PurchaseRequest {
	return taocomputer.PurchaseRequest{
		Resource: "flex-2c2g", Region: "bj-cn", Name: "host", BillingUnit: "HOUR",
		RequestParams: map[string]any{"image": "tao-base-1.0", "disk_gb": disk},
		Computer:      taocomputer.ComputerSpec{Name: "cloud", Visibility: visibility},
	}
}

func TestEnqueueTaskDedupesDiskAndTeam(t *testing.T) {
	computers := &fakeComputerService{}
	svc := newTestService(nil, computers).WithCloudTasks(newFakeTaskStore()).WithTaskKeyResolver(fakeTaskKeys{})
	ctx := context.Background()

	first, err := svc.EnqueuePurchase(ctx, "u1", "t1", "u1", taskPurchaseRequestForTest(50, "private"))
	if err != nil {
		t.Fatal(err)
	}
	same, err := svc.EnqueuePurchase(ctx, "u1", "t1", "u1", taskPurchaseRequestForTest(50, "private"))
	if err != nil || same.ID != first.ID {
		t.Fatalf("same intent task=%v err=%v want %s", same, err, first.ID)
	}
	otherDisk, err := svc.EnqueuePurchase(ctx, "u1", "t1", "u1", taskPurchaseRequestForTest(100, "private"))
	if err != nil || otherDisk.ID == first.ID {
		t.Fatalf("disk must participate in dedupe: first=%s other=%s", first.ID, otherDisk.ID)
	}
	otherTeam, err := svc.EnqueuePurchase(ctx, "u1", "t2", "u1", taskPurchaseRequestForTest(50, "private"))
	if err != nil || otherTeam.ID == first.ID {
		t.Fatalf("team must participate in dedupe: first=%s other=%s", first.ID, otherTeam.ID)
	}
}

func TestEnqueueTaskFailedRetryRules(t *testing.T) {
	computers := &fakeComputerService{}
	store := newFakeTaskStore()
	svc := newTestService(nil, computers).WithCloudTasks(store).WithTaskKeyResolver(fakeTaskKeys{})
	ctx := context.Background()
	task, err := svc.EnqueuePurchase(ctx, "u1", "", "", taskPurchaseRequestForTest(50, "private"))
	if err != nil {
		t.Fatal(err)
	}
	task.Status = taocomputer.TaskStatusFailed
	task.Error = "tao rejected purchase"
	_ = store.Update(ctx, task)

	retried, err := svc.EnqueuePurchase(ctx, "u1", "", "", taskPurchaseRequestForTest(50, "private"))
	if err != nil || retried.ID != task.ID || retried.Status != taocomputer.TaskStatusQueued || retried.Error != "" {
		t.Fatalf("failed no-instance retry=%+v err=%v", retried, err)
	}

	if _, err := store.Get(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	task, _ = store.Get(ctx, task.ID)
	task.Status = taocomputer.TaskStatusFailed
	task.InstanceID = "inst-existing"
	_ = store.Update(ctx, task)
	if _, err := svc.EnqueuePurchase(ctx, "u1", "", "", taskPurchaseRequestForTest(50, "private")); !errors.Is(err, taocomputer.ErrTaskInstanceConflict) {
		t.Fatalf("failed with instance err=%v", err)
	}
}

func TestGetTaskOwnership(t *testing.T) {
	store := newFakeTaskStore()
	svc := newTestService(nil, &fakeComputerService{}).WithCloudTasks(store).WithTaskKeyResolver(fakeTaskKeys{})
	task, err := svc.EnqueuePurchase(context.Background(), "owner", "", "", taskPurchaseRequestForTest(50, "private"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetTask(context.Background(), "other", task.ID); !errors.Is(err, taocomputer.ErrTaskNotFound) {
		t.Fatalf("ownership err=%v", err)
	}
	got, err := svc.GetTask(context.Background(), "owner", task.ID)
	if err != nil || got.ID != task.ID {
		t.Fatalf("owner task=%+v err=%v", got, err)
	}
}

type deletingComputers struct{ fakeComputerService }

func (deletingComputers) Get(context.Context, string) (*computertypes.Computer, error) {
	return nil, modulecomputer.ErrNotFound
}

type workerTai struct{ computers *workerComputers }

func (t *workerTai) Bootstrap(context.Context, computertypes.SSHTarget, computertypes.BootstrapRequest) error {
	return nil
}
func (t *workerTai) Start(context.Context, computertypes.SSHTarget, computertypes.StartTaiRequest) error {
	t.computers.mu.Lock()
	defer t.computers.mu.Unlock()
	t.computers.status = computertypes.StatusOnline
	return nil
}
func (t *workerTai) Stop(context.Context, computertypes.SSHTarget) error { return nil }
func (t *workerTai) Status(context.Context, computertypes.SSHTarget) (*computertypes.TaiStatus, error) {
	return &computertypes.TaiStatus{Running: true}, nil
}
func (t *workerTai) Logs(context.Context, computertypes.SSHTarget, int) (string, error) {
	return "", nil
}

type workerComputers struct {
	mu       sync.Mutex
	computer *computertypes.Computer
	status   string
	purchase *computertypes.PurchaseSnapshot
}

func (c *workerComputers) CreateOrUpdate(_ context.Context, req computertypes.CreateOrUpdateComputerRequest) (*computertypes.UpsertResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.computer == nil {
		c.computer = &computertypes.Computer{ID: "cmp-worker", NodeID: "node-worker", InstanceID: req.InstanceID}
	}
	if req.Status != "" {
		c.computer.Status = req.Status
		c.status = req.Status
	}
	if req.Purchase != nil {
		c.purchase = req.Purchase
		c.computer.Purchase = req.Purchase
	}
	copy := *c.computer
	return &computertypes.UpsertResult{Computer: &copy}, nil
}
func (c *workerComputers) Get(context.Context, string) (*computertypes.Computer, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.computer == nil {
		return nil, errors.New("computer not found")
	}
	copied := *c.computer
	if c.status != "" {
		copied.Status = c.status
	}
	return &copied, nil
}

func TestCloudWorkerCompletesTask(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	computers := &workerComputers{status: computertypes.StatusProvisioning}
	store := newFakeTaskStore()
	svc := newTestService(taocomputer.NewClient(srv.URL()), computers).
		WithTai(&workerTai{computers: computers}).
		WithCloudTasks(store).WithTaskKeyResolver(fakeTaskKeys{key: srv.Token})
	task, err := svc.EnqueuePurchase(context.Background(), "u1", "", "u1", taskPurchaseRequestForTest(50, "private"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartCloudWorker(ctx)
	defer svc.StopCloudWorker()

	deadline := time.Now().Add(10 * time.Second)
	var got *taocomputer.CloudTask
	for time.Now().Before(deadline) {
		got, _ = store.Get(context.Background(), task.ID)
		if got != nil && (got.Status == taocomputer.TaskStatusSucceeded || got.Status == taocomputer.TaskStatusFailed) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if got == nil || got.Status != taocomputer.TaskStatusSucceeded || got.Phase != taocomputer.TaskPhaseReady {
		t.Fatalf("task=%+v", got)
	}
	if got.Error != "" || got.InstanceID == "" || got.ComputerID != "cmp-worker" {
		t.Fatalf("terminal task=%+v", got)
	}
	computers.mu.Lock()
	defer computers.mu.Unlock()
	if computers.purchase == nil || computers.purchase.Resource != "flex-2c2g" || computers.purchase.Image != "tao-base-1.0" {
		t.Fatalf("purchase snapshot=%+v", computers.purchase)
	}
}

func TestReuseTaskDeletedComputerStartsFresh(t *testing.T) {
	store := newFakeTaskStore()
	svc := newTestService(nil, &deletingComputers{}).WithCloudTasks(store).WithTaskKeyResolver(fakeTaskKeys{})
	ctx := context.Background()
	old, err := svc.EnqueuePurchase(ctx, "u1", "", "u1", taskPurchaseRequestForTest(50, "private"))
	if err != nil {
		t.Fatal(err)
	}
	old.Status = taocomputer.TaskStatusSucceeded
	old.ComputerID = "cmp-deleted"
	_ = store.Update(ctx, old)
	fresh, err := svc.EnqueuePurchase(ctx, "u1", "", "u1", taskPurchaseRequestForTest(50, "private"))
	if err != nil || fresh.ID == old.ID || fresh.Status != taocomputer.TaskStatusQueued {
		t.Fatalf("fresh=%+v old=%s err=%v", fresh, old.ID, err)
	}
}

type transientComputerError struct{ fakeComputerService }

func (transientComputerError) Get(context.Context, string) (*computertypes.Computer, error) {
	return nil, errors.New("database temporarily unavailable")
}

func TestReuseTaskTransientComputerErrorDoesNotCreate(t *testing.T) {
	store := newFakeTaskStore()
	svc := newTestService(nil, &transientComputerError{}).WithCloudTasks(store).WithTaskKeyResolver(fakeTaskKeys{})
	ctx := context.Background()
	old, err := svc.EnqueuePurchase(ctx, "u1", "", "u1", taskPurchaseRequestForTest(50, "private"))
	if err != nil {
		t.Fatal(err)
	}
	old.Status = taocomputer.TaskStatusSucceeded
	old.ComputerID = "cmp-1"
	_ = store.Update(ctx, old)
	if _, err := svc.EnqueuePurchase(ctx, "u1", "", "u1", taskPurchaseRequestForTest(50, "private")); err == nil {
		t.Fatal("transient error must not release the dedupe key")
	}
}

type failingTaskKeys struct{}

func (failingTaskKeys) ResolveTaoKey(context.Context, string) (string, error) {
	return "", errors.New("tao key was revoked")
}

func TestCloudWorkerPersistsRealFailure(t *testing.T) {
	store := newFakeTaskStore()
	svc := newTestService(nil, &fakeComputerService{}).WithCloudTasks(store).WithTaskKeyResolver(failingTaskKeys{})
	task, err := svc.EnqueuePurchase(context.Background(), "u1", "", "u1", taskPurchaseRequestForTest(50, "private"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartCloudWorker(ctx)
	defer svc.StopCloudWorker()
	deadline := time.Now().Add(5 * time.Second)
	var got *taocomputer.CloudTask
	for time.Now().Before(deadline) {
		got, _ = store.Get(context.Background(), task.ID)
		if got != nil && got.Status == taocomputer.TaskStatusFailed {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if got == nil || got.Error != "tao key was revoked" || got.ErrorCode != "task_failed" {
		t.Fatalf("failed task=%+v", got)
	}
}

func TestCloudWorkerFailsStaleTask(t *testing.T) {
	store := newFakeTaskStore()
	svc := newTestService(nil, &fakeComputerService{}).WithCloudTasks(store).WithTaskKeyResolver(fakeTaskKeys{})
	task, err := svc.EnqueuePurchase(context.Background(), "u1", "", "u1", taskPurchaseRequestForTest(50, "private"))
	if err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-taocomputer.CloudTaskTimeout - time.Minute)
	task.StartedAt = &old
	_ = store.Update(context.Background(), task)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartCloudWorker(ctx)
	defer svc.StopCloudWorker()
	deadline := time.Now().Add(5 * time.Second)
	var got *taocomputer.CloudTask
	for time.Now().Before(deadline) {
		got, _ = store.Get(context.Background(), task.ID)
		if got != nil && got.Status == taocomputer.TaskStatusFailed {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if got == nil || got.Status != taocomputer.TaskStatusFailed || !strings.Contains(got.Error, "timed out") {
		t.Fatalf("stale task=%+v", got)
	}
}

func TestEnqueuePurchaseRejectsLoopbackServerURL(t *testing.T) {
	svc := newTestService(nil, &fakeComputerService{}).
		WithServerURL("http://127.0.0.1:5099").
		WithCloudTasks(newFakeTaskStore()).
		WithTaskKeyResolver(fakeTaskKeys{})
	if _, err := svc.EnqueuePurchase(context.Background(), "u1", "", "u1", taskPurchaseRequestForTest(50, "private")); !errors.Is(err, taocomputer.ErrCloudServerURL) {
		t.Fatalf("loopback server URL err=%v", err)
	}
}

func TestCloudWorkerCompletesProvisionWhenComputerOnline(t *testing.T) {
	store := newFakeTaskStore()
	computers := &workerComputers{
		status:   computertypes.StatusOnline,
		computer: &computertypes.Computer{ID: "cmp-online", NodeID: "node-online", Status: computertypes.StatusOnline},
	}
	svc := newTestService(nil, computers).WithCloudTasks(store)
	task, err := svc.EnqueuePurchase(context.Background(), "u1", "", "u1", taskPurchaseRequestForTest(50, "private"))
	if err != nil {
		t.Fatal(err)
	}
	task.Status = taocomputer.TaskStatusRunning
	task.Phase = taocomputer.TaskPhaseProvision
	task.ComputerID = "cmp-online"
	if err := store.Update(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	store.updateFailures = 1
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartCloudWorker(ctx)
	defer svc.StopCloudWorker()
	deadline := time.Now().Add(5 * time.Second)
	var got *taocomputer.CloudTask
	for time.Now().Before(deadline) {
		got, _ = store.Get(context.Background(), task.ID)
		if got != nil && got.Status == taocomputer.TaskStatusSucceeded {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got == nil || got.Status != taocomputer.TaskStatusSucceeded || got.Phase != taocomputer.TaskPhaseReady {
		t.Fatalf("task=%+v", got)
	}
}

func TestCloudWorkerCompletesTaiPhaseWhenComputerOnline(t *testing.T) {
	store := newFakeTaskStore()
	computers := &workerComputers{
		status:   computertypes.StatusOnline,
		computer: &computertypes.Computer{ID: "cmp-online-tai", NodeID: "node-online-tai", Status: computertypes.StatusOnline},
	}
	svc := newTestService(nil, computers).WithCloudTasks(store)
	task, err := svc.EnqueuePurchase(context.Background(), "u1", "", "u1", taskPurchaseRequestForTest(50, "private"))
	if err != nil {
		t.Fatal(err)
	}
	task.Status = taocomputer.TaskStatusRunning
	task.Phase = taocomputer.TaskPhaseTai
	task.ComputerID = "cmp-online-tai"
	if err := store.Update(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartCloudWorker(ctx)
	defer svc.StopCloudWorker()
	deadline := time.Now().Add(5 * time.Second)
	var got *taocomputer.CloudTask
	for time.Now().Before(deadline) {
		got, _ = store.Get(context.Background(), task.ID)
		if got != nil && got.Status == taocomputer.TaskStatusSucceeded {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got == nil || got.Status != taocomputer.TaskStatusSucceeded || got.Phase != taocomputer.TaskPhaseReady {
		t.Fatalf("task=%+v", got)
	}
}

func TestCloudWorkerWaitsForComputerOnline(t *testing.T) {
	store := newFakeTaskStore()
	computers := &workerComputers{
		status:   computertypes.StatusPending,
		computer: &computertypes.Computer{ID: "cmp-wait", NodeID: "node-wait", Status: computertypes.StatusPending},
	}
	svc := newTestService(nil, computers).WithCloudTasks(store)
	task, err := svc.EnqueuePurchase(context.Background(), "u1", "", "u1", taskPurchaseRequestForTest(50, "private"))
	if err != nil {
		t.Fatal(err)
	}
	task.Status = taocomputer.TaskStatusRunning
	task.Phase = taocomputer.TaskPhaseTai
	task.ComputerID = "cmp-wait"
	if err := store.Update(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartCloudWorker(ctx)
	defer svc.StopCloudWorker()
	time.Sleep(250 * time.Millisecond)
	if got, _ := store.Get(context.Background(), task.ID); got == nil || got.Status != taocomputer.TaskStatusRunning {
		t.Fatalf("task should still wait: %+v", got)
	}
	computers.mu.Lock()
	computers.status = computertypes.StatusOnline
	computers.computer.Status = computertypes.StatusOnline
	computers.mu.Unlock()
	deadline := time.Now().Add(5 * time.Second)
	var got *taocomputer.CloudTask
	for time.Now().Before(deadline) {
		got, _ = store.Get(context.Background(), task.ID)
		if got != nil && got.Status == taocomputer.TaskStatusSucceeded {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got == nil || got.Status != taocomputer.TaskStatusSucceeded || got.Phase != taocomputer.TaskPhaseReady {
		t.Fatalf("task after online=%+v", got)
	}
}

func TestCloudWorkerRetriesSuccessPersistence(t *testing.T) {
	store := newFakeTaskStore()
	computers := &workerComputers{
		status:   computertypes.StatusOnline,
		computer: &computertypes.Computer{ID: "cmp-retry", NodeID: "node-retry", Status: computertypes.StatusOnline},
	}
	svc := newTestService(nil, computers).WithCloudTasks(store)
	task, err := svc.EnqueuePurchase(context.Background(), "u1", "", "u1", taskPurchaseRequestForTest(50, "private"))
	if err != nil {
		t.Fatal(err)
	}
	task.Status = taocomputer.TaskStatusRunning
	task.Phase = taocomputer.TaskPhaseProvision
	task.ComputerID = "cmp-retry"
	if err := store.Update(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	store.updateFailures = 1
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartCloudWorker(ctx)
	defer svc.StopCloudWorker()
	deadline := time.Now().Add(7 * time.Second)
	var got *taocomputer.CloudTask
	for time.Now().Before(deadline) {
		got, _ = store.Get(context.Background(), task.ID)
		if got != nil && got.Status == taocomputer.TaskStatusSucceeded {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got == nil || got.Status != taocomputer.TaskStatusSucceeded || got.Phase != taocomputer.TaskPhaseReady {
		t.Fatalf("task after update retry=%+v", got)
	}
}

func TestCloudWorkerDoesNotRerunSucceededTaskAfterRestart(t *testing.T) {
	store := newFakeTaskStore()
	computers := &workerComputers{
		status:   computertypes.StatusOnline,
		computer: &computertypes.Computer{ID: "cmp-done", NodeID: "node-done", Status: computertypes.StatusOnline},
	}
	first := newTestService(nil, computers).WithCloudTasks(store)
	task, err := first.EnqueuePurchase(context.Background(), "u1", "", "u1", taskPurchaseRequestForTest(50, "private"))
	if err != nil {
		t.Fatal(err)
	}
	task.Status = taocomputer.TaskStatusRunning
	task.Phase = taocomputer.TaskPhaseProvision
	task.ComputerID = "cmp-done"
	if err := store.Update(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	first.StartCloudWorker(ctx)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		got, _ := store.Get(context.Background(), task.ID)
		if got != nil && got.Status == taocomputer.TaskStatusSucceeded {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	first.StopCloudWorker()
	cancel()
	before, _ := store.Get(context.Background(), task.ID)
	second := newTestService(nil, computers).WithCloudTasks(store)
	secondCtx, secondCancel := context.WithCancel(context.Background())
	defer secondCancel()
	second.StartCloudWorker(secondCtx)
	defer second.StopCloudWorker()
	time.Sleep(300 * time.Millisecond)
	after, _ := store.Get(context.Background(), task.ID)
	if before == nil || after == nil || after.Status != taocomputer.TaskStatusSucceeded || after.Phase != taocomputer.TaskPhaseReady || !after.FinishedAt.Equal(*before.FinishedAt) {
		t.Fatalf("task changed after restart: before=%+v after=%+v", before, after)
	}
}

func TestCloudWorkerRecoversFromPhaseUpdateFailure(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	computers := &workerComputers{status: computertypes.StatusProvisioning}
	store := newFakeTaskStore()
	svc := newTestService(taocomputer.NewClient(srv.URL()), computers).
		WithTai(&workerTai{computers: computers}).
		WithCloudTasks(store).WithTaskKeyResolver(fakeTaskKeys{key: srv.Token})
	task, err := svc.EnqueuePurchase(context.Background(), "u1", "", "u1", taskPurchaseRequestForTest(50, "private"))
	if err != nil {
		t.Fatal(err)
	}
	task.Status = taocomputer.TaskStatusRunning
	task.Phase = taocomputer.TaskPhasePurchase
	if err := store.Update(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	store.updateFailures = 1
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartCloudWorker(ctx)
	defer svc.StopCloudWorker()
	deadline := time.Now().Add(10 * time.Second)
	var got *taocomputer.CloudTask
	for time.Now().Before(deadline) {
		got, _ = store.Get(context.Background(), task.ID)
		if got != nil && got.Status == taocomputer.TaskStatusSucceeded {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got == nil || got.Status != taocomputer.TaskStatusSucceeded || got.Phase != taocomputer.TaskPhaseReady {
		t.Fatalf("task after phase update failure=%+v", got)
	}
}

type attachFailComputers struct{ fakeComputerService }

func (attachFailComputers) CreateOrUpdate(context.Context, computertypes.CreateOrUpdateComputerRequest) (*computertypes.UpsertResult, error) {
	return nil, errors.New("attach failed")
}

type getFailComputers struct{ fakeComputerService }

func (getFailComputers) Get(context.Context, string) (*computertypes.Computer, error) {
	return nil, errors.New("computer lookup failed")
}

func TestCloudWorkerFailsWithoutTaskKeyResolver(t *testing.T) {
	store := newFakeTaskStore()
	svc := newTestService(nil, &fakeComputerService{}).WithCloudTasks(store)
	task, err := svc.EnqueuePurchase(context.Background(), "u1", "", "u1", taskPurchaseRequestForTest(50, "private"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartCloudWorker(ctx)
	defer svc.StopCloudWorker()
	deadline := time.Now().Add(5 * time.Second)
	var got *taocomputer.CloudTask
	for time.Now().Before(deadline) {
		got, _ = store.Get(context.Background(), task.ID)
		if got != nil && got.Status == taocomputer.TaskStatusFailed {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got == nil || got.Status != taocomputer.TaskStatusFailed || !strings.Contains(got.Error, "key resolver is not configured") {
		t.Fatalf("task=%+v", got)
	}
}

func TestCloudWorkerFailsOnTaoCreateError(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	srv.FailNext(http.MethodPost, "/v1/computers", http.StatusInternalServerError, "boom")
	store := newFakeTaskStore()
	svc := newTestService(taocomputer.NewClient(srv.URL()), &fakeComputerService{}).
		WithTai(&fakeTai{}).
		WithCloudTasks(store).WithTaskKeyResolver(fakeTaskKeys{key: srv.Token})
	task, err := svc.EnqueuePurchase(context.Background(), "u1", "", "u1", taskPurchaseRequestForTest(50, "private"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartCloudWorker(ctx)
	defer svc.StopCloudWorker()
	deadline := time.Now().Add(5 * time.Second)
	var got *taocomputer.CloudTask
	for time.Now().Before(deadline) {
		got, _ = store.Get(context.Background(), task.ID)
		if got != nil && got.Status == taocomputer.TaskStatusFailed {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got == nil || got.Status != taocomputer.TaskStatusFailed || !strings.Contains(got.Error, "status 500") {
		t.Fatalf("task=%+v", got)
	}
}

func TestCloudWorkerFailsOnAttachError(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	store := newFakeTaskStore()
	svc := newTestService(taocomputer.NewClient(srv.URL()), &attachFailComputers{}).
		WithTai(&fakeTai{}).
		WithCloudTasks(store).WithTaskKeyResolver(fakeTaskKeys{key: srv.Token})
	task, err := svc.EnqueuePurchase(context.Background(), "u1", "", "u1", taskPurchaseRequestForTest(50, "private"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartCloudWorker(ctx)
	defer svc.StopCloudWorker()
	deadline := time.Now().Add(5 * time.Second)
	var got *taocomputer.CloudTask
	for time.Now().Before(deadline) {
		got, _ = store.Get(context.Background(), task.ID)
		if got != nil && got.Status == taocomputer.TaskStatusFailed {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got == nil || got.Status != taocomputer.TaskStatusFailed || !strings.Contains(got.Error, "attach failed") {
		t.Fatalf("task=%+v", got)
	}
}

func TestCloudWorkerFailsWhenTaiComputerErrors(t *testing.T) {
	store := newFakeTaskStore()
	computers := &workerComputers{
		status:   computertypes.StatusError,
		computer: &computertypes.Computer{ID: "cmp-error", NodeID: "node-error", Status: computertypes.StatusError},
	}
	svc := newTestService(nil, computers).WithCloudTasks(store).WithTaskKeyResolver(fakeTaskKeys{})
	task, err := svc.EnqueuePurchase(context.Background(), "u1", "", "u1", taskPurchaseRequestForTest(50, "private"))
	if err != nil {
		t.Fatal(err)
	}
	task.Status = taocomputer.TaskStatusRunning
	task.Phase = taocomputer.TaskPhaseTai
	task.InstanceID = "inst-error"
	task.ComputerID = "cmp-error"
	if err := store.Update(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartCloudWorker(ctx)
	defer svc.StopCloudWorker()
	deadline := time.Now().Add(5 * time.Second)
	var got *taocomputer.CloudTask
	for time.Now().Before(deadline) {
		got, _ = store.Get(context.Background(), task.ID)
		if got != nil && got.Status == taocomputer.TaskStatusFailed {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got == nil || got.Status != taocomputer.TaskStatusFailed || !strings.Contains(got.Error, "reached error") {
		t.Fatalf("task=%+v", got)
	}
}

func TestCloudWorkerFailsWhenComputerLookupErrors(t *testing.T) {
	store := newFakeTaskStore()
	svc := newTestService(nil, &getFailComputers{}).WithCloudTasks(store).WithTaskKeyResolver(fakeTaskKeys{})
	task, err := svc.EnqueuePurchase(context.Background(), "u1", "", "u1", taskPurchaseRequestForTest(50, "private"))
	if err != nil {
		t.Fatal(err)
	}
	task.Status = taocomputer.TaskStatusRunning
	task.Phase = taocomputer.TaskPhaseTai
	task.InstanceID = "inst-missing"
	task.ComputerID = "cmp-missing"
	if err := store.Update(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartCloudWorker(ctx)
	defer svc.StopCloudWorker()
	deadline := time.Now().Add(5 * time.Second)
	var got *taocomputer.CloudTask
	for time.Now().Before(deadline) {
		got, _ = store.Get(context.Background(), task.ID)
		if got != nil && got.Status == taocomputer.TaskStatusFailed {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got == nil || got.Status != taocomputer.TaskStatusFailed || !strings.Contains(got.Error, "computer lookup failed") {
		t.Fatalf("task=%+v", got)
	}
}

func TestEnqueuePurchaseConfigurationErrors(t *testing.T) {
	ctx := context.Background()
	if _, err := taocomputer.NewService(nil, nil).EnqueuePurchase(ctx, "u", "", "u", taskPurchaseRequestForTest(50, "private")); err == nil {
		t.Fatal("missing task store should fail")
	}
	svc := taocomputer.NewService(nil, nil).WithCloudTasks(newFakeTaskStore())
	if _, err := svc.EnqueuePurchase(ctx, "u", "", "u", taskPurchaseRequestForTest(50, "private")); err == nil {
		t.Fatal("missing computer service should fail")
	}
	svc = newTestService(nil, &fakeComputerService{}).WithCloudTasks(newFakeTaskStore())
	if _, err := svc.EnqueuePurchase(ctx, "u", "", "u", taocomputer.PurchaseRequest{Resource: "r"}); !errors.Is(err, taocomputer.ErrInvalidPurchaseRequest) {
		t.Fatalf("invalid intent err=%v", err)
	}
}

func TestCloudWorkerFailsWithInvalidServerURLInTask(t *testing.T) {
	store := newFakeTaskStore()
	svc := newTestService(nil, &fakeComputerService{}).WithCloudTasks(store).
		WithTaskKeyResolver(fakeTaskKeys{}).WithServerURL("http://valid.example.com")
	task, err := svc.EnqueuePurchase(context.Background(), "u1", "", "u1", taskPurchaseRequestForTest(50, "private"))
	if err != nil {
		t.Fatal(err)
	}
	svc.WithServerURL("http://127.0.0.1:5099")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartCloudWorker(ctx)
	defer svc.StopCloudWorker()
	deadline := time.Now().Add(5 * time.Second)
	var got *taocomputer.CloudTask
	for time.Now().Before(deadline) {
		got, _ = store.Get(context.Background(), task.ID)
		if got != nil && got.Status == taocomputer.TaskStatusFailed {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got == nil || got.Status != taocomputer.TaskStatusFailed || !strings.Contains(got.Error, "cloud-reachable") {
		t.Fatalf("task=%+v", got)
	}
}

func TestCloudWorkerRecoversInstanceWithoutComputer(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	client := taocomputer.NewClient(srv.URL())
	created, err := client.Create(context.Background(), testTAOKey, validCreate("flex-2c2g"))
	if err != nil {
		t.Fatal(err)
	}
	computers := &workerComputers{status: computertypes.StatusProvisioning}
	store := newFakeTaskStore()
	svc := newTestService(client, computers).WithTai(&workerTai{computers: computers}).
		WithCloudTasks(store).WithTaskKeyResolver(fakeTaskKeys{key: srv.Token})
	task := &taocomputer.CloudTask{
		ID: "task_recover", Status: taocomputer.TaskStatusRunning, Phase: taocomputer.TaskPhaseProvision,
		DedupeKey: "dedupe-recover", BillingUserID: "u1", TaoName: "host", Resource: "flex-2c2g",
		Region: "bj-cn", Image: "tao-base-1.0", BillingUnit: "HOUR", ComputerName: "cloud", Visibility: "private",
		RequestParams: map[string]any{"image": "tao-base-1.0"}, InstanceID: created.InstanceID,
		StartedAt: func() *time.Time { now := time.Now().UTC(); return &now }(), CreatedAt: time.Now().UTC(),
	}
	if err := store.Create(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartCloudWorker(ctx)
	defer svc.StopCloudWorker()
	deadline := time.Now().Add(10 * time.Second)
	var got *taocomputer.CloudTask
	for time.Now().Before(deadline) {
		got, _ = store.Get(context.Background(), task.ID)
		if got != nil && got.Status == taocomputer.TaskStatusSucceeded {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got == nil || got.Status != taocomputer.TaskStatusSucceeded || got.ComputerID == "" {
		t.Fatalf("task=%+v", got)
	}
}

func TestCloudWorkerFailsOnTaoStatusError(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	client := taocomputer.NewClient(srv.URL())
	created, err := client.Create(context.Background(), testTAOKey, validCreate("flex-2c2g"))
	if err != nil {
		t.Fatal(err)
	}
	srv.SetInstanceStatus(created.InstanceID, "error", "provider failed")
	computers := &workerComputers{status: computertypes.StatusProvisioning}
	store := newFakeTaskStore()
	svc := newTestService(client, computers).WithTai(&workerTai{computers: computers}).
		WithCloudTasks(store).WithTaskKeyResolver(fakeTaskKeys{key: srv.Token})
	task := &taocomputer.CloudTask{
		ID: "task_error", Status: taocomputer.TaskStatusRunning, Phase: taocomputer.TaskPhaseProvision,
		DedupeKey: "dedupe-error", BillingUserID: "u1", TaoName: "host", Resource: "flex-2c2g",
		Region: "bj-cn", Image: "tao-base-1.0", BillingUnit: "HOUR", ComputerName: "cloud", Visibility: "private",
		RequestParams: map[string]any{"image": "tao-base-1.0"}, InstanceID: created.InstanceID,
		StartedAt: func() *time.Time { now := time.Now().UTC(); return &now }(), CreatedAt: time.Now().UTC(),
	}
	if err := store.Create(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartCloudWorker(ctx)
	defer svc.StopCloudWorker()
	deadline := time.Now().Add(5 * time.Second)
	var got *taocomputer.CloudTask
	for time.Now().Before(deadline) {
		got, _ = store.Get(context.Background(), task.ID)
		if got != nil && got.Status == taocomputer.TaskStatusFailed {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got == nil || got.Status != taocomputer.TaskStatusFailed || !strings.Contains(got.Error, "reached error") {
		t.Fatalf("task=%+v", got)
	}
}

func TestCloudWorkerFailsOnPhaseTaiWithoutComputer(t *testing.T) {
	store := newFakeTaskStore()
	svc := newTestService(nil, &fakeComputerService{}).WithCloudTasks(store).
		WithTaskKeyResolver(fakeTaskKeys{})
	task, err := svc.EnqueuePurchase(context.Background(), "u1", "", "u1", taskPurchaseRequestForTest(50, "private"))
	if err != nil {
		t.Fatal(err)
	}
	task.Status = taocomputer.TaskStatusRunning
	task.Phase = taocomputer.TaskPhaseTai
	if err := store.Update(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartCloudWorker(ctx)
	defer svc.StopCloudWorker()
	deadline := time.Now().Add(5 * time.Second)
	var got *taocomputer.CloudTask
	for time.Now().Before(deadline) {
		got, _ = store.Get(context.Background(), task.ID)
		if got != nil && got.Status == taocomputer.TaskStatusFailed {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got == nil || got.Status != taocomputer.TaskStatusFailed || !strings.Contains(got.Error, "tao computer client is not configured") {
		t.Fatalf("task=%+v", got)
	}
}

func TestCloudTaskServiceDependencyAndStoreErrors(t *testing.T) {
	ctx := context.Background()
	svc := taocomputer.NewService(nil, nil)
	if _, err := svc.GetTask(ctx, "u", "x"); err == nil {
		t.Fatal("GetTask without store should fail")
	}
	store := newFakeTaskStore()
	store.getErr = errors.New("db read failed")
	svc = newTestService(nil, &fakeComputerService{}).WithCloudTasks(store).WithTaskKeyResolver(fakeTaskKeys{})
	if _, err := svc.GetTask(ctx, "u", "x"); err == nil {
		t.Fatal("GetTask with store error should fail")
	}
	createStore := newFakeTaskStore()
	createStore.createErr = errors.New("insert failed")
	svc = newTestService(nil, &fakeComputerService{}).WithCloudTasks(createStore).WithTaskKeyResolver(fakeTaskKeys{})
	if _, err := svc.EnqueuePurchase(ctx, "u", "", "u", taskPurchaseRequestForTest(50, "private")); err == nil {
		t.Fatal("enqueue with create error should fail")
	}
	deleteStore := newFakeTaskStore()
	deleting := &deletingComputers{}
	svc = newTestService(nil, deleting).WithCloudTasks(deleteStore).WithTaskKeyResolver(fakeTaskKeys{})
	old, err := svc.EnqueuePurchase(ctx, "u", "", "u", taskPurchaseRequestForTest(50, "private"))
	if err != nil {
		t.Fatal(err)
	}
	old.Status = taocomputer.TaskStatusSucceeded
	old.Phase = taocomputer.TaskPhaseReady
	old.ComputerID = "cmp-gone"
	if err := deleteStore.Update(ctx, old); err != nil {
		t.Fatal(err)
	}
	deleteStore.deleteErr = errors.New("delete failed")
	if _, err := svc.EnqueuePurchase(ctx, "u", "", "u", taskPurchaseRequestForTest(50, "private")); err == nil {
		t.Fatal("delete error should be returned")
	}
}

func TestCloudWorkerStoreListAndResetErrors(t *testing.T) {
	store := newFakeTaskStore()
	store.listErr = errors.New("list failed")
	store.resetErr = errors.New("reset failed")
	svc := newTestService(nil, &fakeComputerService{}).WithCloudTasks(store)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartCloudWorker(ctx)
	time.Sleep(2300 * time.Millisecond)
	svc.StopCloudWorker()
}

func TestCloudWorkerClaimUpdateFailure(t *testing.T) {
	store := newFakeTaskStore()
	store.updateFailures = 1
	svc := newTestService(nil, &fakeComputerService{}).WithCloudTasks(store).WithTaskKeyResolver(fakeTaskKeys{})
	if _, err := svc.EnqueuePurchase(context.Background(), "u", "", "u", taskPurchaseRequestForTest(50, "private")); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartCloudWorker(ctx)
	defer svc.StopCloudWorker()
	time.Sleep(2300 * time.Millisecond)
	tasks, _ := store.ListRunnable(context.Background(), 10)
	if len(tasks) != 1 || tasks[0].Status != taocomputer.TaskStatusQueued {
		t.Fatalf("task should remain queued after claim failure: %+v", tasks)
	}
}

func TestCloudWorkerRecoveryClientErrors(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	store := newFakeTaskStore()
	svc := newTestService(taocomputer.NewClient(srv.URL()), &workerComputers{}).
		WithCloudTasks(store).WithTaskKeyResolver(fakeTaskKeys{key: srv.Token})
	task := &taocomputer.CloudTask{ID: "task_missing_instance", Status: taocomputer.TaskStatusRunning, Phase: taocomputer.TaskPhaseProvision,
		DedupeKey: "missing", BillingUserID: "u", TaoName: "host", Resource: "flex-2c2g", Region: "bj-cn",
		Image: "tao-base-1.0", BillingUnit: "HOUR", ComputerName: "cloud", Visibility: "private",
		RequestParams: map[string]any{"image": "tao-base-1.0"}, InstanceID: "inst-does-not-exist",
		StartedAt: func() *time.Time { now := time.Now().UTC(); return &now }(), CreatedAt: time.Now().UTC()}
	_ = store.Create(context.Background(), task)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartCloudWorker(ctx)
	defer svc.StopCloudWorker()
	deadline := time.Now().Add(5 * time.Second)
	var got *taocomputer.CloudTask
	for time.Now().Before(deadline) {
		got, _ = store.Get(context.Background(), task.ID)
		if got != nil && got.Status == taocomputer.TaskStatusFailed {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got == nil || got.Status != taocomputer.TaskStatusFailed {
		t.Fatalf("task=%+v", got)
	}
}

func TestCloudWorkerPendingTaoState(t *testing.T) {
	srv := taocloud.New()
	srv.AutoRun = false
	t.Cleanup(srv.Close)
	client := taocomputer.NewClient(srv.URL())
	created, err := client.Create(context.Background(), testTAOKey, validCreate("flex-2c2g"))
	if err != nil {
		t.Fatal(err)
	}
	computers := &workerComputers{status: computertypes.StatusProvisioning}
	store := newFakeTaskStore()
	svc := newTestService(client, computers).WithTai(&workerTai{computers: computers}).
		WithCloudTasks(store).WithTaskKeyResolver(fakeTaskKeys{key: srv.Token})
	task := &taocomputer.CloudTask{ID: "task_pending", Status: taocomputer.TaskStatusRunning, Phase: taocomputer.TaskPhaseProvision,
		DedupeKey: "pending", BillingUserID: "u", TaoName: "host", Resource: "flex-2c2g", Region: "bj-cn",
		Image: "tao-base-1.0", BillingUnit: "HOUR", ComputerName: "cloud", Visibility: "private",
		RequestParams: map[string]any{"image": "tao-base-1.0"}, InstanceID: created.InstanceID,
		StartedAt: func() *time.Time { now := time.Now().UTC(); return &now }(), CreatedAt: time.Now().UTC()}
	_ = store.Create(context.Background(), task)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartCloudWorker(ctx)
	defer svc.StopCloudWorker()
	time.Sleep(300 * time.Millisecond)
	got, _ := store.Get(context.Background(), task.ID)
	if got == nil || got.Status != taocomputer.TaskStatusRunning || got.Phase != taocomputer.TaskPhaseProvision {
		t.Fatalf("pending task=%+v", got)
	}
}

func TestCloudWorkerPhaseTaiTimeout(t *testing.T) {
	store := newFakeTaskStore()
	computers := &workerComputers{status: computertypes.StatusPending, computer: &computertypes.Computer{ID: "cmp-timeout", NodeID: "node-timeout", Status: computertypes.StatusPending}}
	svc := newTestService(nil, computers).WithCloudTasks(store).WithTaskKeyResolver(fakeTaskKeys{})
	task, err := svc.EnqueuePurchase(context.Background(), "u", "", "u", taskPurchaseRequestForTest(50, "private"))
	if err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-taocomputer.CloudTaskTimeout - time.Minute)
	task.Status = taocomputer.TaskStatusRunning
	task.Phase = taocomputer.TaskPhaseTai
	task.InstanceID = "inst-timeout"
	task.ComputerID = "cmp-timeout"
	task.StartedAt = &old
	if err := store.Update(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartCloudWorker(ctx)
	defer svc.StopCloudWorker()
	deadline := time.Now().Add(5 * time.Second)
	var got *taocomputer.CloudTask
	for time.Now().Before(deadline) {
		got, _ = store.Get(context.Background(), task.ID)
		if got != nil && got.Status == taocomputer.TaskStatusFailed {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got == nil || got.Status != taocomputer.TaskStatusFailed || !strings.Contains(got.Error, "timed out") {
		t.Fatalf("task=%+v", got)
	}
}

func TestEnqueuePurchaseServerURLVariants(t *testing.T) {
	for _, raw := range []string{"not-a-url", "ftp://example.com", "http://localhost:5099", "http://0.0.0.0:5099", "http://[::1]:5099"} {
		svc := newTestService(nil, &fakeComputerService{}).WithCloudTasks(newFakeTaskStore()).
			WithTaskKeyResolver(fakeTaskKeys{}).WithServerURL(raw)
		if _, err := svc.EnqueuePurchase(context.Background(), "u", "", "u", taskPurchaseRequestForTest(50, "private")); !errors.Is(err, taocomputer.ErrCloudServerURL) {
			t.Fatalf("%q err=%v", raw, err)
		}
	}
}

func TestCloudWorkerNilComputerDependency(t *testing.T) {
	store := newFakeTaskStore()
	svc := newTestService(nil, nil).WithCloudTasks(store).WithTaskKeyResolver(fakeTaskKeys{})
	task := &taocomputer.CloudTask{ID: "task_nil_dep", Status: taocomputer.TaskStatusQueued, Phase: taocomputer.TaskPhasePurchase,
		DedupeKey: "nil-dep", UserID: "u", BillingUserID: "u", TaoName: "host", Resource: "r", Region: "r", Image: "i",
		BillingUnit: "HOUR", ComputerName: "cloud", Visibility: "private", CreatedAt: time.Now().UTC()}
	_ = store.Create(context.Background(), task)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartCloudWorker(ctx)
	time.Sleep(2300 * time.Millisecond)
	svc.StopCloudWorker()
}

func TestViewNilClient(t *testing.T) {
	if _, err := taocomputer.NewService(nil, nil).View(context.Background(), "key", "instance"); err == nil {
		t.Fatal("nil client should fail")
	}
}

func TestEnqueuePurchaseDefaultsVisibility(t *testing.T) {
	svc := newTestService(nil, &fakeComputerService{}).WithCloudTasks(newFakeTaskStore()).WithTaskKeyResolver(fakeTaskKeys{})
	req := taskPurchaseRequestForTest(50, "")
	req.Computer.Visibility = ""
	task, err := svc.EnqueuePurchase(context.Background(), "u", "", "u", req)
	if err != nil || task.Visibility != computertypes.VisibilityPrivate {
		t.Fatalf("task=%+v err=%v", task, err)
	}
	svc.WithServerURL("")
	if _, err := svc.EnqueuePurchase(context.Background(), "u2", "", "u2", req); !errors.Is(err, taocomputer.ErrCloudServerURL) {
		t.Fatalf("empty server err=%v", err)
	}
}

func TestReuseTaskStoreErrors(t *testing.T) {
	ctx := context.Background()
	store := newFakeTaskStore()
	svc := newTestService(nil, &deletingComputers{}).WithCloudTasks(store).WithTaskKeyResolver(fakeTaskKeys{})
	first, err := svc.EnqueuePurchase(ctx, "u", "", "u", taskPurchaseRequestForTest(50, "private"))
	if err != nil {
		t.Fatal(err)
	}
	first.Status = taocomputer.TaskStatusFailed
	first.ComputerID = ""
	_ = store.Update(ctx, first)
	store.updateFailures = 1
	if _, err := svc.EnqueuePurchase(ctx, "u", "", "u", taskPurchaseRequestForTest(50, "private")); err == nil {
		t.Fatal("requeue update error should be returned")
	}
	store = newFakeTaskStore()
	svc = newTestService(nil, &fakeComputerService{}).WithCloudTasks(store).WithTaskKeyResolver(fakeTaskKeys{})
	active, err := svc.EnqueuePurchase(ctx, "u", "", "u", taskPurchaseRequestForTest(50, "private"))
	if err != nil {
		t.Fatal(err)
	}
	same, err := svc.EnqueuePurchase(ctx, "u", "", "u", taskPurchaseRequestForTest(50, "private"))
	if err != nil || same.ID != active.ID {
		t.Fatalf("active reuse=%+v/%v", same, err)
	}
}

func TestCreateFreshReusedTaskError(t *testing.T) {
	ctx := context.Background()
	store := newFakeTaskStore()
	svc := newTestService(nil, &deletingComputers{}).WithCloudTasks(store).WithTaskKeyResolver(fakeTaskKeys{})
	old, err := svc.EnqueuePurchase(ctx, "u", "", "u", taskPurchaseRequestForTest(50, "private"))
	if err != nil {
		t.Fatal(err)
	}
	old.Status = taocomputer.TaskStatusSucceeded
	old.ComputerID = "cmp-gone"
	_ = store.Update(ctx, old)
	store.createErr = errors.New("insert failed")
	if _, err := svc.EnqueuePurchase(ctx, "u", "", "u", taskPurchaseRequestForTest(50, "private")); err == nil {
		t.Fatal("create fresh error should be returned")
	}
}

func TestStartWorkerStoreErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	taocomputer.NewService(nil, nil).StartCloudWorker(ctx)
	taocomputer.NewService(nil, nil).StopCloudWorker()
	store := newFakeTaskStore()
	store.resetErr = errors.New("reset failed")
	store.listErr = errors.New("list failed")
	svc := newTestService(nil, &fakeComputerService{}).WithCloudTasks(store)
	svc.StartCloudWorker(ctx)
	time.Sleep(2200 * time.Millisecond)
	svc.StopCloudWorker()
}

func TestFailTaskUpdateError(t *testing.T) {
	store := newFakeTaskStore()
	store.updateFailures = 2
	svc := newTestService(nil, &fakeComputerService{}).WithCloudTasks(store).WithTaskKeyResolver(fakeTaskKeys{})
	task := &taocomputer.CloudTask{ID: "task_fail_update", Status: taocomputer.TaskStatusRunning, Phase: taocomputer.TaskPhaseProvision,
		DedupeKey: "fail-update", UserID: "u", BillingUserID: "u", TaoName: "host", Resource: "r", Region: "r", Image: "i",
		BillingUnit: "HOUR", ComputerName: "cloud", Visibility: "private", CreatedAt: time.Now().UTC(),
		StartedAt: func() *time.Time {
			now := time.Now().UTC().Add(-taocomputer.CloudTaskTimeout - time.Minute)
			return &now
		}()}
	_ = store.Create(context.Background(), task)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartCloudWorker(ctx)
	time.Sleep(2300 * time.Millisecond)
	svc.StopCloudWorker()
}
