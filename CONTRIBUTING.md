# Contributing

Thanks for contributing to **pgway**. This guide is the practical checklist for working in the code repository. For deeper architecture and “what to watch for,” see the docs site: [Contributing guide](https://pgway.aknevrnky.dev/docs/guides/contributing/).

## Prerequisites

- Go **1.27+** (`go version`)
- Git
- Optional: [protoc](https://grpc.io/docs/protoc-installation/) if you change `.proto` files
- Optional: Bun/Node only for the experimental dashboard under `frontend/`

## Setup

```bash
git clone https://github.com/aknEvrnky/pgway.git
cd pgway
make build
make tools   # gotestsum (and other make helpers)
make test
```

Ensure `$(go env GOPATH)/bin` is on your `PATH` so `gotestsum` is found.

## Project shape (short)

| Binary | Role |
|--------|------|
| `pgway` | Control Plane + Data Plane (all-in-one) |
| `pgway-cp` | Control Plane only |
| `pgway-dp` | Data Plane agent |
| `pgctl` | CLI against the Control Plane |

Hexagonal layout: domain under `internal/application/core/domain`, CP under `controlplane` / `auth` / `agent`, DP under `dataplane/{api,agenthost,balancer,consumer}`, contracts in `ports/`, I/O in `adapters/`. Wiring lives in `cmd/*/main.go`.

**Hard rule:** `dataplane` packages must not import `controlplane`, `auth`, or `agent`. Import boundaries are checked by `internal/architecture/import_rules_test.go`.

## Before you open a PR

1. Prefer a focused branch and small commits.
2. Add/update tests for every behavior change (`*_test.go`, table-driven + testify).
3. Run:

   ```bash
   make test
   go vet ./...
   ```

4. If you touch `proto/`, regenerate and commit generated code:

   ```bash
   make proto
   ```

   CI fails if `gen/` drifts.

5. If the change is user/operator-facing (YAML schema, binaries, auth, CLI, install, architecture), update the **separate** docs repo ([pgway-docs](https://github.com/aknEvrnky/pgway-docs)) — not this repo’s local `docs/` notes.

6. Do **not** commit editor/agent tooling (`.cursor/`, `.claude/`, `CLAUDE.md`, IDE junk).

7. Fill in the PR template. Link related issues.

## Pull requests

- One concern per PR when practical.
- Describe **why**, not only what.
- Call out breaking changes, proto/schema migrations, and docs updates.
- CI must be green (lint, tests, proto check).

## Issues

Use the GitHub issue templates (bug / feature). Include version (`go version`, how you built), config snippets (redact secrets), and reproduction steps for bugs.

## Code of collaboration

- Be respectful in reviews and issues.
- Prefer questions over assumptions when requirements are unclear.
- Experimental dashboard/REST is not the bar for gateway correctness — prefer `pgctl` / gRPC paths for core features.
