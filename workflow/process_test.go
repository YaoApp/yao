package workflow

// WARNING: the Workflow widget will be removed from yao engine

// func init() {
// 	share.DBConnect(config.Conf.DB)
// 	share.Load(config.Conf)
// 	model.Load(config.Conf)
// 	engineModels := path.Join(os.Getenv("YAO_DEV"), "yao", "models")
// 	model.LoadFrom(engineModels, "xiang.")
// 	query.Load(config.Conf)
// 	flow.Load(config.Conf)
// 	Load(config.Conf)
// }

// func TestProcessFind(t *testing.T) {
// 	assignFlow := Select("assign")
// 	wflow := assignFlow.Save(1, "选择商务负责人", 1, Input{
// 		Data: map[string]interface{}{"id": 1, "name": "云主机"},
// 		Form: map[string]interface{}{"biz_id": 1, "name": "张良明"},
// 	})

// 	args := []interface{}{"assign", wflow["id"]}
// 	res := gou.NewProcess("xiang.workflow.Find", args...).Run()

// 	data := any.Of(res).Map().MapStrAny.Dot()
// 	assert.Equal(t, wflow["id"], data.Get("id"))

// 	// 清理数据
// 	capsule.Query().From("xiang_workflow").Truncate()
// }

// func TestProcessSetting(t *testing.T) {
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

// 	args := []interface{}{"assign", 1, 1}
// 	res := gou.NewProcess("xiang.workflow.Setting", args...).Run()
// 	data := any.Of(res).Map().MapStrAny.Dot()
// 	assert.Equal(t, 2, data.Get("nodes.2.source"))
// 	assert.Equal(t, 2, data.Get("nodes.3.source"))
// 	assert.Equal(t, "assign", data.Get("name"))
// 	assert.Equal(t, "指派商务负责人", data.Get("label"))

// 	// 清理数据
// 	capsule.Query().From("xiang_workflow").Truncate()
// }

// func TestProcessOpen(t *testing.T) {
// 	assignFlow := Select("assign")
// 	wflow := assignFlow.Save(1, "选择商务负责人", 1, Input{
// 		Data: map[string]interface{}{"id": 1, "name": "云主机"},
// 		Form: map[string]interface{}{"biz_id": 1, "name": "张良明"},
// 	})

// 	args := []interface{}{"assign", 1, 1}
// 	res := gou.NewProcess("xiang.workflow.Open", args...).Run()

// 	data := any.Of(res).Map().MapStrAny.Dot()
// 	assert.Equal(t, wflow["id"], data.Get("id"))

// 	// 清理数据
// 	capsule.Query().From("xiang_workflow").Truncate()
// }

// func TestProcessSave(t *testing.T) {
// 	args := []interface{}{"assign",
// 		1, "选择商务负责人", 1,
// 		map[string]interface{}{
// 			"data": map[string]interface{}{"id": 1, "name": "云主机"},
// 			"form": map[string]interface{}{"biz_id": 1, "name": "张良明"},
// 		},
// 		map[string]interface{}{"foo": "bar"},
// 	}
// 	res := gou.NewProcess("xiang.workflow.Save", args...).Run()
// 	data := any.Of(res).Map().MapStrAny.Dot()
// 	assert.Equal(t, int64(1), data.Get("id"))

// 	// 清理数据
// 	capsule.Query().From("xiang_workflow").Truncate()
// }

// func TestProcessNext(t *testing.T) {
// 	assignFlow := Select("assign")
// 	wflow := assignFlow.Save(1, "选择商务负责人", 1, Input{
// 		Data: map[string]interface{}{"id": 1, "name": "云主机"},
// 		Form: map[string]interface{}{"biz_id": 1, "name": "张良明"},
// 	})

// 	args := []interface{}{"assign",
// 		1, wflow["id"], map[string]interface{}{
// 			"项目名称":    "测试项目",
// 			"商务负责人名称": "林明波",
// 		},
// 	}
// 	res := gou.NewProcess("xiang.workflow.Next", args...).Run()
// 	data := any.Of(res).Map().MapStrAny.Dot()
// 	assert.Equal(t, wflow["id"], data.Get("id"))
// 	assert.Equal(t, "项目负责人审批", data.Get("node_name"))

// 	// 清理数据
// 	capsule.Query().From("xiang_workflow").Truncate()
// }

// func TestProcessGoto(t *testing.T) {
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

// 	args := []interface{}{"assign",
// 		2, id, "选择商务负责人",
// 		map[string]interface{}{"审批结果": "驳回"},
// 	}
// 	res := gou.NewProcess("xiang.workflow.Goto", args...).Run()
// 	data := any.Of(res).Map().MapStrAny.Dot()
// 	assert.Equal(t, wflow["id"], data.Get("id"))
// 	assert.Equal(t, "选择商务负责人", data.Get("node_name"))

// 	// 清理数据
// 	capsule.Query().From("xiang_workflow").Truncate()
// }

// func TestProcessStatus(t *testing.T) {
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

// 	args := []interface{}{"assign",
// 		2, id, "已完成", map[string]interface{}{"关闭原因": "测试完成"},
// 	}
// 	res := gou.NewProcess("xiang.workflow.Status", args...).Run()
// 	data := any.Of(res).Map().MapStrAny.Dot()
// 	assert.Equal(t, wflow["id"], data.Get("id"))
// 	assert.Equal(t, "已完成", data.Get("status"))

// 	// 清理数据
// 	capsule.Query().From("xiang_workflow").Truncate()
// }

// func TestProcessDone(t *testing.T) {
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

// 	args := []interface{}{"assign",
// 		2, id, map[string]interface{}{"关闭原因": "测试完成"},
// 	}
// 	res := gou.NewProcess("xiang.workflow.Done", args...).Run()
// 	data := any.Of(res).Map().MapStrAny.Dot()
// 	assert.Equal(t, wflow["id"], data.Get("id"))
// 	assert.Equal(t, "已完成", data.Get("status"))

// 	// 清理数据
// 	capsule.Query().From("xiang_workflow").Truncate()
// }

// func TestProcessClose(t *testing.T) {
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

// 	args := []interface{}{"assign",
// 		2, id, map[string]interface{}{"关闭原因": "测试关闭"},
// 	}
// 	res := gou.NewProcess("xiang.workflow.Close", args...).Run()
// 	data := any.Of(res).Map().MapStrAny.Dot()
// 	assert.Equal(t, wflow["id"], data.Get("id"))
// 	assert.Equal(t, "已关闭", data.Get("status"))

// 	// 清理数据
// 	capsule.Query().From("xiang_workflow").Truncate()
// }

// func TestProcessReset(t *testing.T) {
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

// 	args := []interface{}{"assign",
// 		2, id, map[string]interface{}{"驳回原因": "字段填写错误"},
// 	}
// 	res := gou.NewProcess("xiang.workflow.Reset", args...).Run()
// 	data := any.Of(res).Map().MapStrAny.Dot()
// 	assert.Equal(t, wflow["id"], data.Get("id"))
// 	assert.Equal(t, "进行中", data.Get("status"))
// 	assert.Equal(t, "选择商务负责人", data.Get("node_name"))

// 	// 清理数据
// 	capsule.Query().From("xiang_workflow").Truncate()
// }
