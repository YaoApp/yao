package computer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yaoapp/yao/modules/computer/types"
	computertypes "github.com/yaoapp/yao/modules/computer/types"
)

func TestCloudTaskViewAndHelpers(t *testing.T) {
	task := &CloudTask{ID: "task_1", Status: TaskStatusFailed, Phase: TaskPhasePurchase, Error: "tao api status 402: insufficient"}
	view := task.View()
	if view.TaskID != "task_1" || view.Status != TaskStatusFailed || view.Phase != TaskPhasePurchase ||
		view.Error == nil || *view.Error != task.Error || view.InstanceID != nil || view.ComputerID != nil {
		t.Fatalf("view=%+v", view)
	}
	task.InstanceID, task.ComputerID = "inst_1", "cmp_1"
	view = task.View()
	if view.InstanceID == nil || *view.InstanceID != "inst_1" || view.ComputerID == nil || *view.ComputerID != "cmp_1" {
		t.Fatalf("view with ids=%+v", view)
	}

	if got := uniqueStrings([]string{"a", "a", "b"}); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("unique=%v", got)
	}
	if got := normalizeNumber(float64(50)); got != "50" {
		t.Fatalf("normalize=%q", got)
	}
	if got := normalizeNumber("bad"); got != "bad" {
		t.Fatalf("normalize string=%q", got)
	}
}

func TestValidatePurchaseIntent(t *testing.T) {
	base := PurchaseRequest{
		Resource: "r", Region: "gz", Name: "host", BillingUnit: BillingHour,
		RequestParams: map[string]any{"image": "img"}, Computer: ComputerSpec{Visibility: types.VisibilityPrivate},
	}
	if err := validatePurchaseIntent(base); err != nil {
		t.Fatal(err)
	}
	cases := []PurchaseRequest{
		{},
		{Resource: "r", Region: "gz", Name: "host", BillingUnit: "YEAR", RequestParams: map[string]any{"image": "img"}},
		{Resource: "r", Region: "gz", Name: "host", BillingUnit: BillingMonth, PeriodMonths: 2, RequestParams: map[string]any{"image": "img"}},
		{Resource: "r", Region: "gz", Name: "host", BillingUnit: BillingHour, PeriodMonths: 1, RequestParams: map[string]any{"image": "img"}},
		{Resource: "r", Region: "gz", Name: "host", BillingUnit: BillingHour, RequestParams: map[string]any{}},
		{Resource: "r", Region: "gz", Name: "host", BillingUnit: BillingHour, RequestParams: map[string]any{"image": "img"}, Computer: ComputerSpec{Visibility: "public"}},
		{Resource: "r", Region: "gz", Name: "host", BillingUnit: BillingHour, RequestParams: map[string]any{"image": "img"}, Members: []string{"m"}, Computer: ComputerSpec{Visibility: types.VisibilityPrivate}},
	}
	for index, req := range cases {
		if err := validatePurchaseIntent(req); err == nil {
			t.Fatalf("case %d should fail: %+v", index, req)
		}
	}
	longName := base
	longName.Name = string(make([]rune, 51))
	if err := validatePurchaseIntent(longName); err == nil {
		t.Fatal("long name should fail")
	}
}

func TestTaskErrorCode(t *testing.T) {
	if got := taskErrorCode(&APIError{Status: 402}); got != "tao_402" {
		t.Fatalf("api code=%q", got)
	}
	if got := taskErrorCode(ErrTaiStart); got != "tai_start_error" {
		t.Fatalf("tai code=%q", got)
	}
	if got := taskErrorCode(errors.New("boom")); got != "task_failed" {
		t.Fatalf("generic code=%q", got)
	}
}

func TestCloudTaskFromRowTimestamps(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	task := cloudTaskFromRow(map[string]interface{}{
		"id": "task_1", "status": TaskStatusQueued, "phase": TaskPhasePurchase,
		"request_params": `{"image":"img","disk_gb":50}`, "members": `["b","a"]`,
		"started_at": now, "created_at": now.Format(time.RFC3339),
	})
	if task.RequestParams["image"] != "img" || len(task.Members) != 2 || task.StartedAt == nil || task.CreatedAt.IsZero() {
		t.Fatalf("task=%+v", task)
	}
}

