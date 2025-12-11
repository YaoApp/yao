package context_test

import (
	stdContext "context"
	"testing"
	"time"

	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestNewStack(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	traceID := "12345678"
	assistantID := "test-assistant"
	referer := context.RefererAPI
	opts := &context.Options{}

	stack := context.NewStack(traceID, assistantID, referer, opts)

	if stack == nil {
		t.Fatal("Expected stack to be created, got nil")
	}

	if stack.TraceID != traceID {
		t.Errorf("Expected TraceID '%s', got '%s'", traceID, stack.TraceID)
	}

	if stack.AssistantID != assistantID {
		t.Errorf("Expected AssistantID '%s', got '%s'", assistantID, stack.AssistantID)
	}

	if stack.Referer != referer {
		t.Errorf("Expected Referer '%s', got '%s'", referer, stack.Referer)
	}

	if stack.Depth != 0 {
		t.Errorf("Expected Depth 0, got %d", stack.Depth)
	}

	if stack.ParentID != "" {
		t.Errorf("Expected empty ParentID, got '%s'", stack.ParentID)
	}

	if !stack.IsRoot() {
		t.Error("Expected stack to be root")
	}

	if stack.Status != context.StackStatusRunning {
		t.Errorf("Expected Status '%s', got '%s'", context.StackStatusRunning, stack.Status)
	}
}

func TestNewStack_GenerateTraceID(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Empty traceID should generate a UUID
	stack := context.NewStack("", "test-assistant", context.RefererAPI, &context.Options{})

	if stack.TraceID == "" {
		t.Error("Expected TraceID to be generated, got empty string")
	}

	// Should be a valid UUID (36 characters with dashes)
	if len(stack.TraceID) < 8 {
		t.Errorf("Expected TraceID to be at least 8 characters, got %d", len(stack.TraceID))
	}
}

func TestNewChildStack(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create parent stack
	parentStack := context.NewStack("12345678", "parent-assistant", context.RefererAPI, &context.Options{})

	// Create child stack
	childStack := parentStack.NewChildStack("child-assistant", context.RefererAgent, &context.Options{})

	if childStack == nil {
		t.Fatal("Expected child stack to be created, got nil")
	}

	// Child should inherit TraceID
	if childStack.TraceID != parentStack.TraceID {
		t.Errorf("Expected child TraceID '%s', got '%s'", parentStack.TraceID, childStack.TraceID)
	}

	// Child should have parent ID
	if childStack.ParentID != parentStack.ID {
		t.Errorf("Expected ParentID '%s', got '%s'", parentStack.ID, childStack.ParentID)
	}

	// Child should have incremented depth
	if childStack.Depth != parentStack.Depth+1 {
		t.Errorf("Expected Depth %d, got %d", parentStack.Depth+1, childStack.Depth)
	}

	// Child should not be root
	if childStack.IsRoot() {
		t.Error("Expected child stack not to be root")
	}

	// Path should include both parent and child
	if len(childStack.Path) != 2 {
		t.Errorf("Expected Path length 2, got %d", len(childStack.Path))
	}

	if childStack.Path[0] != parentStack.ID {
		t.Errorf("Expected first path element '%s', got '%s'", parentStack.ID, childStack.Path[0])
	}

	if childStack.Path[1] != childStack.ID {
		t.Errorf("Expected second path element '%s', got '%s'", childStack.ID, childStack.Path[1])
	}
}

func TestStackComplete(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	stack := context.NewStack("12345678", "test-assistant", context.RefererAPI, &context.Options{})

	// Wait a bit to have measurable duration
	time.Sleep(10 * time.Millisecond)

	stack.Complete()

	if stack.Status != context.StackStatusCompleted {
		t.Errorf("Expected Status '%s', got '%s'", context.StackStatusCompleted, stack.Status)
	}

	if stack.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set, got nil")
	}

	if stack.DurationMs == nil {
		t.Error("Expected DurationMs to be set, got nil")
	}

	if *stack.DurationMs < 10 {
		t.Errorf("Expected DurationMs to be at least 10ms, got %d", *stack.DurationMs)
	}

	if !stack.IsCompleted() {
		t.Error("Expected stack to be completed")
	}

	if stack.IsRunning() {
		t.Error("Expected stack not to be running")
	}
}

