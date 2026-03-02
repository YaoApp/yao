# registry

Go client SDK for [Yao Registry](https://github.com/YaoApp/registry).

## Usage

```go
import "github.com/yaoapp/yao/registry"

// Only the server URL is required. API prefix is auto-discovered
// via /.well-known/yao-registry on the first call.
c := registry.New("https://registry.yaoagents.com",
    registry.WithAuth("user", "pass"),  // optional, for push/delete
)

// Push a .yao.zip package
result, err := c.Push("assistants", "@yao", "hello", "1.0.0", zipBytes)

// Pull (by version or dist-tag)
data, digest, err := c.Pull("assistants", "@yao", "hello", "latest")

// Query
pack, err := c.GetPackument("assistants", "@yao", "hello")
ver, err  := c.GetVersion("assistants", "@yao", "hello", "1.0.0")
list, err := c.Search("hello", "assistants", 1, 20)

// Dependencies
deps, err := c.GetDependencies("assistants", "@yao", "hello", "1.0.0", true)

// Dist-tags
c.SetTag("assistants", "@yao", "hello", "stable", "1.0.0")
c.DeleteTag("assistants", "@yao", "hello", "stable")

// Delete
c.DeleteVersion("assistants", "@yao", "hello", "1.0.0")
```

## Options

| Option | Description |
|--------|-------------|
| `WithAuth(user, pass)` | Basic Auth for push/delete |
| `WithHTTPClient(hc)` | Custom `*http.Client` |
| `WithTimeout(d)` | HTTP timeout |

## Environment

Tests require a running registry server. Set `YAO_REGISTRY_URL` (default `http://localhost:8080`) and create user `yaoagents`/`yaoagents`.

```bash
# Start registry with test user
REGISTRY_INIT_USER=yaoagents REGISTRY_INIT_PASS=yaoagents registry start

# Run tests
go test ./registry/... -v
```
