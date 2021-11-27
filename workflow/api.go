package workflow

import (
	"path/filepath"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou"
)

// API:
// 读取工作流 GET /api/xiang/workflow/<工作流名称>/find/:id
// 读取工作流 GET /api/xiang/workflow/<工作流名称>/open/:data_id
// 读取工作流配置 GET /api/xiang/workflow/<工作流名称>/setting/:data_id
// 调用自定义API POST /api/xiang/workflow/<工作流名称>/<自定义API路由>

// SetupAPIs 注册API
func (workflow *WorkFlow) SetupAPIs() error {
	api := &gou.HTTP{
		Name:        "工作流接口",
		Version:     "1.0.0",
		Description: "工作流接口 API",
		Group:       "xiang/workflow",
		Guard:       "bearer-jwt",
		Paths:       []gou.Path{},
	}

	workflow.AddAPI(api, workflow.apiFind())
	workflow.AddAPI(api, workflow.apiOpen())
	workflow.AddAPI(api, workflow.apiSetting())

	for name, custAPI := range workflow.APIs {
		workflow.AddAPI(api, gou.Path{
			Label:       custAPI.Label,
			Description: custAPI.Description,
			Path:        filepath.Join("/", workflow.Name, name),
			Method:      "POST",
			Process:     custAPI.Process,
			In:          []string{workflow.Name, "$query.data_id", ":payload"},
			Out: gou.Out{
				Status: 200,
				Type:   "application/json",
			},
		})
	}

	// 注册API
	source, err := jsoniter.Marshal(api)
	if err != nil {
		return err
	}

	gou.LoadAPI(string(source), "xiang.workflow."+workflow.Name)
	return nil
}

// AddAPI 添加API
func (workflow *WorkFlow) AddAPI(api *gou.HTTP, path gou.Path) {
	api.Paths = append(api.Paths, path)
}

// apiFind 使用ID读取工作流
func (workflow *WorkFlow) apiFind() gou.Path {
	return gou.Path{
		Label:       "读取工作流",
		Description: "使用工作流ID读取工作流",
		Path:        filepath.Join("/", workflow.Name, "find", ":id"),
		Method:      "GET",
		Process:     "xiang.workflow.Find",
		In:          []string{workflow.Name, "$param.id"},
		Out: gou.Out{
			Status: 200,
			Type:   "application/json",
		},
	}
}

// apiOpen 使用用户ID和数据ID 读取工作流
func (workflow *WorkFlow) apiOpen() gou.Path {
	return gou.Path{
		Label:       "读取工作流",
		Description: "使用数据ID读取工作流",
		Path:        filepath.Join("/", workflow.Name, "open", ":data_id"),
		Method:      "GET",
		Process:     "xiang.workflow.Open",
		In:          []string{workflow.Name, "$session.user_id", "$param.data_id"},
		Out: gou.Out{
			Status: 200,
			Type:   "application/json",
		},
	}
}

// apiSetting 读取工作流配置
func (workflow *WorkFlow) apiSetting() gou.Path {
	return gou.Path{
		Label:       "读取工作流配置",
		Description: "使用数据ID读取工作流配置",
		Path:        filepath.Join("/", workflow.Name, "setting", ":data_id"),
		Method:      "GET",
		Process:     "xiang.workflow.Setting",
		In:          []string{workflow.Name, "$session.user_id", "$param.data_id"},
		Out: gou.Out{
			Status: 200,
			Type:   "application/json",
		},
	}
}
