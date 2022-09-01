package workflow

import (
	"fmt"
	"path/filepath"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou"
	gshare "github.com/yaoapp/gou/query/share"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/kun/str"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/helper"
	"github.com/yaoapp/yao/share"
)

// WARNING: the Workflow widget will be removed from yao engine

// Process
// 读取工作流 xiang.workflow.Find(name, workflow_id)
// 读取工作流 xiang.workflow.Open(name, uid, data_id)
// 保存工作流 xiang.workflow.Save(name, uid, node_name, data_id, input, ...output)
// 进入下一个节点 xiang.workflow.Next(name, uid, workflow_id, output)
// 跳转到指定节点 xiang.workflow.Goto(name, uid, workflow_id, node_name, output)
// 更新工作流状态 xiang.workflow.Status(name, uid, workflow_id, status_name, output)
// 标记结束流程 xiang.workflow.Done(name, uid, workflow_id, output)
// 标记关闭流程 xiang.workflow.Close(name, uid, workflow_id, output)
// 标记重置流程 xiang.workflow.Reset(name, uid, workflow_id, output)

// API:
// 读取工作流 GET /api/xiang/workflow/<工作流名称>/find/:id
// 读取工作流 GET /api/xiang/workflow/<工作流名称>/open
// 读取工作流配置 GET /api/xiang/workflow/<工作流名称>/setting
// 调用自定义API POST /api/xiang/workflow/<工作流名称>/<自定义API路由>

// WorkFlows 工作流列表
var WorkFlows = map[string]*WorkFlow{}

// Load 加载数据表格
func Load(cfg config.Config) {
	LoadFrom(filepath.Join(cfg.Root, "workflows"), "")
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
			log.With(log.F{"root": root, "filename": filename}).Error(err.Error())
		}
	})
}

