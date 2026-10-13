package context

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// NewStack creates a new root stack
func NewStack(assistantID, referer string, opts *Options) *Stack {

	stackID := uuid.New().String()
	now := time.Now().UnixMilli()

	return &Stack{
		ID:          stackID,
		AssistantID: assistantID,
		Referer:     referer,
		Depth:       0,
		ParentID:    "",
		Path:        []string{stackID},
		Options:     opts,
		CreatedAt:   now,
		Status:      StackStatusRunning,
	}
}

// NewChildStack creates a child stack from the current stack
func (s *Stack) NewChildStack(assistantID, referer string, opts *Options) *Stack {
	stackID := uuid.New().String()
	now := time.Now().UnixMilli()

	// Build path by appending current stack's path with new ID
	path := make([]string, len(s.Path)+1)
	copy(path, s.Path)
	path[len(s.Path)] = stackID

	return &Stack{
		ID:          stackID,
		AssistantID: assistantID,
		Referer:     referer,
		Depth:       s.Depth + 1,
		ParentID:    s.ID,
		Path:        path,
		Options:     opts,
		CreatedAt:   now,
		Status:      StackStatusRunning,
	}
}

// NewChildStackFromForkParent creates a child stack from ForkParentInfo
// This is used by forked contexts (ctx.agent.Call) to create a child stack
// without sharing the actual Stack reference (which would cause race conditions)
func NewChildStackFromForkParent(parent *ForkParentInfo, assistantID, referer string, opts *Options) *Stack {
	stackID := uuid.New().String()
	now := time.Now().UnixMilli()

	// Build path by appending parent's path with new ID
	path := make([]string, len(parent.Path)+1)
	copy(path, parent.Path)
	path[len(parent.Path)] = stackID

	return &Stack{
		ID:          stackID,
		AssistantID: assistantID,
		Referer:     referer,
		Depth:       parent.Depth + 1,
		ParentID:    parent.StackID, // Use parent's stack ID
		Path:        path,
		Options:     opts,
		CreatedAt:   now,
		Status:      StackStatusRunning,
	}
}

// Complete marks the stack as completed and calculates duration
func (s *Stack) Complete() {
	now := time.Now().UnixMilli()
	s.CompletedAt = &now
	s.Status = StackStatusCompleted
	duration := now - s.CreatedAt
	s.DurationMs = &duration
}

// Fail marks the stack as failed with an error message
func (s *Stack) Fail(err error) {
	now := time.Now().UnixMilli()
	s.CompletedAt = &now
	s.Status = StackStatusFailed
	if err != nil {
		s.Error = err.Error()
	}
	duration := now - s.CreatedAt
	s.DurationMs = &duration
}

// Timeout marks the stack as timeout
func (s *Stack) Timeout() {
	now := time.Now().UnixMilli()
	s.CompletedAt = &now
	s.Status = StackStatusTimeout
	duration := now - s.CreatedAt
	s.DurationMs = &duration
}

// IsRoot returns true if this is a root stack (no parent)
func (s *Stack) IsRoot() bool {
	return s.ParentID == ""
}

// IsCompleted returns true if the stack has completed (success, failed, or timeout)
func (s *Stack) IsCompleted() bool {
	return s.Status == StackStatusCompleted ||
		s.Status == StackStatusFailed ||
		s.Status == StackStatusTimeout
}

// IsRunning returns true if the stack is currently running
func (s *Stack) IsRunning() bool {
	return s.Status == StackStatusRunning
}

// GetPathString returns the path as a string (e.g., "root_id -> parent_id -> current_id")
func (s *Stack) GetPathString() string {
	if len(s.Path) == 0 {
		return s.ID
	}

	result := s.Path[0]
	for i := 1; i < len(s.Path); i++ {
		result += " -> " + s.Path[i]
	}
	return result
}

