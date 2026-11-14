package computer_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/yaoapp/yao/e2e/taocloud"
	computertypes "github.com/yaoapp/yao/modules/computer/types"
	taocomputer "github.com/yaoapp/yao/modules/tao/computer"
)

func testControlKey() *taocomputer.ControlKey {
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	signer, _ := ssh.NewSignerFromKey(privateKey)
	return &taocomputer.ControlKey{
		KeyID:      "test-control",
		PublicKey:  string(ssh.MarshalAuthorizedKey(signer.PublicKey())),
		PrivateKey: string(mustMarshalPrivateKey(privateKey)),
	}
}

func mustMarshalPrivateKey(key any) []byte {
	block, err := ssh.MarshalPrivateKey(key, "")
	if err != nil {
		panic(err)
	}
	return pemEncode(block)
}

func pemEncode(block *pem.Block) []byte {
	return pem.EncodeToMemory(block)
}

func newTestService(client taocomputer.TaoComputersClient, computers taocomputer.ComputerService) *taocomputer.Service {
	return taocomputer.NewService(client, computers).WithControlKey(testControlKey())
}

type fakeComputerService struct {
	computer *computertypes.Computer
	calls    []computertypes.CreateOrUpdateComputerRequest
}

func (f *fakeComputerService) CreateOrUpdate(_ context.Context, req computertypes.CreateOrUpdateComputerRequest) (*computertypes.UpsertResult, error) {
	f.calls = append(f.calls, req)
	if f.computer == nil {
		f.computer = &computertypes.Computer{ID: "cmp_test", NodeID: "node_test", InstanceID: req.InstanceID, Status: req.Status, Metadata: req.Metadata}
	} else if req.Status != "" {
		f.computer.Status = req.Status
	}
	return &computertypes.UpsertResult{Computer: f.computer, NodeToken: "token-test", ExpiresAt: time.Now().Add(time.Hour)}, nil
}

func (f *fakeComputerService) Get(_ context.Context, _ string) (*computertypes.Computer, error) {
	return f.computer, nil
}

type fakeTai struct {
	started      bool
	bootstrapErr error
	req          computertypes.StartTaiRequest
	target       computertypes.SSHTarget
}

func (f *fakeTai) Bootstrap(context.Context, computertypes.SSHTarget, computertypes.BootstrapRequest) error {
	return f.bootstrapErr
}
func (f *fakeTai) Start(_ context.Context, target computertypes.SSHTarget, req computertypes.StartTaiRequest) error {
	f.started = true
	f.target = target
	f.req = req
	return nil
}
func (f *fakeTai) Stop(context.Context, computertypes.SSHTarget) error { return nil }
func (f *fakeTai) Status(context.Context, computertypes.SSHTarget) (*computertypes.TaiStatus, error) {
	return &computertypes.TaiStatus{Running: true}, nil
}
func (f *fakeTai) Logs(context.Context, computertypes.SSHTarget, int) (string, error) { return "", nil }

