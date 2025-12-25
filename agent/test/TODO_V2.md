# Agent Test Framework V2 - Implementation TODO

## Format Rules Summary

| Context               | Format                   | Example                                                 |
| --------------------- | ------------------------ | ------------------------------------------------------- |
| `-i` flag (CLI)       | Prefix required          | `agents:workers.test.gen`, `scripts:tests.gen`          |
| JSONL assertion `use` | Prefix required          | `"use": "agents:workers.test.validator"`                |
| JSONL `simulator.use` | No prefix (agent only)   | `"use": "workers.test.user-simulator"`                  |
| `--simulator` flag    | No prefix (agent only)   | `--simulator workers.test.user-simulator`               |
| `t.assert.Agent()`    | No prefix (method-bound) | `t.assert.Agent(resp, "workers.test.validator", {...})` |

## Phase 1: Message History Support ✅ (Already Implemented)

The `input` field already supports:
- `string`: Simple text input
- `object`: Single message `{role, content}`  
- `array`: Message history `[{role, content}, ...]`

See `input.go` → `ParseInputWithOptions()` for implementation.

**Remaining tasks:**
- [ ] Add `options` field support (aligned with `context.Options`) - partially done via `CaseOptions`
- [x] Support attachments in message content parts (file:// protocol)
- [ ] Update console output to show message count
- [ ] Update JSON output format with `messages_count`

## Phase 2: Agent-Driven Input

- [ ] Parse `agents:` prefix in `-i` flag
- [ ] Parse `scripts:` prefix in `-i` flag
- [ ] Use standard `context.Options` for generator invocation
- [ ] Pass `test_mode: "generator"` in `options.metadata`
- [ ] Pass target agent info (description, tools) in `options.metadata`
- [ ] Support query parameters (`?count=10&focus=...`) → merged into `options.metadata`
- [ ] Add `--dry-run` flag to save generated cases without running
- [ ] Create example generator agent with prompt template

## Phase 3: Dynamic Mode (Checkpoints)

- [ ] Add `checkpoints` array to test case parser
- [ ] Add `simulator` field to test case parser
- [ ] Implement checkpoint matching against agent responses
- [ ] Support `after` field for order constraints
- [ ] Track pending/reached checkpoints during execution
- [ ] Implement termination conditions:
  - [ ] All checkpoints reached → PASSED
  - [ ] Simulator signals goal_achieved but checkpoints missing → FAILED
  - [ ] max_turns exceeded → FAILED
  - [ ] timeout exceeded → FAILED
- [ ] Implement simulator invocation via `Assistant.Stream()`
- [ ] `simulator.use` is direct agent ID (no prefix needed)
- [ ] Pass `test_mode: "simulator"` in `options.metadata`
- [ ] Pass persona, goal, turn_count from `simulator.options.metadata`
- [ ] Pass conversation history as messages
- [ ] Create example simulator agent with prompt template

## Phase 4: Agent-Driven Assertions

### In JSONL Test Cases

- [ ] Add `agent` assertion type to assertion parser
- [ ] Support `options` field in assertion (aligned with `context.Options`)
- [ ] Implement validator agent invocation via `Assistant.Stream()`
- [ ] Pass `test_mode: "validator"` in `options.metadata`
- [ ] Pass conversation context and criteria in `options.metadata`
- [ ] Support score-based pass/fail threshold (configurable in `options.metadata`)
- [ ] Add `suggestions` to assertion error output

### In Script Tests

- [ ] Add `t.assert.Agent(response, agentID, options?)` method
- [ ] `agentID` is direct ID (e.g., `workers.test.validator`), no prefix needed
- [ ] Invoke validator agent with context
- [ ] Return `ValidatorResult` object to JavaScript
- [ ] Support passing conversation history in options

### Shared

- [ ] Create example validator agent with prompt template
- [ ] Document `ValidatorResult` interface

## Phase 5: Error Handling & Reporting

- [ ] Implement test-level error handling
- [ ] Add detailed error messages with hints
- [ ] Support `--parallel` flag for concurrent test execution
- [ ] Support `--fail-fast` flag to stop on first failure
- [ ] Add verbose mode (`-v`) for detailed output

## Open Questions

1. **Message Generation**: Should we provide a helper to generate message history from a script?

2. **Snapshot Testing**: Should we support "golden file" comparison for responses?

3. **Retry Logic**: If a test fails, should we support automatic retry?
