package types

import "context"

// Job interface
type Job interface {
	Run(ctx context.Context) error
	AddTask(ctx context.Context, task Task) error
}

// Task interface
type Task func(ctx context.Context, job Job) error
