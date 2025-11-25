package jsapi

// func init() {
// 	// Auto-register Output JavaScript API when package is imported
// 	v8.RegisterFunction("Output", ExportFunction)
// }

// // Usage from JavaScript:
// //
// //	const output = new Output(ctx)
// //	output.Send({ type: "text", props: { content: "Hello" } })
// //	output.Send("Hello") // shorthand for text message
// //	output.SendGroup({ id: "group1", messages: [...] })
// //
// // Objects:
// //   - Output: Output manager (constructor)

// // ExportFunction exports the Output constructor function template
// // This is used by v8.RegisterFunction
// func ExportFunction(iso *v8go.Isolate) *v8go.FunctionTemplate {
// 	return v8go.NewFunctionTemplate(iso, outputConstructor)
// }

// // outputConstructor is the JavaScript constructor for Output
// // Usage: new Output(ctx)
// func outputConstructor(info *v8go.FunctionCallbackInfo) *v8go.Value {
// 	v8ctx := info.Context()
// 	args := info.Args()

// 	// Require ctx argument
// 	if len(args) < 1 {
// 		return bridge.JsException(v8ctx, "Output constructor requires a context argument")
// 	}

// 	// Get the context object from JavaScript
// 	ctxObj, err := args[0].AsObject()
// 	if err != nil {
// 		return bridge.JsException(v8ctx, fmt.Sprintf("context must be an object: %s", err))
// 	}

// 	// Get the goValueID from internal field (index 0)
// 	if ctxObj.InternalFieldCount() < 1 {
// 		return bridge.JsException(v8ctx, "context object is missing internal fields")
// 	}

// 	goValueIDValue := ctxObj.GetInternalField(0)
// 	if goValueIDValue == nil || !goValueIDValue.IsString() {
// 		return bridge.JsException(v8ctx, "context object is missing goValueID")
// 	}

// 	goValueID := goValueIDValue.String()

// 	// Retrieve the Go context object from bridge registry
// 	goObj := bridge.GetGoObject(goValueID)
// 	if goObj == nil {
// 		return bridge.JsException(v8ctx, "context object not found in registry")
// 	}

// 	// Type assert to *agentContext.Context
// 	ctx, ok := goObj.(*agentContext.Context)
// 	if !ok {
// 		return bridge.JsException(v8ctx, fmt.Sprintf("object is not a Context, got %T", goObj))
// 	}

// 	// Create output object
// 	outputObj, err := NewOutputObject(v8ctx, ctx)
// 	if err != nil {
// 		return bridge.JsException(v8ctx, err.Error())
// 	}

// 	return outputObj
// }

// // NewOutputObject creates a JavaScript Output object
// func NewOutputObject(v8ctx *v8go.Context, ctx *agentContext.Context) (*v8go.Value, error) {
// 	jsObject := v8go.NewObjectTemplate(v8ctx.Isolate())

// 	// Set internal field count to 1 to store the __go_id
// 	// Internal fields are not accessible from JavaScript, providing better security
// 	jsObject.SetInternalFieldCount(1)

// 	// Register context in global bridge registry for efficient Go object retrieval
// 	// The goValueID will be stored in internal field (index 0) after instance creation
// 	goValueID := bridge.RegisterGoObject(ctx)

// 	// Set methods
// 	jsObject.Set("Send", outputSendMethod(v8ctx.Isolate(), ctx))
// 	jsObject.Set("SendGroup", outputSendGroupMethod(v8ctx.Isolate(), ctx))

// 	// Set release function that will be called when JavaScript object is released
// 	jsObject.Set("__release", outputGoRelease(v8ctx.Isolate()))

// 	// Create instance
// 	instance, err := jsObject.NewInstance(v8ctx)
// 	if err != nil {
// 		// Clean up: release from global registry if instance creation failed
// 		bridge.ReleaseGoObject(goValueID)
// 		return nil, err
// 	}

// 	// Store the goValueID in internal field (index 0)
// 	// This is not accessible from JavaScript, providing better security
// 	obj, err := instance.Value.AsObject()
// 	if err != nil {
// 		bridge.ReleaseGoObject(goValueID)
// 		return nil, err
// 	}

// 	err = obj.SetInternalField(0, goValueID)
// 	if err != nil {
// 		bridge.ReleaseGoObject(goValueID)
// 		return nil, err
// 	}

// 	return instance.Value, nil
// }

// // outputGoRelease releases the Go object from the global bridge registry
// // It retrieves the goValueID from internal field (index 0) and releases the Go object
// func outputGoRelease(iso *v8go.Isolate) *v8go.FunctionTemplate {
// 	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
// 		// Get the output object (this)
// 		thisObj, err := info.This().AsObject()
// 		if err == nil && thisObj.InternalFieldCount() > 0 {
// 			// Get goValueID from internal field (index 0)
// 			goValueIDValue := thisObj.GetInternalField(0)
// 			if goValueIDValue != nil && goValueIDValue.IsString() {
// 				goValueID := goValueIDValue.String()
// 				// Release from global bridge registry
// 				bridge.ReleaseGoObject(goValueID)
// 			}
// 		}

