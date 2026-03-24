# Agent Assistants API — 问题分析

## 概述

本文档梳理 `GET /agent/assistants`（List API）和 `GET /agent/assistants/tags`（Tags API）
在前后端交互中存在的问题，涉及三个核心议题：

1. **分页参数冲突** — 前端请求的 pagesize 超出后端上限，实际返回数据量与预期不符
2. **Tags 列表与查询条件不一致** — 标签始终展示全量，搜索/筛选后不会动态更新
3. **Sandbox V2 识别断裂** — List API 返回的 `sandbox` 布尔值只反映 V1，V2 被遗漏
4. **Sandbox 锁定判断未适配 V2** — Card/详情页仍用 docker 判断，未考虑 `kind=host` 场景
5. **AgentPicker 组件未使用服务端能力** — 搜索、标签过滤、分页均在前端完成，数据不完整

---

## 一、前端调用场景对比

`AgentPicker` 组件被 3 个场景使用，各自的 filter 不同：

| 场景 | 文件 | mode | filter | 预期查询范围 |
|------|------|------|--------|-------------|
| 聊天框切换助手 | `chatbox/components/InputArea/AgentTag.tsx` | single | 无（不传 filter） | 与助手页面一致：`type=assistant` |
| MC 身份设定 — 可协作智能体 | `pages/mission-control/.../IdentityPanel.tsx` | multiple | `{ types: ['assistant', 'robot'], automated: true }` | assistant + robot 中 automated 的 |
| MC 添加智能体 — 可协作智能体 | `pages/mission-control/.../AddAgentModal/index.tsx` | multiple | `{ types: ['assistant', 'robot'], automated: true }` | 同上 |

聊天框场景不传 filter，后端默认 `type=assistant`，结果范围与助手页面一致（这是正确的）。

另外，**助手页面**（`pages/assistants/index.tsx`）不使用 AgentPicker，它有独立的列表实现：

| 维度 | 助手页面 | AgentPicker（聊天框） | AgentPicker（MC） |
|------|---------|---------------------|-------------------|
| pagesize | 12（正确分页） | 200（超出后端上限） | 200（超出后端上限） |
| 搜索 | `keywords`（服务端） | 前端内存过滤 | 前端内存过滤 |
| 标签过滤 | `tags`（服务端） | 前端内存过滤 | 前端内存过滤 |
| 标签列表 | `tags.List()` API | 从已加载数据聚合 | 从已加载数据聚合 |
| type | `type: 'assistant'` | 不传（默认 assistant） | `types: ['assistant', 'robot']` |
| 其他 filter | 无 | 无 | `automated: true` |

---

## 二、分页参数冲突（核心 Bug）

### 后端限制

```go
// openapi/agent/assistant.go:44-47
pagesize := 20
if pagesizeStr := c.Query("pagesize"); pagesizeStr != "" {
    if ps, err := strconv.Atoi(pagesizeStr); err == nil && ps > 0 && ps <= 100 {
        pagesize = ps
    }
}
```

后端 handler 对 `pagesize` 有 **硬上限 100**：当请求值 > 100 时，条件 `ps <= 100` 不满足，
`pagesize` **静默回落到默认值 20**，不报错。

`ValidatePagination` 也同样限制 `pagesize > 100` 报错，但 handler 的预处理已经把它截断为 20 了，
所以验证永远不会触发。

`BuildAssistantFilter` 再兜底：`PageSize > 100 → 100`。三层保护逻辑叠加，最终效果是
**超过 100 的请求静默变成 20**。

### 前端请求

```typescript
// AgentPicker/index.tsx:66-71
api.assistants.List({
    select: ['assistant_id', 'name', 'avatar', 'description', 'tags', 'connector', 'sandbox', 'built_in'],
    locale: is_cn ? 'zh-cn' : 'en-us',
    pagesize: 200,    // ← 超出后端上限
    ...filter
})
```

### 实际效果

| 前端期望 | 后端实际行为 |
|---------|------------|
| 一次拉取 200 条 | 静默回落为 pagesize=20，只返回 20 条 |
| 聊天框场景（无 filter）不传 type | type 默认 `assistant`，不影响 |
| MC 场景传 `types: ['assistant', 'robot']` | 不传 type，后端不设默认 type，只用 types IN 查询，结果正确但仅 20 条 |

**AgentPicker 显示的最多只有 20 个助手**，而不是用户期望的全部。
左侧分类标签和计数也只基于这 20 条数据聚合，严重不准。

