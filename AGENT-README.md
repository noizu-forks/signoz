# AGENT-README

> Orientation for AI coding agents working in this repo. Human docs: README.md.

## Type

go+node-web

## Purpose

SigNoz fork — open-source observability platform on OpenTelemetry: logs, metrics, traces, alerts, dashboards in one place (enterprise-grade DataDog/NewRelic alternative).

## Stack

Go backend (`cmd/`, `pkg/`, ClickHouse storage, module `github.com/SigNoz/signoz`) + TypeScript React frontend (`frontend/`: Vite, Ant Design, react-query, pnpm, Node >= 22). Makefile drives devenv and builds.

## Dev mode

```bash
make devenv-up                        # clickhouse + signoz-otel-collector via docker compose (.devenv/)
make go-test                          # go test -race ./...
cd frontend && pnpm install && pnpm dev   # vite dev server
```

## Ports / URLs

- http://localhost:8123 — devenv ClickHouse (Makefile `devenv-up` output).
- 6006 — Storybook dev server (`frontend/package.json`, `storybook dev -p 6006`).

## Tests

```bash
make go-test                  # Go: go test -race ./...
cd frontend && pnpm test      # jest
```

## Gotchas

- Frontend is pnpm-enforced (`preinstall` runs `only-allow pnpm`); engines: Node >= 22, pnpm 10.
- Frontend API hooks/types are generated with orval (`pnpm generate:api`); i18n hash is generated on postinstall — don't hand-edit generated dirs (`src/api/generated`, `src/schemas/generated`).
- Devenv services each live in their own compose file under `.devenv/docker/<service>/`; `devenv-signoz-otel-collector` requires ClickHouse to be up first.
