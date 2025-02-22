package assistant

import (
	"context"
	"fmt"

	"github.com/fatih/color"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	v8plan "github.com/yaoapp/gou/runtime/v8/objects/plan"
	"rogchap.com/v8go"
)

// TaskFn is the task function
func TaskFn(plan_id string, task_id string, source bool, method string, args ...interface{}) (interface{}, error) {

	if !source {
		return v8plan.DefaultTaskFn(plan_id, task_id, source, method, args...)
	}

	// Data
	plan, err := v8plan.GetPlan(plan_id)
	if err != nil {
		return nil, err
	}

	global, ok := plan.Data().(*GlobalVariables)
	if !ok {
		return nil, fmt.Errorf("plan data is not a GlobalVariables")
	}

	if global.Assistant == nil {
		return nil, fmt.Errorf("assistant is not set")
	}

	if global.Assistant.Script == nil {
		return nil, fmt.Errorf("script is not set")
	}

	scriptCtx, err := global.Assistant.Script.NewContext(global.ChatContext.Sid, nil)
	if err != nil {
		return nil, err
	}
	defer scriptCtx.Close()

	// Initialize the object
	global.Assistant.InitObject(scriptCtx, global.GinContext, global.ChatContext, global.Contents)

	fnargs := []interface{}{plan_id, task_id}
	fnargs = append(fnargs, args...)

	// Execute the anonymous function
	return scriptCtx.CallAnonymousWith(context.Background(), method, fnargs...)

}

// SubscribeFn is the default subscribe function
func SubscribeFn(plan_id string, key string, value interface{}, source bool, method string, args ...interface{}) {

	if !source {
		v8plan.DefaultSubscribeFn(plan_id, key, value, source, method, args...)
		return
	}

	// Data
	plan, err := v8plan.GetPlan(plan_id)
	if err != nil {
		color.Red("Failed to get the plan: %s", err.Error())
		return
	}

	global, ok := plan.Data().(*GlobalVariables)
	if !ok {
		color.Red("plan data is not a GlobalVariables")
		return
	}

	if global.Assistant == nil {
		color.Red("assistant is not set")
		return
	}

	if global.Assistant.Script == nil {
		color.Red("script is not set")
		return
	}

	scriptCtx, err := global.Assistant.Script.NewContext(global.ChatContext.Sid, nil)
	if err != nil {
		color.Red("Failed to create the script context: %s", err.Error())
		return
	}
	defer scriptCtx.Close()

	fnargs := []interface{}{plan_id, key, value}
	fnargs = append(fnargs, args...)

	// Initialize the object
	global.Assistant.InitObject(scriptCtx, global.GinContext, global.ChatContext, global.Contents)
	_, err = scriptCtx.CallAnonymousWith(context.Background(), method, fnargs...)
	if err != nil {
		return
	}
}

// jsNewPlan create a plan object and return it
func jsPlan(info *v8go.FunctionCallbackInfo) *v8go.Value {

	global, err := global(info)
	if err != nil {
		return bridge.JsException(info.Context(), err.Error())
	}

	obj := newPlanObject()

	// Export the object
	objectTmpl := obj.ExportObject(info.Context().Isolate())

	args := info.Args()
	if len(args) < 1 {
		return bridge.JsException(info.Context(), "the first parameter should be a string")
	}

	if !args[0].IsString() {
		return bridge.JsException(info.Context(), "the first parameter should be a string")
	}

	id := args[0].String()
	plan, err := objectTmpl.NewInstance(info.Context())
	if err != nil {
		return bridge.JsException(info.Context(), fmt.Sprintf("failed to create plan object %s", err.Error()))
	}

	return obj.NewInstance(id, plan, global)
}

func newPlanObject() *v8plan.Object {
	obj := v8plan.New(v8plan.Options{
		TaskFn:      TaskFn,
		SubscribeFn: SubscribeFn,
	})
	return obj
}
