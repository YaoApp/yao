# Agent Test Framework V2 - TODO

> 详细实施计划见 [UPGRADE_PLAN.md](./UPGRADE_PLAN.md)

## Format Rules

| Context               | Format                   | Example                                        |
| --------------------- | ------------------------ | ---------------------------------------------- |
| `-i` flag (CLI)       | Prefix required          | `agents:workers.test.gen`, `scripts:tests.gen` |
| JSONL assertion `use` | Prefix required          | `"use": "agents:workers.test.validator"`       |
| JSONL `simulator.use` | No prefix (agent only)   | `"use": "workers.test.user-simulator"`         |
| `--simulator` flag    | No prefix (agent only)   | `--simulator workers.test.user-simulator`      |
| `t.assert.Agent()`    | No prefix (method-bound) | `t.assert.Agent(resp, "workers.test.val")`     |
| JSONL `before/after`  | No prefix (in src/)      | `"before": "env_test.Before"`                  |
| `--before/--after`    | No prefix (in src/)      | `--before env_test.BeforeAll`                  |

## Phase 1: Before/After Scripts ✅

**新增文件**: `script_hooks.go`

- [x] `types.go`: 添加 `Before`, `After` 字段到 `Case`
- [x] `types.go`: 添加 `BeforeAll`, `AfterAll` 字段到 `Options`
- [x] `script_hooks.go`: 实现 `HookExecutor`
- [x] `script_hooks.go`: 通过 V8 直接执行 `*_test.ts` 脚本
- [x] `runner.go`: 集成 before/after 到 `runSingleTest`
- [x] `runner.go`: 集成 beforeAll/afterAll 到 `RunTests`
- [x] `cmd/agent/test.go`: 添加 `--before`, `--after` flags
- [x] `test/utils.go`: 添加 `LoadAgentTestScripts()` 通用函数
- [x] 创建示例脚本 `assistants/tests/hooks-test/src/env_test.ts`
- [x] 创建单元测试 `script_hooks_test.go` (黑盒测试)

## Phase 2: Agent-Driven Assertions ✅

**修改文件**: `assert.go`, `script_assert.go`

- [x] `types.go`: 添加 `Use`, `Options` 字段到 `Assertion`
- [x] `assert.go`: 实现 `assertAgent` 方法
- [x] `assert.go`: 在 `evaluateAssertion` 添加 `agent` 类型
- [x] `assert.go`: 使用 `goutext.ExtractJSON` 容错解析 LLM 响应
- [x] `script_assert.go`: 添加 `assertAgentMethod` 到 `newAssertObject`
- [x] 创建示例 validator agent (`assistants/tests/validator-agent`)
- [x] 创建单元测试 `assert_agent_test.go` (JSONL 断言 + JSAPI 断言)

## Phase 3: Dynamic Mode (Simulator + Checkpoints)

**新增文件**: `dynamic_runner.go`, `dynamic_types.go`

- [ ] `types.go`: 添加 `Simulator`, `Checkpoints` 字段到 `Case`
- [ ] `dynamic_types.go`: 定义 `Checkpoint`, `DynamicResult` 等类型
- [ ] `dynamic_runner.go`: 实现 `DynamicRunner`
- [ ] `dynamic_runner.go`: 实现 checkpoint 匹配逻辑
- [ ] `dynamic_runner.go`: 实现终止条件判断
- [ ] `runner.go`: 在 `runSingleTest` 判断并调用动态模式
- [ ] 创建示例 simulator agent

## Phase 4: Agent-Driven Input

**新增文件**: `input_source.go`

- [ ] `input_source.go`: 实现 `ParseInputSource`
- [ ] `input_source.go`: 实现 `GenerateTestCases`
- [ ] `loader.go`: 添加 `LoadFromAgent` 方法
- [ ] `loader.go`: 添加 `LoadFromScript` 方法
- [ ] `runner.go`: 在 `RunTests` 支持不同输入源
- [ ] `cmd/agent/agent.go`: 添加 `--dry-run` flag
- [ ] 创建示例 generator agent

## Phase 5: Console Output Optimization

**修改文件**: `output.go`

- [ ] `output.go`: 添加 `DynamicTestStart` 方法
- [ ] `output.go`: 添加 `DynamicTurn` 方法
- [ ] `output.go`: 添加 `DynamicTestResult` 方法
- [ ] `output.go`: 添加 `ParallelResults` 方法
- [ ] 测试并行模式输出效果

## Already Implemented ✅

- [x] Message history support (`input` as array)
- [x] File attachments (`file://` protocol)
- [x] `--parallel` flag
- [x] `--fail-fast` flag
- [x] `-v` verbose mode
- [x] Script testing (`*_test.ts`)
- [x] Before/After hooks (Phase 1)
- [x] Agent-driven assertions (Phase 2)

## Open Questions

1. **Message Generation**: 是否提供 helper 从脚本生成 message history?
2. **Snapshot Testing**: 是否支持 "golden file" 对比?
3. **Retry Logic**: 测试失败是否支持自动重试?
