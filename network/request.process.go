package network

import (
	"fmt"

	"github.com/yaoapp/gou"
)

// *******************************************************
// * DEPRECATED	→ http								     *
// *******************************************************

// ProcessPost  xiang.helper.Post HTTP Post
func ProcessPost(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	var data interface{}
	var headers = map[string]string{}
	url := process.ArgsString(0)
	if process.NumOfArgs() > 1 {
		data = process.Args[1]
	}
	if process.NumOfArgs() > 2 {
		inputHeaders := process.ArgsMap(2)
		for name, value := range inputHeaders {
			headers[name] = fmt.Sprintf("%v", value)
		}
	}
	return RequestPost(url, data, headers)
}

// ProcessPostJSON  xiang.helper.PostJSON HTTP Post
func ProcessPostJSON(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	var data interface{}
	var headers = map[string]string{}
	url := process.ArgsString(0)
	if process.NumOfArgs() > 1 {
		data = process.Args[1]
	}
	if process.NumOfArgs() > 2 {
		inputHeaders := process.ArgsMap(2)
		for name, value := range inputHeaders {
			headers[name] = fmt.Sprintf("%v", value)
		}
	}
	return RequestPostJSON(url, data, headers)
}

// ProcessPut  xiang.helper.Put HTTP PUT
func ProcessPut(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	var data interface{}
	var headers = map[string]string{}
	url := process.ArgsString(0)
	if process.NumOfArgs() > 1 {
		data = process.Args[1]
	}
	if process.NumOfArgs() > 2 {
		inputHeaders := process.ArgsMap(2)
		for name, value := range inputHeaders {
			headers[name] = fmt.Sprintf("%v", value)
		}
	}
	return RequestPut(url, data, headers)
}

// ProcessPutJSON  xiang.helper.PutJSON HTTP PUT
func ProcessPutJSON(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	var data interface{}
	var headers = map[string]string{}
	url := process.ArgsString(0)
	if process.NumOfArgs() > 1 {
		data = process.Args[1]
	}
	if process.NumOfArgs() > 2 {
		inputHeaders := process.ArgsMap(2)
		for name, value := range inputHeaders {
			headers[name] = fmt.Sprintf("%v", value)
		}
	}
	return RequestPutJSON(url, data, headers)
}

// ProcessSend  xiang.helper.Send HTTP Send
func ProcessSend(process *gou.Process) interface{} {
	process.ValidateArgNums(2)
	var data interface{}
	var params = map[string]interface{}{}
	var headers = map[string]string{}
	method := process.ArgsString(0)
	url := process.ArgsString(1)
	if process.NumOfArgs() > 2 {
		params = process.ArgsMap(2)
	}

	if process.NumOfArgs() > 3 {
		data = process.Args[3]
	}

	if process.NumOfArgs() > 4 {
		inputHeaders := process.ArgsMap(4)
		for name, value := range inputHeaders {
			headers[name] = fmt.Sprintf("%v", value)
		}
	}
	return RequestSend(method, url, params, data, headers)
}

// ProcessGet  xiang.helper.Get HTTP Get
func ProcessGet(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	var params = map[string]interface{}{}
	var headers = map[string]string{}
	url := process.ArgsString(0)
	if process.NumOfArgs() > 1 {
		params = process.ArgsMap(1)
	}
	if process.NumOfArgs() > 2 {
		inputHeaders := process.ArgsMap(2)
		for name, value := range inputHeaders {
			headers[name] = fmt.Sprintf("%v", value)
		}
	}

	return RequestGet(url, params, headers)
}
