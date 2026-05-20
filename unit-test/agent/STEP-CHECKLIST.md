# Agent 测试重构 -- 每步执行要点

> 每一步规划时必须读一遍此文件，确保不遗漏。

## 进度总览

| Step | 说明                                         | 状态      |
| ---- | -------------------------------------------- | --------- |
| 0    | 基础骨架：testprepare + CI + Load 验证       | ✅ 完成   |
| 1    | Tier 1 纯单元：types/citation/safe_writer/ID | ✅ 完成   |
| 2    | store/xun + context                          | ✅ 完成   |
| 3    | i18n + memory                                | ✅ 完成   |
| 4    | llm                                          | ✅ 完成   |
| 5    | content                                      | ✅ 完成   |
| 6    | assistant/hook                               | ✅ 完成   |
| 7    | assistant 主循环                             | ✅ 完成   |
| 8    | caller                                       | ✅ 完成   |
| 9    | assistant/mcp+loop                           | ✅ 完成   |
| 10   | output                                       | ✅ 完成   |
| 11   | robot                                        | ✅ 完成   |
| 12   | search                                       | ✅ 完成   |
| 清理 | 删除 180 个 .go.bak 旧测试备份               | ✅ 完成   |
| 补充 | 覆盖率补充 + E2E + Stress fixture 适配        | ✅ 完成   |
| 收尾 | 覆盖率收集：各 Tier coverprofile 合并 + CI 上传 | ✅ 完成   |

## 核心约束

1. **testprepare 是唯一入口** -- 所有新测试用 `testprepare.PrepareUnit/PrepareSandbox/PrepareE2E`，禁止用 `testutils.PrepareAgent` 或 `test.Prepare`
2. **Build Tag 必须标注** -- `//go:build unit`、`//go:build integration`、`//go:build sandbox`、`//go:build e2e`、`//go:build stress`
3. **旧测试必须删除** -- 每步做完后，该包下所有使用 `testutils.PrepareAgent` 的旧测试文件全部删除，不允许新旧共存
4. **新测试必须完全覆盖旧测试的范围** -- 删除前逐个检查旧测试的测试场景，确保新测试已覆盖
5. **CI 只关注 Linux** -- 现阶段只适配 `agent-unit-test.yml`，Windows 后续单独处理
6. **外部测试包** -- 所有新测试必须使用 `package 包名_test`（如 `package agent_test`、`package assistant_test`），不使用 `package 包名`。这是 Go 的外部测试包机制，强制只通过导出 API 验证行为，同时避免循环依赖（`testprepare` import 了 `agent`，白盒会形成 `agent -> testprepare -> agent` 循环）

## 每步流程

```
1. 构建依赖的 assistant（package.yao + src/*.ts + prompts/ + locales/）
2. 写新测试（testprepare + Build Tag）
3. 删除该包下所有旧测试文件
4. 本地跑通
5. 适配 CI（与本地环境差异：env、hosts、mock-llm、Docker）
6. 提交推送，等 CI
7. CI 不通则迭代修复，全绿后进入下一步
```

## 技术要点

### testprepare 层级

| 函数                | 用途                    | 启动的服务               |
| ------------------- | ----------------------- | ------------------------ |
| （无 Prepare）      | Tier 0 基础设施验证      | sandbox/v2 自带 TestMain |
| `PrepareUnit(t)`    | Tier 1 纯函数           | 无（只加载 env）         |
| `PrepareSandbox(t)` | Tier 2/3 集成+sandbox   | App + DB + V8 + Mock LLM |
| `PrepareE2E(t)`     | Tier 4 端到端           | App + DB + V8 + 真实 LLM |

### TestMain 约定

```go
// 每个包的 main_test.go（无 Build Tag，外部测试包）
// 例如 agent/ 下用 package agent_test，assistant/ 下用 package assistant_test
package agent_test // 按实际包名替换

import (
    "os"
    "testing"
    "github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestMain(m *testing.M) {
    testprepare.MustLoadEnv()
    os.Exit(m.Run())
}
```

### 文件命名

- 小模块：`xxx_test.go`（头部标一个 tag）
- 大模块：按功能拆 `load_test.go`、`hook_test.go`（每文件一个 tag）
- 跨层级：`resolve_unit_test.go` + `resolve_integration_test.go`

### Mock LLM

- Alpha 团队 `openai.mock` 指向 `http://host.tai.internal:6920`
- CI 中 mock-llm 在本机启动，`host.tai.internal` 映射到 `127.0.0.1`
- 本地开发同理（需在 `/etc/hosts` 中配置）

### Assistant 路径约定

- 所有测试 assistant 放在 `unit-test/agent/app/assistants/tests/` 下
- ID 格式：`tests.<name>`（目录结构 `tests/<name>/package.yao`）

### Connector 约定

- `openai.mock` -- 主 mock connector（已有）
- `openai.mock-validator` -- 第二个 mock connector（Step 0 创建，用于 hook 切换验证）
- `use::default` / `use::light` / `use::vision` -- 角色解析（Beta 团队配置）

### 旧测试识别方法

