package workflow

// WARNING: the Workflow widget will be removed from yao engine

// func init() {
// 	share.DBConnect(config.Conf.DB)
// 	share.Load(config.Conf)
// 	model.Load(config.Conf)
// 	engineModels := path.Join(os.Getenv("YAO_DEV"), "xiang", "models")
// 	model.LoadFrom(engineModels, "xiang.")
// 	query.Load(config.Conf)
// 	flow.Load(config.Conf)
// 	Load(config.Conf)
// }

// func TestLoad(t *testing.T) {
// 	share.DBConnect(config.Conf.DB)
// 	share.Load(config.Conf)
// 	model.Load(config.Conf)
// 	query.Load(config.Conf)
// 	Load(config.Conf)
// 	LoadFrom("not a path", "404.")
// 	check(t)
// }

// func TestSave(t *testing.T) {
// 	assignFlow := Select("assign")
// 	wflow := assignFlow.Save(1, "选择商务负责人", 1, Input{
// 		Data: map[string]interface{}{"id": 1, "name": "云主机"},
// 		Form: map[string]interface{}{"biz_id": 1, "name": "张良明"},
// 	})

// 	data := maps.Of(wflow).Dot()
// 	assert.Equal(t, int64(1), data.Get("id"))
// 	assert.Equal(t, "assign", data.Get("name"))
// 	assert.Equal(t, "选择商务负责人", data.Get("node_name"))
// 	assert.Equal(t, "进行中", data.Get("node_status"))
// 	assert.Equal(t, "进行中", data.Get("status"))
// 	assert.Equal(t, float64(1), data.Get("input.选择商务负责人.data.id"))
// 	assert.Equal(t, "云主机", data.Get("input.选择商务负责人.data.name"))
// 	assert.Equal(t, float64(1), data.Get("input.选择商务负责人.form.biz_id"))
// 	assert.Equal(t, "张良明", data.Get("input.选择商务负责人.form.name"))
// 	// 清理数据
// 	capsule.Query().From("xiang_workflow").Truncate()
// }

// func TestSaveUpdate(t *testing.T) {
// 	assignFlow := Select("assign")
// 	wflow := assignFlow.Save(1, "选择商务负责人", 1, Input{
// 		Data: map[string]interface{}{"id": 1, "name": "云主机"},
// 		Form: map[string]interface{}{"biz_id": 1, "name": "张良明"},
// 	})

// 	wflow = assignFlow.Save(1, "选择商务负责人", 1, Input{
// 		Data: map[string]interface{}{"id": 1, "name": "云存储"},
// 		Form: map[string]interface{}{"biz_id": 1, "name": "李明博"},
// 	})

// 	data := maps.Of(wflow).Dot()
// 	assert.Equal(t, int64(1), data.Get("id"))
// 	assert.Equal(t, "assign", data.Get("name"))
// 	assert.Equal(t, "选择商务负责人", data.Get("node_name"))
// 	assert.Equal(t, "进行中", data.Get("node_status"))
// 	assert.Equal(t, "进行中", data.Get("status"))
// 	assert.Equal(t, float64(1), data.Get("input.选择商务负责人.data.id"))
// 	assert.Equal(t, "云存储", data.Get("input.选择商务负责人.data.name"))
// 	assert.Equal(t, float64(1), data.Get("input.选择商务负责人.form.biz_id"))
// 	assert.Equal(t, "李明博", data.Get("input.选择商务负责人.form.name"))

// 	// 清理数据
// 	capsule.Query().From("xiang_workflow").Truncate()
// }

// func TestOpen(t *testing.T) {
// 	assignFlow := Select("assign")
// 	assignFlow.Save(1, "选择商务负责人", 1, Input{
// 		Data: map[string]interface{}{"id": 1, "name": "云主机"},
// 		Form: map[string]interface{}{"biz_id": 1, "name": "张良明"},
// 	})
// 	wflow := assignFlow.Open(1, 1)
// 	data := maps.Of(wflow).Dot()
// 	assert.Equal(t, int64(1), data.Get("id"))
// 	assert.Equal(t, "assign", data.Get("name"))
// 	assert.Equal(t, "选择商务负责人", data.Get("node_name"))
// 	assert.Equal(t, "进行中", data.Get("node_status"))
// 	assert.Equal(t, "进行中", data.Get("status"))
// 	assert.Equal(t, int64(1), data.Get("user_id"))
// 	assert.Equal(t, float64(1), data.Get("users.选择商务负责人"))

// 	// 清理数据
// 	capsule.Query().From("xiang_workflow").Truncate()
// }