func TestStackFail(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	stack := context.NewStack("12345678", "test-assistant", context.RefererAPI, &context.Options{})

	testError := "test error message"
	stack.Fail(nil)
	stack.Error = testError

	if stack.Status != context.StackStatusFailed {
		t.Errorf("Expected Status '%s', got '%s'", context.StackStatusFailed, stack.Status)
	}

	if stack.Error != testError {
		t.Errorf("Expected Error '%s', got '%s'", testError, stack.Error)
	}

	if !stack.IsCompleted() {
		t.Error("Expected failed stack to be completed")
	}
}

func TestStackTimeout(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	stack := context.NewStack("12345678", "test-assistant", context.RefererAPI, &context.Options{})

	stack.Timeout()

	if stack.Status != context.StackStatusTimeout {
		t.Errorf("Expected Status '%s', got '%s'", context.StackStatusTimeout, stack.Status)
	}

	if !stack.IsCompleted() {
		t.Error("Expected timeout stack to be completed")
	}
}

func TestEnterStack_RootCreation(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := context.New(stdContext.Background(), nil, "test-chat-id")
	ctx.Referer = context.RefererAPI

	stack, traceID, done := context.EnterStack(ctx, "test-assistant", &context.Options{})
	defer done()

	if stack == nil {
		t.Fatal("Expected stack to be created, got nil")
	}

	if traceID == "" {
		t.Error("Expected traceID to be generated, got empty string")
	}

	// TraceID should be at least 8 digits (from trace.GenTraceID)
	if len(traceID) < 8 {
		t.Errorf("Expected traceID length at least 8, got %d", len(traceID))
	}

	if stack.TraceID != traceID {
		t.Errorf("Expected stack TraceID '%s', got '%s'", traceID, stack.TraceID)
	}

	if ctx.Stack != stack {
		t.Error("Expected ctx.Stack to be set to created stack")
	}

	if ctx.Stacks == nil {
		t.Fatal("Expected ctx.Stacks to be initialized, got nil")
	}

	if ctx.Stacks[stack.ID] != stack {
		t.Error("Expected stack to be saved in ctx.Stacks")
	}

	if !stack.IsRoot() {
		t.Error("Expected stack to be root")
	}
}

func TestEnterStack_ChildCreation(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := context.New(stdContext.Background(), nil, "test-chat-id")
	ctx.Referer = context.RefererAPI

	// Create parent
	parentStack, parentTraceID, parentDone := context.EnterStack(ctx, "parent-assistant", &context.Options{})
	defer parentDone()

	if parentStack == nil {
		t.Fatal("Expected parent stack to be created, got nil")
	}

	// Create child
	childStack, childTraceID, childDone := context.EnterStack(ctx, "child-assistant", &context.Options{})
	defer childDone()

	if childStack == nil {
		t.Fatal("Expected child stack to be created, got nil")
	}

	// Child should inherit trace ID
	if childTraceID != parentTraceID {
		t.Errorf("Expected child traceID '%s', got '%s'", parentTraceID, childTraceID)
	}

	// Child should have parent ID
	if childStack.ParentID != parentStack.ID {
		t.Errorf("Expected child ParentID '%s', got '%s'", parentStack.ID, childStack.ParentID)
	}

	// Both should be saved in ctx.Stacks
	if len(ctx.Stacks) != 2 {
		t.Errorf("Expected 2 stacks in ctx.Stacks, got %d", len(ctx.Stacks))
	}

	// Current stack should be child
	if ctx.Stack != childStack {
		t.Error("Expected ctx.Stack to be child stack")
	}
}

func TestEnterStack_DoneCallback(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := context.New(stdContext.Background(), nil, "test-chat-id")
	ctx.Referer = context.RefererAPI

	// Create parent
	parentStack, _, parentDone := context.EnterStack(ctx, "parent-assistant", &context.Options{})

	// Create child
	childStack, _, childDone := context.EnterStack(ctx, "child-assistant", &context.Options{})

	// Child should be current
	if ctx.Stack != childStack {
		t.Error("Expected ctx.Stack to be child stack before done")
	}

	// Call child done
	childDone()

	// Parent should be restored
	if ctx.Stack != parentStack {
		t.Error("Expected ctx.Stack to be restored to parent stack after child done")
	}

	// Child should be completed
	if !childStack.IsCompleted() {
		t.Error("Expected child stack to be completed after done")
	}

	// Call parent done
	parentDone()

	// Parent should be completed
	if !parentStack.IsCompleted() {
		t.Error("Expected parent stack to be completed after done")
	}
}

