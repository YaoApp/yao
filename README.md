# Yao — App Runtime for the AI Era

Yao is an open-source runtime for building AI agents and web applications — shipped as a single binary.

![Mission Control](docs/mission-control.png)

**🏠 Homepage:** [https://yaoagents.com](https://yaoagents.com)

**📚 Docs:** [https://yaoagents.com/docs](https://yaoagents.com/docs)

**🖥️ Yao Desktop:** [https://yaoagents.com/download](https://yaoagents.com/download)

---

## How It Works

Think of Yao Agent as a **cage, not an animal**. What you put inside determines the behavior; the cage keeps it controlled.

Every request flows through the same pipeline:

![Pipeline](docs/pipeline.png)

`Create Hook` runs before the executor — inject context, enforce constraints, route requests.  
`Next Hook` runs after — validate output, trigger downstream actions, drive multi-step loops.  
**The AI does the heavy lifting. You define the boundaries.**

### Three Modes

| Mode | Executor | When to use |
|------|----------|-------------|
| **LLM** | OpenAI, Anthropic, etc. | Conversational assistants, Q&A, content generation |
| **CLI Agent** | OpenCode, Claude Code, Codex in a container | Computer use, sandbox isolation, SKILL ecosystem |
| **Pure Hook** | Your own TypeScript code | Deterministic logic, routing, menu flows — no AI needed |

All three share the same Hook interface. You can mix them freely — route some requests through the LLM, handle others with pure code, all inside a single `Create Hook`.

---

## Features

### Agent Framework

- **TypeScript Hooks** — `Create` and `Next` hooks intercept every request; built-in V8 engine
- **Native MCP Support** — Connect tools via process, SSE, or STDIO transport
- **Memory API** — Four scopes: request-level, session, user, team
- **Multi-Agent** — Delegate to specialist agents or call agents in parallel
- **CLI Agent / Sandbox** — Run Claude Code (or other CLI runners) in an isolated container with VNC desktop support
- **Skills Ecosystem** — Drop reusable capability packs (`SKILL.md`) into any CLI Agent

### Full-Stack Runtime

Everything in a single executable:

- **Data Models** — Define database tables and relations in JSON/YAML
- **REST APIs** — Map routes to model queries or TypeScript processors
- **SUI Pages** — Component-based web UI with server-side rendering
- **Chat UI (CUI)** — Built-in conversation interface for agents
- **TypeScript** — Built-in V8 engine; no Node.js required
- **Single Binary** — Runs on ARM64/x64; no Python, Node, or containers needed on the host

### Built-in Search

- **Vector Search** — Embeddings with OpenAI or FastEmbed
- **Knowledge Graph** — Entity-relationship retrieval
- **GraphRAG** — Hybrid vector + graph search

---

## About the Name

Yao (爻, yáo) is the fundamental symbol in the I Ching — the building block of the eight trigrams. Like a binary digit, it has two states. Their combinations describe the patterns of everything.
