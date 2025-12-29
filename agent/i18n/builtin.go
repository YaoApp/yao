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
			"assistant.agent.stream.label":           "{{name}}",
			"assistant.agent.stream.description":     "{{name}} is processing the request",
			"assistant.agent.stream.history":         "Get Chat History",
			"assistant.agent.stream.capabilities":    "Get Connector Capabilities",
			"assistant.agent.stream.create_hook":     "Call Create Hook",
			"assistant.agent.stream.closing":         "Closing output (root call)",
			"assistant.agent.stream.skipping":        "Skipping output close (nested call)",
			"assistant.agent.stream.close_error":     "Failed to close output",
			"assistant.agent.completion.label":       "Agent Completion",
			"assistant.agent.completion.description": "Final output from {{name}}",

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

			// MCP: context/mcp.go - Resource operations
			"mcp.list_resources.label":       "MCP: List Resources",
			"mcp.list_resources.description": "List resources from MCP client '%s'",
			"mcp.read_resource.label":        "MCP: Read Resource",
			"mcp.read_resource.description":  "Read resource '%s' from MCP client '%s'",

			// MCP: context/mcp.go - Tool operations
			"mcp.list_tools.label":                "MCP: List Tools",
			"mcp.list_tools.description":          "List tools from MCP client '%s'",
			"mcp.call_tool.label":                 "MCP: Call Tool",
			"mcp.call_tool.description":           "Call tool '%s' from MCP client '%s'",
			"mcp.call_tools.label":                "MCP: Call Tools",
			"mcp.call_tools.description":          "Call %d tools sequentially from MCP client '%s'",
			"mcp.call_tools_parallel.label":       "MCP: Call Tools (Parallel)",
			"mcp.call_tools_parallel.description": "Call %d tools in parallel from MCP client '%s'",

			// MCP: context/mcp.go - Prompt operations
			"mcp.list_prompts.label":       "MCP: List Prompts",
			"mcp.list_prompts.description": "List prompts from MCP client '%s'",
			"mcp.get_prompt.label":         "MCP: Get Prompt",
			"mcp.get_prompt.description":   "Get prompt '%s' from MCP client '%s'",

			// MCP: context/mcp.go - Sample operations
			"mcp.list_samples.label":       "MCP: List Samples",
			"mcp.list_samples.description": "List samples for '%s' from MCP client '%s'",
			"mcp.get_sample.label":         "MCP: Get Sample",
			"mcp.get_sample.description":   "Get sample #%d for '%s' from MCP client '%s'",

			// KB: Chat collection
			"kb.chat.name":        "Chat Knowledge Base",
			"kb.chat.description": "Auto-created knowledge base collection for chat sessions",

			// Content: content/image/image.go - Image processing messages
			"content.image.analyzing": "Analyzing image...",

			// Content: content/pdf/pdf.go - PDF processing messages
			"content.pdf.analyzing_page": "Analyzing PDF page %d/%d...",

			// Search: assistant/search.go - Output messages
			"search.loading":     "Searching...",
			"search.success":     "Found %d references",
			"search.success.one": "Found 1 reference",
			"search.partial":     "Found %d references (some sources failed)",
			"search.failed":      "Search failed",
			"search.no_results":  "No references found",

			// Search Intent: assistant/search.go - Intent detection messages
			"search.intent.loading":     "Checking if references are needed...",
			"search.intent.need_search": "Searching for references...",
			"search.intent.no_search":   "No references needed",

			// Keyword Extraction: assistant/search.go - Keyword extraction messages
			"search.keyword.loading": "Analyzing conversation...",
			"search.keyword.done":    "Analysis complete",

			// Search: assistant/search.go - Trace labels
			"search.trace.label":           "Search",
			"search.trace.description":     "Search the web and knowledge base for relevant information",
			"search.trace.web.label":       "Web Search",
			"search.trace.web.description": "Searching the web",
			"search.trace.kb.label":        "KB Search",
			"search.trace.kb.description":  "Searching knowledge base",
			"search.trace.db.label":        "DB Search",
			"search.trace.db.description":  "Searching database",
		},
	}

	// Built-in Chinese (Simplified) messages
	Locales["__global__"]["zh-cn"] = I18n{
		Locale: "zh-cn",
		Messages: map[string]any{
			// Assistant: agent.go Stream() function
			"assistant.agent.stream.label":           "{{name}}",
			"assistant.agent.stream.description":     "{{name}} 正在处理请求",
			"assistant.agent.stream.history":         "获取聊天历史",
			"assistant.agent.stream.capabilities":    "获取连接器能力",
			"assistant.agent.stream.create_hook":     "调用 Create Hook",
			"assistant.agent.stream.closing":         "关闭输出（根调用）",
			"assistant.agent.stream.skipping":        "跳过输出关闭（嵌套调用）",
			"assistant.agent.stream.close_error":     "关闭输出失败",
			"assistant.agent.completion.label":       "智能体完成",
			"assistant.agent.completion.description": "{{name}} 最终输出",

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

			// KB: Chat collection
			"kb.chat.name":        "聊天知识库",
			"kb.chat.description": "自动为聊天会话创建的知识库集合",

			// Content: content/image/image.go - Image processing messages
			"content.image.analyzing": "正在分析图片...",

			// Content: content/pdf/pdf.go - PDF processing messages
			"content.pdf.analyzing_page": "正在分析 PDF 第 %d/%d 页...",

			// Search: assistant/search.go - Output messages
			"search.loading":     "正在搜索...",
			"search.success":     "找到 %d 条参考资料",
			"search.success.one": "找到 1 条参考资料",
			"search.partial":     "找到 %d 条参考资料（部分来源失败）",
			"search.failed":      "搜索失败",
			"search.no_results":  "未找到相关资料",

			// Search Intent: assistant/search.go - Intent detection messages
			"search.intent.loading":     "检查是否需要查询资料...",
			"search.intent.need_search": "正在查询相关资料...",
			"search.intent.no_search":   "无需查询资料",

			// Keyword Extraction: assistant/search.go - Keyword extraction messages
			"search.keyword.loading": "正在分析对话内容...",
			"search.keyword.done":    "分析完成",

			// Search: assistant/search.go - Trace labels
			"search.trace.label":           "搜索",
			"search.trace.description":     "搜索网络和知识库获取相关信息",
			"search.trace.web.label":       "网页搜索",
			"search.trace.web.description": "搜索网页获取相关信息",
			"search.trace.kb.label":        "知识库搜索",
			"search.trace.kb.description":  "搜索知识库获取相关信息",
			"search.trace.db.label":        "数据库搜索",
			"search.trace.db.description":  "搜索数据库获取相关信息",
		},
	}

	// Built-in Chinese (short code) - same as zh-cn
	Locales["__global__"]["zh"] = I18n{
		Locale: "zh",
		Messages: map[string]any{
			// Assistant: agent.go Stream() function
			"assistant.agent.stream.label":           "{{name}}",
			"assistant.agent.stream.description":     "{{name}} 正在处理请求",
			"assistant.agent.stream.history":         "获取聊天历史",
			"assistant.agent.stream.capabilities":    "获取连接器能力",
			"assistant.agent.stream.create_hook":     "调用 Create Hook",
			"assistant.agent.stream.closing":         "关闭输出（根调用）",
			"assistant.agent.stream.skipping":        "跳过输出关闭（嵌套调用）",
			"assistant.agent.stream.close_error":     "关闭输出失败",
			"assistant.agent.completion.label":       "智能体完成",
			"assistant.agent.completion.description": "{{name}} 最终输出",

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

			// MCP: context/mcp.go - Resource operations
			"mcp.list_resources.label":       "MCP: 列出资源",
			"mcp.list_resources.description": "从 MCP 客户端 '%s' 列出资源",
			"mcp.read_resource.label":        "MCP: 读取资源",
			"mcp.read_resource.description":  "从 MCP 客户端 '%s' 读取资源 '%s'",

			// MCP: context/mcp.go - Tool operations
			"mcp.list_tools.label":                "MCP: 列出工具",
			"mcp.list_tools.description":          "从 MCP 客户端 '%s' 列出工具",
			"mcp.call_tool.label":                 "MCP: 调用工具",
			"mcp.call_tool.description":           "从 MCP 客户端 '%s' 调用工具 '%s'",
			"mcp.call_tools.label":                "MCP: 调用工具",
			"mcp.call_tools.description":          "从 MCP 客户端 '%s' 顺序调用 %d 个工具",
			"mcp.call_tools_parallel.label":       "MCP: 调用工具（并行）",
			"mcp.call_tools_parallel.description": "从 MCP 客户端 '%s' 并行调用 %d 个工具",

			// MCP: context/mcp.go - Prompt operations
			"mcp.list_prompts.label":       "MCP: 列出提示词",
			"mcp.list_prompts.description": "从 MCP 客户端 '%s' 列出提示词",
			"mcp.get_prompt.label":         "MCP: 获取提示词",
			"mcp.get_prompt.description":   "从 MCP 客户端 '%s' 获取提示词 '%s'",

			// MCP: context/mcp.go - Sample operations
			"mcp.list_samples.label":       "MCP: 列出示例",
			"mcp.list_samples.description": "从 MCP 客户端 '%s' 列出 '%s' 的示例",
			"mcp.get_sample.label":         "MCP: 获取示例",
			"mcp.get_sample.description":   "从 MCP 客户端 '%s' 获取 '%s' 的第 %d 个示例",

			// KB: Chat collection
			"kb.chat.name":        "聊天知识库",
			"kb.chat.description": "自动为聊天会话创建的知识库集合",

			// Content: content/image/image.go - Image processing messages
			"content.image.analyzing": "正在分析图片...",

			// Content: content/pdf/pdf.go - PDF processing messages
			"content.pdf.analyzing_page": "正在分析 PDF 第 %d/%d 页...",

			// Search: assistant/search.go - Output messages
			"search.loading":     "正在搜索...",
			"search.success":     "找到 %d 条参考资料",
			"search.success.one": "找到 1 条参考资料",
			"search.partial":     "找到 %d 条参考资料（部分来源失败）",
			"search.failed":      "搜索失败",
			"search.no_results":  "未找到相关资料",

			// Search Intent: assistant/search.go - Intent detection messages
			"search.intent.loading":     "检查是否需要查询资料...",
			"search.intent.need_search": "正在查询相关资料...",
			"search.intent.no_search":   "无需查询资料",

			// Keyword Extraction: assistant/search.go - Keyword extraction messages
			"search.keyword.loading": "正在分析对话内容...",
			"search.keyword.done":    "分析完成",

			// Search: assistant/search.go - Trace labels
			"search.trace.label":           "搜索",
			"search.trace.description":     "搜索网络和知识库获取相关信息",
			"search.trace.web.label":       "网页搜索",
			"search.trace.web.description": "搜索网页获取相关信息",
			"search.trace.kb.label":        "知识库搜索",
			"search.trace.kb.description":  "搜索知识库获取相关信息",
			"search.trace.db.label":        "数据库搜索",
			"search.trace.db.description":  "搜索数据库获取相关信息",
		},
	}
}
