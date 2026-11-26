//go:build e2e

package computer_test

import (
	"context"
	"errors"
	"testing"
	"time"

	taocomputer "github.com/yaoapp/yao/modules/tao/computer"
)

func TestE2ECloudTaskModelStore(t *testing.T) {
	_ = testEnvironment(t)
	ctx := context.Background()
	store := taocomputer.CloudTaskModelStore()
	task := &taocomputer.CloudTask{
		ID: "task_e2e_" + time.Now().Format("150405000000000"), Status: taocomputer.TaskStatusQueued,
		Phase: taocomputer.TaskPhasePurchase, DedupeKey: "dedupe-e2e-" + time.Now().Format("150405000000000"),
		UserID: "user-e2e", BillingUserID: "user-e2e", TaoName: "host", Resource: "flex-2c2g",
		Region: "bj-cn", Image: "tao-base-1.0", BillingUnit: "HOUR", ComputerName: "cloud",
		Visibility: "private", RequestParams: map[string]any{"image": "tao-base-1.0", "disk_gb": float64(50)},
		CreatedAt: time.Now().UTC(),
	}
	if err := store.Create(ctx, task); err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { _ = store.Delete(context.Background(), task.ID) })
	got, err := store.Get(ctx, task.ID)
	if err != nil || got.TaoName != "host" || got.RequestParams["image"] != "tao-base-1.0" {
		t.Fatalf("get=%+v err=%v", got, err)
	}
	if _, err := store.GetByDedupe(ctx, task.DedupeKey); err != nil {
		t.Fatalf("dedupe lookup: %v", err)
	}
	if _, err := store.GetByDedupe(ctx, "missing"); !errors.Is(err, taocomputer.ErrTaskNotFound) {
		t.Fatalf("missing dedupe err=%v", err)
	}
	runnable, err := store.ListRunnable(ctx, 100)
	if err != nil {
		t.Fatalf("list runnable: %v", err)
	}
	found := false
	for _, item := range runnable {
		if item.ID == task.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("task not in runnable: %+v", runnable)
	}
	task.Status = taocomputer.TaskStatusRunning
	started := time.Now().UTC()
	task.StartedAt = &started
	task.Attempts = 1
	if err := store.Update(ctx, task); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := store.ResetRunning(ctx); err != nil {
		t.Fatalf("reset: %v", err)
	}
	got, err = store.Get(ctx, task.ID)
	if err != nil || got.Status != taocomputer.TaskStatusQueued || got.StartedAt != nil {
		t.Fatalf("reset task=%+v err=%v", got, err)
	}
	if view := got.View(); view.TaskID != task.ID || view.Status != taocomputer.TaskStatusQueued {
		t.Fatalf("view=%+v", view)
	}
}

func TestE2ECloudTaskModelStoreErrorBranches(t *testing.T) {
	_ = testEnvironment(t)
	ctx := context.Background()
	store := taocomputer.CloudTaskModelStore()
	if _, err := store.Get(ctx, "task-missing"); !errors.Is(err, taocomputer.ErrTaskNotFound) {
		t.Fatalf("missing get err=%v", err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	task := &taocomputer.CloudTask{ID: "task_canceled", Status: taocomputer.TaskStatusQueued, Phase: taocomputer.TaskPhasePurchase, DedupeKey: "dedupe-canceled"}
	for name, fn := range map[string]func() error{
		"create":       func() error { return store.Create(canceled, task) },
		"get":          func() error { _, err := store.Get(canceled, "task-missing"); return err },
		"getByDedupe":  func() error { _, err := store.GetByDedupe(canceled, "missing"); return err },
		"update":       func() error { return store.Update(canceled, task) },
		"listRunnable": func() error { _, err := store.ListRunnable(canceled, 1); return err },
		"resetRunning": func() error { return store.ResetRunning(canceled) },
		"delete":       func() error { return store.Delete(canceled, "task-missing") },
	} {
		if err := fn(); !errors.Is(err, context.Canceled) {
			t.Fatalf("%s canceled err=%v", name, err)
		}
	}
}

func TestE2ECloudTaskModelStoreLimitAndDuplicate(t *testing.T) {
	_ = testEnvironment(t)
	ctx := context.Background()
	store := taocomputer.CloudTaskModelStore()
	now := time.Now().Format("150405000000000")
	tasks := []*taocomputer.CloudTask{
		{ID: "task_limit_a_" + now, Status: taocomputer.TaskStatusQueued, Phase: taocomputer.TaskPhasePurchase, DedupeKey: "dedupe-limit-a-" + now, UserID: "u", TaoName: "a", Resource: "r", Region: "r", Image: "i", BillingUnit: "HOUR", ComputerName: "a", Visibility: "private", CreatedAt: time.Now().UTC()},
		{ID: "task_limit_b_" + now, Status: taocomputer.TaskStatusQueued, Phase: taocomputer.TaskPhasePurchase, DedupeKey: "dedupe-limit-b-" + now, UserID: "u", TaoName: "b", Resource: "r", Region: "r", Image: "i", BillingUnit: "HOUR", ComputerName: "b", Visibility: "private", CreatedAt: time.Now().UTC()},
	}
	for _, task := range tasks {
		if err := store.Create(ctx, task); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func(t *taocomputer.CloudTask) func() { return func() { _ = store.Delete(context.Background(), t.ID) } }(task))
	}
	limited, err := store.ListRunnable(ctx, 1)
	if err != nil || len(limited) != 1 {
		t.Fatalf("limit list=%+v err=%v", limited, err)
	}
	if all, err := store.ListRunnable(ctx, 0); err != nil || len(all) < 2 {
		t.Fatalf("unlimited list=%+v err=%v", all, err)
	}
	duplicate := *tasks[0]
	duplicate.ID = duplicate.ID + "_dup"
	if err := store.Create(ctx, &duplicate); err == nil {
		t.Fatal("duplicate dedupe key should fail")
	}
	if err := store.ResetRunning(ctx); err != nil {
		t.Fatal(err)
	}
}