// func TestOpenEmpty(t *testing.T) {
// 	assignFlow := Select("assign")
// 	wflow := assignFlow.Open(1, 1)
// 	data := maps.Of(wflow).Dot()
// 	assert.Equal(t, false, data.Has("id"))
// 	assert.Equal(t, "assign", data.Get("name"))
// 	assert.Equal(t, "选择商务负责人", data.Get("node_name"))
// 	assert.Equal(t, "进行中", data.Get("node_status"))
// 	assert.Equal(t, "进行中", data.Get("status"))
// 	assert.Equal(t, 1, data.Get("user_id"))
// 	assert.Equal(t, 1, data.Get("users.选择商务负责人"))
// }

// func TestNext(t *testing.T) {
// 	assignFlow := Select("assign")
// 	wflow := assignFlow.Save(1, "选择商务负责人", 1, Input{
// 		Data: map[string]interface{}{"id": 1, "name": "云主机"},
// 		Form: map[string]interface{}{"biz_id": 1, "name": "张良明"},
// 	})
// 	wflow = assignFlow.Next(1, any.Of(wflow["id"]).CInt(), map[string]interface{}{
// 		"项目名称":    "测试项目",
// 		"商务负责人名称": "林明波",
// 	})
// 	data := maps.Of(wflow).Dot()
// 	assert.Equal(t, int64(1), data.Get("id"))
// 	assert.Equal(t, "assign", data.Get("name"))
// 	assert.Equal(t, "项目负责人审批", data.Get("node_name"))
// 	assert.Equal(t, "进行中", data.Get("node_status"))
// 	assert.Equal(t, "进行中", data.Get("status"))
// 	assert.Equal(t, int64(2), data.Get("user_id"))
// 	assert.Equal(t, true, data.Has("users"))
// 	assert.Equal(t, "测试项目", data.Get("output.项目名称"))
// 	assert.Equal(t, "林明波", data.Get("output.商务负责人名称"))

// 	// 清理数据
// 	capsule.Query().From("xiang_workflow").Truncate()
// }

// func TestNextWhen(t *testing.T) {
// 	assignFlow := Select("assign")
// 	wflow := assignFlow.Save(1, "选择商务负责人", 1, Input{
// 		Data: map[string]interface{}{"id": 1, "name": "云主机"},
// 		Form: map[string]interface{}{"biz_id": 1, "name": "张良明"},
// 	})
// 	id := any.Of(wflow["id"]).CInt()
// 	assignFlow.Next(1, id, map[string]interface{}{
// 		"项目名称":    "测试项目",
// 		"商务负责人名称": "林明波",
// 	})

// 	// Next When
// 	wflow = assignFlow.Next(2, id, map[string]interface{}{"审批结果": "通过"})
// 	data := maps.Of(wflow).Dot()
// 	assert.Equal(t, int64(1), data.Get("id"))
// 	assert.Equal(t, "assign", data.Get("name"))
// 	assert.Equal(t, "审批通过", data.Get("node_name"))
// 	assert.Equal(t, "进行中", data.Get("node_status"))
// 	assert.Equal(t, "进行中", data.Get("status"))
// 	assert.Equal(t, int64(1), data.Get("user_id"))
// 	assert.Equal(t, true, data.Has("users"))
// 	assert.Equal(t, "测试项目", data.Get("output.项目名称"))
// 	assert.Equal(t, "林明波", data.Get("output.商务负责人名称"))
// 	assert.Equal(t, "通过", data.Get("output.审批结果"))

// 	// 清理数据
// 	capsule.Query().From("xiang_workflow").Truncate()
// }

// func TestGoto(t *testing.T) {
// 	assignFlow := Select("assign")
// 	wflow := assignFlow.Save(1, "选择商务负责人", 1, Input{
// 		Data: map[string]interface{}{"id": 1, "name": "云主机"},
// 		Form: map[string]interface{}{"biz_id": 1, "name": "张良明"},
// 	})
// 	id := any.Of(wflow["id"]).CInt()
// 	assignFlow.Next(1, id, map[string]interface{}{
// 		"项目名称":    "测试项目",
// 		"商务负责人名称": "林明波",
// 	})

// 	wflow = assignFlow.Goto(2, id, "选择商务负责人", map[string]interface{}{"审批结果": "驳回"})
// 	data := maps.Of(wflow).Dot()
// 	assert.Equal(t, true, data.Has("id"))
// 	assert.Equal(t, "assign", data.Get("name"))
// 	assert.Equal(t, "选择商务负责人", data.Get("node_name"))
// 	assert.Equal(t, "进行中", data.Get("node_status"))
// 	assert.Equal(t, "进行中", data.Get("status"))
// 	assert.Equal(t, int64(1), data.Get("user_id"))
// 	assert.Equal(t, float64(1), data.Get("users.选择商务负责人"))
// 	assert.Equal(t, float64(2), data.Get("users.项目负责人审批"))

// 	// 清理数据
// 	capsule.Query().From("xiang_workflow").Truncate()
// }

