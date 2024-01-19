package form

import (
	"fmt"
	"net/url"
	"os"
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
	gouProcess.Register("yao.form.setting", processSetting)
	gouProcess.Register("yao.form.xgen", processXgen)
	gouProcess.Register("yao.form.component", processComponent)
	gouProcess.Register("yao.form.upload", processUpload)
	gouProcess.Register("yao.form.download", processDownload)
	gouProcess.Register("yao.form.find", processFind)
	gouProcess.Register("yao.form.save", processSave)
	gouProcess.Register("yao.form.create", processCreate)
	gouProcess.Register("yao.form.update", processUpdate)
	gouProcess.Register("yao.form.delete", processDelete)
	gouProcess.Register("yao.form.load", processLoad)
}

func processXgen(process *gouProcess.Process) interface{} {

	form := MustGet(process)
	data := process.ArgsMap(1, map[string]interface{}{})
	excludes := app.Permissions(process, "forms", form.ID)
	setting, err := form.Xgen(data, excludes)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return setting
}

func processComponent(process *gouProcess.Process) interface{} {

	process.ValidateArgNums(3)
	form := MustGet(process)
	xpath, _ := url.QueryUnescape(process.ArgsString(1))
	method, _ := url.QueryUnescape(process.ArgsString(2))
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

func processDownload(process *gouProcess.Process) interface{} {

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
	p, err := gouProcess.Of(name, file)
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

func processUpload(process *gouProcess.Process) interface{} {

	process.ValidateArgNums(4)
	form := MustGet(process)
	xpath, _ := url.QueryUnescape(process.ArgsString(1))
	method, _ := url.QueryUnescape(process.ArgsString(2))
	key := fmt.Sprintf("%s.$%s", xpath, method)

	// get cloud props
	cProp, has := form.CProps[key]
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
		file = fmt.Sprintf("/api/__yao/form/%s/download/%s?name=%s", form.ID, url.QueryEscape(field), file)
		return file
	}

	return res
}

func processSetting(process *gouProcess.Process) interface{} {
	form := MustGet(process)
	process.Args = append(process.Args, process.Args[0]) // form name
	return form.Action.Setting.MustExec(process)
}

func processSave(process *gouProcess.Process) interface{} {
	form := MustGet(process)
	return form.Action.Save.MustExec(process)
}

func processCreate(process *gouProcess.Process) interface{} {
	form := MustGet(process)
	return form.Action.Create.MustExec(process)
}

func processFind(process *gouProcess.Process) interface{} {
	form := MustGet(process)
	return form.Action.Find.MustExec(process)
}

func processUpdate(process *gouProcess.Process) interface{} {
	form := MustGet(process)
	return form.Action.Update.MustExec(process)
}

func processDelete(process *gouProcess.Process) interface{} {
	form := MustGet(process)
	return form.Action.Delete.MustExec(process)
}

// processLoad yao.table.Load (:file)
func processLoad(process *gouProcess.Process) interface{} {
	process.ValidateArgNums(1)
	file := process.ArgsString(0)
	if file == "" {
		exception.New("file is required", 400).Throw()
	}

	file = strings.TrimPrefix(file, string(os.PathSeparator))
	return LoadFileSync("forms", file)
}