// LoadWorkFlow 载入工作流
func LoadWorkFlow(source []byte, name string) (*WorkFlow, error) {
	workflow := WorkFlow{Name: name, Source: source}
	err := jsoniter.Unmarshal(source, &workflow)
	if err != nil {
		log.With(log.F{"name": name, "source": source}).Error("LoadWorkFlow: %s", err.Error())
		return nil, err
	}

	workflow.SetupAPIs()
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

// Find 读取给定ID的工作流
// uid 当前处理人ID, id 数据ID
func (workflow *WorkFlow) Find(id int) map[string]interface{} {
	wflow := gou.Select("xiang.workflow")
	res := wflow.MustFind(id, gou.QueryParam{
		Select: []interface{}{
			"data_id", "id", "input", "output", "name",
			"user_id", "users",
			"node_name", "node_status",
			"status",
			"updated_at", "created_at",
		}})
	return res
}

// Open 读取当前工作流(未完成的)
// uid 当前处理人ID, id 数据ID
func (workflow *WorkFlow) Open(uid int, id interface{}) map[string]interface{} {
	wflow := gou.Select("xiang.workflow")
	params := gou.QueryParam{
		Select: []interface{}{
			"data_id", "id", "input", "output", "name",
			"user_id", "users",
			"node_name", "node_status",
			"status",
			"updated_at", "created_at",
		},
		Wheres: []gou.QueryWhere{
			{Column: "name", Value: workflow.Name},
			{Column: "data_id", Value: id},
			{Column: "user_ids", OP: "like", Value: fmt.Sprintf("%%|%d|%%", uid)},
			{Column: "status", Value: "进行中"},
		},
		Limit: 1,
	}
	rows := wflow.MustGet(params)
	if len(rows) > 0 {
		return rows[0]
	}

	if workflow.Len() == 0 {
		exception.New("工作流没有任何节点", 400).Throw()
	}

	users := map[string]interface{}{}
	users[workflow.Nodes[0].Name] = uid
	return map[string]interface{}{
		"name":        workflow.Name,
		"data_id":     id,
		"node_name":   workflow.Nodes[0].Name,
		"user_id":     uid,
		"users":       users,
		"status":      "进行中",
		"node_status": "进行中",
		"input":       map[string]interface{}{},
		"output":      map[string]interface{}{},
	}
}

// Setting 返回配置信息
// uid 当前处理人ID, id 数据ID
func (workflow *WorkFlow) Setting(uid int, id interface{}) map[string]interface{} {
	wflow := workflow.Open(1, 1)
	data := maps.MapStr{
		"$in":     wflow["input"],
		"$input":  wflow["input"],
		"$out":    wflow["output"],
		"$outupt": wflow["output"],
		"$data":   wflow["output"],
	}

	nodes := workflow.FlowNodes(data.Dot())
	_, idx := workflow.pickNode(string(str.Of(wflow["node_name"])))
	return map[string]interface{}{
		"nodes":      nodes,
		"node":       nodes[idx],
		"node_index": idx,
		"read":       true,
		"write":      any.Of(wflow["user_id"]).CInt() == uid,
		"actions":    workflow.Actions,
		"name":       workflow.Name,
		"version":    workflow.Version,
		"label":      workflow.Label,
		"decription": workflow.Decription,
	}
}

// Save 保存工作流节点数据 此版本使用Like实现
// uid 当前处理人ID, id 数据ID
func (workflow *WorkFlow) Save(uid int, name string, id interface{}, input Input, outputs ...map[string]interface{}) map[string]interface{} {
	wflow := gou.Select("xiang.workflow")
	params := gou.QueryParam{
		Select: []interface{}{"id", "input", "output", "users"},
		Wheres: []gou.QueryWhere{
			{Column: "name", Value: workflow.Name},
			{Column: "data_id", Value: id},
			{Column: "user_ids", OP: "like", Value: fmt.Sprintf("%%|%d|%%", uid)},
			{Column: "status", Value: "进行中"},
		},
		Limit: 1,
	}

	rows := wflow.MustGet(params)
	data := map[string]interface{}{
		"name":      workflow.Name,
		"data_id":   id,
		"node_name": name,
		"user_id":   uid,
	}
	users := map[string]interface{}{}
	users[name] = uid
	output := map[string]interface{}{}
	nodeInput := map[string]interface{}{}
	nodeInput[name] = input
	if len(outputs) > 0 {
		output = outputs[0]
	}
	if len(rows) > 0 {
		data["id"] = rows[0].Get("id")
		nodeInput = workflow.MergeData(rows[0].Get("input"), nodeInput)
		users = workflow.MergeUsers(rows[0].Get("users"), users)
		output = workflow.MergeData(rows[0].Get("output"), output)
	} else {
		data["status"] = "进行中"
		data["node_status"] = "进行中"
	}
	data["users"] = users
	data["user_ids"] = workflow.UserIDs(users)
	data["input"] = nodeInput
	data["output"] = output
	id = wflow.MustSave(data)
	return wflow.MustFind(id, gou.QueryParam{})
}

// Done 标记完成
// uid 当前处理人ID, id 工作流ID
func (workflow *WorkFlow) Done(uid int, id int, output map[string]interface{}) map[string]interface{} {
	return workflow.Status(uid, id, "已完成", output)
}

// Close 标记关闭
// uid 当前处理人ID, id 工作流ID
func (workflow *WorkFlow) Close(uid int, id int, output map[string]interface{}) map[string]interface{} {
	return workflow.Status(uid, id, "已关闭", output)
}

// Status 设定状态
// uid 当前处理人ID, id 工作流ID
func (workflow *WorkFlow) Status(uid int, id int, status string, output map[string]interface{}) map[string]interface{} {
	wflow := workflow.Find(id)
	mod := gou.Select("xiang.workflow")
	output = workflow.MergeData(wflow["output"], output)
	mod.Save(map[string]interface{}{
		"id":     wflow["id"],
		"output": output,
		"status": status,
	})
	return workflow.Find(id)
}

// Reset 重新开始
// uid 当前处理人ID, id 工作流ID
func (workflow *WorkFlow) Reset(uid int, id int, output map[string]interface{}) map[string]interface{} {
	return workflow.Goto(uid, id, workflow.Nodes[0].Name, output)
}

// Goto 工作流跳转
func (workflow *WorkFlow) Goto(uid int, id int, name string, output map[string]interface{}) map[string]interface{} {
	wflow := workflow.Find(id)
	users := map[string]interface{}{}
	curr, _ := workflow.pickNode(wflow["node_name"].(string))
	users[curr.Name] = uid
	node, _ := workflow.pickNode(name)
	output = workflow.MergeData(wflow["output"], output)
	users = workflow.MergeUsers(wflow["users"], users)
	nodeUID := node.GetUID(users)
	mod := gou.Select("xiang.workflow")
	mod.Save(map[string]interface{}{
		"id":          wflow["id"],
		"output":      output,
		"node_name":   name,
		"node_status": "进行中",
		"user_id":     nodeUID,
		"users":       users,
		"user_ids":    workflow.UserIDs(users),
	})
	return workflow.Find(id)
}

// Next 下一个工作流
// uid 当前处理人ID, id 工作流ID
func (workflow *WorkFlow) Next(uid int, id int, output map[string]interface{}) map[string]interface{} {
	wflow := workflow.Find(id)
	currNode, ok := wflow["node_name"].(string)
	if !ok {
		exception.New("流程数据异常: 当前节点信息错误", 500).Ctx(currNode).Throw()
	}

	users := map[string]interface{}{}
	users[currNode] = uid
	output = workflow.MergeData(wflow["output"], output)

	// 读取下一个节点
	data := map[string]interface{}{
		"$in":     wflow["input"],
		"$input":  wflow["input"],
		"$out":    output,
		"$outupt": output,
		"$data":   output,
	}
	nextNode := workflow.nextNode(currNode, data)
	nextUID := nextNode.MakeUID()
	users[nextNode.Name] = nextUID

	// 更新数据
	users = workflow.MergeUsers(wflow["users"], users)
	mod := gou.Select("xiang.workflow")
	mod.Save(map[string]interface{}{
		"id":          wflow["id"],
		"output":      output,
		"node_name":   nextNode.Name,
		"node_status": "进行中",
		"user_id":     nextUID,
		"users":       users,
		"user_ids":    workflow.UserIDs(users),
	})
	return workflow.Find(id)
}

// GetUID 读取节点相关人
func (node *Node) GetUID(users interface{}) int {
	if users, ok := users.(map[string]interface{}); ok {
		if uid, has := users[node.Name]; has {
			return any.Of(uid).CInt()
		}
	}
	return node.MakeUID()
}

// MakeUID 根据条件选择节点处理人
func (node *Node) MakeUID() int {
	res := gou.NewProcess(node.User.Process, node.User.Args...).Run()
	return any.Of(res).CInt()
}

func (workflow *WorkFlow) pickNode(name string) (*Node, int) {
	for i, node := range workflow.Nodes {
		if node.Name == name {
			return &node, i
		}
	}
	exception.New("流程数据异常: 未找到节点 %s", 500, name).Throw()
	return nil, 0
}

// nextNode 查找下一个节点
func (workflow *WorkFlow) nextNode(currentNode string, data map[string]interface{}) *Node {
	curr, index := workflow.pickNode(currentNode)
	nextIndex := index + 1
	if nextIndex == workflow.Len() {
		exception.New("流程数据异常: 当前节点为最后一个节点", 500).Ctx(currentNode).Throw()
	}

	// 未声明 Next 节点, 转到下一个节点
	if curr.Next == nil {
		return &workflow.Nodes[nextIndex]
	}

	// 声明 Next 节点, 按条件到指定节点
	data = maps.Of(data).Dot()
	for _, next := range curr.Next {
		node := workflow.GetNodeWhen(next, data)
		if node != nil {
			return node
		}
	}

	exception.New("流程数据异常: 未找到符合条件的工作流节点", 500).Ctx(map[string]interface{}{"current": currentNode, "data": data}).Throw()
	return nil
}

// GetNodeWhen 读取节点
func (workflow *WorkFlow) GetNodeWhen(next Next, data map[string]interface{}) *Node {
	nextNode := ""
	conditions := workflow.Conditions(next.Conditions, data)
	if helper.When(conditions) {
		nextNode = next.Goto
		for i := 0; i < workflow.Len(); i++ {
			node := workflow.Nodes[i]
			if node.Name == nextNode {
				return &node
			}
		}
	}
	return nil
}

// Conditions 处理绑定参数
func (workflow *WorkFlow) Conditions(conds []helper.Condition, data map[string]interface{}) []helper.Condition {
	conditions := []helper.Condition{}
	for _, cond := range conds {
		if left, ok := cond.Left.(string); ok {
			cond.Left = gshare.Bind(left, data)
		}
		if right, ok := cond.Right.(string); ok {
			cond.Right = gshare.Bind(right, data)
		}
		conditions = append(conditions, cond)
	}
	return conditions
}

// UserIDs 读取用户ID
func (workflow *WorkFlow) UserIDs(users map[string]interface{}) string {
	userIDs := []string{}
	for _, u := range users {
		userIDs = append(userIDs, fmt.Sprintf("|%d|", u))
	}
	userIDs = helper.ArrayStringUnique(userIDs)
	return strings.Join(userIDs, ",")
}

// MergeUsers 合并数据
func (workflow *WorkFlow) MergeUsers(data interface{}, new interface{}) map[string]interface{} {
	res, ok := data.(map[string]interface{})
	if !ok {
		return map[string]interface{}{}
	}
	if new, ok := new.(map[string]interface{}); ok {
		for name, value := range new {
			res[name] = value
		}
	}
	return res
}

// MergeData 合并数据
func (workflow *WorkFlow) MergeData(data interface{}, new interface{}) map[string]interface{} {

	res, ok := data.(map[string]interface{})
	if !ok {
		return map[string]interface{}{}
	}

	if new, ok := new.(map[string]interface{}); ok {
		for key, value := range new {
			res[key] = value
		}
	}
	return res
}

// IsLastNode 检查是否为最后一个节点
func (workflow *WorkFlow) IsLastNode(name string) bool {
	length := workflow.Len()
	return workflow.Nodes[length-1].Name == name
}

// FlowNodes 转换为 flow
func (workflow *WorkFlow) FlowNodes(data map[string]interface{}) []map[string]interface{} {
	res := []map[string]interface{}{}
	nodes, err := jsoniter.Marshal(workflow.Nodes)
	if err != nil {
		exception.New("JSON解析错误 %s", 500, nodes).Throw()
	}

	err = jsoniter.Unmarshal(nodes, &res)
	if err != nil {
		exception.New("JSON解析错误 %s", 500, nodes).Throw()
	}

	nameMaps := map[string]int{}
	for i, node := range workflow.Nodes {
		nameMaps[node.Name] = i
	}

	for i, node := range res {
		v := gshare.Bind(node, data)
		if v, ok := v.(map[string]interface{}); ok {
			res[i] = v
		}
		res[i]["id"] = i + 1
		res[i]["label"] = res[i]["name"]
		delete(res[i], "user")
		delete(res[i], "next")
		if workflow.Nodes[i].Next != nil {
			for _, next := range workflow.Nodes[i].Next {
				name := next.Goto
				if id, has := nameMaps[name]; has {
					res[id]["source"] = i + 1
				}
			}
		}
	}
	return res
}

// Len 节点数量
func (workflow *WorkFlow) Len() int {
	return len(workflow.Nodes)
}

// InputOf 映射表转换为Input
func InputOf(in map[string]interface{}) Input {
	input := Input{Data: map[string]interface{}{}, Form: map[string]interface{}{}}
	if data, ok := in["data"].(map[string]interface{}); ok {
		input.Data = data
	}
	if form, ok := in["form"].(map[string]interface{}); ok {
		input.Form = form
	}
	return input
}