// 		return v8go.Undefined(info.Context().Isolate())
// 	})
// }

// // outputSendMethod implements the Send method
// // Usage: output.Send(message)
// // message can be an object with { type: string, props: object, ... } or a simple string (will be converted to text message)
// func outputSendMethod(iso *v8go.Isolate, ctx *agentContext.Context) *v8go.FunctionTemplate {
// 	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
// 		v8ctx := info.Context()
// 		args := info.Args()

// 		if len(args) < 1 {
// 			return bridge.JsException(v8ctx, "Send requires a message argument")
// 		}

// 		// Parse message argument
// 		msg, err := parseMessage(v8ctx, args[0])
// 		if err != nil {
// 			return bridge.JsException(v8ctx, fmt.Sprintf("invalid message: %s", err))
// 		}

// 		// Call output.Send
// 		if err := output.Send(ctx, msg); err != nil {
// 			return bridge.JsException(v8ctx, fmt.Sprintf("Send failed: %s", err))
// 		}

// 		return info.This().Value
// 	})
// }

// // outputSendGroupMethod implements the SendGroup method
// // Usage: output.SendGroup(group)
// // group must be an object with { id: string, messages: [], ... }
// func outputSendGroupMethod(iso *v8go.Isolate, ctx *agentContext.Context) *v8go.FunctionTemplate {
// 	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
// 		v8ctx := info.Context()
// 		args := info.Args()

// 		if len(args) < 1 {
// 			return bridge.JsException(v8ctx, "SendGroup requires a group argument")
// 		}

// 		// Parse group argument
// 		group, err := parseGroup(v8ctx, args[0])
// 		if err != nil {
// 			return bridge.JsException(v8ctx, fmt.Sprintf("invalid group: %s", err))
// 		}

// 		// Call output.SendGroup
// 		if err := output.SendGroup(ctx, group); err != nil {
// 			return bridge.JsException(v8ctx, fmt.Sprintf("SendGroup failed: %s", err))
// 		}

// 		return info.This().Value
// 	})
// }

// // parseMessage parses a JavaScript value into a message.Message
// func parseMessage(v8ctx *v8go.Context, jsValue *v8go.Value) (*message.Message, error) {
// 	// Handle string shorthand: convert to text message
// 	if jsValue.IsString() {
// 		return &message.Message{
// 			Type: message.TypeText,
// 			Props: map[string]interface{}{
// 				"content": jsValue.String(),
// 			},
// 		}, nil
// 	}

// 	// Handle object
// 	if !jsValue.IsObject() {
// 		return nil, fmt.Errorf("message must be a string or object")
// 	}

// 	// Convert to Go map
// 	goValue, err := bridge.GoValue(jsValue, v8ctx)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to convert message: %w", err)
// 	}

// 	msgMap, ok := goValue.(map[string]interface{})
// 	if !ok {
// 		return nil, fmt.Errorf("message must be an object")
// 	}

// 	// Build message
// 	msg := &message.Message{}

// 	// Type field (required)
// 	if msgType, ok := msgMap["type"].(string); ok {
// 		msg.Type = msgType
// 	} else {
// 		return nil, fmt.Errorf("message.type is required and must be a string")
// 	}

// 	// Props field (optional)
// 	if props, ok := msgMap["props"].(map[string]interface{}); ok {
// 		msg.Props = props
// 	}

// 	// Optional fields
// 	if id, ok := msgMap["id"].(string); ok {
// 		msg.ID = id
// 	}
// 	if delta, ok := msgMap["delta"].(bool); ok {
// 		msg.Delta = delta
// 	}
// 	if done, ok := msgMap["done"].(bool); ok {
// 		msg.Done = done
// 	}
// 	if deltaPath, ok := msgMap["delta_path"].(string); ok {
// 		msg.DeltaPath = deltaPath
// 	}
// 	if deltaAction, ok := msgMap["delta_action"].(string); ok {
// 		msg.DeltaAction = deltaAction
// 	}
// 	if typeChange, ok := msgMap["type_change"].(bool); ok {
// 		msg.TypeChange = typeChange
// 	}
// 	if groupID, ok := msgMap["group_id"].(string); ok {
// 		msg.GroupID = groupID
// 	}
// 	if groupStart, ok := msgMap["group_start"].(bool); ok {
// 		msg.GroupStart = groupStart
// 	}
// 	if groupEnd, ok := msgMap["group_end"].(bool); ok {
// 		msg.GroupEnd = groupEnd
// 	}

