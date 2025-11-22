package i18n

// init registers built-in global messages
func init() {
	// Initialize __global__ if not exists
	if Locales["__global__"] == nil {
		Locales["__global__"] = make(map[string]I18n)
	}

	// Built-in English messages
	Locales["__global__"]["en"] = I18n{
		Locale: "en",
		Messages: map[string]any{
			// Assistant: agent.go Stream() function
			"assistant.agent.stream.label":        "Assistant {{name}}",
			"assistant.agent.stream.description":  "Assistant {{name}} is processing the request",
			"assistant.agent.stream.history":      "Get Chat History",
			"assistant.agent.stream.capabilities": "Get Connector Capabilities",
			"assistant.agent.stream.create_hook":  "Call Create Hook",
			"assistant.agent.stream.closing":      "Closing output (root call)",
			"assistant.agent.stream.skipping":     "Skipping output close (nested call)",
			"assistant.agent.stream.close_error":  "Failed to close output",

			// LLM: providers/openai/openai.go Stream() function
			"llm.openai.stream.label":        "LLM %s",
			"llm.openai.stream.description":  "LLM %s is processing the request",
			"llm.openai.stream.starting":     "Starting stream request",
			"llm.openai.stream.request":      "Stream Request",
			"llm.openai.stream.retry":        "Stream request failed, retrying",
			"llm.openai.stream.api_error":    "OpenAI API returned error response",
			"llm.openai.stream.error":        "OpenAI Stream Error",
			"llm.openai.stream.no_data":      "Request body that caused empty response",
			"llm.openai.stream.no_data_info": "Request details",
			"llm.openai.post.api_error":      "OpenAI API error response",

			// LLM: handlers/stream.go (general LLM stream handler)
			"llm.handlers.stream.info":       "LLM Stream",
			"llm.handlers.stream.raw_output": "LLM Raw Output",

			// Output: adapters/openai/writer.go
			"output.openai.writer.sending_chunk": "Sending chunk to client",
			"output.openai.writer.sending_done":  "Sending [DONE] to client",
			"output.openai.writer.adapt_error":   "Failed to adapt message",
			"output.openai.writer.chunk_error":   "Failed to send chunk",
			"output.openai.writer.group_error":   "Failed to write message in group",
			"output.openai.writer.send_error":    "Failed to send data to client",
			"output.openai.writer.marshal_error": "Failed to marshal chunk",
			"output.openai.writer.done_error":    "Failed to send [DONE] to client",

			// Output: adapters/cui/writer.go
			"output.cui.writer.sending_chunk": "Sending chunk to client",
			"output.cui.writer.adapt_error":   "Failed to adapt message",
			"output.cui.writer.chunk_error":   "Failed to send chunk",
			"output.cui.writer.group_error":   "Failed to send message group",
			"output.cui.writer.send_error":    "Failed to send data to client",
			"output.cui.writer.marshal_error": "Failed to marshal chunk",

			// Output: Stream event messages
			"output.stream_start": "Assistant is processing",
			"output.view_trace":   "View process",

			// Common status messages
			"common.status.processing": "Processing",
			"common.status.completed":  "Completed",
			"common.status.failed":     "Failed",
			"common.status.retrying":   "Retrying",
		},
	}

	// Built-in Chinese (Simplified) messages
	Locales["__global__"]["zh-cn"] = I18n{
		Locale: "zh-cn",
		Messages: map[string]any{
			// Assistant: agent.go Stream() function
			"assistant.agent.stream.label":        "助手 {{name}}",
			"assistant.agent.stream.description":  "助手 {{name}} 正在处理请求",
			"assistant.agent.stream.history":      "获取聊天历史",
			"assistant.agent.stream.capabilities": "获取连接器能力",
			"assistant.agent.stream.create_hook":  "调用 Create Hook",
			"assistant.agent.stream.closing":      "关闭输出（根调用）",
			"assistant.agent.stream.skipping":     "跳过输出关闭（嵌套调用）",
			"assistant.agent.stream.close_error":  "关闭输出失败",

			// LLM: providers/openai/openai.go Stream() function
			"llm.openai.stream.label":        "LLM %s",
			"llm.openai.stream.description":  "LLM %s 正在处理请求",
			"llm.openai.stream.starting":     "开始流式请求",
			"llm.openai.stream.request":      "流式请求",
			"llm.openai.stream.retry":        "流式请求失败，正在重试",
			"llm.openai.stream.api_error":    "OpenAI API 返回错误响应",
			"llm.openai.stream.error":        "OpenAI 流错误",
			"llm.openai.stream.no_data":      "导致空响应的请求体",
			"llm.openai.stream.no_data_info": "请求详情",
			"llm.openai.post.api_error":      "OpenAI API 错误响应",

			// LLM: handlers/stream.go (general LLM stream handler)
			"llm.handlers.stream.info":       "LLM 流式输出",
			"llm.handlers.stream.raw_output": "LLM 原始输出",

			// Output: adapters/openai/writer.go
			"output.openai.writer.sending_chunk": "向客户端发送数据块",
			"output.openai.writer.sending_done":  "向客户端发送 [DONE]",
			"output.openai.writer.adapt_error":   "适配消息失败",
			"output.openai.writer.chunk_error":   "发送数据块失败",
			"output.openai.writer.group_error":   "写入消息组中的消息失败",
			"output.openai.writer.send_error":    "发送数据到客户端失败",
			"output.openai.writer.marshal_error": "序列化数据块失败",
			"output.openai.writer.done_error":    "发送 [DONE] 到客户端失败",

			// Output: adapters/cui/writer.go
			"output.cui.writer.sending_chunk": "向客户端发送数据块",
			"output.cui.writer.adapt_error":   "适配消息失败",
			"output.cui.writer.chunk_error":   "发送数据块失败",
			"output.cui.writer.group_error":   "发送消息组失败",
			"output.cui.writer.send_error":    "发送数据到客户端失败",
			"output.cui.writer.marshal_error": "序列化数据块失败",

			// Output: Stream event messages
			"output.stream_start": "智能体正在处理",
			"output.view_trace":   "查看处理详情",

			// Common status messages
			"common.status.processing": "处理中",
			"common.status.completed":  "已完成",
			"common.status.failed":     "失败",
			"common.status.retrying":   "重试中",
		},
	}

	// Built-in Chinese (short code) - same as zh-cn
	Locales["__global__"]["zh"] = I18n{
		Locale: "zh",
		Messages: map[string]any{
			// Assistant: agent.go Stream() function
			"assistant.agent.stream.label":        "助手 {{name}}",
			"assistant.agent.stream.description":  "助手 {{name}} 正在处理请求",
			"assistant.agent.stream.history":      "获取聊天历史",
			"assistant.agent.stream.capabilities": "获取连接器能力",
			"assistant.agent.stream.create_hook":  "调用 Create Hook",
			"assistant.agent.stream.closing":      "关闭输出（根调用）",
			"assistant.agent.stream.skipping":     "跳过输出关闭（嵌套调用）",
			"assistant.agent.stream.close_error":  "关闭输出失败",

			// LLM: providers/openai/openai.go Stream() function
			"llm.openai.stream.label":        "LLM %s",
			"llm.openai.stream.description":  "LLM %s 正在处理请求",
			"llm.openai.stream.starting":     "开始流式请求",
			"llm.openai.stream.request":      "流式请求",
			"llm.openai.stream.retry":        "流式请求失败，正在重试",
			"llm.openai.stream.api_error":    "OpenAI API 返回错误响应",
			"llm.openai.stream.error":        "OpenAI 流错误",
			"llm.openai.stream.no_data":      "导致空响应的请求体",
			"llm.openai.stream.no_data_info": "请求详情",
			"llm.openai.post.api_error":      "OpenAI API 错误响应",

			// LLM: handlers/stream.go (general LLM stream handler)
			"llm.handlers.stream.info":       "LLM 流式输出",
			"llm.handlers.stream.raw_output": "LLM 原始输出",

			// Output: adapters/openai/writer.go
			"output.openai.writer.sending_chunk": "向客户端发送数据块",
			"output.openai.writer.sending_done":  "向客户端发送 [DONE]",
			"output.openai.writer.adapt_error":   "适配消息失败",
			"output.openai.writer.chunk_error":   "发送数据块失败",
			"output.openai.writer.group_error":   "写入消息组中的消息失败",
			"output.openai.writer.send_error":    "发送数据到客户端失败",
			"output.openai.writer.marshal_error": "序列化数据块失败",
			"output.openai.writer.done_error":    "发送 [DONE] 到客户端失败",

			// Output: adapters/cui/writer.go
			"output.cui.writer.sending_chunk": "向客户端发送数据块",
			"output.cui.writer.adapt_error":   "适配消息失败",
			"output.cui.writer.chunk_error":   "发送数据块失败",
			"output.cui.writer.group_error":   "发送消息组失败",
			"output.cui.writer.send_error":    "发送数据到客户端失败",
			"output.cui.writer.marshal_error": "序列化数据块失败",

			// Output: Stream event messages
			"output.stream_start": "智能体正在处理",
			"output.view_trace":   "查看处理详情",

			// Common status messages
			"common.status.processing": "处理中",
			"common.status.completed":  "已完成",
			"common.status.failed":     "失败",
			"common.status.retrying":   "重试中",
		},
	}
}
