# pgway

A proxy gateway that manages HTTP/SOCKS5 upstream proxies through a centralized entry point.

```
Client → pgway (Gateway) → Upstream Proxy Pool → Target Server
```

## What it does

- Single entry point for your proxy infrastructure
- Manage proxy pools with static lists or dynamic label-based selection
- Route requests by host, path, method, headers, or custom rules
- Round-robin load balancing (weighted and least-bytes coming soon)
- Control Plane / Data Plane separation with gRPC communication
- CLI tool for resource management
- All-in-one or distributed deployment

## Architecture

pgway has three binaries:

| Binary | Description |
|--------|-------------|
| `pgway` | All-in-one: runs CP + DP in a single process |
| `pgway-cp` | Standalone Control Plane (gRPC server + BadgerDB) |
| `pgway-dp` | Standalone Data Plane (gateway, connects to CP via gRPC) |
| `pgctl` | CLI tool for managing resources via gRPC |

## Quick Start

### Build

```bash
go build -o pgway ./cmd/pgway
go build -o pgctl ./cmd/pgctl
```

### Run (all-in-one)

```bash
./pgway
```

### Create resources

```yaml
# stack.yaml
kind: Proxy
version: v1
metadata:
  name: proxy-1
  labels:
    provider: luminati
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

```bash
./pgctl apply -f stack.yaml
./pgctl get proxy
./pgctl get pool
./pgctl get balancer
./pgctl get flow
./pgctl get entrypoint
```

### Distributed mode

```bash
# Terminal 1 — Control Plane
./pgway-cp

# Terminal 2 — Data Plane
./pgway-dp

# Terminal 3 — Manage
./pgctl apply -f stack.yaml
```

## Resource Types

- **Proxy** — upstream proxy server (HTTP or SOCKS5)
- **Pool** — group of proxies (static list or dynamic label selector)
- **LoadBalancer** — balancing algorithm over a pool (round-robin, weighted, least-bytes)
- **Router** — request matching rules that route to different balancers
- **Flow** — connects a router or balancer to an entrypoint
- **Entrypoint** — listener that accepts client connections

## Testing

```bash
# Unit tests
go test ./internal/...

# Integration tests (BadgerDB, gRPC, full pipeline)
go test ./integration/...

# All tests
go test ./...
```

## License

MIT