---

## 三、Tags API 返回范围与查询条件不一致

### 问题描述

助手页面（`pages/assistants/index.tsx`）的标签 Tab 栏显示的是 **所有** 标签，
而不是当前查询条件下的标签。当用户在搜索框中输入关键词后，标签 Tab 没有变化，
仍然展示全量标签，其中很多标签对应的搜索结果可能为零。

### 前端调用

```typescript
// pages/assistants/index.tsx:66-69 — Tags 加载（只在组件挂载时调用一次）
const response = await agent.tags.List({
    locale: is_cn ? 'zh-cn' : 'en-us',
    type: 'assistant'
})
```

Tags 加载在 `useEffect(() => { ... }, [is_cn])` 中，只依赖 `is_cn`，
**不会在搜索/筛选条件变化时重新加载**。

### 后端 Tags API 支持的参数

Tags handler（`ListAssistantTags`）支持以下过滤参数：

| 参数 | 支持 | 说明 |
|------|------|------|
| `type` | 是 | 单个类型，默认 `assistant` |
| `types` | **否** | 不支持多类型 IN 查询 |
| `connector` | 是 | |
| `keywords` | 是 | 搜索 name/description |
| `built_in` | 是 | |
| `mentionable` | 是 | |
| `automated` | 是 | |
| `sandbox` | **否** | Tags API 不支持 |
| `tags` | **否** | Tags API 不接受（合理） |

### 应有的行为

当用户输入搜索关键词或切换其他筛选条件时，标签列表应该只显示
**在当前查询条件下存在助手的标签**。例如：

- 搜索 "Keeper" → 标签只显示 Data、Query、Ingestion 等 Keeper 相关助手的标签
- 没有匹配助手的标签应该消失（或显示为 0）

### 修复方向

1. **前端**：当搜索/筛选条件变化时，重新调用 `tags.List()` 并透传 `keywords` 等参数
2. **后端**（可选）：Tags API 增加 `types` 参数支持，与 List API 对齐

---

## 四、Sandbox V2 识别断裂

### 数据流断裂点

```
加载时（load.go）                DB 读回（xun/assistant.go）         List API（filter.go）
┌─────────────┐                 ┌───────────────────┐              ┌────────────────────┐
│ sandbox.yao │                 │ data["sandbox"]   │              │ hasSandbox :=      │
│ version:2.0 │                 │     ↓             │              │   a.Sandbox != nil │
│     ↓       │                 │ ToSandbox()       │              │     ↓              │
│ SandboxV2 ✓ │                 │ → model.Sandbox   │              │ sandbox: bool      │
│ IsSandbox ✓ │                 │   (V1 struct)     │              │ (只看 V1)          │
│             │                 │                   │              │                    │
│ 不走 DB     │                 │ 不调 ToSandboxV2  │              │ 不看 IsSandbox     │
└─────────────┘                 └───────────────────┘              └────────────────────┘
```

### AssistantModel 中的字段定义

```go
// agent/store/types/types.go:458-462
Sandbox        *Sandbox                    `json:"sandbox,omitempty"`   // V1 — 持久化到 DB
SandboxV2      *sandboxTypes.SandboxConfig `json:"-"`                   // V2 — 运行时，json:"-"
IsSandbox      bool                        `json:"-"`                   // 运行时标记
ComputerFilter *sandboxTypes.ComputerFilter `json:"-"`                  // 运行时
```

`SandboxV2`、`IsSandbox`、`ComputerFilter` 都标了 `json:"-"`，仅在运行时内存中存在。

### List API 的处理

```go
// openapi/agent/filter.go:194-195
hasSandbox := a.Sandbox != nil   // ← 只看 V1 的 Sandbox 指针
```

### GetInfo API 的处理（对比）

```go
// agent/assistant/assistant.go:488
Sandbox: ast.IsSandbox   // ← 看的是运行时 IsSandbox（V2 会为 true）
```

### DB 读回路径

```go
// agent/store/xun/assistant.go:636-641（ToAssistantModel 中）
if sandbox, has := data["sandbox"]; has && sandbox != nil {
    sb, err := types.ToSandbox(sandbox)   // ← 只用 ToSandbox（V1）
    if err == nil {
        model.Sandbox = sb
    }
}
// 没有 ToSandboxV2 调用，没有检查 version 字段
```

### V2 Sandbox 的两种配置方式