```bash
# 查找该包下所有使用旧入口的测试文件
rg "testutils\.PrepareAgent|testutils\.Prepare|test\.Prepare" agent/<package>/ --files-with-matches
```

### CI 分级（agent-unit-test.yml）

| Tier | 命令 | LLM | 说明 |
|------|------|-----|------|
| 0 | `go test ./sandbox/v2/...` | 不需要 | 基础设施验证，无 tag，失败则后续无意义 |
| 1 | `go test -tags unit ./agent/...` | 不需要 | 纯单元测试 |
| 2 | `go test -tags integration ./agent/...` | mock-llm | App 集成测试 |
| 3 | `go test -tags sandbox ./agent/...` | mock-llm | Docker+Tai sandbox 测试 |
| 4 | `go test -tags e2e ./agent/...` | 真实 API | 端到端测试 |
| 5 | `go test -tags stress ./agent/...` | mock-llm | 压力测试 + 内存/goroutine 泄漏检测 |

### CI 环境差异（Linux vs 本地 macOS）

- Docker：Linux CI 原生支持，需 pull `yaoapp/tai-sandbox-base:latest`
- Docker Desktop (macOS)：VirtioFS bind mount 的 `/workspace` owner 显示为 UID 0，`waitEntrypoint` probe 已兼容此场景
- hosts：CI 中 `echo "127.0.0.1 host.tai.internal" | sudo tee -a /etc/hosts`
- env：CI 从 template 生成 `agent-test.env`，注入 secrets
- DB 矩阵：SQLite3 + Postgres14

## .go.bak 旧测试参考文件（已清理）

Step 1 将 `agent/` 下所有旧 `*_test.go` 文件（排除 Step 0 产出和 `agent/sandbox/`）批量改名为 `*.go.bak`，使其脱离编译但保留在 git 中作为后续重构的参考。

- **数量**：180 个文件
- **状态**：✅ 已全部删除（所有 Step 完成后统一清理）
- **原目的**：让 `go test ./agent/...` 安全运行，不会因旧测试依赖 `testutils` 而编译失败
- **生命周期**：每步重构时参考对应 `.go.bak` 文件的测试场景，确保新测试覆盖 → 清理步统一删除

## 覆盖率收集

### 策略

各 Tier 独立生成 `-coverprofile`，合并后上传 Codecov。

**关键**：`go test -coverprofile` 不能直接用 `./agent/...`——没有测试文件的包会导致 `exit 1`。
必须先用 `go list` 过滤出有测试文件的包，再传给 `go test`。

### 本地命令

```bash
mkdir -p .build/coverage

# Tier 1: unit
PKGS=$(go list -f '{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}' -tags unit ./agent/... | tr '\n' ' ')
go test -count=1 -timeout 120s -tags unit -coverprofile=.build/coverage/tier1.out -coverpkg=./agent/... $PKGS

# Tier 2: integration
PKGS=$(go list -f '{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}' -tags integration ./agent/... | tr '\n' ' ')
go test -count=1 -timeout 300s -tags integration -coverprofile=.build/coverage/tier2.out -coverpkg=./agent/... $PKGS

# Tier 5: stress (optional, long-running)
PKGS=$(go list -f '{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}' -tags stress ./agent/... | tr '\n' ' ')
go test -count=1 -timeout 1800s -tags stress -coverprofile=.build/coverage/tier5.out -coverpkg=./agent/... $PKGS

# Merge
head -1 .build/coverage/tier1.out > .build/coverage/all.out
tail -n +2 -q .build/coverage/tier*.out >> .build/coverage/all.out
go tool cover -func=.build/coverage/all.out | tail -1
```

### CI 集成（已启用）

在 `agent-unit-test.yml` 中：
1. Tier 1~5 每层用 `go list` 过滤包后运行 `go test -coverprofile`
2. "Merge Coverage Profiles" step 合并所有 tier*.out 为 all.out
3. `codecov/codecov-action@v5` 上传 all.out，flag 为 `agent-{db}`
4. 需要在 GitHub repo secrets 中配置 `CODECOV_TOKEN`

**Tier 5 (stress) 说明**：压力测试 timeout 设为 1800s（30 分钟），包含高迭代循环（100~1000 次）和高并发（100 goroutines x 10 iterations）的内存/goroutine 泄漏检测。使用 `PrepareSandbox` + mock-llm，依赖 `tests.create` 和 `tests.realworld` 两个 fixture assistant。

### Stress Test Fixture

| Assistant ID | 用途 | 场景 |
|-------------|------|------|
| `tests.create` | Hook 性能基准 + 泄漏检测 | `return_process`, `nested_script_call`, `deep_nested_call` |
| `tests.realworld` | 真实场景压力 + MCP 集成 | `simple`, `mcp_health`, `mcp_tools`, `full_workflow`, `trace_intensive`, `resource_heavy` |

## 参考文档

- 完整设计：`unit-test/agent/AGENT-TEST-DESIGN.md`
- 实施计划：`.cursor/plans/agent_测试逐步实施_dd9eb301.plan.md`
- 测试环境：`unit-test/AGENT-TEST-ENV.md`
- 环境配置：`unit-test/agent/env/agent-test.env.template`
