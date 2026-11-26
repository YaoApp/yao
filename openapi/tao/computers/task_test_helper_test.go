package computers_test

import (
	"context"
	"errors"
	"sync"

	taocomputer "github.com/yaoapp/yao/modules/tao/computer"
)

type memTaskStore struct {
	mu    sync.Mutex
	tasks map[string]*taocomputer.CloudTask
}

func newMemTaskStore() *memTaskStore {
	return &memTaskStore{tasks: map[string]*taocomputer.CloudTask{}}
}

func (s *memTaskStore) Create(_ context.Context, task *taocomputer.CloudTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.tasks {
		if existing.DedupeKey == task.DedupeKey {
			return errors.New("duplicate dedupe key")
		}
	}
	copy := *task
	s.tasks[task.ID] = &copy
	return nil
}

func (s *memTaskStore) Get(_ context.Context, id string) (*taocomputer.CloudTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.tasks[id]
	if !ok {
		return nil, taocomputer.ErrTaskNotFound
	}
	copy := *task
	return &copy, nil
}

func (s *memTaskStore) GetByDedupe(_ context.Context, key string) (*taocomputer.CloudTask, error) {
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

func (s *memTaskStore) Update(_ context.Context, task *taocomputer.CloudTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	copy := *task
	s.tasks[task.ID] = &copy
	return nil
}

func (s *memTaskStore) ListRunnable(_ context.Context, limit int) ([]*taocomputer.CloudTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []*taocomputer.CloudTask{}
	for _, task := range s.tasks {
		if task.Status == taocomputer.TaskStatusQueued || task.Status == taocomputer.TaskStatusRunning {
			copy := *task
			out = append(out, &copy)
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (s *memTaskStore) ResetRunning(context.Context) error { return nil }

func (s *memTaskStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tasks, id)
	return nil
}