// String returns a string representation of the stack for debugging
func (s *Stack) String() string {
	status := s.Status
	if s.IsCompleted() && s.DurationMs != nil {
		status = fmt.Sprintf("%s (%dms)", s.Status, *s.DurationMs)
	}

	return fmt.Sprintf("Stack[ID=%s, Assistant=%s, Depth=%d, Status=%s]",
		s.ID[:8], s.AssistantID, s.Depth, status)
}

// Clone creates a deep copy of the stack
func (s *Stack) Clone() *Stack {
	clone := &Stack{
		ID:          s.ID,
		AssistantID: s.AssistantID,
		Referer:     s.Referer,
		Depth:       s.Depth,
		ParentID:    s.ParentID,
		Path:        make([]string, len(s.Path)),
		Options:     s.Options, // Shallow copy of Options pointer
		CreatedAt:   s.CreatedAt,
		Status:      s.Status,
		Error:       s.Error,
	}

	copy(clone.Path, s.Path)

	if s.CompletedAt != nil {
		completedAt := *s.CompletedAt
		clone.CompletedAt = &completedAt
	}

	if s.DurationMs != nil {
		durationMs := *s.DurationMs
		clone.DurationMs = &durationMs
	}

	return clone
}

// EnterStack initializes or creates a child stack and returns it along with a completion function
// This is a helper function to manage stack context for nested calls
//
// Returns:
//   - *Stack: current stack
//   - func(): completion function to be deferred
//
// Usage:
//
//	stack, done := context.EnterStack(ctx, assistantID, opts)
//	defer done()
//	// ... your code here ...
func EnterStack(ctx *Context, assistantID string, opts *Options) (*Stack, func()) {
	var stack *Stack
	var parentStack *Stack

	// Get referer from ctx (request source)
	referer := ctx.Referer

	// Initialize Stacks map if not exists
	if ctx.Stacks == nil {
		ctx.Stacks = make(map[string]*Stack)
	}

	if ctx.Stack == nil {
		// Check if this is a forked context with parent stack info
		if ctx.ForkParent != nil {
			// Create child stack using ForkParent info
			// This is for forked contexts (ctx.agent.Call) to have proper ThreadID
			stack = NewChildStackFromForkParent(ctx.ForkParent, assistantID, referer, opts)
			ctx.Stack = stack
		} else {
			// Create root stack for this assistant call (entry point)
			stack = NewStack(assistantID, referer, opts)
			ctx.Stack = stack
		}
	} else {
		// Create child stack for nested agent call (delegate)
		parentStack = ctx.Stack
		stack = ctx.Stack.NewChildStack(assistantID, referer, opts)
		ctx.Stack = stack
	}

	// Mark stack as running (in case it was pending)
	if stack.Status == StackStatusPending {
		stack.Status = StackStatusRunning
	}

	// Save stack to collection
	ctx.Stacks[stack.ID] = stack

	// Return completion function
	done := func() {
		// Mark as completed if no panic occurred
		if !stack.IsCompleted() {
			stack.Complete()
		}

		// Restore parent stack
		if parentStack != nil {
			ctx.Stack = parentStack
		}
	}

	return stack, done
}

// GetAllStacks returns all stacks collected during the request
// This is useful for trace logging after the request completes
func (ctx *Context) GetAllStacks() []*Stack {
	if ctx.Stacks == nil {
		return nil
	}

	stacks := make([]*Stack, 0, len(ctx.Stacks))
	for _, s := range ctx.Stacks {
		stacks = append(stacks, s)
	}
	return stacks
}

// GetStackByID returns a specific stack by its ID
// This is useful for querying stack information during request processing
func (ctx *Context) GetStackByID(id string) *Stack {
	if ctx.Stacks == nil {
		return nil
	}
	return ctx.Stacks[id]
}

// GetRootStack returns the root stack (depth = 0)
func (ctx *Context) GetRootStack() *Stack {
	if ctx.Stacks == nil {
		return nil
	}

	for _, s := range ctx.Stacks {
		if s.IsRoot() {
			return s
		}
	}
	return nil
}
