## Yao Sandbox Environment

### Environment Variables

These environment variables are set by the Yao sandbox. **Always use these variables — never hardcode paths.**

| Variable            | Purpose                                    | Example                                |
| ------------------- | ------------------------------------------ | -------------------------------------- |
| `$WORKDIR`          | Sandbox working directory (project root)   | `/workspace`                           |
| `$HOME`             | Same as `$WORKDIR` (redirected by sandbox) | `/workspace`                           |
| `$CTX_SKILLS_DIR`   | Skills directory for this assistant        | `$WORKDIR/.yao/assistants/<id>/skills` |
| `$CTX_ASSISTANT_ID` | Current assistant ID                       | `yao.agent-smith`                      |
| `$CTX_WORKSPACE_ID` | Current workspace ID                       | `ws-abc123`                            |

### Path Rules

- **Use `$WORKDIR`** for all file paths — never hardcode `/workspace`
- **Use `$CTX_SKILLS_DIR`** for assistant-specific skills (custom skills provided by the assistant)
- System tool skills are in `$HOME/.claude/skills/` and are **auto-discovered** — you do not need to read them manually
- The `Read` and `Write` tools do **NOT** expand shell variables.
  Resolve first: `echo "$WORKDIR"`, then use the printed value.
- On Windows, use `$env:WORKDIR` / `$env:CTX_SKILLS_DIR` syntax instead.

### Workspace Path in Replies

When replying to users, **never expose raw `/workspace/...` paths**. Rewrite them using the `workspace://` scheme so the frontend can render them correctly.

**Format**: `workspace://<workspace-id>/relative/path`

**Step 1 — resolve the real workspace ID** (do this once per session):

```bash
echo "$CTX_WORKSPACE_ID"
```

Use the **actual printed value** (e.g. `ws-bfc4c2de-b53...`) in all subsequent replies.

**Step 2 — rewrite paths in replies**:

- `/workspace/output/result.png` → `workspace://<real-id>/output/result.png`

**Common mistakes** (all wrong):

- `workspace://$CTX_WORKSPACE_ID/...` ← shell variable literally in reply
- `workspace://ws-bfc4c2de-b53/...` ← example/placeholder ID instead of the real one
- `workspace://<workspace-id>/...` ← template placeholder instead of the real one
- `/workspace/output/...` ← raw path without `workspace://` scheme

You **must** run `echo "$CTX_WORKSPACE_ID"` and use the exact output.

### Attachments

User-uploaded files are placed in `$WORKDIR/.attachments/{chatID}/`.
When the user references an attached file, read it from this directory.

### Image Files

When you need to read, analyze, or describe an image (screenshot, photo, chart, diagram, etc.), **always use `image_read`** instead of trying to read binary files directly. The tool sends the image to a vision model and returns a text description.

```bash
tai tool image_read '{"image_path": "<file_path_or_url>", "prompt": "describe this image"}'
```

## Yao System Tools

You have access to Yao system tools via the `tai` command in bash.

**Calling convention**: `tai tool <name> '<json_args>'`

| Tool              | Skill (auto-loaded) | Description                                         |
| ----------------- | ------------------- | --------------------------------------------------- |
| `web_search`      | yao-web             | Search the web for real-time information            |
| `web_fetch`       | yao-web             | Fetch and read a web page by URL                    |
| `process_call`    | yao-process         | Execute a Yao Process (server-side function)        |
| `process_allowed` | yao-process         | Check which processes are allowed                   |
| `doc_list`        | yao-doc             | Search/list available process documentation         |
| `doc_inspect`     | yao-doc             | Get detailed docs for a specific process            |
| `doc_validate`    | yao-doc             | Validate a process name and get suggestions         |
| `image_read`      | yao-image           | Read and analyze images using a vision model        |
| `image_generate`  | yao-image           | Generate images from text prompts                   |
| `image_providers` | yao-image           | List available image generation or vision providers |

The system skills (`yao-web`, `yao-process`, `yao-doc`, `yao-image`) in `$HOME/.claude/skills/` are **auto-discovered** — they contain detailed parameter docs and workflow guidance. You do not need to manually read them; they are loaded automatically when your task matches their description.
