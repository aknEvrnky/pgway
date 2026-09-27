[![CI](https://github.com/aknEvrnky/pgway/actions/workflows/ci.yml/badge.svg)](https://github.com/aknEvrnky/pgway/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/aknEvrnky/pgway/graph/badge.svg?token=5STIIF7V68)](https://codecov.io/gh/aknEvrnky/pgway)
[![GitHub Release](https://img.shields.io/github/v/release/aknEvrnky/pgway?include_prereleases)](https://github.com/aknEvrnky/pgway/releases)
[![GHCR](https://img.shields.io/badge/GHCR-aknevrnky%2Fpgway-blue)](https://github.com/aknEvrnky/pgway/pkgs/container/pgway)

# pgway

Proxy gateway for upstream HTTP and SOCKS5 proxies — one stable entry point while provider URLs and credentials change.

```text
Client → pgway → Upstream Proxy Pool → Target
```

> **Experimental** — the core gateway works; the **dashboard** (HTTP API + UI) is not production-ready.

## Docs

Guides, configuration, resource reference, auth, CLI, distributed mode:

**https://pgway.aknevrnky.dev/**

## Install

Binaries: `pgway` (all-in-one), `pgway-cp`, `pgway-dp`, `pgctl`.

**GitHub Release** (preferred for trying a tagged build):

1. Download the archive for your OS/arch from [Releases](https://github.com/aknEvrnky/pgway/releases) (`linux` / `darwin` × `amd64` / `arm64`).
2. Extract and put the binaries on your `PATH`.

**Container** (multi-arch image via GoReleaser → GHCR):

```bash
docker pull ghcr.io/aknevrnky/pgway:0.1.0-beta.1   # use a tag from the release
# default entrypoint is all-in-one pgway; override for pgctl:
docker run --rm --entrypoint /usr/local/bin/pgctl ghcr.io/aknevrnky/pgway:0.1.0-beta.1 version
```

Tags are pushed when a SemVer git tag is published (pre-release tags create GitHub pre-releases). Full install notes: **[Installation](https://pgway.aknevrnky.dev/getting-started/installation/)**.

## Quick start (from source)

Requires **Go 1.27+**.

```bash
git clone https://github.com/aknEvrnky/pgway.git
cd pgway
cp config/default.toml ./config.toml
make build

./build/pgway
# bootstrap token is printed on stderr (not structured logs):
./build/pgctl init --bootstrap-token <token>
./build/pgctl apply -f stack.yaml

curl -x http://localhost:8080 https://example.com
```

Configuration and operations: **[docs](https://pgway.aknevrnky.dev/)**.

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
- ✅ OpenTelemetry metrics
- ✅ OpenTelemetry tracing
- ✅ GitHub Releases + multi-arch GHCR images (GoReleaser, tag-triggered)
- 🚧 Dashboard — experimental; embedded UI in release builds; resource CRUD + flow editor + admin Users shipped; agents/settings later
- 🔜 Health checks with automatic pool recovery
- 🔜 Homebrew formula ([#100](https://github.com/aknEvrnky/pgway/issues/100)) — after dashboard stabilizes

## License

[Apache License 2.0](LICENSE)
