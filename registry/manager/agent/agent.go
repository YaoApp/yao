// Package agent implements the assistant package manager for the Yao registry.
package agent

import (
	"github.com/yaoapp/yao/registry"
	"github.com/yaoapp/yao/registry/manager/common"
)

// Manager handles assistant package operations (add, update, push, fork).
type Manager struct {
	client   *registry.Client
	appRoot  string
	prompter common.Prompter
}

// New creates an agent Manager.
func New(client *registry.Client, appRoot string, prompter common.Prompter) *Manager {
	if prompter == nil {
		prompter = &common.StdinPrompter{}
	}
	return &Manager{
		client:   client,
		appRoot:  appRoot,
		prompter: prompter,
	}
}