| 方式 | DB 中 sandbox 列 | 加载时 SandboxV2 | List API sandbox 布尔 |
|------|------------------|-----------------|---------------------|
| **独立 `sandbox.yao` 文件** | 可能为 NULL（sandbox 配置不在 package 里） | ✓（从文件加载） | **false**（DB 列为空 → Sandbox==nil） |
| **package.yao 内嵌 `sandbox` 块（version:2.0）** | 有 JSON（含 version:2.0） | ✓（从 DB JSON 解析） | **true**（ToSandbox 用 jsoniter 反序列化，忽略未知字段，返回空但非 nil 的 `*Sandbox`） |

对于 **独立 `sandbox.yao` 文件** 的 V2 助手：
- DB `sandbox` 列为 NULL 或 JSON null
- `ToSandbox` 返回 nil
- `hasSandbox = false`
- **List API 返回 `sandbox: false`，但 GetInfo API 返回 `sandbox: true`**
- 前端助手页卡片上不会显示电脑图标，AgentPicker 也无法识别

对于 **package.yao 内嵌 `sandbox` 块（version:2.0）** 的 V2 助手：
- DB `sandbox` 列有 JSON（含 version、computer、runner 等 V2 字段）
- `ToSandbox` 用 `jsoniter.Unmarshal` 到 V1 `Sandbox` 结构体，**忽略未知字段**，
  返回一个空但非 nil 的 `*Sandbox{}`（command=""、image="" 等零值）
- `hasSandbox = true`（指针非 nil）
- **List API 返回 `sandbox: true`，凑巧正确，但依据错误**（实际是空 V1 对象，不是真的 V1 配置）

注意：`FilterBuiltInAssistant` 会对内置助手清除 `assistant.Sandbox = nil`，
但 `hasSandbox` 在清除前捕获（filter.go:195-196），所以不影响布尔值。
修复后若使用 `SandboxV2`/`IsSandbox`，因其标记 `json:"-"` 不会被 `FilterBuiltInAssistant` 清除，
也不会被 `json.Marshal` 输出，需在 `AssistantToResponse` 中手动追加到 result map。

### GetAssistantTags 的 Sandbox 情况

Tags API (`GET /agent/assistants/tags`) 不接受 `sandbox` 参数，也不返回 sandbox 相关信息。
这本身没问题，但 AgentPicker 没有使用 Tags API。

---

## 五、Sandbox 锁定判断未适配 V2

### 问题描述

助手 Card 和详情页的「聊天」按钮禁用逻辑仍使用 V1 时代的判断方式，
V2 引入了 `computer_filter.kind` 区分 `host`（宿主机）和 `box`（容器），
但列表页没有使用这个信息。

### V1 的判断（当前 Card 和详情页）

```typescript
// pages/assistants/components/Card.tsx:30-31
const dockerAvailable = (global.app_info as any)?.tools?.docker?.available === true
const chatDisabled = data.sandbox === true && !dockerAvailable
```

逻辑：sandbox 助手 + 没有 docker → 禁用聊天。**对 V1 是正确的**（V1 全部走容器）。

### V2 的变化

V2 sandbox 有 `ComputerFilter`，其中 `kind` 决定执行环境：

| kind | 含义 | 需要 docker |
|------|------|------------|
| `"host"` | 在宿主机执行 | 不需要 |
| `"box"` | 在容器中执行 | 需要 |
| `["host", "box"]` | 两种都支持 | 有一种匹配即可 |

InputArea 里**已经正确实现**了基于 `computer_filter` 的工作区兼容性检查：

```typescript
// chatbox/components/InputArea/index.tsx:198-201
const kinds = Array.isArray(filter.kind) ? filter.kind : [filter.kind]
return !kinds.some((k) =>
    k === 'host' ? caps.host_exec : k === 'box' ? caps.docker || caps.k8s : false
)
```

但 Card 和详情页**没有使用 `computer_filter`**，因为：
1. List API 不返回 `computer_filter`（它是运行时字段，`json:"-"`）
2. Card 只拿到了 `sandbox: boolean`，没有 kind 信息
3. 所以 Card 只能用旧的 `docker.available` 做兜底判断

### 实际影响

| 助手类型 | Card 上的判断 | 实际能否聊天 |
|---------|--------------|------------|
| V1 sandbox（容器） | `sandbox && !docker` → 正确禁用 | 确实不行 |
| V2 `kind=box`（容器） | `sandbox && !docker` → 正确禁用 | 确实不行 |
| V2 `kind=host`（宿主机） | `sandbox && !docker` → **错误禁用** | 其实可以（不需要 docker） |
| V2 `kind=["host","box"]` | `sandbox && !docker` → **错误禁用** | host 方式可以 |

