# control-plane

the go opamp control plane for helmsman. it runs an
[opamp-go](https://github.com/open-telemetry/opamp-go) server that collectors
connect to over the opamp protocol. on connect it upserts a row in the
postgres `agents` table, keeps `status` / `last_seen` current while the
connection lives, and flips the agent to `disconnected` when the connection
closes.

this is a sprint 2 scaffold. it compiles, is unit tested, and wires the real
opamp-go callback api, but it has not been run against a live database or a
live collector yet. end-to-end runtime validation is sprint 3 scope.

## layout

- `cmd/controlplane` - main entrypoint: load config, connect postgres, start
  the opamp server, block until a signal, shut down cleanly.
- `internal/config` - env loading with defaults.
- `internal/opamp` - opamp-go server wiring plus pure attribute-mapping
  helpers (`attrs.go`).
- `internal/store` - pgxpool wrapper: `UpsertAgent` and `MarkDisconnected`.

## config

read from the environment, with defaults:

- `DATABASE_URL` - postgres url, default
  `postgres://helmsman:helmsman@localhost:5432/helmsman`.
- `OPAMP_LISTEN` - opamp listen endpoint, default `:4320`.

## agent attribute mapping

opamp agent descriptions carry identifying attributes by opentelemetry
semantic convention. they map to `agents` columns as:

- `host.name` (falls back to `host.hostname`) -> `hostname`
- `service.name` -> `agent_type`
- `service.version` -> `version`

missing attributes fall back to `unknown` so the not-null `hostname` column is
always satisfied. the instance uid arrives as 16 raw bytes and is formatted as
a canonical uuid string for the `instance_uid` primary key.

## develop

standard go module, toolchain go 1.26.

```
go build ./...
go vet ./...
go test ./...
```

## not yet done (sprint 3)

- running the binary against a live postgres and a live collector.
- reconciling desired vs effective config and pushing remote config over
  opamp.
- authenticating collector connections; today every connection is accepted.
