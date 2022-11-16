package form

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/helper"
)

// Export process
func exportProcess() {
	gou.RegisterProcessHandler("yao.form.setting", processSetting)
	gou.RegisterProcessHandler("yao.form.xgen", processXgen)
	gou.RegisterProcessHandler("yao.form.component", processComponent)
	gou.RegisterProcessHandler("yao.form.upload", processUpload)
	gou.RegisterProcessHandler("yao.form.download", processDownload)
	gou.RegisterProcessHandler("yao.form.find", processFind)
	gou.RegisterProcessHandler("yao.form.save", processSave)
	gou.RegisterProcessHandler("yao.form.create", processCreate)
	gou.RegisterProcessHandler("yao.form.update", processUpdate)
	gou.RegisterProcessHandler("yao.form.delete", processDelete)
}

func processXgen(process *gou.Process) interface{} {

	form := MustGet(process)
	setting, err := form.Xgen()
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return setting
}

func processComponent(process *gou.Process) interface{} {

	process.ValidateArgNums(3)
	form := MustGet(process)
	xpath := process.ArgsString(1)
	method := process.ArgsString(2)
	key := fmt.Sprintf("%s.$%s", xpath, method)

	// get cloud props
	cProp, has := form.CProps[key]
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
	form := MustGet(process)
	field := process.ArgsString(1)
	file := process.ArgsString(2)
	tokenString := process.ArgsString(3)

	// checking
	ext := fs.ExtName(file)
	if _, has := fs.DownloadWhitelist[ext]; !has {
		exception.New("%s.%s .%s file does not allow", 403, form.ID, field, ext).Throw()
	}

	// Auth
	tokenString = strings.TrimSpace(strings.TrimPrefix(tokenString, "Bearer "))
	if tokenString == "" {
		exception.New("%s.%s No permission", 403, form.ID, field).Throw()
	}
	claims := helper.JwtValidate(tokenString)

	// Get Process name
	name := "fs.system.Download"
	if form.Action.Download.Process != "" {
		name = form.Action.Download.Process
	}

	// Create process
	p, err := gou.ProcessOf(name, file)
	if err != nil {
		log.Error("[downalod] %s.%s %s", form.ID, field, err.Error())
		exception.New("[downalod] %s.%s %s", 400, form.ID, field, err.Error()).Throw()
	}

	// Excute process
	res, err := p.WithGlobal(process.Global).WithSID(claims.SID).Exec()
	if err != nil {
		log.Error("[downalod] %s.%s %s", form.ID, field, err.Error())
		exception.New("[downalod] %s.%s %s", 500, form.ID, field, err.Error()).Throw()
	}

	return res
}

func processUpload(process *gou.Process) interface{} {

	process.ValidateArgNums(4)
	form := MustGet(process)
	xpath := process.ArgsString(1)
	method := process.ArgsString(2)
	key := fmt.Sprintf("%s.$%s", xpath, method)

	// get cloud props
	cProp, has := form.CProps[key]
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
		file = fmt.Sprintf("/api/__yao/form/%s/download/%s?name=%s", form.ID, url.QueryEscape(field), file)
		return file
	}

	return res
}

func processSetting(process *gou.Process) interface{} {
	form := MustGet(process)
	process.Args = append(process.Args, process.Args[0]) // formle name
	return form.Action.Setting.MustExec(process)
}

func processSave(process *gou.Process) interface{} {
	form := MustGet(process)
	return form.Action.Save.MustExec(process)
}

func processCreate(process *gou.Process) interface{} {
	form := MustGet(process)
	return form.Action.Create.MustExec(process)
}

func processFind(process *gou.Process) interface{} {
	form := MustGet(process)
	return form.Action.Find.MustExec(process)
}

func processUpdate(process *gou.Process) interface{} {
	form := MustGet(process)
	return form.Action.Update.MustExec(process)
}

func processDelete(process *gou.Process) interface{} {
	form := MustGet(process)
	return form.Action.Delete.MustExec(process)
}
