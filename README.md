[![CI](https://github.com/aknEvrnky/pgway/actions/workflows/ci.yml/badge.svg)](https://github.com/aknEvrnky/pgway/actions/workflows/ci.yml)

# pgway

A proxy gateway that manages HTTP/SOCKS5 upstream proxies through a centralized entry point.

> ⚠️ **Experimental** — pgway is under active development. The core proxy gateway is working and tested; **the Dashboard and REST API surface are subject to breaking changes**. Use with caution if you plan to self-host.

```
Client → pgway (Gateway) → Upstream Proxy Pool → Target Server
```

## What it does

- Single, stable entry point for your upstream proxy infrastructure
- Static or dynamic (label-based) proxy pools
- Request routing by host, path, method, header or custom rules
- Round-robin load balancing (weighted and least-bytes in progress)
- Control Plane / Data Plane separation over gRPC (single-process or distributed)
- CLI (`pgctl`) for declarative resource management
- Web dashboard for visual configuration (work in progress)

## Dashboard

![pgway dashboard](.art/dashboard.png)

The dashboard is **actively in development**. It is built with Nuxt 4, Vue 3, PrimeVue and Vue Flow, and talks to the Control Plane over REST. Expect rough edges and breaking changes.

## Architecture

```mermaid
flowchart LR
  Client -->|HTTP / CONNECT| EP[Entrypoint]
  Dashboard -->|REST| CP[Control Plane]
  CLI[pgctl] -->|gRPC| CP
  CP -->|gRPC| DP[Data Plane]
  EP --> DP
  CP -->|read / write| DB[(BadgerDB)]
  DP -->|forward| UP[Upstream Proxy Pool]
  UP --> Target[Target Server]
```

The Control Plane owns configuration (stored in BadgerDB) and exposes gRPC + REST APIs. The Data Plane serves traffic on configured entrypoints and resolves routing decisions against the Control Plane (directly in single-process mode, or over gRPC in distributed mode).

## Flow Model

Resources are composed into a pipeline that routes client traffic to an upstream proxy:

```mermaid
flowchart LR
  EP[Entrypoint] --> FL[Flow]
  FL -.optional.-> RT[Router]
  FL --> LB[LoadBalancer]
  RT --> LB
  LB --> PL[Pool]
  PL --> PX[Proxy]
```

- **Entrypoint** — required start of the pipeline (host:port listener)
- **Router** — optional; matches on host / path / method / header and picks a target balancer
- **LoadBalancer** — required; sits in front of exactly one pool
- **Pool** — group of proxies (static list or dynamic label selector)
- **Proxy** — single upstream proxy (HTTP or SOCKS5)

A minimal valid flow is `Entrypoint → LoadBalancer → Pool → Proxy`. Multiple entrypoints can be served from the same process on different ports.

## Binaries

| Binary     | Description |
|------------|-------------|
| `pgway`    | All-in-one: Control Plane + Data Plane in a single process |
| `pgway-cp` | Standalone Control Plane (gRPC + REST server, BadgerDB) |
| `pgway-dp` | Standalone Data Plane (gateway, connects to CP via gRPC) |
| `pgctl`    | CLI client for applying, reading and deleting resources |

## Installation

### Prerequisites

- Go **1.25** or newer (`go version`)
- Optional, for dashboard development: [Bun](https://bun.sh/) or Node.js 20+

### From source

```bash
git clone https://github.com/aknEvrnky/pgway.git
cd pgway
make build            # builds all four binaries into ./build/
```

`make build` produces `build/pgway`, `build/pgway-cp`, `build/pgway-dp` and
`build/pgctl`. To build a single binary directly:

```bash
go build -o pgway ./cmd/pgway
go build -o pgway-cp ./cmd/pgway-cp
go build -o pgway-dp ./cmd/pgway-dp
go build -o pgctl ./cmd/pgctl
```

### Via `go install`

```bash
go install github.com/aknEvrnky/pgway/cmd/pgway@latest
go install github.com/aknEvrnky/pgway/cmd/pgway-cp@latest
go install github.com/aknEvrnky/pgway/cmd/pgway-dp@latest
go install github.com/aknEvrnky/pgway/cmd/pgctl@latest
```

## Quick Start

pgway needs a config file on startup. Copy the default one into a location it will find (see [Configuration](#configuration)):

```bash
cp config/default.yml ./config.yml
# or system-wide:
# sudo mkdir -p /etc/pgway && sudo cp config/default.yml /etc/pgway/config.yml
```

Start the all-in-one binary:

```bash
./pgway
```

On first start the server logs a one-time **bootstrap token**. Use it to create
the first admin user (this also logs you in):

```bash
./pgctl init --bootstrap-token <token-from-server-log>
```

Describe your stack in YAML — the example below defines one upstream proxy, a static pool, a round-robin balancer and an entrypoint listening on `:8080`:

```yaml
# stack.yaml
kind: Proxy
version: v1
metadata:
  name: proxy-1
  labels:
    provider: example
    region: us-east
spec:
  url: http://user:pass@1.2.3.4:8080
---
kind: Pool
version: v1
metadata:
  name: main-pool
spec:
  title: Main Pool
  type: static
  proxy_ids:
    - proxy-1
---
kind: LoadBalancer
version: v1
metadata:
  name: main-rr
spec:
  title: Main Round Robin
  type: round-robin
  pool_id: main-pool
---
kind: Flow
version: v1
metadata:
  name: main-flow
spec:
  balancer_id: main-rr
---
kind: Entrypoint
version: v1
metadata:
  name: main-ep
spec:
  title: Main Gateway
  protocol: http
  host: 0.0.0.0
  port: 8080
  flow_id: main-flow
```

Apply and inspect:

```bash
./pgctl apply -f stack.yaml
./pgctl get proxy
./pgctl get pool
./pgctl get balancer
./pgctl get flow
./pgctl get entrypoint
```

Send traffic through the gateway:

```bash
curl -x http://localhost:8080 https://example.com
```

## Resource Types

### Proxy

A single upstream proxy. Either the `url` shorthand or explicit fields.

```yaml
kind: Proxy
version: v1
metadata:
  name: proxy-1
  labels:
    provider: example
spec:
  url: http://user:pass@1.2.3.4:8080
  # — or —
  # protocol: http
  # host: 1.2.3.4
  # port: 8080
  # username: user
  # password: pass
```

### Pool

Static list by ID, or dynamic by matching labels on proxies.

```yaml
# static
kind: Pool
version: v1
metadata:
  name: static-pool
spec:
  type: static
  proxy_ids: [proxy-1, proxy-2]
---
# dynamic
kind: Pool
version: v1
metadata:
  name: dynamic-pool
spec:
  type: dynamic
  selector:
    allow:
      provider: example
      region: us-east
```

### LoadBalancer

Sits in front of exactly one pool.

```yaml
kind: LoadBalancer
version: v1
metadata:
  name: rr
spec:
  type: round-robin   # round-robin | weighted (wip) | least-bytes (wip)
  pool_id: dynamic-pool
```

### Router

Optional rule engine between entrypoint and load balancer. Rules are evaluated in order; the first match wins.

Supported match types: `host`, `host_suffix`, `path_prefix`, `path_regex`, `method`, `header`, `catch_all`, plus composite `all` (AND), `any` (OR) and `not`.

```yaml
kind: Router
version: v1
metadata:
  name: main-router
spec:
  rules:
    - id: api-traffic
      match:
        type: host_suffix
        value: api.example.com
      target: api-rr
    - id: catch-all
      match:
        type: catch_all
      target: main-rr
```

### Flow

Wires a router and/or balancer behind an entrypoint.

```yaml
kind: Flow
version: v1
metadata:
  name: main-flow
spec:
  router_id: main-router   # optional
  balancer_id: main-rr
```

### Entrypoint

A network listener that serves the flow.

```yaml
kind: Entrypoint
version: v1
metadata:
  name: main-ep
spec:
  protocol: http
  host: 0.0.0.0
  port: 8080
  flow_id: main-flow
```

## Configuration

All four binaries share the same configuration keys and loader. Values are
resolved with the following precedence (highest wins):

1. **Flags** — `--config <path>` (all binaries) and `--token` (`pgctl`)
2. **Environment variables** — any key with the `PGWAY_` prefix (e.g. `PGWAY_TOKEN`)
3. **Config file** — see the search order below
4. **Built-in defaults** — the values in the table below

When `--config <path>` is given, exactly that file is loaded. Otherwise the
binaries search for a `config.{yml,yaml,json,toml}` file in these directories,
first match wins:

1. `/etc/pgway/`
2. `$HOME/.pgway/`
3. `.` (current directory)

A config file must exist on one of these paths (or be passed via `--config`)
even when every value is supplied through environment variables.

| Key                | Default          | Description |
|--------------------|------------------|-------------|
| `badger_path`      | `/var/pgway/lib` | BadgerDB storage directory (`pgway`, `pgway-cp`) |
| `grpc_listen_addr` | `:9090`          | gRPC Control Plane listen/dial address |
| `rest_listen_addr` | `:8081`          | REST API listen address (`pgway`, `pgway-cp`) |
| `token`            | *(empty)*        | Bearer token for outgoing CP calls (`pgctl`, `pgway-dp`) |
| `token_ttl`        | `720h`           | Default lifetime of login-issued tokens (`pgway`, `pgway-cp`) |

Example `config.yml` (copy `config/default.yml` as a starting point):

```yaml
badger_path: /var/pgway/lib
grpc_listen_addr: ":9090"
rest_listen_addr: ":8081"
token: ""          # only needed by pgway-dp / pgctl in distributed mode
token_ttl: 720h
```

## Authentication

All gRPC endpoints require a bearer token; only `AuthService/Login` and
`AuthService/InitAdmin` are exempt. The REST API does not enforce
authentication yet (tracked separately for the dashboard integration).

**Bootstrap.** When the server starts with no users, it logs a one-time
bootstrap token and only `InitAdmin` is usable — there is no unauthenticated
window. A restart before init generates a fresh token.

```bash
./pgctl init --bootstrap-token <token>   # creates the admin, stores a token in ~/.pgctl/credentials
```

**Sessions.** `pgctl login` exchanges credentials for an opaque token (stored
hashed server-side, revocable at any time):

```bash
pgctl login --username admin             # default TTL (token_ttl)
pgctl login --username dp-agent --no-expiry   # long-lived token for automation
pgctl logout                             # revoke current token
```

`pgctl` resolves its token in this order: `--token` flag → `PGWAY_TOKEN` env /
`token` config key → `~/.pgctl/credentials` (written by `pgctl login`). The
credentials file lives under `~/.pgctl/` (the CLI's own state), separate from
the shared config search path `~/.pgway/`.

**Users.** Passwords are stored bcrypt-hashed. Changing a password revokes all
of that user's tokens.

```bash
pgctl user create bob                    # admin only; prints a one-time temporary password
pgctl user create alice --role admin
pgctl user list                          # admin only
pgctl user change-password               # own password
pgctl user change-password bob           # admin reset
pgctl user delete bob                    # admin only; the last admin cannot be deleted
```

**Distributed mode.** `pgway-dp` also authenticates to the CP: create a user
for it, log in with `--no-expiry`, and put the token in the DP's config
(`token` key or `PGWAY_TOKEN`).

## Distributed Mode

Run the Control Plane and Data Plane as separate processes. Each binary loads
config the same way (see [Configuration](#configuration)); a distinct file can
be pointed at with `--config`:

```bash
# Terminal 1 — Control Plane
./pgway-cp --config /etc/pgway/cp.yml

# Terminal 2 — Data Plane
./pgway-dp --config /etc/pgway/dp.yml

# Terminal 3 — Manage
./pgctl apply -f stack.yaml
```

The Data Plane dials the CP at `grpc_listen_addr` and authenticates with
`token`. Both can come from the config file, an environment variable, or a
dedicated `--config` file — pick whichever fits your deployment:

```yaml
# dp.yml — data plane config
grpc_listen_addr: "cp-host:9090"
token: "<long-lived token from: pgctl login --no-expiry>"
```

```bash
# equivalently, entirely via environment variables
PGWAY_GRPC_LISTEN_ADDR="cp-host:9090" PGWAY_TOKEN="<token>" ./pgway-dp --config ./config.yml
```

> The CLI credentials file moved from `~/.pgway/credentials` to
> `~/.pgctl/credentials`. If you upgraded from an earlier build, re-run
> `pgctl login` (or `pgctl init`) to write the token to the new location.

## Development

### Tests

```bash
# Unit tests
go test ./internal/...

# Integration tests (BadgerDB, gRPC, end-to-end flow)
go test ./integration/...

# Everything
go test ./...
```

### Dashboard

```bash
cd frontend
bun install
bun run dev    # http://localhost:3000
```

### Project layout

```
cmd/                  # entry points: pgway, pgway-cp, pgway-dp, pgctl
internal/
  application/        # core business logic (control plane, balancer, routing)
  adapters/           # grpc, rest, http, cli, repository implementations
  ports/              # interface definitions
  platform/           # config, logger
frontend/             # Nuxt 4 dashboard (WIP)
proto/                # gRPC protobuf definitions
integration/          # end-to-end tests
```

The codebase follows a hexagonal (ports & adapters) layout: `internal/ports` defines interfaces, `internal/adapters` contains concrete implementations, and the application core has no framework or I/O dependencies.

## Status & Roadmap

- ✅ HTTP proxy, CONNECT tunneling, round-robin load balancing
- ✅ Router with multiple match types and composite conditions
- ✅ Static and dynamic (label-selector) pools
- ✅ gRPC Control Plane, CLI, BadgerDB storage
- ✅ Token authentication for gRPC, user management (`pgctl init/login/user`)
- 🚧 Dashboard (in progress), REST API (in flux)
- 🚧 Weighted and least-bytes load balancing
- 🔜 SOCKS5 support
- 🔜 Authentication for REST / dashboard
- 🔜 Health checks with automatic pool recovery
- 🔜 Prometheus metrics

## License

MIT
