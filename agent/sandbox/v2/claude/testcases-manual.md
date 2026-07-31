# Claude Sandbox 回归测试用例

本文档用于人工验证 memory 注入、工具调用、技能发现等功能。
在沙箱 CC 对话中逐条输入提示词，对比预期行为。

---

## 1. 验证 workspace:// 链接（environment-context.md recall）

**提示词 A**：

```
帮我看一下当前工作目录下有哪些文件，列出来并给我可点击的链接
```

**提示词 B**：

```
当前的工作空间 ID 是什么？
```

**预期行为**：

- CC 输出正确的 `workspace://<实际ID>/path` 链接，而不是 `workspace://<workspace-id>/path` 占位符
- 不出现 `$CTX_WORKSPACE_ID` 等未展开的变量

---

## 2. 验证 tai tool 调用格式（environment-context.md recall）

**提示词 A**：

```
搜索一下"Yao Framework"的最新信息
```

**提示词 B**：

```
帮我列出所有可用的 Yao Process
```

**预期行为**：

- CC 使用 `tai tool web_search '{"query":"Yao Framework"}'` 格式调用，不乱猜参数
- CC 使用 `tai tool doc_list` 或 `tai tool process_allowed` 查 Process

---

## 3. 验证 skill_list 工具

**提示词 A**：

```
我有哪些可用的技能？列出来看看
```

**提示词 B**：

```
运行 tai tool skill_list '{}' 看看有什么技能
```

**预期行为**：

- 返回 JSON，包含 `skills[]`、`directories{}`、`usage` 字段
- skills 按 system/assistant/extension 分类

---

## 4. 验证扩展技能发现（extension-skills.md recall）

> 仅在 `.yao/skills/` 下安装了扩展技能时有效

**提示词**：

```
你能帮我做什么？有什么特殊能力吗？
```

**预期行为**：

- CC 主动提及 extension skills 列表中的能力，而不是只说通用能力

---

## 5. 验证跨工作空间访问

**提示词**：

```
帮我看看还有哪些其他可用的工作空间
```

**预期行为**：

- CC 调用 `tai tool workspace_list '{}'` 并列出结果

---

## 6. 验证 kanban/任务工具

**提示词**：

```
帮我看看有哪些看板，以及当前正在进行的任务
```

**预期行为**：

- CC 先调 `tai tool board_list '{}'`，再调 `tai tool task_list '{"run_status":"running"}'`
- 无 "MCP client not found" 错误

---

## 7. 验证 memory 文件存在性（底层验证）

**提示词 A**：

```
帮我看看 .yao/assistants/ 目录下有哪些文件，特别是 memory 子目录
```

**提示词 B**：

```
读一下当前助手的 memory 目录下所有文件的内容
```

**预期行为**：

- 能看到 `environment-context.md`、`extension-skills.md`（有扩展技能时）、`MEMORY.md`
- `MEMORY.md` 为索引文件，含指向上述文件的链接条目
- `environment-context.md` 含正确的 workspace ID 和 tai tool 清单

---

## 8. 快速冒烟测试（一条提示覆盖多功能）

**提示词**：

```
你好，请帮我：1) 告诉我当前工作空间ID  2) 列出可用的技能  3) 查看有哪些看板  4) 搜索"Yao 低代码"的最新信息。每一步都给我详细结果。
```

**预期行为**：

- 4 个子任务分别触发 memory recall、skill_list、board_list、web_search
- workspace ID 正确
- 无工具调用错误
- 全链路正常返回结果