### 修复方向

需要让 List API 返回足够的信息，使 Card 能做出正确判断：

**方案 A：List API 返回 `computer_filter`**

在 `AssistantsToResponse` 中增加 `computer_filter` 字段。
需要在 `ToAssistantModel`（DB 读回）时从 V2 sandbox 配置中提取 filter。

**方案 B：List API 返回 `sandbox_kind`**

新增一个简化字段 `sandbox_kind`（`"host"` / `"box"` / `["host","box"]` / `null`），
Card 用它替代单纯的 `sandbox` 布尔值做判断。

---

## 六、AgentPicker 的其他问题

### 6.1 搜索仅前端过滤

后端 API 支持 `keywords` 参数（搜索 name/description/capabilities/locales），
但 AgentPicker 没用，搜索只在已加载的 ≤20 条数据上做前端 filter。

### 6.2 标签过滤仅前端聚合

后端有独立的 Tags API (`GET /agent/assistants/tags`)，支持权限过滤，
但 AgentPicker 没调用，左侧分类列表从已加载的 ≤20 条数据聚合。

### 6.3 loadedRef 阻止 filter 变化时重新请求

```typescript
useEffect(() => {
    if (!visible || loadedRef.current || !window.$app?.openapi) return
    loadedRef.current = true
    // ... API call
}, [visible, type, is_cn])  // ← 不包含 filter
```

如果调用方在同一会话中改变 filter props，组件不会重新请求。

---

## 七、修复建议

### 7.1 后端：List API 的 sandbox 布尔值应使用 IsSandbox

当前 `AssistantsToResponse` 只看 `a.Sandbox != nil`（V1），应同时考虑 V2：

```go
// 建议修改
hasSandbox := a.Sandbox != nil || a.IsSandbox
```

但问题是 **从 DB 读回的 model 没有填充 IsSandbox**（只有加载时的运行时路径才填充）。
需要在 `ToAssistantModel`（convert.go）中增加 V2 检测：

```go
if sandbox, ok := data["sandbox"]; ok && sandbox != nil {
    version := extractSandboxVersion(sandbox)
    if version == "2.0" {
        sb, err := types.ToSandboxV2(sandbox)
        if err == nil {
            model.SandboxV2 = sb
            model.IsSandbox = true
        }
    } else {
        sb, err := types.ToSandbox(sandbox)
        if err == nil {
            model.Sandbox = sb
        }
    }
}
```

然后在 `AssistantsToResponse` 中：

```go
hasSandbox := a.Sandbox != nil || a.IsSandbox
```

对于独立 `sandbox.yao` 文件的助手（DB sandbox 列为空），需要额外机制将 sandbox 标记
持久化到 DB，或在 List 查询中从运行时 assistant 实例补充 IsSandbox 信息。

### 7.2 前端：AgentPicker 应正确使用分页和服务端能力

**方案 A — 使用正确的 pagesize + 滚动加载（推荐）**

参考助手页面的实现：
- pagesize 设为 20-50（不超过 100）
- 实现滚动加载更多（参考 `loadMoreData` 模式）
- 使用 `keywords` 参数做服务端搜索（带防抖）
- 使用 `tags` 参数做服务端标签过滤
- 调用 `tags.List()` 获取准确的标签列表
- Tags API 调用需传入与助手列表相同的 filter 条件（如 `types`、`automated`）

**方案 B — 取消 pagesize 上限（不推荐）**

放宽后端 `pagesize` 限制到 200-500。不推荐，因为：
- 数据量大时响应慢
- 内存占用高
- 不符合分页设计初衷

### 7.3 前端：助手页面标签应随查询条件动态更新

当搜索/筛选条件变化时，重新调用 `tags.List()` 并透传 `keywords` 等参数，
使标签 Tab 只显示当前条件下有结果的标签。

### 7.4 后端：Tags API 增加 types 参数支持

当前 Tags handler 只支持 `type`（单类型），不支持 `types`（多类型 IN 查询）。
AgentPicker 在 MC 场景需要 `types: ['assistant', 'robot']`，如果要让 AgentPicker
也用 Tags API，需要后端增加 `types` 支持。

### 7.5 后端：List API 返回 computer_filter

