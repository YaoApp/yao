package workflow

import (
	"fmt"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/share"
	"github.com/yaoapp/xiang/xlog"
)

// WorkFlows 工作流列表
var WorkFlows = map[string]*WorkFlow{}

// Load 加载数据表格
func Load(cfg config.Config) {
	LoadFrom(cfg.RootWorkFlow, "")
}

// LoadFrom 从特定目录加载
func LoadFrom(dir string, prefix string) {
	if share.DirNotExists(dir) {
		return
	}
	share.Walk(dir, ".json", func(root, filename string) {
		name := prefix + share.SpecName(root, filename)
		content := share.ReadFile(filename)
		_, err := LoadWorkFlow(content, name)
		if err != nil {
			exception.New("%s 工作流格式错误", 400, name).Ctx(filename).Throw()
		}
	})
}

// LoadWorkFlow 载入工作流
func LoadWorkFlow(source []byte, name string) (*WorkFlow, error) {
	workflow := WorkFlow{Name: name, Source: source}
	err := jsoniter.Unmarshal(source, &workflow)
	if err != nil {
		xlog.Println(name)
		xlog.Println(err.Error())
		xlog.Println(string(source))
		return nil, err
	}
	WorkFlows[workflow.Name] = &workflow
	return WorkFlows[workflow.Name], nil
}

// Select 读取已加载图表
func Select(name string) *WorkFlow {
	workflow, has := WorkFlows[name]
	if !has {
		exception.New(
			fmt.Sprintf("工作流:%s; 尚未加载", name),
			400,
		).Throw()
	}
	return workflow
}

// Reload 重新载入工作流
func (workflow *WorkFlow) Reload() *WorkFlow {
	new, err := LoadWorkFlow(workflow.Source, workflow.Name)
	if err != nil {
		exception.New(
			fmt.Sprintf("工作流:%s; 加载失败", workflow.Name),
			400,
		).Throw()
	}
	WorkFlows[workflow.Name] = new
	return new
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
func (workflow *WorkFlow) Get(uid int, name string, id interface{}) map[string]interface{} {
	wflow := gou.Select("xiang.workflow")
	params := gou.QueryParam{
		Select: []interface{}{"*"},
		Wheres: []gou.QueryWhere{
			{Column: "name", Value: workflow.Name},
			{Column: "data_id", Value: id},
			{Column: "user_id", Value: uid},
			{Column: "status", Value: "进行中"},
		},
	}
	rows := wflow.MustGet(params)
	if len(rows) > 0 {
		return rows[0]
	}
	return map[string]interface{}{
		"name":        workflow.Name,
		"data_id":     id,
		"node_name":   name,
		"user_id":     uid,
		"status":      "进行中",
		"node_status": "进行中",
		"input":       map[string]interface{}{},
	}
}

// Save 保存工作流节点数据
func (workflow *WorkFlow) Save(uid int, name string, id interface{}, input Input) map[string]interface{} {
	wflow := gou.Select("xiang.workflow")
	params := gou.QueryParam{
		Select: []interface{}{"id", "input"},
		Wheres: []gou.QueryWhere{
			{Column: "name", Value: workflow.Name},
			{Column: "data_id", Value: id},
			{Column: "user_id", Value: uid},
			{Column: "status", Value: "进行中"},
		},
	}

	rows := wflow.MustGet(params)
	data := map[string]interface{}{
		"name":      workflow.Name,
		"data_id":   id,
		"node_name": name,
		"user_id":   uid,
	}
	if len(rows) > 0 {
		nodeInput := map[string]interface{}{}
		if history, ok := rows[0].Get("input").(map[string]interface{}); ok {
			nodeInput = history
		}
		nodeInput[name] = input
		data["id"] = rows[0].Get("id")
		data["input"] = nodeInput
	} else {

		nodeInput := map[string]interface{}{}
		nodeInput[name] = input
		data["status"] = "进行中"
		data["node_status"] = "进行中"
		data["input"] = nodeInput
	}

	id = wflow.MustSave(data)
	return wflow.MustFind(id, gou.QueryParam{})
}

// Next 下一个工作流
func (workflow *WorkFlow) Next(uid int, id int, input map[string]interface{}) {}

// Goto 工作流跳转
func (workflow *WorkFlow) Goto(uid int, id int, node string, input map[string]interface{}) {}
