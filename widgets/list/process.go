package list

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/helper"
	"github.com/yaoapp/yao/widgets/app"
)

// Export process
func exportProcess() {
	gou.RegisterProcessHandler("yao.list.setting", processSetting)
	gou.RegisterProcessHandler("yao.list.xgen", processXgen)
	gou.RegisterProcessHandler("yao.list.component", processComponent)
	gou.RegisterProcessHandler("yao.list.upload", processUpload)
	gou.RegisterProcessHandler("yao.list.download", processDownload)
	gou.RegisterProcessHandler("yao.list.save", processSave)
}

func processXgen(process *gou.Process) interface{} {

	list := MustGet(process)
	data := process.ArgsMap(1, map[string]interface{}{})
	excludes := app.Permissions(process, "lists", list.ID)
	setting, err := list.Xgen(data, excludes)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return setting
}

func processComponent(process *gou.Process) interface{} {

	process.ValidateArgNums(3)
	list := MustGet(process)
	xpath := process.ArgsString(1)
	method := process.ArgsString(2)
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

func processDownload(process *gou.Process) interface{} {

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
		exception.New("%s.%s No permission", 403, list.ID, field).Throw()
	}
	claims := helper.JwtValidate(tokenString)

	// Get Process name
	name := "fs.system.Download"
	if list.Action.Download.Process != "" {
		name = list.Action.Download.Process
	}

	// Create process
	p, err := gou.ProcessOf(name, file)
	if err != nil {
		log.Error("[downalod] %s.%s %s", list.ID, field, err.Error())
		exception.New("[downalod] %s.%s %s", 400, list.ID, field, err.Error()).Throw()
	}

	// Excute process
	res, err := p.WithGlobal(process.Global).WithSID(claims.SID).Exec()
	if err != nil {
		log.Error("[downalod] %s.%s %s", list.ID, field, err.Error())
		exception.New("[downalod] %s.%s %s", 500, list.ID, field, err.Error()).Throw()
	}

	return res
}

func processUpload(process *gou.Process) interface{} {

	process.ValidateArgNums(4)
	list := MustGet(process)
	xpath := process.ArgsString(1)
	method := process.ArgsString(2)
	key := fmt.Sprintf("%s.$%s", xpath, method)

	// get cloud props
	cProp, has := list.CProps[key]
	if !has {
		exception.New("%s does not exist", 400, key).Throw()
	}

	// $file.file
	tmpfile, ok := process.Args[3].(gou.UploadFile)
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

func processSetting(process *gou.Process) interface{} {
	list := MustGet(process)
	process.Args = append(process.Args, process.Args[0]) // list name
	return list.Action.Setting.MustExec(process)
}

func processGet(process *gou.Process) interface{} {
	list := MustGet(process)
	return list.Action.Get.MustExec(process)
}

func processSave(process *gou.Process) interface{} {
	list := MustGet(process)
	return list.Action.Save.MustExec(process)
}