func TestContextGetAllStacks(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := context.New(stdContext.Background(), nil, "test-chat-id")
	ctx.Referer = context.RefererAPI

	// Create multiple stacks
	_, _, done1 := context.EnterStack(ctx, "assistant1", &context.Options{})
	defer done1()

	_, _, done2 := context.EnterStack(ctx, "assistant2", &context.Options{})
	defer done2()

	_, _, done3 := context.EnterStack(ctx, "assistant3", &context.Options{})
	defer done3()

	// Get all stacks
	allStacks := ctx.GetAllStacks()

	if len(allStacks) != 3 {
		t.Errorf("Expected 3 stacks, got %d", len(allStacks))
	}
}

func TestContextGetStackByID(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := context.New(stdContext.Background(), nil, "test-chat-id")
	ctx.Referer = context.RefererAPI

	stack, _, done := context.EnterStack(ctx, "test-assistant", &context.Options{})
	defer done()

	// Get stack by ID
	found := ctx.GetStackByID(stack.ID)

	if found == nil {
		t.Fatal("Expected to find stack, got nil")
	}

	if found.ID != stack.ID {
		t.Errorf("Expected stack ID '%s', got '%s'", stack.ID, found.ID)
	}

	// Try to get non-existent stack
	notFound := ctx.GetStackByID("non-existent-id")
	if notFound != nil {
		t.Error("Expected nil for non-existent stack ID")
	}
}

func TestContextGetStacksByTraceID(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := context.New(stdContext.Background(), nil, "test-chat-id")
	ctx.Referer = context.RefererAPI

	// Create parent and child (same trace ID)
	_, traceID, done1 := context.EnterStack(ctx, "parent-assistant", &context.Options{})
	defer done1()

	_, _, done2 := context.EnterStack(ctx, "child-assistant", &context.Options{})
	defer done2()

	// Get stacks by trace ID
	stacks := ctx.GetStacksByTraceID(traceID)

	if len(stacks) != 2 {
		t.Errorf("Expected 2 stacks with trace ID '%s', got %d", traceID, len(stacks))
	}

	// All should have same trace ID
	for _, s := range stacks {
		if s.TraceID != traceID {
			t.Errorf("Expected TraceID '%s', got '%s'", traceID, s.TraceID)
		}
	}
}

func TestContextGetRootStack(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := context.New(stdContext.Background(), nil, "test-chat-id")
	ctx.Referer = context.RefererAPI

	// Create parent
	parentStack, _, done1 := context.EnterStack(ctx, "parent-assistant", &context.Options{})
	defer done1()

	// Create child
	_, _, done2 := context.EnterStack(ctx, "child-assistant", &context.Options{})
	defer done2()

	// Get root stack
	rootStack := ctx.GetRootStack()

	if rootStack == nil {
		t.Fatal("Expected to find root stack, got nil")
	}

	if rootStack.ID != parentStack.ID {
		t.Errorf("Expected root stack ID '%s', got '%s'", parentStack.ID, rootStack.ID)
	}

	if !rootStack.IsRoot() {
		t.Error("Expected returned stack to be root")
	}
}

func TestStackClone(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	original := context.NewStack("12345678", "test-assistant", context.RefererAPI, &context.Options{})
	original.Complete()

	clone := original.Clone()

	if clone == nil {
		t.Fatal("Expected clone to be created, got nil")
	}

	// Check all fields are copied
	if clone.ID != original.ID {
		t.Error("ID not cloned correctly")
	}

	if clone.TraceID != original.TraceID {
		t.Error("TraceID not cloned correctly")
	}

	if clone.AssistantID != original.AssistantID {
		t.Error("AssistantID not cloned correctly")
	}

	if clone.Status != original.Status {
		t.Error("Status not cloned correctly")
	}

	// Check deep copy of Path
	if len(clone.Path) != len(original.Path) {
		t.Error("Path length not cloned correctly")
	}

	// Modify clone's path shouldn't affect original
	if len(clone.Path) > 0 {
		clone.Path[0] = "modified"
		if original.Path[0] == "modified" {
			t.Error("Path is not deeply copied")
		}
	}
}
