package workflow

import "github.com/yaoapp/xiang/share"

// WorkFlow 工作流配置结构
type WorkFlow struct {
	Name       string `json:"-"`
	Source     string `json:"-"`
	Version    string `json:"version"`
	Label      string `json:"label,omitempty"`
	Decription string `json:"decription,omitempty"`
	Nodes      []Node `json:"nodes"`
	APIs       []API  `json:"apis"`
}

// Node 工作流节点
type Node struct {
	Name    string       `json:"name"`
	Body    share.Render `json:"body,omitempty"`
	Actions []string     `json:"actions,omitempty"`
	User    User         `json:"user,omitempty"`
}

// User 工作相关用户读取条件
type User struct {
	Process string        `json:"process"`
	Args    []interface{} `json:"args"`
}

// API 工作相关API
type API struct {
	Name    string        `json:"name"`
	Process string        `json:"process"`
	Args    []interface{} `json:"args"`
}
