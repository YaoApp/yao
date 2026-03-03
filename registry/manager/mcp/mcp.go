// Package mcp implements the MCP package manager for the Yao registry.
package mcp

import (
	"github.com/yaoapp/yao/registry"
	"github.com/yaoapp/yao/registry/manager/common"
)

// Manager handles MCP package operations (add, update, push, fork).
type Manager struct {
	client   *registry.Client
	appRoot  string
	prompter common.Prompter
}

// New creates an MCP Manager.
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
