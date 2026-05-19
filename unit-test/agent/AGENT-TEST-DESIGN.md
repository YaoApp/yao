# Agent 测试体系设计文档

## 目录

- [1. 现状问题](#1-现状问题)
- [2. 设计原则](#2-设计原则)
- [3. 测试分层](#3-测试分层)
- [4. 测试 Assistant 体系设计](#4-测试-assistant-体系设计)
- [5. 子包测试覆盖矩阵](#5-子包测试覆盖矩阵)
- [6. Mock 策略](#6-mock-策略)
- [7. CI 分级方案](#7-ci-分级方案)
- [8. 统一 Prepare 迁移路径](#8-统一-prepare-迁移路径)
- [9. 实施顺序](#9-实施顺序)

---

## 1. 现状问题

- 203 个测试文件，大量硬编码 assistant ID（如 `tests.agent-caller`、`tests.mcpload`、`tests.sandbox.basic` 等）在磁盘上无对应 `package.yao`
- CI 仅跑 `sandbox/v2`，其余 agent 测试完全未纳入
- 测试加载方式割裂：sandbox/v2 用 `testprepare`，其余用 `testutils.PrepareAgent` + `test.Prepare`
- mock vs 真实 API 边界模糊

---

## 2. 设计原则

- **mock 优先**：验证逻辑（hook、解析、路由、合并）一律用 mock，只有 E2E 用真实 API
- **assistant 即测试契约**：每个测试 assistant 定义一个明确的测试场景，而非一个万能 assistant
- **统一加载路径**：所有测试走 `testprepare`（`unit-test/agent/app`），废弃 `testutils.PrepareAgent`
- **分层执行**：Tier 0（sandbox 基础设施验证）-> Tier 1（纯逻辑，无外部依赖）-> Tier 2（需 App + DB + Mock LLM）-> Tier 3（需 Docker/Tai + Mock LLM）-> Tier 4（需真实 LLM API）

---

## 3. 测试分层

```
Tier 0: Sandbox Infra    -- sandbox/v2 基础设施验证，无 Build Tag，失败则后续无意义
  |
  v
Tier 1: Pure Unit        -- 无需 App/DB/LLM，数据结构、解析、合并
  |
  v
Tier 2: App Integration  -- 需 App + DB + Mock LLM，Hook、Caller、History、Search
  |
  v
Tier 3: Sandbox           -- 需 Docker + Tai + Mock LLM，Claude/OpenCode/Yao Runner
  |
  v
Tier 4: E2E               -- 需真实 LLM API，端到端验证
```

### Tier 1 -- 工具函数单元测试（无 Prepare）

**极少数**纯工具函数测试，不加载应用、不需要 assistant：

| 能力             | 覆盖内容                                          | 对应 Go 包                      |
| ---------------- | ------------------------------------------------- | ------------------------------- |
| 消息序列化       | Message JSON marshal/unmarshal                    | `context/message*.go`           |
| MCP 工具名       | `MCPToolName`/`ParseMCPToolName` 往返             | `assistant/mcp.go`              |
| Output safe writer | 并发写安全                                      | `output/safe_writer.go`         |
| ID 生成          | NanoID、MessageID 等                              | `output/message/utils.go`       |

> **注意**：Yao 是 Runtime，脱离应用上下文测试容易遗漏真实问题。除上述纯工具函数外，其余测试都应加载应用（至少 `PrepareUnit`），归入 Tier 2+。

### Tier 2 -- App 集成测试（PrepareSandbox，使用 Mock LLM）

**绝大多数测试归入此层**。需要加载应用（App + DB + V8 + Mock LLM），使用测试 assistant 定义。包括但不限于：

- Hook 触发与响应解析（`HookCreateResponse`、`NextHookResponse`）
- 配置合并（`loadMap`、`mergeSearchConfig`、`uses` merge）
- LLM 连接器解析与角色路由
- Output 格式化（OpenAI/CUI adapter）
- I18n 模板解析与 locale 合并
- Memory 命名空间（需要 store 后端）
- Caller 编排（All/Any/Race 需要加载目标 assistant）
- Buffer/Stack 消息缓冲
- Content/Attachments 解析
- Search 各模式
- History/Resume

### Tier 3 -- Sandbox 测试（PrepareSandbox + Docker/Tai）

**已有良好覆盖**，保持现有 5 个 sandbox-v2 assistant。

### Tier 4 -- E2E 测试（PrepareE2E，真实 LLM）

**使用 Beta 团队的真实连接器**，验证端到端链路。

---

## 4. 测试 Assistant 体系设计

所有 assistant 定义位于 `unit-test/agent/app/assistants/tests/` 下。

### 4.1 Tier 2 -- 新建 Assistant（需 App + Mock LLM）

**基础设施要求：**

- 所有 Tier 2 assistant 使用 `openai.mock` 指向 mock-llm server
- 测试环境需注册 `openai.mock-validator`（第二个 mock connector，用于验证 connector 切换）
- MCP 测试需配置一个内置 echo MCP server（返回固定响应）
- 搜索测试中 `__yao.needsearch` 系统 agent 走 mock-llm，web handler 用 mock HTTP 或跳过实际网络调用
- 附件测试使用本地测试文件（放在 `unit-test/agent/app/data/` 下），不做真实网络下载

#### 4.1.1 基础加载与环境

| Assistant ID | connector | 关键文件 | Hook 行为 | 测试目标 | 对应 Go 实现 |
|-------------|-----------|---------|-----------|---------|-------------|
| `tests.simple-greeting` | `openai.mock` | `package.yao` | 无 hook | 最小化环境验证：加载、LLM 调用、响应 | `agent/load.go`, `assistant/agent.go` |
| `tests.no-prompt` | `openai.mock` | `package.yao`（无 prompts、无 mcp） | 无 hook | 边界场景：prompts=nil && mcp=nil 时 LLM 调用被跳过（`agent.go` L277） | `assistant/agent.go` |
| `tests.connector-resolve` | `use::default` | `package.yao` | 无 hook | `use::default` / `use::light` / `use::vision` 角色解析 | `llm/resolve.go` |

> `tests.no-prompt` 的 connector 字段不会被真正使用（LLM 调用被跳过），仅作为加载时必填项存在。

#### 4.1.2 Hook 系统

| Assistant ID | connector | 关键文件 | Hook 行为 | 测试目标 | 对应 Go 实现 |
|-------------|-----------|---------|-----------|---------|-------------|
| `tests.hook-echo` | `openai.mock` | `package.yao`, `src/index.ts` | Create: 加 `[hook-echo]` 前缀; Next: 返回 null（标准响应） | Create/Next hook 基础触发、脚本执行、返回值解析 | `hook/create.go`, `hook/next.go` |
| `tests.hook-error` | `openai.mock` | `package.yao`, `src/index.ts` | Create: `throw new Error("hook-crash")` | Hook JS 异常 -> Go error 传播 | `hook/script.go` |
| `tests.hook-delegate` | `openai.mock` | `package.yao`, `src/index.ts` | Create: 返回 `{ delegate: { agent_id: "tests.simple-greeting", messages: [...] } }` | Create Hook delegate：跳过 LLM，转发到目标 agent | `agent.go` delegate 分支, `next.go` handleDelegation |
| `tests.hook-next-delegate` | `openai.mock` | `package.yao`, `src/index.ts` | Next: 返回 `{ delegate: { agent_id: "tests.simple-greeting", messages: [...] } }` | Next Hook delegate：LLM 完成后再转发 | `next.go` processNextResponse delegate 路径 |
| `tests.hook-next-data` | `openai.mock` | `package.yao`, `src/index.ts` | Next: 返回 `{ data: { custom_field: "value" } }` | Next Hook custom data：自定义返回数据 | `next.go` processNextResponse data 路径 |
| `tests.hook-connector-override` | `openai.mock` | `package.yao`, `src/index.ts` | Create: 返回 `{ connector: "openai.mock-validator" }` | Hook 动态切换连接器（需第二个 mock connector） | `hook/create.go` applyOptionsAdjustments |
| `tests.hook-context-adjust` | `openai.mock` | `package.yao`, `src/index.ts` | Create: 返回 `{ locale: "zh-cn", theme: "dark", route: "/test", metadata: { key: "value" } }` | Create Hook 上下文覆盖：locale/theme/route/metadata | `hook/create.go` applyContextAdjustments |
| `tests.hook-search-control` | `openai.mock` | `package.yao`(`uses.search: auto`), `src/index.ts` | Create: 按输入关键字返回不同 search intent | Hook 控制搜索：enable/disable/custom intent | `assistant/search.go` |
| `tests.hook-prompt-preset` | `openai.mock` | `package.yao`, `prompts/`(含多 preset), `src/index.ts` | Create: 返回 `{ prompt_preset: "formal" }` | prompt_preset 选择 + 全局 prompt 合并 | `build.go` getAssistantPrompts + buildSystemPrompts |
| `tests.hook-disable-global-prompts` | `openai.mock` | `package.yao`(`disable_global_prompts: true`) | 无 hook | 禁用全局 prompt：验证 global prompts 不被注入 | `build.go` shouldDisableGlobalPrompts |

**processNextResponse 三条路径覆盖：**

```
processNextResponse(npc)
  ├── NextResponse == nil        → buildStandardResponse     ← tests.hook-echo (Next 返回 null)
  ├── NextResponse.Delegate != nil → handleDelegation        ← tests.hook-next-delegate
  └── NextResponse.Data != nil    → 自定义 data 返回        ← tests.hook-next-data
```

**applyContextAdjustments 四字段覆盖（tests.hook-context-adjust）：**

```
ctx.Locale   ← response.Locale
ctx.Theme    ← response.Theme
ctx.Route    ← response.Route
ctx.Metadata ← response.Metadata (merge)
```

**shouldDisableGlobalPrompts 三级优先级（tests.hook-disable-global-prompts + tests.hook-prompt-preset）：**

```
1. createResponse.DisableGlobalPrompts  (最高)
2. ctx.Metadata["__disable_global_prompts"]
3. ast.DisableGlobalPrompts (package.yao 配置)
```

#### 4.1.3 Agent-to-Agent 调用

| Assistant ID | connector | 关键文件 | Hook 行为 | 测试目标 | 对应 Go 实现 |
|-------------|-----------|---------|-----------|---------|-------------|
| `tests.caller-target` | `openai.mock` | `package.yao` | 无 hook（简单响应） | 作为被调用方，验证 `ctx.agent.Call` 链路 | `caller/orchestrator.go` |
| `tests.caller-orchestrator` | `openai.mock` | `package.yao`, `src/index.ts` | Next: 调用 `ctx.agent.Call/All/Any/Race` | A2A 调用、并发编排、Fork context、skip 策略 | `caller/jsapi.go`, `caller/orchestrator.go` |

#### 4.1.4 MCP 工具

| Assistant ID | connector | 关键文件 | Hook 行为 | 测试目标 | 对应 Go 实现 |
|-------------|-----------|---------|-----------|---------|-------------|
| `tests.mcp-tools` | `openai.mock` | `package.yao`(mcp 配 echo server), `src/index.ts` | Next: `ctx.mcp.CallTool(...)` | MCP tool 发现、调用、结果解析 | `assistant/mcp.go`, `context/jsapi_mcp.go` |
| `tests.tool-loop` | `openai.mock` | `package.yao`(mcp 配置), 无 hook | 无 hook -- 依赖 mock-llm 返回 tool_calls | Tool Loop：LLM -> tool_calls -> results -> LLM 循环 | `loop.go` executeToolLoop |

**Tool Loop 触发条件（agent.go L543）：**

```
无 Next hook + 有 tool_calls + 非 sandbox + 未禁用 loop
→ executeToolLoop
→ 失败时 fallback 到 __yao.loop_fallback delegation
```

#### 4.1.5 搜索

| Assistant ID | connector | 关键文件 | Hook 行为 | 测试目标 | 对应 Go 实现 |
|-------------|-----------|---------|-----------|---------|-------------|
| `tests.search-web` | `openai.mock` | `package.yao`(`uses.search: auto`) | 无 hook | auto 判意图 + web 搜索（mock `__yao.needsearch`） | `search/`, `assistant/search.go` |
| `tests.search-disabled` | `openai.mock` | `package.yao`(`uses.search: disabled`) | 无 hook | disabled 模式：auto 判断被完全跳过 | `assistant/search.go` |
| `tests.search-hook` | `openai.mock` | `package.yao`(`uses.search: auto`), `src/index.ts` | Create: 返回 search intent 覆盖 | Hook 覆盖 auto 搜索行为 | `assistant/search.go` |

#### 4.1.6 历史与附件

| Assistant ID | connector | 关键文件 | Hook 行为 | 测试目标 | 对应 Go 实现 |
|-------------|-----------|---------|-----------|---------|-------------|
| `tests.history-basic` | `openai.mock` | `package.yao` | 无 hook | 多轮历史加载、重叠检测、historySize 优先级 | `assistant/history.go` |
| `tests.attachment-handler` | `openai.mock` | `package.yao` | 无 hook | 附件路由（image/pdf/docx/text）、本地测试文件 | `content/`, `build_content.go` |

#### 4.1.7 国际化

| Assistant ID | connector | 关键文件 | Hook 行为 | 测试目标 | 对应 Go 实现 |
|-------------|-----------|---------|-----------|---------|-------------|
| `tests.i18n-multilang` | `openai.mock` | `package.yao`, `locales/zh-cn.yml`, `locales/en-us.yml` | 无 hook | 多语言加载、`${}` 模板渲染、locale 合并 | `i18n/i18n.go`, `assistant/load.go` |

#### Tier 2 Assistant 汇总

| 分组 | 数量 | Assistant IDs |
|------|------|---------------|
| 基础加载 | 3 | simple-greeting, no-prompt, connector-resolve |
| Hook 系统 | 10 | hook-echo, hook-error, hook-delegate, hook-next-delegate, hook-next-data, hook-connector-override, hook-context-adjust, hook-search-control, hook-prompt-preset, hook-disable-global-prompts |
| A2A 调用 | 2 | caller-target, caller-orchestrator |
| MCP 工具 | 2 | mcp-tools, tool-loop |
| 搜索 | 3 | search-web, search-disabled, search-hook |
| 历史附件 | 2 | history-basic, attachment-handler |
| 国际化 | 1 | i18n-multilang |
| **合计** | **23** | |

### 4.2 Tier 3 -- 已有 Sandbox V2 Assistant（需 Docker + Tai）

| Assistant ID | runner | lifecycle | connector | 测试目标 | 状态 |
|-------------|--------|-----------|-----------|---------|------|
| `tests.sandbox-v2.oneshot-cli` | claude | oneshot | `use::default` | Claude CLI oneshot + ctx.sandbox | 已有 |
| `tests.sandbox-v2.opencode-oneshot-cli` | opencode | oneshot | `use::default` | OpenCode CLI oneshot | 已有 |
| `tests.sandbox-v2.opencode-session-cli` | opencode | session | `use::default` | OpenCode CLI session（容器复用） | 已有 |
| `tests.sandbox-v2.jsapi-v2` | yao | oneshot | `use::default` | Yao runner JSAPI（computer + workspace） | 已有 |

---

## 5. 子包测试覆盖矩阵

每个 Go 子包需要的测试类型、依赖的 assistant 和 Prepare 层级。

### 5.1 `agent/assistant/` -- 核心主循环

assistant 包是整个 agent 的主调度器（`agent.go` 的 `Stream` 方法），完整流程：

```
权限检查 -> 初始化(Stack/Buffer/Capabilities) -> 历史 -> 沙箱V2
-> Create Hook -> BuildRequest(Prompt组装) -> BuildContent(附件解析)
-> 搜索 -> LLM 调用 -> Tool Loop -> Next Hook -> delegate/return -> Buffer flush
```

| 测试场景 | 使用的 Assistant | Prepare | 关键代码 |
|----------|-----------------|---------|---------|
| 加载与配置合并 | 所有 assistant（验证 Load） | PrepareSandbox | `load.go`, `load_system.go`, `load_merge` |
| 权限检查（auth required） | `tests.simple-greeting` | PrepareSandbox | `permission.go` |
| 无 prompt 无 MCP 边界 | `tests.no-prompt` | PrepareSandbox | `agent.go` L277 LLM 调用被跳过 |
| Create Hook 基础 | `tests.hook-echo` | PrepareSandbox | `hook/create.go` Execute + getHookCreateResponse |
| Create Hook 委派 | `tests.hook-delegate` | PrepareSandbox | `agent.go` delegate 分支 + `next.go` handleDelegation |
| Create Hook 切换连接器 | `tests.hook-connector-override` | PrepareSandbox | `hook/create.go` applyOptionsAdjustments |
| Create Hook 上下文覆盖 | `tests.hook-context-adjust` | PrepareSandbox | `hook/create.go` applyContextAdjustments |
| Create Hook 搜索控制 | `tests.hook-search-control` | PrepareSandbox | `search.go` shouldAutoSearch + parseSearchField |
| Create Hook prompt 预设 | `tests.hook-prompt-preset` | PrepareSandbox | `build.go` buildSystemPrompts + PromptPreset 选择 |
| 禁用全局 prompt | `tests.hook-disable-global-prompts` | PrepareSandbox | `build.go` shouldDisableGlobalPrompts |
| Hook 异常容错 | `tests.hook-error` | PrepareSandbox | `hook/script.go` JS throw -> Go error |
| Next Hook return null | `tests.hook-echo` | PrepareSandbox | `next.go` processNextResponse nil -> buildStandardResponse |
| Next Hook delegate | `tests.hook-next-delegate` | PrepareSandbox | `next.go` processNextResponse delegate 路径 |
| Next Hook custom data | `tests.hook-next-data` | PrepareSandbox | `next.go` processNextResponse data 路径 |
| Source 脚本加载 | `tests.hook-echo` | PrepareSandbox | `source.go` loadSource |
| 历史加载与去重 | `tests.history-basic` | PrepareSandbox | `history.go` WithHistory + overlap 检测 |
| BuildContent 附件解析 | `tests.attachment-handler` | PrepareSandbox | `build_content.go` + content.ParseUserInput |
| 自动搜索 shouldAutoSearch | `tests.search-web` | PrepareSandbox | `search.go` shouldAutoSearch |
| 搜索禁用 | `tests.search-disabled` | PrepareSandbox | `search.go` uses.search="disabled" |
| 搜索 Hook 覆盖 | `tests.search-hook` | PrepareSandbox | `search.go` createResponse.Search 覆盖 |
| MCP 工具构建 | `tests.mcp-tools` | PrepareSandbox | `mcp.go` buildMCPTools |
| Tool Loop 循环（无 Hook） | `tests.tool-loop` | PrepareSandbox | `loop.go` executeToolLoop |
| Tool Loop fallback | `tests.tool-loop` | PrepareSandbox | `loop.go` fallback to `__yao.loop_fallback` |
| 中断处理 | `tests.hook-echo` | PrepareSandbox | `agent.go` handleInterrupt |
| Trace 集成 | `tests.simple-greeting` | PrepareSandbox | `trace.go` initAgentTraceNode |
| Buffer flush | `tests.simple-greeting` | PrepareSandbox | `chat.go` InitBuffer + FlushBuffer |

### 5.2 `agent/context/` -- 运行时上下文与 JSAPI

context 包定义了 `Context` 结构（700+ 行的 `types.go`），包含 Buffer、Stack、Interrupt、JSAPI 等子系统。

| 测试场景 | 使用的 Assistant | Prepare | 关键代码 |
|----------|-----------------|---------|---------|
| Context New/Release 生命周期 | 无需特定 assistant | PrepareSandbox | `context.go` New() + Release() |
| Context Register/Get/Unregister | 无需特定 assistant | PrepareSandbox | `context.go` contextRegistry |
| Context Fork（并发安全） | 无需特定 assistant | PrepareSandbox | `context.go` Fork() |
| ChatBuffer 消息生命周期 | 无需特定 assistant | PrepareSandbox | `buffer.go` AddMessage/Append/Complete |
| ChatBuffer Step 跟踪 | 无需特定 assistant | PrepareSandbox | `buffer.go` BeginStep/CompleteStep |
| Stack root/child/fork | 无需特定 assistant | PrepareSandbox | `stack.go` NewStack/NewChildStack |
| Stack EnterStack 全流程 | 无需特定 assistant | PrepareSandbox | `stack.go` root vs delegate vs fork |
| Message/ContentPart 序列化 | 无（Tier 1） | 无 | `types.go` JSON marshal/unmarshal |
| HookCreateResponse 解析 | 无（Tier 1） | 无 | `types.go` 各字段序列化 |
| NextHookResponse.Action() | 无（Tier 1） | 无 | `types.go` return vs delegate |
| OpenAPI 请求解析 | 无需特定 assistant | PrepareSandbox | `openapi.go` GetCompletionRequest |
| messageMetadataStore | 无需特定 assistant | PrepareSandbox | `message.go` 线程安全 metadata |
| Interrupt 控制器 | 无需特定 assistant | PrepareSandbox | `interrupt.go` SendSignal/Check/CheckWithMerge |
| Interrupt Force 取消 | 无需特定 assistant | PrepareSandbox | `interrupt.go` force -> context cancel |
| JSAPI ctx.agent | `tests.caller-orchestrator` | PrepareSandbox | `jsapi_agent.go` Call/All/Any/Race |
| JSAPI ctx.agent 回调 | `tests.caller-orchestrator` | PrepareSandbox | `jsapi_agent.go` CallWithHandler |
| JSAPI ctx.mcp | `tests.mcp-tools` | PrepareSandbox | `jsapi_mcp.go` CallTool/ListTools |
| JSAPI ctx.llm | `tests.simple-greeting` | PrepareSandbox | `jsapi_llm.go` Stream/Post |
| JSAPI ctx.search | `tests.search-web` | PrepareSandbox | `jsapi_search.go` Search |
| JSAPI ctx.memory | 无需特定 assistant | PrepareSandbox | `jsapi_memory.go` Get/Set/Delete |
| JSAPI ctx.workspace | 需 sandbox assistant | PrepareSandbox | `jsapi_workspace.go` ReadFile/WriteFile |
| JSAPI ctx 输出方法 | 无需特定 assistant | PrepareSandbox | `jsapi.go` Send/SendStream/Replace/Append |
| Output 初始化 | 无需特定 assistant | PrepareSandbox | `output.go` InitOutput/CloseOutput |
| gRPC 上下文 | 无需特定 assistant | PrepareSandbox | `grpc.go` gRPC context 适配 |

### 5.3 `agent/caller/` -- Agent 调用与编排

caller 包提供 agent-to-agent 调用能力，核心是 `Orchestrator`（All/Any/Race）和 `JSAPI`（V8 绑定）。

| 测试场景 | 使用的 Assistant | Prepare | 关键代码 |
|----------|-----------------|---------|---------|
| Orchestrator.All | `tests.caller-orchestrator` + `tests.caller-target` | PrepareSandbox | `orchestrator.go` All() |
| Orchestrator.Any | 同上 | PrepareSandbox | `orchestrator.go` Any() |
| Orchestrator.Race | 同上 | PrepareSandbox | `orchestrator.go` Race() |
| callAgentWithForkedContext | 同上 | PrepareSandbox | `orchestrator.go` ctx.Fork() |
| Process agent.Call | `tests.caller-target` | PrepareSandbox | `process.go` processAgentCall |
| JSAPI Call/All/Any/Race | `tests.caller-orchestrator` | PrepareSandbox | `jsapi.go` V8 绑定 |
| JSAPI CallWithHandler 回调 | `tests.caller-orchestrator` | PrepareSandbox | `jsapi.go` OnMessage 回调 |
| forceSkipForSubAgent | 无需特定 assistant | PrepareSandbox | `jsapi.go` A2A skip 策略 |
| types 序列化 | 无（Tier 1） | 无 | `types.go` Request/Result 序列化 |
| 错误处理 | `tests.hook-error` | PrepareSandbox | agent 不存在/getter 未初始化 |

### 5.4 `agent/content/` -- 附件与多模态

content 包是附件处理的分发器，`content.go` 的 `ParseUserInput` 按 ContentPart.Type 路由到子处理器。

| 测试场景 | 使用的 Assistant | Prepare | 关键代码 |
|----------|-----------------|---------|---------|
| ParseUserInput 路由（string） | `tests.attachment-handler` | PrepareSandbox | `content.go` string case |
| ParseUserInput 路由（[]ContentPart） | `tests.attachment-handler` | PrepareSandbox | `content.go` parseContentParts |
| ParseUserInput 路由（[]interface{}） | `tests.attachment-handler` | PrepareSandbox | `content.go` convertToContentParts |
| Image 解析（vision 直通） | `tests.attachment-handler` | PrepareSandbox | `image/image.go` vision=true |
| Image 解析（non-vision 降级） | `tests.attachment-handler` | PrepareSandbox | `image/image.go` vision agent |
| PDF 文本提取 | `tests.attachment-handler` | PrepareSandbox | `pdf/pdf.go` |
| DOCX 文本提取 | `tests.attachment-handler` | PrepareSandbox | `docx/docx.go` |
| PPTX 文本提取 | `tests.attachment-handler` | PrepareSandbox | `pptx/pptx.go` |
| Text/Code 文件 | `tests.attachment-handler` | PrepareSandbox | `text/text.go` |
| 未知类型降级 | `tests.attachment-handler` | PrepareSandbox | `content.go` fallback text.ParseRaw |
| tools 配置 | 无需特定 assistant | PrepareSandbox | `tools/tools.go` |

### 5.5 `agent/llm/` -- LLM 连接器与 Provider

`resolve.go` 是连接器解析核心：`use::role` -> `llmprovider.GetRoleBy(role, identity)` -> `GetRole(role)` -> default fallback。

| 测试场景 | 使用的 Assistant | Prepare | 关键代码 |
|----------|-----------------|---------|---------|
| ResolveConnector 显式 ID | `tests.connector-resolve` | PrepareSandbox | `resolve.go` connectorID 非空 |
| ResolveConnector use::role | `tests.connector-resolve` | PrepareSandbox | `resolve.go` use::light/vision |
| ResolveConnector identity 优先级 | `tests.connector-resolve` | PrepareSandbox | `resolve.go` GetRoleBy > GetRole |
| ResolveConnector default 降级 | `tests.connector-resolve` | PrepareSandbox | `resolve.go` 空 ID -> "default" |
| GetCapabilities 各解析路径 | 无需特定 assistant | PrepareSandbox | `capabilities.go` |
| capabilitiesFromMap 转换 | 无（Tier 1） | 无 | `capabilities.go` JSON map -> struct |
| Mock Provider stream/post | 无需 assistant | PrepareSandbox | `llm.go` mock-llm 交互 |
| 真实 Provider E2E | 无需 assistant（Tier 4） | PrepareE2E | OpenAI/Anthropic/DeepSeek |
| Image base64/URL 提取 | 无需特定 assistant | PrepareSandbox | `image.go` |

### 5.6 `agent/robot/` -- 自主 Robot 系统

Robot 是独立子系统（自主执行、调度、事件驱动），包含：`types/`、`manager/`、`executor/standard/`、`store/`、`cache/`、`pool/`、`events/integrations/`、`api/`。

| 测试场景 | 使用的 Assistant | Prepare | 关键代码 |
|----------|-----------------|---------|---------|
| Robot 类型（CanRun/Slot 管理） | 无（Tier 1） | 无 | `types/robot.go` |
| Config 解析 | 无（Tier 1） | 无 | `types/config.go` |
| Execution 状态机 | 无（Tier 1） | 无 | `types/robot.go` |
| Manager 启动/停止/Tick | `tests.simple-greeting` | PrepareSandbox | `manager/manager.go` |
| Manager 事件调度 | `tests.simple-greeting` | PrepareSandbox | `manager/manager.go` |
| Manager 并发控制 | `tests.simple-greeting` | PrepareSandbox | `manager/manager.go` + pool |
| Manager 恢复 | `tests.simple-greeting` | PrepareSandbox | `manager/recovery.go` |
| Executor Run 标准流 | `tests.simple-greeting` | PrepareSandbox | `executor/standard/run.go` |
| Executor Goals/Input/Delivery | `tests.simple-greeting` | PrepareSandbox | `executor/standard/` |
| Executor AgentCaller | `tests.simple-greeting` | PrepareSandbox | `executor/standard/agent.go` |
| Executor 暂停/恢复 | `tests.simple-greeting` | PrepareSandbox | `executor/standard/suspend.go` |
| Executor Workspace | `tests.simple-greeting` | PrepareSandbox | `executor/standard/workspace.go` |
| Executor Host 通信 | `tests.simple-greeting` | PrepareSandbox | `executor/standard/host.go` |
| Cache 加载/刷新 | 无需特定 assistant | PrepareSandbox | `cache/cache.go` |
| Store CRUD | 无需特定 assistant | PrepareSandbox | `store/robot.go` |
| Pool Worker 管理 | 无需特定 assistant | PrepareSandbox | `pool/worker.go` |
| Watcher 超时监控 | 无需特定 assistant | PrepareSandbox | `watcher.go` |
| API lifecycle/interact/trigger | `tests.simple-greeting` | PrepareSandbox | `api/` |
| Events dispatcher | 无需特定 assistant | PrepareSandbox | `events/integrations/dispatcher.go` |
| Telegram（E2E） | `tests.simple-greeting` | PrepareE2E | `events/integrations/telegram/` |
| Discord（E2E） | `tests.simple-greeting` | PrepareE2E | `events/integrations/discord/` |
| Feishu（E2E） | `tests.simple-greeting` | PrepareE2E | `events/integrations/feishu/` |
| DingTalk（E2E） | `tests.simple-greeting` | PrepareE2E | `events/integrations/dingtalk/` |
| Weixin（E2E） | `tests.simple-greeting` | PrepareE2E | `events/integrations/weixin/` |

### 5.7 `agent/output/` -- 输出与流式

output 包包含 `output.go`（Accept 路由）、`safe_writer.go`（并发安全 SSE）、`adapters/`（openai/cui）、`message/`、`jsapi/`。

| 测试场景 | 使用的 Assistant | Prepare | 关键代码 |
|----------|-----------------|---------|---------|
| NewOutput Accept 路由 | 无需特定 assistant | PrepareSandbox | `output.go` 路由到 adapter |
| OpenAI adapter SSE chunk | 无需特定 assistant | PrepareSandbox | `adapters/openai/` |
| CUI adapter 消息转换 | 无需特定 assistant | PrepareSandbox | `adapters/cui/` |
| Safe writer 并发安全 | 无（Tier 1） | 无 | `safe_writer.go` Write/Close |
| Safe writer Context 取消 | 无（Tier 1） | 无 | `safe_writer.go` context drain |
| IDGenerator | 无（Tier 1） | 无 | `message/utils.go` |
| Output JSAPI 绑定 | 无需特定 assistant | PrepareSandbox | `jsapi/output.go` |

### 5.8 `agent/store/` -- 持久化

store 包是 agent 持久化层，`xun/` 是唯一活跃实现（SQLite/PG）。

| 测试场景 | 使用的 Assistant | Prepare | 关键代码 |
|----------|-----------------|---------|---------|
| Chat CRUD | 无需特定 assistant | PrepareSandbox | `xun/chat.go` |
| Message SaveMessages/GetMessages | 无需特定 assistant | PrepareSandbox | `xun/message.go` |
| Message 搜索 | 无需特定 assistant | PrepareSandbox | `xun/search.go` |
| Resume SaveSteps/GetSteps | 无需特定 assistant | PrepareSandbox | `xun/resume.go` |
| Assistant Store CRUD | 无需特定 assistant | PrepareSandbox | `xun/assistant.go` |
| types Convert | 无（Tier 1） | 无 | `types/convert.go` |
| types Fields | 无（Tier 1） | 无 | `types/fields.go` |
| types Prompt | 无（Tier 1） | 无 | `types/prompt.go` |
| types MCP 配置 | 无（Tier 1） | 无 | `types/mcp_test.go` |

### 5.9 `agent/memory/` -- 运行时内存

memory 包提供四级命名空间（User/Team/Chat/Context），各自有独立 TTL 和 store 后端。

| 测试场景 | 使用的 Assistant | Prepare | 关键代码 |
|----------|-----------------|---------|---------|
| New 四命名空间初始化 | 无需特定 assistant | PrepareSandbox | `memory.go` New() |
| Namespace Get/Set/Delete | 无需特定 assistant | PrepareSandbox | `namespace.go` |
| 命名空间隔离性 | 无需特定 assistant | PrepareSandbox | User/Team/Chat/Context 互不可见 |
| 默认 TTL 验证 | 无需特定 assistant | PrepareSandbox | UserTTL=0, ChatTTL=24h, ContextTTL=30m |
| Manager GetMemory | 无需特定 assistant | PrepareSandbox | `manager.go` |
| 空 ID 跳过初始化 | 无需特定 assistant | PrepareSandbox | userID="" -> User=nil |

### 5.10 `agent/i18n/` -- 国际化

i18n 包包含 `i18n.go`（加载/解析/合并/模板）和 `builtin.go`（内置翻译）。

| 测试场景 | 使用的 Assistant | Prepare | 关键代码 |
|----------|-----------------|---------|---------|
| Parse 模板解析 | `tests.i18n-multilang` | PrepareSandbox | `i18n.go` `${ }` 变量替换 |
| Parse 递归解析 | `tests.i18n-multilang` | PrepareSandbox | `i18n.go` map/slice/string |
| Load 从应用加载 | `tests.i18n-multilang` | PrepareSandbox | `i18n.go` YAML 文件加载 |
| Merge assistant + global | `tests.i18n-multilang` | PrepareSandbox | `i18n.go` Merge |
| Tr 运行时翻译 | `tests.i18n-multilang` | PrepareSandbox | `i18n.go` Tr() |
| 内置翻译 | 无需特定 assistant | PrepareSandbox | `builtin.go` |

### 5.11 `agent/search/` -- 搜索

search 包包含 `Searcher`、`Registry`、`reference.go`、`citation.go`、`handlers/web/`（Web 搜索）。

| 测试场景 | 使用的 Assistant | Prepare | 关键代码 |
|----------|-----------------|---------|---------|
| Searcher.Search | `tests.search-web` | PrepareSandbox | `search.go` |
| SearchAll/SearchAny 并发 | `tests.search-web` | PrepareSandbox | `search.go` |
| Web handler | `tests.search-web` | PrepareSandbox | `handlers/web/` |
| Registry 注册/获取 | 无（Tier 1） | 无 | `registry.go` |
| BuildReferences | 无（Tier 1） | 无 | `reference.go` |
| FormatReferencesXML | 无（Tier 1） | 无 | `reference.go` |
| BuildReferenceContext | 无（Tier 1） | 无 | `reference.go` |
| CitationGenerator 原子计数 | 无（Tier 1） | 无 | `citation.go` |
| JSAPI ctx.search | `tests.search-web` | PrepareSandbox | `jsapi.go` |

---

## 6. Mock 策略

| 层级   | LLM                                   | MCP                   | Search        | DB        |
| ------ | ------------------------------------- | --------------------- | ------------- | --------- |
| Tier 0 | 不需要                                | 不需要                | 不需要        | SQLite    |
| Tier 1 | 不需要                                | 不需要                | 不需要        | 不需要    |
| Tier 2 | mock-llm server（openai.mock）        | 内置 echo server      | mock handler  | SQLite    |
| Tier 3 | mock-llm server（openai.mock）        | N/A                   | disabled      | SQLite    |
| Tier 4 | 真实 API（Beta 团队）                 | 真实 MCP              | 真实 provider | SQLite/PG |

---

## 7. CI 分级方案（Build Tag 自动发现）

使用 Go Build Tag 隔离测试层级，CI 无需列包名，新增子包自动发现。

### 7.1 Build Tag 约定

| Tag | 层级 | 依赖 | 说明 |
|-----|------|------|------|
| （无 tag） | Tier 0 | Docker + Tai | `sandbox/v2` 基础设施验证，环境不通后续无意义 |
| `//go:build unit` | Tier 1 | 无 | 纯函数、数据结构、序列化 |
| `//go:build integration` | Tier 2 | App + DB + Mock LLM | Hook、Caller、History、Search、Store 等 |
| `//go:build sandbox` | Tier 3 | Docker + Tai + Mock LLM | Claude/OpenCode/Yao Runner |
| `//go:build e2e` | Tier 4 | 真实 LLM API | 端到端验证（已有约定） |

### 7.2 CI 运行命令

```yaml
- name: "Tier 0: Sandbox Infra Tests (sandbox/v2)"
  run: go test -v -count=1 -timeout 600s ./sandbox/v2/...

- name: "Tier 1: Pure Unit Tests"
  run: go test -v -count=1 -timeout 120s -tags unit ./agent/...

- name: "Tier 2: App Integration Tests (mock-llm)"
  run: go test -v -count=1 -timeout 300s -tags integration ./agent/...

- name: "Tier 3: Agent Sandbox Tests (mock-llm)"
  run: go test -v -count=1 -timeout 600s -tags sandbox ./agent/...

- name: "Tier 4: E2E Tests (real LLM)"
  run: go test -v -count=1 -timeout 900s -tags e2e ./agent/...
```

### 7.3 文件命名约定

- **小模块**（测试场景少）：单文件 `xxx_test.go`，头部标 tag
- **大模块**（如 `assistant/`）：按功能拆分 `load_test.go`、`hook_test.go`、`search_test.go`，每个文件头标对应 tag
- **同一文件可以只有一个 tag**：一个功能文件内所有测试属于同一层级
- **跨层级的功能**（如 `llm/resolve.go` 有 unit + integration）：拆为 `resolve_unit_test.go` + `resolve_integration_test.go`

### 7.4 示例

```go
// agent/search/citation_test.go
//go:build unit

package search_test

func TestCitationGenerator_Next(t *testing.T) { ... }
func TestBuildReferences(t *testing.T) { ... }
func TestFormatReferencesXML(t *testing.T) { ... }
```

```go
// agent/assistant/hook_integration_test.go
//go:build integration

package assistant_test

func TestCreateHook_Echo(t *testing.T) { ... }
func TestCreateHook_Delegate(t *testing.T) { ... }
func TestCreateHook_ConnectorOverride(t *testing.T) { ... }
```

### 7.5 向后兼容

- 现有 `//go:build e2e` 标记的文件无需修改，已符合约定
- 无 tag 的测试文件默认不属于任何层级（`go test` 不加 `-tags` 时仍可运行）
- 每个包的 `TestMain` 放在无 tag 的文件中（所有层级共享初始化逻辑）

---

## 8. 统一 Prepare 迁移路径

目标：与 sandbox/v2 现有测试模式完全一致，所有 `agent/` 下的测试统一到 `testprepare` 一条路径。

### 现状对比

| | sandbox/v2（已统一） | agent/ 其余包（待迁移） |
|--|---------------------|----------------------|
| TestMain | `testprepare.MustLoadEnv()` | `test.Prepare(nil, config.Conf)` |
| 每个测试 | `testprepare.PrepareUnit/Sandbox/E2E` | `testutils.PrepareAgent(t)` |
| 应用路径 | `unit-test/agent/app`（固定） | `YAO_AGENT_TEST_APPLICATION`（环境变量） |
| 加载逻辑 | `testprepare/apploader.go` | `testutils/testutils.go` + `test.Prepare` |

### 迁移策略

1. `testprepare` 成为唯一入口（与 sandbox/v2 一致）
2. Tier 1 测试：不调 Prepare 或只调 `PrepareUnit`
3. Tier 2 测试：调 `PrepareSandbox`（加载 App + DB + V8 + Mock LLM）
4. 废弃 `testutils.PrepareAgent`，逐包迁移
5. 每个包的 `TestMain` 统一为 `testprepare.MustLoadEnv` 或更高级别

---

## 9. 实施顺序（纵向切片）

**策略**：每一步完成一个完整闭环，可独立提交。每步流程：

1. 构建依赖的 assistant（如需要）
2. 写新测试（使用 `testprepare`，标 Build Tag）
3. **删除该包下所有旧测试文件**（使用 `testutils.PrepareAgent` 的）
4. **本地跑通**
5. **适配 CI**（环境变量、hosts、mock-llm、Build Tag 等与本地的差异）
6. **提交推送，等待 CI 验证**
7. **CI 不通则迭代修复**，直到全绿后进入下一步

> **CI 范围**：现阶段只关注 Linux（`agent-unit-test.yml`）。Windows 适配在所有 Step 完成后单独进行（可能涉及路径、Docker 检测等代码改动）。

| Step | 涉及包 | 新建 Assistant | 删除旧测试(文件数) | 说明 | 状态 |
|------|--------|---------------|-------------------|------|------|
| 0 | `agent/`（根） | simple-greeting | ~2 | 基础骨架：testprepare 统一 + CI Build Tag + Load 验证 | ✅ 完成 |
| 1 | 多包（Tier 1） | 无 | 0（新增） | 纯单元：types 序列化、citation、safe_writer、ID 生成 | ✅ 完成 |
| 2 | `store/xun/` + `context/` | 无 | ~28 | 持久化 CRUD + Buffer/Stack/Interrupt/JSAPI/MCP/Search | ✅ 完成 |
| 3 | `i18n/` + `memory/` | i18n-multilang | ~2 | 国际化模板 + 四级内存 | ✅ 完成 |
| 4 | `llm/` | connector-resolve | ~12 | ResolveConnector 全路径 + Capabilities | ✅ 完成 |
| 5 | `content/` | attachment-handler | ~5 | ParseUserInput + image/pdf/docx/pptx/text | ✅ 完成 |
| 6 | `assistant/hook/` | 8 个 hook-* | ~9 | Create/Next hook 全路径 | ⏳ 待开始 |
| 7 | `assistant/` | no-prompt, disable-global-prompts, history-basic, search-* (3) | ~22 | 主循环：load/build/search/history/permission | ⏳ 待开始 |
| 8 | `caller/` | caller-target, caller-orchestrator | ~3 | A2A 调用 + All/Any/Race | ⏳ 待开始 |
| 9 | `assistant/mcp+loop` | mcp-tools, tool-loop | 0（新增） | MCP + Tool Loop | ⏳ 待开始 |
| 10 | `output/` | 无 | ~1 | Accept 路由 + adapter + JSAPI | ⏳ 待开始 |
| 11 | `robot/` | 无 | ~40 | manager/executor/store/cache/pool/api/events | ⏳ 待开始 |
| 12 | `search/` | 无 | ~16 | Searcher + web handler + JSAPI | ⏳ 待开始 |
| 清理 | `agent/test/` + `agent/testutils/` | -- | ~9 | 删除跨包遗留，废弃 testutils + 删除所有 .go.bak | ⏳ 待开始 |

**约 110 个旧测试文件被新测试替代并删除。每步一个独立 PLAN。**
