package workflow

import "github.com/yaoapp/xiang/share"

// WorkFlow 工作流配置结构
type WorkFlow struct {
	Name       string                  `json:"-"`
	Source     []byte                  `json:"-"`
	Version    string                  `json:"version"`
	Label      string                  `json:"label,omitempty"`
	Decription string                  `json:"decription,omitempty"`
	Nodes      []Node                  `json:"nodes"`
	APIs       map[string]API          `json:"apis"`
	Actions    map[string]share.Render `json:"actions"`
}

// Node 工作流节点
type Node struct {
	Name    string       `json:"name"`
	Body    share.Render `json:"body,omitempty"`
	Actions []string     `json:"actions,omitempty"`
	User    User         `json:"user,omitempty"`
}

// User 工作流相关用户读取条件
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

// Input 用户输入数据
type Input struct {
	Data map[string]interface{} `json:"data"` // 记录数据
	Form map[string]interface{} `json:"form"` // 表单数据
}
