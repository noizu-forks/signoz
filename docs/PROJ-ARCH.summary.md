# SigNoz — Architecture (summary)

- **Shape**: Go modular monolith (`cmd/community` server binary) + React/Vite SPA
  served by the same binary; MIT-only fork, no licensing, no enterprise tree.
- **Storage planes**: metadata in SQLite (default) or Postgres via bun;
  telemetry (traces/logs/metrics) in ClickHouse. Never mixed.
- **Request path**: apiserver :8080 → `identn` (tokenizer / API key /
  impersonation) → `authz` (OpenFGA tuples) → domain module → `sqlstore` or
  `querier`.
- **Telemetry ingest**: OTel collector writes to ClickHouse directly, not
  through the backend; queries are built by `querybuilder` +
  `statementbuilder`.
- **Alerting**: `ruler` evaluates rules against ClickHouse; `alertmanager`
  stores config/state in the metastore and fans out notifications.
- **Config**: one YAML (`--config`, repeatable, later wins) with a section per
  subsystem; `conf/example.yaml` is the reference.
- **Deploy**: single image built and published by `noizu-image` (GHCR + optional
  ops/Docker Hub); migrations run at boot via `sqlmigrator`.
- **Key decisions**: provider/factory pattern enables noop providers; generated
  `docs/api/openapi.yml` gates CI and generates the frontend client (orval);
  dashboard plugins are allowlisted by `pluginId`, never by script URL.
