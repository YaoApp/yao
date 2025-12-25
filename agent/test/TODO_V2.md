# Agent Test Framework V2 - Implementation TODO

## Phase 1: Static Multi-Turn

- [ ] Extend test case parser for `turns` array
- [ ] Implement turn-by-turn execution
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
- [ ] Define generator agent interface (GeneratorInput/GeneratorOutput)
- [ ] Implement generator invocation
- [ ] Support query parameters (`?count=10&focus=...`)
- [ ] Pass target agent metadata to generator (description, tools)
- [ ] Add `--dry-run` flag to save generated cases without running
- [ ] Create example generator agent

## Phase 3: Dynamic Simulator

- [ ] Define simulator agent interface (SimulatorInput/SimulatorOutput)
- [ ] Implement simulator invocation with full context
- [ ] Pass conversation history and tool results to simulator
- [ ] Add goal completion detection
- [ ] Add max_turns limit and timeout
- [ ] Support hybrid mode (static turns + simulator fallback)
- [ ] Create example simulator agent

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

## Phase 6: Error Handling & Reporting

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