func TestPurchaseCreatesProvisioningComputer(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	computers := &fakeComputerService{}
	svc := newTestService(taocomputer.NewClient(srv.URL()), computers)
	result, err := svc.Purchase(context.Background(), testTAOKey, taocomputer.PurchaseRequest{
		Resource: "flex-2c2g", Region: "bj-cn", Name: "host", BillingUnit: "HOUR",
		RequestParams: map[string]any{"image": "tao-base-1.0"},
		Computer:      taocomputer.ComputerSpec{Name: "my-cloud", Visibility: "private", LeaseSource: "tao"},
		UserID:        "user-1",
	})
	if err != nil {
		t.Fatalf("Purchase: %v", err)
	}
	if result.Instance == nil || result.Computer == nil || result.Computer.Status != computertypes.StatusProvisioning {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(computers.calls) != 1 || computers.calls[0].InstanceID != result.Instance.InstanceID {
		t.Fatalf("unexpected CreateOrUpdate calls: %+v", computers.calls)
	}
}

func TestGetAdvancesRunningAndStartsTai(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	client := taocomputer.NewClient(srv.URL())
	created, err := client.Create(context.Background(), testTAOKey, validCreate("flex-2c2g"))
	if err != nil {
		t.Fatal(err)
	}
	computers := &fakeComputerService{computer: &computertypes.Computer{ID: "cmp_test", NodeID: "node_test", InstanceID: created.InstanceID, Status: computertypes.StatusProvisioning, Metadata: map[string]interface{}{"data_dir": "/data"}}}
	tai := &fakeTai{}
	svc := newTestService(client, computers).WithTai(tai).WithServerURL("https://yao.example.com")

	result, err := svc.Get(context.Background(), testTAOKey, created.InstanceID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if result.Computer == nil || result.Computer.Status != computertypes.StatusPending {
		t.Fatalf("unexpected computer: %+v", result.Computer)
	}
	if !tai.started || tai.req.NodeToken != "token-test" || tai.req.NodeID != "node_test" || tai.req.ServerURL == "" {
		t.Fatalf("tai not started correctly: %+v", tai)
	}
	if !computers.calls[len(computers.calls)-1].IssueNodeToken {
		t.Fatal("expected issue_node_token on running")
	}
}

func TestSSHTargetAndStartRequest(t *testing.T) {
	instance := &taocomputer.Instance{PublicIP: "203.0.113.10", SSH: taocomputer.SSHInfo{Host: "203.0.113.10", Port: 22, User: "root"}}
	target := taocomputer.SSHTargetFrom(instance)
	if target.Host != "203.0.113.10" || target.Port != 22 || target.User != "root" {
		t.Fatalf("target=%+v", target)
	}
	req := taocomputer.StartTaiRequestFor(&computertypes.Computer{NodeID: "node-x", Metadata: map[string]interface{}{"data_dir": "/data", "sandbox_image": "img"}}, "tok", "https://yao")
	if req.NodeID != "node-x" || req.DataDir != "/data" || req.NodeToken != "tok" || req.ServerURL != "https://yao" || req.SandboxImage != "img" || req.ShutdownTimeout == 0 {
		t.Fatalf("start request=%+v", req)
	}
}

func TestPurchaseInvalidResource(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	svc := newTestService(taocomputer.NewClient(srv.URL()), &fakeComputerService{})
	if _, err := svc.Purchase(context.Background(), testTAOKey, taocomputer.PurchaseRequest{}); err == nil {
		t.Fatal("expected resource error")
	}
}

func TestListAndServiceErrors(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	svc := newTestService(taocomputer.NewClient(srv.URL()), &fakeComputerService{})
	if _, err := svc.List(context.Background(), testTAOKey, taocomputer.ListOptions{}); err != nil {
		t.Fatalf("List: %v", err)
	}
	if _, err := newTestService(nil, nil).List(context.Background(), testTAOKey, taocomputer.ListOptions{}); err == nil {
		t.Fatal("expected config error")
	}
	if _, err := newTestService(nil, nil).Get(context.Background(), testTAOKey, "x"); err == nil {
		t.Fatal("expected config error")
	}
}

func TestPurchaseTeamScope(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	computers := &fakeComputerService{}
	svc := newTestService(taocomputer.NewClient(srv.URL()), computers)
	_, err := svc.Purchase(context.Background(), testTAOKey, taocomputer.PurchaseRequest{
		Resource: "flex-2c2g", Region: "bj-cn", Name: "host", BillingUnit: "HOUR",
		RequestParams: map[string]any{"image": "tao-base-1.0"},
		TeamID:        "team-1",
	})
	if err != nil {
		t.Fatalf("purchase: %v", err)
	}
	if computers.calls[0].Scope != computertypes.ScopeTeam || computers.calls[0].Name == "" {
		t.Fatalf("scope/name: %+v", computers.calls[0])
	}
}

type errorTai struct{ fakeTai }

func (e *errorTai) Start(context.Context, computertypes.SSHTarget, computertypes.StartTaiRequest) error {
	return errors.New("ssh failed")
}

func TestGetTaiFailureMarksError(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	client := taocomputer.NewClient(srv.URL())
	created, err := client.Create(context.Background(), testTAOKey, validCreate("flex-2c2g"))
	if err != nil {
		t.Fatal(err)
	}
	computers := &fakeComputerService{computer: &computertypes.Computer{ID: "cmp_test", NodeID: "node_test", InstanceID: created.InstanceID, Status: computertypes.StatusProvisioning}}
	svc := newTestService(client, computers).WithTai(&errorTai{})
	if _, err := svc.Get(context.Background(), testTAOKey, created.InstanceID); err == nil {
		t.Fatal("expected tai error")
	}
	if computers.computer.Status != computertypes.StatusError {
		t.Fatalf("status=%s", computers.computer.Status)
	}
}

func TestPurchaseClientError(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	svc := newTestService(taocomputer.NewClient(srv.URL()), &fakeComputerService{})
	if _, err := svc.Purchase(context.Background(), testTAOKey, taocomputer.PurchaseRequest{Resource: "missing"}); err == nil {
		t.Fatal("expected Tao error")
	}
	if _, err := newTestService(nil, nil).Purchase(context.Background(), testTAOKey, taocomputer.PurchaseRequest{Resource: "x"}); err == nil {
		t.Fatal("expected config error")
	}
}

func TestGetErrorBranches(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	client := taocomputer.NewClient(srv.URL())
	created, err := client.Create(context.Background(), testTAOKey, validCreate("flex-2c2g"))
	if err != nil {
		t.Fatal(err)
	}
	computers := &fakeComputerService{computer: &computertypes.Computer{ID: "cmp_test", NodeID: "node_test", InstanceID: created.InstanceID, Status: computertypes.StatusProvisioning}}
	svc := newTestService(client, computers)
	if _, err := svc.Get(context.Background(), testTAOKey, "missing"); err == nil {
		t.Fatal("expected Tao not found")
	}
}

type failingComputerService struct{}

func (failingComputerService) CreateOrUpdate(context.Context, computertypes.CreateOrUpdateComputerRequest) (*computertypes.UpsertResult, error) {
	return nil, errors.New("computer failed")
}
func (failingComputerService) Get(context.Context, string) (*computertypes.Computer, error) {
	return nil, errors.New("computer failed")
}

func TestPurchaseComputerFailure(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	svc := newTestService(taocomputer.NewClient(srv.URL()), failingComputerService{})
	if _, err := svc.Purchase(context.Background(), testTAOKey, taocomputer.PurchaseRequest{Resource: "flex-2c2g", Region: "bj-cn", Name: "host", BillingUnit: "HOUR", RequestParams: map[string]any{"image": "tao-base-1.0"}}); err == nil {
		t.Fatal("expected computer failure")
	}
}

func TestSSHTargetNilAndNames(t *testing.T) {
	if target := taocomputer.SSHTargetFrom(nil); target.Host != "" {
		t.Fatalf("target=%+v", target)
	}
	req := taocomputer.PurchaseRequest{Name: "host", Computer: taocomputer.ComputerSpec{Name: "computer"}}
	_ = req
}

func TestGetUsesControlKey(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	client := taocomputer.NewClient(srv.URL())
	created, err := client.Create(context.Background(), testTAOKey, validCreate("flex-2c2g"))
	if err != nil {
		t.Fatal(err)
	}
	computers := &fakeComputerService{computer: &computertypes.Computer{ID: "cmp_test", NodeID: "node_test", InstanceID: created.InstanceID, Status: computertypes.StatusProvisioning}}
	tai := &fakeTai{}
	key := testControlKey()
	svc := taocomputer.NewService(client, computers).WithTai(tai).WithControlKey(key)
	if _, err := svc.Get(context.Background(), testTAOKey, created.InstanceID); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if tai.target.PrivateKey != key.PrivateKey {
		t.Fatal("target did not use Tai control key")
	}
}

func TestGetStatusMapping(t *testing.T) {
	cases := []struct {
		tao      string
		expected string
	}{
		{"pending", computertypes.StatusProvisioning},
		{"running", computertypes.StatusPending},
		{"stopped", computertypes.StatusSuspended},
		{"error", computertypes.StatusError},
		{"deleted", computertypes.StatusReleased},
	}
	for _, tc := range cases {
		t.Run(tc.tao, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(taocomputer.Instance{InstanceID: "inst-x", Status: tc.tao, Region: "bj-cn"})
			}))
			defer srv.Close()
			computers := &fakeComputerService{computer: &computertypes.Computer{ID: "cmp_x", NodeID: "node_x", InstanceID: "inst-x", Status: computertypes.StatusProvisioning}}
			tai := &fakeTai{}
			svc := newTestService(taocomputer.NewClient(srv.URL), computers).WithTai(tai)
			result, err := svc.Get(context.Background(), testTAOKey, "inst-x")
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			if result.Computer.Status != tc.expected {
				t.Fatalf("status=%s want %s", result.Computer.Status, tc.expected)
			}
		})
	}
}

func TestListServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	svc := newTestService(taocomputer.NewClient(srv.URL), &fakeComputerService{})
	if _, err := svc.List(context.Background(), testTAOKey, taocomputer.ListOptions{}); err == nil {
		t.Fatal("expected list error")
	}
}

func TestGetUnknownStatusDefault(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(taocomputer.Instance{InstanceID: "inst-x", Status: "unknown"})
	}))
	defer srv.Close()
	computers := &fakeComputerService{computer: &computertypes.Computer{ID: "cmp_x", InstanceID: "inst-x", Status: computertypes.StatusProvisioning}}
	svc := newTestService(taocomputer.NewClient(srv.URL), computers)
	result, err := svc.Get(context.Background(), testTAOKey, "inst-x")
	if err != nil || result.Computer.Status != computertypes.StatusProvisioning {
		t.Fatalf("status=%v err=%v", result, err)
	}
}

func TestOrchestrateDelegatesToGet(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	client := taocomputer.NewClient(srv.URL())
	created, err := client.Create(context.Background(), testTAOKey, validCreate("flex-2c2g"))
	if err != nil {
		t.Fatal(err)
	}
	computers := &fakeComputerService{computer: &computertypes.Computer{ID: "cmp_test", NodeID: "node_test", InstanceID: created.InstanceID, Status: computertypes.StatusProvisioning}}
	svc := newTestService(client, computers).WithTai(&fakeTai{})
	result, err := svc.Orchestrate(context.Background(), testTAOKey, created.InstanceID)
	if err != nil || result.Computer.Status != computertypes.StatusPending {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestGetDeletedMarksComputerReleased(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	client := taocomputer.NewClient(srv.URL())
	created, err := client.Create(context.Background(), testTAOKey, validCreate("flex-2c2g"))
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodDelete, srv.URL()+"/v1/computers/"+created.InstanceID, nil)
	req.Header.Set("Authorization", "Bearer "+srv.Token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	computers := &fakeComputerService{computer: &computertypes.Computer{ID: "cmp_test", NodeID: "node_test", InstanceID: created.InstanceID, Status: computertypes.StatusPending}}
	svc := newTestService(client, computers)
	if _, err := svc.Get(context.Background(), testTAOKey, created.InstanceID); !errors.Is(err, taocomputer.ErrInstanceDeleted) {
		t.Fatalf("err=%v", err)
	}
	if computers.computer.Status != computertypes.StatusReleased {
		t.Fatalf("status=%s", computers.computer.Status)
	}
}

func TestPurchaseMonthRequest(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	computers := &fakeComputerService{}
	svc := newTestService(taocomputer.NewClient(srv.URL()), computers)
	result, err := svc.Purchase(context.Background(), testTAOKey, taocomputer.PurchaseRequest{
		Resource: "starter-2c2g", Region: "bj-cn", Name: "month", BillingUnit: "MONTH", PeriodMonths: 3,
		RequestParams:  map[string]any{"image": "tao-base-1.0"},
		IdempotencyKey: "purchase-test-1",
	})
	if err != nil {
		t.Fatalf("purchase month: %v", err)
	}
	if result.Instance.BillingUnit != "MONTH" || result.Instance.PeriodMonths != 3 {
		t.Fatalf("instance=%+v", result.Instance)
	}
}

func TestServiceContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	svc := newTestService(nil, nil)
	if _, err := svc.List(ctx, testTAOKey, taocomputer.ListOptions{}); err == nil {
		t.Fatal("list should fail")
	}
	if _, err := svc.Get(ctx, testTAOKey, "x"); err == nil {
		t.Fatal("get should fail")
	}
	if _, err := svc.Purchase(ctx, testTAOKey, taocomputer.PurchaseRequest{}); err == nil {
		t.Fatal("purchase should fail")
	}
}

func TestServiceActionAndDelete(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	client := taocomputer.NewClient(srv.URL())
	created, err := client.Create(context.Background(), testTAOKey, validCreate("flex-2c2g"))
	if err != nil {
		t.Fatal(err)
	}
	computers := &fakeComputerService{computer: &computertypes.Computer{ID: "cmp_test", NodeID: "node_test", InstanceID: created.InstanceID, Status: computertypes.StatusProvisioning}}
	svc := taocomputer.NewService(client, computers).WithControlKey(testControlKey())
	srv.SetInstanceStatus(created.InstanceID, "running", "")
	if _, err := svc.Action(context.Background(), testTAOKey, created.InstanceID, "start"); err != nil {
		t.Fatal(err)
	}
	if computers.calls[len(computers.calls)-1].Status != computertypes.StatusPending {
		t.Fatalf("start status=%+v", computers.calls[len(computers.calls)-1])
	}
	if _, err := svc.Action(context.Background(), testTAOKey, created.InstanceID, "stop"); err != nil {
		t.Fatal(err)
	}
	if computers.calls[len(computers.calls)-1].Status != computertypes.StatusSuspended {
		t.Fatalf("stop status=%+v", computers.calls[len(computers.calls)-1])
	}
	if _, err := svc.Action(context.Background(), testTAOKey, created.InstanceID, "bad"); err == nil {
		t.Fatal("expected invalid action")
	}
	if _, err := svc.Delete(context.Background(), testTAOKey, created.InstanceID); err != nil {
		t.Fatal(err)
	}
	if computers.calls[len(computers.calls)-1].Status != computertypes.StatusReleased {
		t.Fatalf("delete status=%+v", computers.calls[len(computers.calls)-1])
	}
}

type errorComputerService struct{ err error }

func (e *errorComputerService) CreateOrUpdate(context.Context, computertypes.CreateOrUpdateComputerRequest) (*computertypes.UpsertResult, error) {
	return nil, e.err
}
func (e *errorComputerService) Get(context.Context, string) (*computertypes.Computer, error) {
	return nil, e.err
}

func TestActionDeleteErrorBranches(t *testing.T) {
	ctx := context.Background()
	svc := taocomputer.NewService(nil, nil)
	if _, err := svc.Action(ctx, testTAOKey, "x", "start"); err == nil {
		t.Fatal("expected config error")
	}
	if _, err := svc.Delete(ctx, testTAOKey, "x"); err == nil {
		t.Fatal("expected config error")
	}
	if _, err := svc.Action(ctx, testTAOKey, "x", "bad"); err == nil {
		t.Fatal("expected action error")
	}

	srv := taocloud.New()
	t.Cleanup(srv.Close)
	client := taocomputer.NewClient(srv.URL())
	created, err := client.Create(ctx, testTAOKey, validCreate("flex-2c2g"))
	if err != nil {
		t.Fatal(err)
	}
	computers := &fakeComputerService{computer: &computertypes.Computer{ID: "cmp", NodeID: "node", InstanceID: created.InstanceID}}
	svc = taocomputer.NewService(client, computers).WithControlKey(testControlKey())
	if _, err := svc.Action(ctx, testTAOKey, created.InstanceID, "bad"); !errors.Is(err, taocomputer.ErrInvalidAction) {
		t.Fatalf("bad action err=%v", err)
	}
	srv.FailNext(http.MethodPost, "/start", http.StatusInternalServerError, "boom")
	if _, err := svc.Action(ctx, testTAOKey, created.InstanceID, "start"); err == nil {
		t.Fatal("expected client error")
	}
	srv.FailNext(http.MethodDelete, "/computers/", http.StatusInternalServerError, "boom")
	if _, err := svc.Delete(ctx, testTAOKey, created.InstanceID); err == nil {
		t.Fatal("expected delete error")
	}

	badSvc := taocomputer.NewService(client, &errorComputerService{err: errors.New("db")}).WithControlKey(testControlKey())
	srv.SetInstanceStatus(created.InstanceID, "running", "")
	if _, err := badSvc.Action(ctx, testTAOKey, created.InstanceID, "stop"); err == nil {
		t.Fatal("expected computer error")
	}
	if _, err := badSvc.Delete(ctx, testTAOKey, created.InstanceID); err == nil {
		t.Fatal("expected computer delete error")
	}
}

func TestPurchaseRequiresControlKey(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	computers := &fakeComputerService{}
	svc := taocomputer.NewService(taocomputer.NewClient(srv.URL()), computers)
	if _, err := svc.Purchase(context.Background(), testTAOKey, taocomputer.PurchaseRequest{Resource: "flex-2c2g"}); !errors.Is(err, taocomputer.ErrTaiControlKey) {
		t.Fatalf("err=%v", err)
	}
}

func TestGetRequiresControlKey(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	client := taocomputer.NewClient(srv.URL())
	created, err := client.Create(context.Background(), testTAOKey, validCreate("flex-2c2g"))
	if err != nil {
		t.Fatal(err)
	}
	computers := &fakeComputerService{computer: &computertypes.Computer{ID: "cmp", NodeID: "node", InstanceID: created.InstanceID}}
	svc := taocomputer.NewService(client, computers).WithTai(&fakeTai{})
	if _, err := svc.Get(context.Background(), testTAOKey, created.InstanceID); !errors.Is(err, taocomputer.ErrTaiStart) {
		t.Fatalf("err=%v", err)
	}
}

func TestPurchaseInjectsControlPublicKey(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	computers := &fakeComputerService{}
	key := testControlKey()
	svc := taocomputer.NewService(taocomputer.NewClient(srv.URL()), computers).WithControlKey(key)
	if _, err := svc.Purchase(context.Background(), testTAOKey, taocomputer.PurchaseRequest{
		Resource: "flex-2c2g", Region: "bj-cn", Name: "host", BillingUnit: "HOUR",
		RequestParams: map[string]any{"image": "tao-base-1.0"},
	}); err != nil {
		t.Fatal(err)
	}
	requests := srv.Requests()
	if len(requests) == 0 {
		t.Fatal("no requests")
	}
	var body struct {
		SSH struct {
			PublicKey string `json:"public_key"`
		} `json:"ssh"`
	}
	if err := json.Unmarshal(requests[0].Body, &body); err != nil {
		t.Fatal(err)
	}
	if body.SSH.PublicKey != key.PublicKey {
		t.Fatalf("public key not injected")
	}
	if computers.calls[0].Metadata["tai_control_key_id"] != key.KeyID {
		t.Fatalf("metadata=%v", computers.calls[0].Metadata)
	}
}

func TestGetBootstrapFailureMarksError(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	client := taocomputer.NewClient(srv.URL())
	created, err := client.Create(context.Background(), testTAOKey, validCreate("flex-2c2g"))
	if err != nil {
		t.Fatal(err)
	}
	computers := &fakeComputerService{computer: &computertypes.Computer{ID: "cmp_boot", NodeID: "node_boot", InstanceID: created.InstanceID, Status: computertypes.StatusProvisioning}}
	tai := &fakeTai{bootstrapErr: errors.New("tai missing")}
	svc := taocomputer.NewService(client, computers).WithTai(tai).WithControlKey(testControlKey())
	if _, err := svc.Get(context.Background(), testTAOKey, created.InstanceID); !errors.Is(err, taocomputer.ErrTaiStart) {
		t.Fatalf("err=%v", err)
	}
	if computers.calls[len(computers.calls)-1].Status != computertypes.StatusError {
		t.Fatalf("last=%+v", computers.calls[len(computers.calls)-1])
	}
}

func TestControlKeyRingSelectsComputerKey(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	client := taocomputer.NewClient(srv.URL())
	created, err := client.Create(context.Background(), testTAOKey, validCreate("flex-2c2g"))
	if err != nil {
		t.Fatal(err)
	}
	oldKey := testControlKey()
	oldKey.KeyID = "old-control"
	newKey := testControlKey()
	newKey.KeyID = "new-control"
	computers := &fakeComputerService{computer: &computertypes.Computer{
		ID: "cmp_ring", NodeID: "node_ring", InstanceID: created.InstanceID,
		Status: computertypes.StatusProvisioning, Metadata: map[string]interface{}{"tai_control_key_id": "old-control"},
	}}
	tai := &fakeTai{}
	svc := taocomputer.NewService(client, computers).WithTai(tai).WithControlKeys("new-control", oldKey, newKey)
	if _, err := svc.Get(context.Background(), testTAOKey, created.InstanceID); err != nil {
		t.Fatal(err)
	}
	if tai.target.PrivateKey != oldKey.PrivateKey {
		t.Fatal("existing computer did not keep old control key")
	}
	if svc == nil {
		t.Fatal("service nil")
	}
}
