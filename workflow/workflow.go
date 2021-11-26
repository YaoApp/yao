package workflow

import "github.com/yaoapp/xiang/config"

// WorkFlows 工作流列表
var WorkFlows = map[string]*WorkFlow{}

// Load 加载数据表格
func Load(cfg config.Config) {
	LoadFrom(cfg.RootWorkFlow, "")
}

// LoadFrom 从特定目录加载
func LoadFrom(dir string, prefix string) {}

// Select 读取已加载图表
func Select(name string) *WorkFlow {
	return WorkFlows[name]
}

// Process
// 读取工作流 xiang.workflow.Get(uid, name, data_id)
// 保存工作流 xiang.workflow.Save(uid, name, node, input)
// 进入下一个节点 xiang.workflow.Next(uid, id, input)
// 跳转到指定节点 xiang.workflow.Goto(uid, id, node, input)

// API:
// 读取工作流 GET /api/xiang/workflow/<工作流名称>/get
// 读取工作流配置 GET /api/xiang/workflow/<工作流名称>/setting
// 调用自定义API POST /api/xiang/workflow/<工作流名称>/<自定义API路由>

// Setting 返回配置信息
func (workflow *WorkFlow) Setting(id int) {}

// SetupAPIs 注册API
func (workflow *WorkFlow) SetupAPIs(id int) {}

// Get 读取当前工作流(未完成的)
func (workflow *WorkFlow) Get(uid int, name string, dataID interface{}) {}

// Save 保存工作流节点数据
func (workflow *WorkFlow) Save(uid int, name string, node string, dataID interface{}, input map[string]interface{}) {
}

// Next 下一个工作流
func (workflow *WorkFlow) Next(uid int, id int, input map[string]interface{}) {}

// Goto 工作流跳转
func (workflow *WorkFlow) Goto(uid int, id int, node string, input map[string]interface{}) {}
