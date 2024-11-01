package list

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/yaoapp/gou/fs"
	gouProcess "github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/types"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/helper"
	"github.com/yaoapp/yao/widgets/app"
)

// Export process
func exportProcess() {
	gouProcess.Register("yao.list.setting", processSetting)
	gouProcess.Register("yao.list.xgen", processXgen)
	gouProcess.Register("yao.list.component", processComponent)
	gouProcess.Register("yao.list.upload", processUpload)
	gouProcess.Register("yao.list.download", processDownload)
	gouProcess.Register("yao.list.save", processSave)
}

func processXgen(process *gouProcess.Process) interface{} {
	list := MustGet(process)
	query := process.ArgsMap(1, map[string]interface{}{})
	data := process.ArgsMap(2, map[string]interface{}{})
	excludes := app.Permissions(process, "lists", list.ID)
	setting, err := list.Xgen(data, excludes, query)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return setting
}

func processComponent(process *gouProcess.Process) interface{} {

	process.ValidateArgNums(3)
	list := MustGet(process)
	xpath, _ := url.QueryUnescape(process.ArgsString(1))
	method, _ := url.QueryUnescape(process.ArgsString(2))
	key := fmt.Sprintf("%s.$%s", xpath, method)

	// get cloud props
	cProp, has := list.CProps[key]
	if !has {
		exception.New("%s does not exist", 400, key).Throw()
	}

	// :query
	query := map[string]interface{}{}
	if process.NumOfArgsIs(4) {
		query = process.ArgsMap(3)
	}

	// execute query
	res, err := cProp.ExecQuery(process, query)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return res
}

func processDownload(process *gouProcess.Process) interface{} {

	process.ValidateArgNums(4)
	list := MustGet(process)
	field := process.ArgsString(1)
	file := process.ArgsString(2)
	tokenString := process.ArgsString(3)

	// checking
	ext := fs.ExtName(file)
	if _, has := fs.DownloadWhitelist[ext]; !has {
		exception.New("%s.%s .%s file does not allow", 403, list.ID, field, ext).Throw()
	}

	// Auth
	tokenString = strings.TrimSpace(strings.TrimPrefix(tokenString, "Bearer "))
	if tokenString == "" {
		exception.New("%s.%s not authenticated", 401, list.ID, field).Throw()
	}
	claims := helper.JwtValidate(tokenString)

	// Get Process name
	name := "fs.system.Download"
	if list.Action.Download.Process != "" {
		name = list.Action.Download.Process
	}

	// Create process
	p, err := gouProcess.Of(name, file)
	if err != nil {
		log.Error("[downalod] %s.%s %s", list.ID, field, err.Error())
		exception.New("[downalod] %s.%s %s", 400, list.ID, field, err.Error()).Throw()
	}

	// Excute process
	err = p.WithGlobal(process.Global).WithSID(claims.SID).Execute()
	if err != nil {
		log.Error("[downalod] %s.%s %s", list.ID, field, err.Error())
		exception.New("[downalod] %s.%s %s", 500, list.ID, field, err.Error()).Throw()
	}
	defer p.Release()
	return p.Value()
}

func processUpload(process *gouProcess.Process) interface{} {

	process.ValidateArgNums(4)
	list := MustGet(process)
	xpath, _ := url.QueryUnescape(process.ArgsString(1))
	method, _ := url.QueryUnescape(process.ArgsString(2))
	key := fmt.Sprintf("%s.$%s", xpath, method)

	// get cloud props
	cProp, has := list.CProps[key]
	if !has {
		exception.New("%s does not exist", 400, key).Throw()
	}

	// $file.file
	tmpfile, ok := process.Args[3].(types.UploadFile)
	if !ok {
		exception.New("parameters error: %v", 400, process.Args[3]).Throw()
	}

	// execute upload
	res, err := cProp.ExecUpload(process, tmpfile)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	if file, ok := res.(string); ok {
		field := strings.TrimSuffix(xpath, ".edit.props")
		file = fmt.Sprintf("/api/__yao/list/%s/download/%s?name=%s", list.ID, url.QueryEscape(field), file)
		return file
	}

	return res
}

func processSetting(process *gouProcess.Process) interface{} {
	list := MustGet(process)
	name := process.ArgsString(0)
	params := process.ArgsMap(1, map[string]interface{}{})
	query := map[string]string{}
	for key, value := range params {
		switch v := value.(type) {
		case string:
			query[key] = v
		case []string:
			query[key] = strings.Join(v, ",")
		default:
			query[key] = fmt.Sprintf("%v", value)
		}
	}

	process.Args = []interface{}{name, name, query} // list name
	return list.Action.Setting.MustExec(process)
}

func processGet(process *gouProcess.Process) interface{} {
	list := MustGet(process)
	return list.Action.Get.MustExec(process)
}

func processSave(process *gouProcess.Process) interface{} {
	list := MustGet(process)
	return list.Action.Save.MustExec(process)
}
