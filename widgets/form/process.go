package form

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/yaoapp/gou/application"
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
	gouProcess.Register("yao.form.reload", processReload)
	gouProcess.Register("yao.form.unload", processUnload)
	gouProcess.Register("yao.form.read", processRead)
	gouProcess.Register("yao.form.exists", processExists)
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
	isAppRoot := process.ArgsInt(4, 0)

	// checking
	ext := fs.ExtName(file)
	if _, has := fs.DownloadWhitelist[ext]; !has {
		exception.New("%s.%s .%s file does not allow", 403, form.ID, field, ext).Throw()
	}

	// Auth
	tokenString = strings.TrimSpace(strings.TrimPrefix(tokenString, "Bearer "))
	if tokenString == "" {
		exception.New("%s.%s not authenticated", 401, form.ID, field).Throw()
	}
	claims := helper.JwtValidate(tokenString)

	// Get Process name
	name := "fs.system.Download"
	if form.Action.Download.Process != "" {
		name = form.Action.Download.Process
	}

	// The root path of the application the Upload Component props.appRoot=true
	if isAppRoot == 1 {
		name = "fs.app.Download"
	}

	// Create process
	p, err := gouProcess.Of(name, file)
	if err != nil {
		log.Error("[downalod] %s.%s %s", form.ID, field, err.Error())
		exception.New("[downalod] %s.%s %s", 400, form.ID, field, err.Error()).Throw()
	}

	// Excute process
	err = p.WithGlobal(process.Global).WithSID(claims.SID).Execute()
	if err != nil {
		log.Error("[downalod] %s.%s %s", form.ID, field, err.Error())
		exception.New("[downalod] %s.%s %s", 500, form.ID, field, err.Error()).Throw()
	}
	defer p.Release()
	return p.Value()
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

// processLoad yao.form.Load form_name file <source>
func processLoad(process *gouProcess.Process) interface{} {
	process.ValidateArgNums(1)

	// Load from source
	if process.NumOfArgs() >= 3 {
		id := process.ArgsString(0)
		source := process.ArgsString(2)
		_, err := LoadSourceSync([]byte(source), id)
		if err != nil {
			exception.New(err.Error(), 500).Throw()
		}
		return nil
	}

	// Load from file
	file := process.ArgsString(0)
	if file == "" {
		exception.New("file is required", 400).Throw()
	}

	file = strings.TrimPrefix(file, string(os.PathSeparator))
	return LoadFileSync("forms", file)
}

// processReload yao.form.Reload form_name
func processReload(process *gouProcess.Process) interface{} {
	process.ValidateArgNums(1)
	tab := MustGet(process) // 0
	_, err := tab.Reload()
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return nil
}

// processUnload yao.form.Unload form_name
func processUnload(process *gouProcess.Process) interface{} {
	process.ValidateArgNums(1)
	Unload(process.ArgsString(0))
	return nil
}

// processRead yao.form.Read form_name
func processRead(process *gouProcess.Process) interface{} {
	process.ValidateArgNums(1)
	tab := MustGet(process) // 0
	source := map[string]interface{}{}
	err := application.Parse(tab.file, tab.Read(), &source)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return source
}

// processExists yao.form.Exists form_name
func processExists(process *gouProcess.Process) interface{} {
	process.ValidateArgNums(1)
	return Exists(process.ArgsString(0))
}