func TestValueHelpers(t *testing.T) {
	if v, ok := intValue(42); !ok || v != 42 {
		t.Fatalf("int=%d/%v", v, ok)
	}
	if v, ok := intValue(int64(43)); !ok || v != 43 {
		t.Fatalf("int64=%d/%v", v, ok)
	}
	if v, ok := intValue(float64(44)); !ok || v != 44 {
		t.Fatalf("float=%d/%v", v, ok)
	}
	if v, ok := intValue(uint64(45)); !ok || v != 45 {
		t.Fatalf("uint=%d/%v", v, ok)
	}
	if v, ok := intValue("46"); !ok || v != 46 {
		t.Fatalf("string=%d/%v", v, ok)
	}
	if v, ok := intValue("bad"); ok || v != 0 {
		t.Fatalf("bad=%d/%v", v, ok)
	}
	if v, ok := intValue(nil); ok || v != 0 {
		t.Fatalf("nil=%d/%v", v, ok)
	}
	if v := stringValue(map[string]interface{}{"s": 42}, "s"); v != "42" {
		t.Fatalf("stringValue=%q", v)
	}
	if v := stringValue(nil, "s"); v != "" {
		t.Fatalf("nil stringValue=%q", v)
	}
	if v := normalizeNumber(int(1)); v != "1" {
		t.Fatalf("norm int=%q", v)
	}
	if v := normalizeNumber(int64(2)); v != "2" {
		t.Fatalf("norm int64=%q", v)
	}
	if v := normalizeNumber(float32(3)); v != "3" {
		t.Fatalf("norm float32=%q", v)
	}
	if v := normalizeNumber(uint64(4)); v != "4" {
		t.Fatalf("norm uint=%q", v)
	}
}

func TestTimeAndJSONHelpers(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	if got := timeValue(now); got == nil || !got.Equal(now) {
		t.Fatalf("time.Time=%v", got)
	}
	if got := timeValue(&now); got == nil || !got.Equal(now) {
		t.Fatalf("*time.Time=%v", got)
	}
	if got := timeValue(now.Format(time.RFC3339)); got == nil || !got.Equal(now) {
		t.Fatalf("RFC3339=%v", got)
	}
	if got := timeValue("2026-11-26 10:00:00"); got == nil {
		t.Fatal("legacy timestamp should parse")
	}
	if got := timeValue("bad"); got != nil {
		t.Fatalf("bad time=%v", got)
	}
	var members []string
	if !decodeJSONValue(`["a"]`, &members) || len(members) != 1 {
		t.Fatalf("json string=%v", members)
	}
	var values []string
	if !decodeJSONValue([]byte(`["b"]`), &values) || len(values) != 1 {
		t.Fatalf("json bytes=%v", values)
	}
	var object map[string]any
	if !decodeJSONValue(map[string]any{"a": "b"}, &object) || object["a"] != "b" {
		t.Fatalf("json object=%v", object)
	}
	if decodeJSONValue(nil, &object) || decodeJSONValue("", &object) || decodeJSONValue("bad", &object) {
		t.Fatal("invalid JSON should fail")
	}
}

func TestDedupeKeyVariants(t *testing.T) {
	base := PurchaseRequest{Resource: "r", Region: "gz", Name: "host", BillingUnit: BillingHour, RequestParams: map[string]any{"image": "img"}}
	defaultKey := cloudDedupeKey("u", "", base)
	withDisk := base
	withDisk.RequestParams = map[string]any{"image": "img", "disk_gb": float64(50)}
	if cloudDedupeKey("u", "", withDisk) == defaultKey {
		t.Fatal("disk must change dedupe key")
	}
	withTeam := cloudDedupeKey("u", "t", base)
	if withTeam == defaultKey {
		t.Fatal("team must change dedupe key")
	}
	withName := base
	withName.Computer.Name = "alias"
	if cloudDedupeKey("u", "", withName) == defaultKey {
		t.Fatal("computer name must change dedupe key")
	}
}

func TestErrorAndParamHelpers(t *testing.T) {
	if errorText(nil) != "" {
		t.Fatal("nil error text")
	}
	if errorText(context.DeadlineExceeded) == "" {
		t.Fatal("deadline text")
	}
	if got := errorText(context.DeadlineExceeded); got == "" {
		t.Fatal("deadline error text is empty")
	}
	if taskErrorCode(ErrInstanceDeleted) == "" {
		t.Fatal("deleted code must not be empty")
	}
	if taskErrorCode(ErrTaiControlKey) == "" {
		t.Fatal("control key error code must not be empty")
	}
	if stringParam(nil, "image") != "" || stringParam(map[string]any{"image": 42}, "image") != "" {
		t.Fatal("invalid string param")
	}
	if cloneParams(nil) != nil {
		t.Fatal("nil params clone")
	}
	params := map[string]any{"a": 1}
	cloned := cloneParams(params)
	cloned["a"] = 2
	if params["a"] != 1 {
		t.Fatal("clone params must copy")
	}
}

func TestControlKeyFallbacks(t *testing.T) {
	key := &ControlKey{KeyID: "key1", PrivateKey: "x"}
	svc := NewService(nil, nil).WithControlKeys("missing", key)
	if svc.activeControlKey() != key {
		t.Fatal("registered key should remain available")
	}
	svc.WithControlKey(key)
	if svc.activeControlKey() != key {
		t.Fatal("active key fallback failed")
	}
	svc = NewService(nil, nil).WithControlKeys("key1", key)
	if svc.activeControlKey() != key || svc.controlKeyFor(&computertypes.Computer{Metadata: map[string]interface{}{"tai_control_key_id": "key1"}}) != key {
		t.Fatal("control key ring failed")
	}
}
