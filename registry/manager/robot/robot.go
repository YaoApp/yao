// Package robot implements the Robot package manager for the Yao registry.
package robot

import (
	"github.com/yaoapp/yao/registry"
	agentmgr "github.com/yaoapp/yao/registry/manager/agent"
	"github.com/yaoapp/yao/registry/manager/common"
	mcpmgr "github.com/yaoapp/yao/registry/manager/mcp"
)

// Manager handles robot package operations (add only for P0).
type Manager struct {
	client   *registry.Client
	appRoot  string
	prompter common.Prompter
	agentMgr *agentmgr.Manager
	mcpMgr   *mcpmgr.Manager
}

// New creates a Robot Manager.
func New(client *registry.Client, appRoot string, prompter common.Prompter) *Manager {
	if prompter == nil {
		prompter = &common.StdinPrompter{}
	}
	return &Manager{
		client:   client,
		appRoot:  appRoot,
		prompter: prompter,
		agentMgr: agentmgr.New(client, appRoot, prompter),
		mcpMgr:   mcpmgr.New(client, appRoot, prompter),
	}
}
