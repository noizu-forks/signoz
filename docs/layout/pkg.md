# Backend packages (`pkg/`)

One line per package. Everything is wired through the provider/factory pattern
(`pkg/factory`): a component declares a `Config` + `ConfigFactory` and one or more
named providers, and `cmd/community/server.go` chooses the provider per config.

## Entry and wiring

| Package | Purpose |
|---------|---------|
| `signoz` | Top-level `Config` and `SigNoz` struct holding every constructed provider |
| `factory` | `ConfigFactory`/`ProviderFactory` registries and lifecycle (`Service`, `Registry`) |
| `config` | Runtime config loading and precedence for YAML files + env |
| `global` | Global settings (`external_url`, `ingestion_url`, allowed origins) |
| `version` | Build version/commit metadata |
| `instrumentation` | Logger, tracer, and metrics bootstrap |

## HTTP and API

| Package | Purpose |
|---------|---------|
| `apiserver` | HTTP server config plus `signozapiserver/` route registration per domain |
| `http` | `server/`, `middleware/` (authz, identn, audit, timeout, cors, recovery), `render/`, `binding/` |
| `web` | Serves the built frontend (`routerweb`) or a noop |
| `pprof` | Profiling endpoints (`httppprof`) or a noop |

## Identity and permissions

| Package | Purpose |
|---------|---------|
| `authn` | Authentication: password (`emailpasswordauthn`), callback/OIDC (`googlecallbackauthn`), store |
| `authz` | Authorization: OpenFGA provider, schema, store, role/relation definitions |
| `identn` | Request identity resolvers: API key, tokenizer, impersonation |
| `modules/{session,user,serviceaccount}` | Domain modules for sessions, users, and service accounts |
| `tokenizer` | JWT (`jwttokenizer`) or opaque token issuance |
| `sharder` | Single-node (`singlesharder`) or noop sharding of org data |

## Storage

| Package | Purpose |
|---------|---------|
| `sqlstore` | Metastore: `sqlite` (default) or `postgres` providers via bun |
| `sqlschema` | Dialect-aware DDL builder (`sqlitesqlschema/`) used by migrations |
| `sqlmigrator` | Fumpt-migrator wrapper that runs `sqlmigration` sets |
| `sqlmigration` | 135 numbered migrations (000–135), each a registered provider |
| `telemetrystore` | ClickHouse connection + query execution |
| `clickhousesql` | ClickHouse-flavoured statement builder helpers |
| `telemetryschema` | Per-signal ClickHouse schemas: traces, logs, metrics, meters, audit, AI telemetry |
| `cache` | `memorycache` and `rediscache` providers |
| `valuer` | Strongly typed value wrappers (UUIDs, names) used in models |

## Query path

| Package | Purpose |
|---------|---------|
| `querier` | v5 query entry points (`QueryRange`, `QueryRangePreview`) |
| `querybuilder` | Builds ClickHouse queries from v5 query envelopes |
| `queryparser` | Parses filter/having expressions |
| `parser/filterquery`, `parser/havingexpression` | ANTLR-generated filter grammars |
| `statementbuilder` | ClickHouse SQL statement construction |
| `units`, `semconv` | Unit conversion and generated semantic conventions |
| `telemetrymetadata` | Field/attribute metadata used to validate query keys |
| `variables` | Dashboard/alert variable substitution |
| `contextlinks` | Alert/dashboard contextual link generation |

## Alerting

| Package | Purpose |
|---------|---------|
| `alertmanager` | Alertmanager config/state, notify, template, batching, HTTP handler |
| `ruler` | Rule evaluation (`signozruler`), rule store, HTTP handler |
| `emailing`, `smtp` | Outbound mail |
| `templating` | Notification templating helpers |

## Platform services

| Package | Purpose |
|---------|---------|
| `gateway` | Ingestion-key gateway (`sqlgateway`, clean-room MIT implementation) + noop |
| `plugins` | Web Component plugin catalog allowlist and `/api/v1/plugins` handler |
| `licensing` | `nooplicensing` (fork runs without a license) |
| `zeus` | Upstream/hosted-service integration; `noopzeus` in the community build |
| `subscription` | Subscription handler; `noopsubscription` in the community build |
| `analytics`, `statsreporter`, `meterreporter`, `metercollector` | Usage telemetry and metering |
| `auditor` | Audit event emission |
| `flagger` | Feature flags (`configflagger`) |
| `transition` | Legacy v3/v4 → v5 migrations for dashboards, alerts, saved views |
| `errors`, `jsonschema` | Structured error codes and reflection helpers |
