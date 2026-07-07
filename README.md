# helmsman

self-hosted control plane for opentelemetry collector fleets. speaks opamp
to every managed collector: live fleet health, remote config push, immutable
version history, one-click rollback, rbac, and an append-only audit trail.

csc 4350 group project - team lokk (group 14), gsu, summer 2026.

## layout

| path | what |
|---|---|
| `ui/` | react + vite + typescript spa (login, fleet dashboard) |
| `api/` | fastapi management plane (auth, rbac, agents, configurations) |
| `control-plane/` | go + opamp-go agent control plane skeleton |
| `database/` | postgres schema + seed (source of truth: docs/submissions/data-model.md) |
| `docs/` | assignment submissions, instructions, meeting notes |

## quickstart

requires docker, uv, node 20+, and go.

```sh
# postgres (schema + seed auto-applied on first boot)
docker compose up -d db

# management api -> http://localhost:8000 (swagger at /docs)
cd api && uv sync && uv run uvicorn app.main:app --reload

# ui -> http://localhost:5173
cd ui && npm install && npm run dev
```

seeded demo logins:

| email | password | role |
|---|---|---|
| admin@helmsman.local | admin123! | admin |
| operator@helmsman.local | operator123! | operator |
| viewer@helmsman.local | viewer123! | viewer |

## tests

```sh
cd api && uv run pytest
cd ui && npm test
cd control-plane && go test ./...
```

## workflow

main + feature branches, PRs required into main, at least one non-author
review before merge. see docs/submissions/team-info-and-contract.md.