// func TestReset(t *testing.T) {
// 	assignFlow := Select("assign")
// 	wflow := assignFlow.Save(1, "选择商务负责人", 1, Input{
// 		Data: map[string]interface{}{"id": 1, "name": "云主机"},
// 		Form: map[string]interface{}{"biz_id": 1, "name": "张良明"},
// 	})
// 	id := any.Of(wflow["id"]).CInt()
// 	assignFlow.Next(1, id, map[string]interface{}{
// 		"项目名称":    "测试项目",
// 		"商务负责人名称": "林明波",
// 	})

// 	wflow = assignFlow.Reset(2, id, map[string]interface{}{"审批结果": "驳回"})
// 	data := maps.Of(wflow).Dot()
// 	assert.Equal(t, true, data.Has("id"))
// 	assert.Equal(t, "assign", data.Get("name"))
// 	assert.Equal(t, "选择商务负责人", data.Get("node_name"))
// 	assert.Equal(t, "进行中", data.Get("node_status"))
// 	assert.Equal(t, "进行中", data.Get("status"))
// 	assert.Equal(t, int64(1), data.Get("user_id"))
// 	assert.Equal(t, float64(1), data.Get("users.选择商务负责人"))
// 	assert.Equal(t, float64(2), data.Get("users.项目负责人审批"))

// 	// 清理数据
// 	capsule.Query().From("xiang_workflow").Truncate()
// }

// func TestDone(t *testing.T) {
// 	assignFlow := Select("assign")
// 	wflow := assignFlow.Save(1, "选择商务负责人", 1, Input{
// 		Data: map[string]interface{}{"id": 1, "name": "云主机"},
// 		Form: map[string]interface{}{"biz_id": 1, "name": "张良明"},
// 	})
// 	id := any.Of(wflow["id"]).CInt()
// 	assignFlow.Next(1, id, map[string]interface{}{
// 		"项目名称":    "测试项目",
// 		"商务负责人名称": "林明波",
// 	})

// 	wflow = assignFlow.Done(2, id, map[string]interface{}{"关闭原因": "测试完成"})
// 	data := maps.Of(wflow).Dot()
// 	assert.Equal(t, "已完成", data.Get("status"))
// 	assert.Equal(t, "测试完成", data.Get("output.关闭原因"))
// 	// 清理数据
// 	capsule.Query().From("xiang_workflow").Truncate()
// }

// func TestClose(t *testing.T) {
// 	assignFlow := Select("assign")
// 	wflow := assignFlow.Save(1, "选择商务负责人", 1, Input{
// 		Data: map[string]interface{}{"id": 1, "name": "云主机"},
// 		Form: map[string]interface{}{"biz_id": 1, "name": "张良明"},
// 	})
// 	id := any.Of(wflow["id"]).CInt()
// 	assignFlow.Next(1, id, map[string]interface{}{
// 		"项目名称":    "测试项目",
// 		"商务负责人名称": "林明波",
// 	})

// 	wflow = assignFlow.Close(2, id, map[string]interface{}{"关闭原因": "测试关闭"})
// 	data := maps.Of(wflow).Dot()
// 	assert.Equal(t, "已关闭", data.Get("status"))
// 	assert.Equal(t, "测试关闭", data.Get("output.关闭原因"))
// 	// 清理数据
// 	capsule.Query().From("xiang_workflow").Truncate()
// }

// func TestSetting(t *testing.T) {
// 	assignFlow := Select("assign")
// 	wflow := assignFlow.Save(1, "选择商务负责人", 1, Input{
// 		Data: map[string]interface{}{"id": 1, "name": "云主机"},
// 		Form: map[string]interface{}{"biz_id": 1, "name": "张良明"},
// 	})

// 	setting := assignFlow.Setting(1, 1)
// 	data := maps.Of(setting).Dot()
// 	assert.Equal(t, true, data.Get("read"))
// 	assert.Equal(t, "选择商务负责人", data.Get("node.label"))
// 	assert.Equal(t, true, data.Get("write"))

// 	id := any.Of(wflow["id"]).CInt()
// 	assignFlow.Next(1, id, map[string]interface{}{
// 		"项目名称":    "测试项目",
// 		"商务负责人名称": "林明波",
// 	})

// 	setting = assignFlow.Setting(1, 1)
// 	data = maps.Of(setting).Dot()
// 	assert.Equal(t, true, data.Get("read"))
// 	assert.Equal(t, false, data.Get("write"))
// 	assert.Equal(t, "项目负责人审批", data.Get("node.label"))
// 	assert.Equal(t, 2, data.Get("nodes.2.source"))
// 	assert.Equal(t, 2, data.Get("nodes.3.source"))
// 	assert.Equal(t, "assign", data.Get("name"))
// 	assert.Equal(t, "指派商务负责人", data.Get("label"))

// 	// 清理数据
// 	capsule.Query().From("xiang_workflow").Truncate()
// }

// func check(t *testing.T) {
// 	keys := []string{}
// 	for key, workflow := range WorkFlows {
// 		keys = append(keys, key)
// 		workflow.Reload()
// 	}
// 	assert.Equal(t, 1, len(keys))
// }