在 `AssistantsToResponse` 中，从 V2 sandbox 配置提取 `computer_filter` 返回给前端。
这样 Card/详情页可以用 `computer_filter.kind` 做准确的锁定判断，
与 InputArea 的逻辑对齐。

需要在 `ToAssistantModel`（DB 读回）时：
- 检测 `version: "2.0"` → 解析 V2 配置 → 提取 `filter` 字段
- 将 `computer_filter` 放入响应 map

注意：`computer_filter` 不在 `availableAssistantFields` 白名单中（types.go），
也不在 `defaultAssistantFields` 中。它不是 DB 列，无法通过 `select` 参数获取。
必须在 `AssistantToResponse` 阶段从解析后的 V2 配置中额外附加到 result map。
（`SandboxV2` 和 `ComputerFilter` 标记 `json:"-"`，`json.Marshal` 不会输出它们。）

### 7.6 前端：Card/详情页使用 computer_filter 替代 docker 判断

```typescript
// 当前（V1 逻辑）
const chatDisabled = data.sandbox === true && !dockerAvailable

// 应改为
const chatDisabled = data.sandbox === true && !hasCompatibleNode(data.computer_filter)
```

`hasCompatibleNode` 应与 InputArea 的工作区兼容性检查对齐，
检查 `kind` 是否有匹配的节点能力（`host_exec` / `docker` / `k8s`）。

### 7.7 后端：pagesize 超限时应返回错误而非静默回落

当前行为：前端传 `pagesize=200`，后端静默用 20，不报错。
建议：handler 预处理中，当 `pagesize > 100` 时直接返回 400 错误，让前端能感知到问题。

或者至少在响应中返回实际使用的 pagesize（当前已返回），前端应检查
`response.pagesize !== requestedPagesize` 的情况。

---

## 八、影响范围

| 组件/页面 | 受影响 | 说明 |
|-----------|--------|------|
| 助手页面 Card | 严重 | 1) V2 sandbox 电脑图标缺失；2) `kind=host` 的助手被错误禁用聊天；3) 搜索后标签 Tab 不更新 |
| 助手详情页 | 严重 | 同 Card：V2 电脑图标缺失 + `kind=host` 错误禁用 |
| AgentPicker — 聊天框 | 严重 | 只显示 20 条，搜索/分类不完整 |
| AgentPicker — MC 身份设定 | 严重 | 只显示 20 条符合条件的，搜索/分类不完整 |
| AgentPicker — MC 添加智能体 | 严重 | 同上 |
| Chatbox InputArea | 不受影响 | 已正确使用 `computer_filter.kind` + 节点能力匹配 |
| GetInfo API | 不受影响 | 已正确使用 IsSandbox + ComputerFilter |

---

## 九、相关文件

### 后端

| 文件 | 说明 |
|------|------|
| `openapi/agent/assistant.go` | List/Tags handler，分页参数解析 |
| `openapi/agent/types.go` | pagesize 上限、ValidatePagination、BuildAssistantFilter |
| `openapi/agent/filter.go` | AssistantsToResponse — sandbox 布尔化 |
| `agent/store/xun/assistant.go` | DB 查询、ToAssistantModel、sandbox 列过滤 |
| `agent/store/types/types.go` | AssistantModel 定义（Sandbox vs SandboxV2） |
| `agent/store/types/convert.go` | ToAssistantModel — 只调用 ToSandbox |
| `agent/store/types/sandbox_v2.go` | ToSandboxV2、LoadSandboxConfig |
| `agent/assistant/load.go` | 加载时 V1/V2 分支处理 |
| `agent/assistant/assistant.go` | GetInfo — 使用 IsSandbox |

### 前端

| 文件 | 说明 |
|------|------|
| `components/AgentPicker/index.tsx` | 组件实现 — pagesize:200、前端过滤 |
| `components/AgentPicker/types.ts` | AgentPickerProps、AgentPickerFilter |
| `chatbox/components/InputArea/AgentTag.tsx` | 聊天框调用 — 无 filter |
| `pages/mission-control/.../IdentityPanel.tsx` | MC 调用 — filter={types,automated} |
| `pages/mission-control/.../AddAgentModal/index.tsx` | MC 调用 — 同上 |
| `pages/assistants/index.tsx` | 助手页面 — 正确的分页实现（参考） |
| `openapi/agent/assistants.ts` | API 封装 |
| `openapi/agent/tags.ts` | Tags API 封装 |
| `openapi/agent/types.ts` | AgentFilter 类型定义 |