// 	// Metadata (optional)
// 	if metadataMap, ok := msgMap["metadata"].(map[string]interface{}); ok {
// 		metadata := &message.Metadata{}
// 		if timestamp, ok := metadataMap["timestamp"].(float64); ok {
// 			metadata.Timestamp = int64(timestamp)
// 		}
// 		if sequence, ok := metadataMap["sequence"].(float64); ok {
// 			metadata.Sequence = int(sequence)
// 		}
// 		if traceID, ok := metadataMap["trace_id"].(string); ok {
// 			metadata.TraceID = traceID
// 		}
// 		msg.Metadata = metadata
// 	}

// 	return msg, nil
// }

// // parseGroup parses a JavaScript value into a message.Group
// func parseGroup(v8ctx *v8go.Context, jsValue *v8go.Value) (*message.Group, error) {
// 	// Must be an object
// 	if !jsValue.IsObject() {
// 		return nil, fmt.Errorf("group must be an object")
// 	}

// 	// Convert to Go map
// 	goValue, err := bridge.GoValue(jsValue, v8ctx)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to convert group: %w", err)
// 	}

// 	groupMap, ok := goValue.(map[string]interface{})
// 	if !ok {
// 		return nil, fmt.Errorf("group must be an object")
// 	}

// 	// Build group
// 	group := &message.Group{}

// 	// ID field (required)
// 	if id, ok := groupMap["id"].(string); ok {
// 		group.ID = id
// 	} else {
// 		return nil, fmt.Errorf("group.id is required and must be a string")
// 	}

// 	// Messages field (required)
// 	if messagesArray, ok := groupMap["messages"].([]interface{}); ok {
// 		group.Messages = make([]*message.Message, 0, len(messagesArray))
// 		for i, msgInterface := range messagesArray {
// 			// Convert to map
// 			msgMap, ok := msgInterface.(map[string]interface{})
// 			if !ok {
// 				return nil, fmt.Errorf("group.messages[%d] must be an object", i)
// 			}

// 			// Convert map to Message
// 			msg := &message.Message{}

// 			// Type field (required)
// 			if msgType, ok := msgMap["type"].(string); ok {
// 				msg.Type = msgType
// 			} else {
// 				return nil, fmt.Errorf("group.messages[%d].type is required", i)
// 			}

// 			// Props field (optional)
// 			if props, ok := msgMap["props"].(map[string]interface{}); ok {
// 				msg.Props = props
// 			}

// 			// Optional fields
// 			if id, ok := msgMap["id"].(string); ok {
// 				msg.ID = id
// 			}
// 			if delta, ok := msgMap["delta"].(bool); ok {
// 				msg.Delta = delta
// 			}
// 			if done, ok := msgMap["done"].(bool); ok {
// 				msg.Done = done
// 			}
// 			if deltaPath, ok := msgMap["delta_path"].(string); ok {
// 				msg.DeltaPath = deltaPath
// 			}
// 			if deltaAction, ok := msgMap["delta_action"].(string); ok {
// 				msg.DeltaAction = deltaAction
// 			}
// 			if typeChange, ok := msgMap["type_change"].(bool); ok {
// 				msg.TypeChange = typeChange
// 			}
// 			if groupID, ok := msgMap["group_id"].(string); ok {
// 				msg.GroupID = groupID
// 			}
// 			if groupStart, ok := msgMap["group_start"].(bool); ok {
// 				msg.GroupStart = groupStart
// 			}
// 			if groupEnd, ok := msgMap["group_end"].(bool); ok {
// 				msg.GroupEnd = groupEnd
// 			}

// 			// Metadata (optional)
// 			if metadataMap, ok := msgMap["metadata"].(map[string]interface{}); ok {
// 				metadata := &message.Metadata{}
// 				if timestamp, ok := metadataMap["timestamp"].(float64); ok {
// 					metadata.Timestamp = int64(timestamp)
// 				}
// 				if sequence, ok := metadataMap["sequence"].(float64); ok {
// 					metadata.Sequence = int(sequence)
// 				}
// 				if traceID, ok := metadataMap["trace_id"].(string); ok {
// 					metadata.TraceID = traceID
// 				}
// 				msg.Metadata = metadata
// 			}

// 			group.Messages = append(group.Messages, msg)
// 		}
// 	} else {
// 		return nil, fmt.Errorf("group.messages is required and must be an array")
// 	}

// 	// Metadata (optional)
// 	if metadataMap, ok := groupMap["metadata"].(map[string]interface{}); ok {
// 		metadata := &message.Metadata{}
// 		if timestamp, ok := metadataMap["timestamp"].(float64); ok {
// 			metadata.Timestamp = int64(timestamp)
// 		}
// 		if sequence, ok := metadataMap["sequence"].(float64); ok {
// 			metadata.Sequence = int(sequence)
// 		}
// 		if traceID, ok := metadataMap["trace_id"].(string); ok {
// 			metadata.TraceID = traceID
// 		}
// 		group.Metadata = metadata
// 	}

// 	return group, nil
// }
