# Yao Developer Scripts

Helper scripts for running and testing Yao from source. All scripts live in
`yao/bin/` and resolve their own paths, so they work from anywhere (direct
invocation, symlinks, or `PATH`).

## Prerequisites

| Tool | Version | Notes |
|------|---------|-------|
| Go | 1.25+ | Required for `yao-dev` and `yao-test` |
| Docker | 20+ | Required for `yao-test` only |
| Local repos | — | `yao`, `gou`, `xun`, `kun`, `v8go` must be sibling directories |

Expected directory layout:

```
Yao/
├── yao/          # this repo
├── gou/          # go.mod replace dependency
├── xun/          # go.mod replace dependency
├── kun/          # go.mod replace dependency
├── v8go/         # go.mod replace dependency
└── tai/          # optional, mounted if present
```

---

## yao-dev — Run Yao from Source

Compiles and runs Yao on-the-fly using `go -C`. The current working directory
is used as the application root (`YAO_ROOT`).

### Usage

```bash
# Initialize a new app
mkdir /tmp/myapp && cd /tmp/myapp
/path/to/yao/bin/yao-dev init

# Start the server
cd /tmp/myapp
/path/to/yao/bin/yao-dev start

# Run a process
/path/to/yao/bin/yao-dev run -s models.__yao.member.Find 1 1

# Any yao subcommand works
/path/to/yao/bin/yao-dev agent test -n myagent -i "hello" -v
```

### How It Works

1. Resolves the Yao source directory from the script's own location (`bin/` → parent).
2. Walks up from `$PWD` to find `app.yao` and sets `YAO_ROOT`.
3. Converts relative path arguments to absolute (since `go -C` changes CWD).
4. Runs `go -C $YAO_SOURCE run . <args>`.

---

## yao-test — Run Go Tests in a Container

Spins up a throwaway Docker container with all local repos volume-mounted,
so `go test` runs in a clean Linux environment while reflecting live host
edits instantly.

### Quick Start

```bash
# Run a specific test
bin/yao-test -v -run TestPickNode ./agent/sandbox/v2/...

# Run all tests in a package
bin/yao-test ./agent/assistant/...

# Run the full suite
bin/yao-test ./...

# Verify compilation only
bin/yao-test --build

# Drop into an interactive shell for debugging
bin/yao-test --shell
```

### Volume Mounts

Mounts are built automatically from `yao/go.mod`:

| Host Path | Container Path | Source |
|-----------|---------------|--------|
| `Yao/yao` | `/workspace/yao` | Always mounted |
| `Yao/gou` | `/workspace/gou` | Parsed from `replace ../gou` |
| `Yao/xun` | `/workspace/xun` | Parsed from `replace ../xun` |
| `Yao/kun` | `/workspace/kun` | Parsed from `replace ../kun` |
| `Yao/v8go` | `/workspace/v8go` | Parsed from `replace ../v8go` |
| `Yao/tai` | `/workspace/tai` | Mounted if directory exists |
| `~/.cache/yao-test-gomod` | `/go/pkg/mod` | Persistent module cache |
| `~/.cache/yao-test-gobuild` | `/root/.cache/go-build` | Persistent build cache |

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GO_IMAGE` | `golang:1.25` | Docker image for the container |
| `EXTRA_ARGS` | _(empty)_ | Extra `docker run` flags, e.g. `--cpus 4` |

### Modes

| Flag | Action |
|------|--------|
| _(default)_ | `go test <args>` |
| `--build` | `go build <args>` (compilation check) |
| `--shell` | Interactive `bash` session inside the container |

---

## Integration Testing Workflow

For full integration tests that need a running Yao server and Tai node
(e.g. sandbox / agent tests), use a long-lived container instead of
the throwaway `yao-test` approach:

```bash
# 1. Start a persistent container with all repos mounted
docker run -d --name yao-dev-test \
  --platform linux/arm64 \
  -v ~/Yao/yao:/workspace/yao \
  -v ~/Yao/kun:/workspace/kun \
  -v ~/Yao/xun:/workspace/xun \
  -v ~/Yao/gou:/workspace/gou \
  -v ~/Yao/v8go:/workspace/v8go \
  -v ~/Yao/tai:/workspace/tai \
  -v ~/Yao/yaoagents-releases/app:/data/app \
  -v ~/.cache/yao-test-gomod:/go/pkg/mod \
  -v ~/.cache/yao-test-gobuild:/root/.cache/go-build \
  -e GOFLAGS=-mod=mod \
  golang:1.25 bash -c 'trap "kill 0" EXIT; sleep infinity'

# 2. Start Yao from source
docker exec -d yao-dev-test bash -c \
  'cd /data/app && /workspace/yao/bin/yao-dev start > /tmp/yao.log 2>&1'

# 3. Generate Tai credentials and start Tai
docker exec yao-dev-test bash -c '
  cd /data/app
  INFO=$(/workspace/yao/bin/yao-dev run -s models.__yao.member.Find 1 1 2>&1 | grep "^{")
  TEAM=$(echo "$INFO" | grep -o "\"team_id\":\"[^\"]*\"" | head -1 | cut -d\" -f4)
  MEMBER=$(echo "$INFO" | grep -o "\"member_id\":\"[^\"]*\"" | head -1 | cut -d\" -f4)
  TOKEN=$(/workspace/yao/bin/yao-dev run -s oauth.token.MakeByUser "$TEAM" "$MEMBER" 86400 2>&1 | sed -n "/^eyJ/p" | head -1)
  mkdir -p /data/app/.tai/volumes
  printf "{\"client_id\":\"dev\",\"tai_id\":\"dev-node\",\"machine_id\":\"dev\",\"server\":\"http://127.0.0.1:5099\",\"access_token\":\"%s\",\"scope\":\"tai:tunnel\",\"expires_at\":\"2099-01-01T00:00:00Z\",\"registered\":true}" "$TOKEN" \
    | base64 -w0 > /data/app/.tai/credentials
'

cat > /tmp/tai-dev.yml <<'EOF'
yao:
  server: "http://127.0.0.1:5099"
  node_id: "dev-node"
  display_name: "Dev HostExec"
credentials: "/data/app/.tai/credentials"
data: "/data/app/.tai/volumes"
host_exec:
  enabled: true
  full_access: true
log:
  level: debug
EOF
docker cp /tmp/tai-dev.yml yao-dev-test:/tmp/tai-dev.yml

docker exec -d yao-dev-test bash -c \
  '/workspace/tai/bin/tai-dev server -config /tmp/tai-dev.yml > /tmp/tai.log 2>&1'

# 4. Verify both services are running
docker exec yao-dev-test bash -c '
  curl -sf http://127.0.0.1:5099/.well-known/yao | head -c 60 && echo
  grep "registered with Yao" /tmp/tai.log && echo "Tai OK"
'

# 5. Run agent tests
docker exec yao-dev-test bash -c \
  'cd /data/app && /workspace/yao/bin/yao-dev agent test -n yao.general -i "hello" -c default -v'

# 6. Clean up
docker rm -f yao-dev-test
```

> **Tip**: Mount `yaoagents-releases/app` as `/data/app` for a ready-made
> application with assistants, models, and seeds already configured. This
> avoids the need to run `yao init` and set up agents from scratch.
