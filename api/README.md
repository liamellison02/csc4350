# helmsman api

fastapi management plane for helmsman.
serves the auth, fleet, and configuration endpoints the react ui consumes,
backed by postgres (schema in ../database/).

## stack

python 3.12, fastapi, sqlalchemy 2, pyjwt, bcrypt. managed with uv.

## run (dev)

start the database from the repo root first:

```sh
docker compose up -d db
```

then, from api/:

```sh
uv sync
uv run uvicorn app.main:app --reload
```

api listens on http://localhost:8000. interactive docs at /docs.

## env

copy `.env.example` to `.env` and adjust as needed.

| var | default | notes |
|---|---|---|
| DATABASE_URL | postgresql+psycopg://helmsman:helmsman@localhost:5432/helmsman | sqlalchemy url |
| JWT_SECRET | dev-secret-change-me | hs256 signing key, change outside dev |
| JWT_TTL_MINUTES | 60 | token lifetime |
| CORS_ORIGINS | http://localhost:5173 | comma separated allowed origins |

## endpoints

| method | path | auth | notes |
|---|---|---|---|
| GET | /healthz | none | liveness probe |
| POST | /auth/login | none | email + password, returns jwt + user |
| GET | /auth/me | bearer | current user |
| GET | /agents | bearer, any role | full agent rows |
| POST | /configurations | bearer, operator+ | creates config, writes audit row |

demo credentials (seeded by ../database/seed.sql):
admin@helmsman.local / admin123!,
operator@helmsman.local / operator123!,
viewer@helmsman.local / viewer123!

## tests + lint

```sh
uv run pytest
uv run ruff check .
```

tests run against in-memory sqlite, no database needed.
