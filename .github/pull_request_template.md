## Summary

<!-- What changed and why (1–3 bullets). -->

-

## Related issues

<!-- e.g. Fixes #123 -->

-

## Type of change

- [ ] Bug fix
- [ ] Feature / enhancement
- [ ] Refactor (no user-facing behavior change)
- [ ] Docs / CI / chore
- [ ] Breaking change (describe below)

## Architecture checklist

- [ ] Code lives in the right plane/package (`dataplane` vs `controlplane` / `auth` / `agent`)
- [ ] No `dataplane` → `controlplane`/`auth`/`agent` imports
- [ ] New capabilities define or use a port in `ports/` before adapters
- [ ] Domain stays free of adapter/framework I/O

## Tests

- [ ] Unit/integration tests added or updated
- [ ] `make test` passes locally
- [ ] `go vet ./...` clean

## Proto / schema (if applicable)

- [ ] `make proto` run and `gen/` committed
- [ ] YAML/schema validation + examples updated if fields changed

## Docs

- [ ] Not needed
- [ ] Updated in [pgway-docs](https://github.com/aknEvrnky/pgway-docs) (link PR/commit)

## Notes for reviewers

<!-- Risk areas, how you tested, screenshots if UI (experimental). -->
