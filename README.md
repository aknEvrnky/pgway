[![CI](https://github.com/aknEvrnky/pgway/actions/workflows/ci.yml/badge.svg)](https://github.com/aknEvrnky/pgway/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/aknEvrnky/pgway/graph/badge.svg?token=5STIIF7V68)](https://codecov.io/gh/aknEvrnky/pgway)

# pgway

Proxy gateway for upstream HTTP and SOCKS5 proxies — one stable entry point while provider URLs and credentials change.

```text
Client → pgway → Upstream Proxy Pool → Target
```

> **Experimental** — the core gateway works; the **dashboard** and **REST API** are not production-ready.

## Docs

Guides, configuration, resource reference, auth, CLI, distributed mode:

**https://pgway.aknevrnky.dev/**

## Quick start

Requires **Go 1.27+**.

```bash
git clone https://github.com/aknEvrnky/pgway.git
cd pgway
cp config/default.toml ./config.toml
make build

./build/pgway
# use the bootstrap token from the server log:
./build/pgctl init --bootstrap-token <token>
./build/pgctl apply -f stack.yaml

curl -x http://localhost:8080 https://example.com
```

Binaries: `pgway` (all-in-one), `pgway-cp`, `pgway-dp`, `pgctl`.  
More install options, configuration, and operations: **[docs](https://pgway.aknevrnky.dev/)**.

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md) and the [contributing guide](https://pgway.aknevrnky.dev/guides/contributing/).

## Status & Roadmap

- ✅ HTTP proxy, CONNECT tunneling, round-robin load balancing
- ✅ SOCKS5 upstream proxies (clients still speak HTTP proxy to pgway)
- ✅ Router with multiple match types and composite conditions
- ✅ Static and dynamic (label-selector) pools
- ✅ gRPC Control Plane, CLI, BadgerDB storage
- ✅ Token authentication for gRPC, user management (`pgctl init/login/user`)
- ✅ Weighted load balancing
- ✅ Least-bytes load balancing
- 🚧 Dashboard (in progress), REST API (in flux)
- 🔜 Prometheus metrics
- 🔜 Health checks with automatic pool recovery
- 🔜 Authentication for REST / dashboard

## License

[Apache License 2.0](LICENSE)
