# Agent Test Framework V2 - Implementation TODO

## Format Rules Summary

| Context | Format | Example |
|---------|--------|---------|
| `-i` flag (CLI) | Prefix required | `agents:workers.test.gen`, `scripts:tests.gen` |
| JSONL assertion `use` | Prefix required | `"use": "agents:workers.test.validator"` |
| JSONL `simulator.use` | No prefix (agent only) | `"use": "workers.test.user-sim"` |
| `--simulator` flag | No prefix (agent only) | `--simulator workers.test.user-sim` |
| `t.assert.Agent()` | No prefix (method is explicit) | `t.assert.Agent(resp, "workers.test.validator", {...})` |

## Phase 1: Static Multi-Turn

- [ ] Extend test case parser for `turns` array
- [ ] Add `options` field support (aligned with `context.Options`)
- [ ] Support test-level `options` and per-turn `options` override
- [ ] Implement turn-by-turn execution with options passing
- [ ] Implement conversation context management
- [ ] Add per-turn assertions
- [ ] Support attachments at turn level
- [ ] Implement awaiting input detection (heuristics)
- [ ] Add `on_missing_input` handling (`skip`, `fail`, `end`)
- [ ] Implement mode priority (static → simulator → interactive → skip)
- [ ] Update console output for multi-turn display
- [ ] Update JSONL output format for turns

## Phase 2: Agent-Driven Input

- [ ] Parse `agents:` prefix in `-i` flag
- [ ] Parse `scripts:` prefix in `-i` flag
- [ ] Use standard `context.Options` for all agent invocations
- [ ] Pass `test_mode: "generator"` in `options.metadata`
- [ ] Pass target agent info (description, tools) in `options.metadata`
- [ ] Support query parameters (`?count=10&focus=...`) → merged into `options.metadata`
- [ ] Add `--dry-run` flag to save generated cases without running
- [ ] Create example generator agent with prompt template

## Phase 3: Dynamic Simulator

- [ ] Implement simulator invocation via `Assistant.Stream()` with `context.Options`
- [ ] `simulator.use` is direct agent ID (no prefix needed)
- [ ] Pass `test_mode: "simulator"` in `options.metadata`
- [ ] Pass persona, goal, turn_count from `simulator.options.metadata`
- [ ] Pass conversation history as messages
- [ ] Pass tool results in `options.metadata`
- [ ] Add goal completion detection (`goal_achieved` in response)
- [ ] Add max_turns limit and timeout
- [ ] Support hybrid mode (static turns + simulator fallback)
- [ ] Create example simulator agent with prompt template

## Phase 4: Interactive Mode

- [ ] Add `--interactive` flag
- [ ] Implement terminal input prompt with context display
- [ ] Add input timeout handling
- [ ] Support input history/editing
- [ ] Add `--skip-interactive` for CI/CD mode

## Phase 5: Enhanced Detection

- [ ] Add `awaiting_input` field to agent response schema
- [ ] Implement tool-based detection (confirmation tools)
- [ ] Add configurable detection rules
- [ ] Support custom detection via script/agent

## Phase 6: Agent-Driven Assertions

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

## Phase 7: Error Handling & Reporting

- [ ] Implement turn-level error handling
- [ ] Implement test-level error aggregation
- [ ] Add detailed error messages with hints
- [ ] Support custom reporter agent

## Open Questions

1. **Session Management**: How to handle session state across turns? Use existing session or create new per-test?

2. **Timeout Strategy**: Per-turn timeout vs. total test timeout?

3. **Parallel Execution**: Can multi-turn tests run in parallel, or must they be sequential?

4. **Retry Logic**: If a turn fails, retry just that turn or restart entire conversation?

5. **Snapshot Testing**: Should we support "golden file" comparison for conversation flows?

