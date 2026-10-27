# SigNoz — Project Layout (summary)

```
signoz/
├── cmd/                  # Cobra entrypoints; community/ = server binary
├── pkg/                  # Go backend → layout/pkg.md
│   ├── apiserver/        #   HTTP server + routes
│   ├── modules/          #   29 domain modules
│   ├── sqlstore/         #   metastore (SQLite / Postgres)
│   ├── telemetrystore/   #   ClickHouse
│   ├── sqlmigration/     #   135 migrations (000–135)
│   ├── authn/ authz/ identn/
│   └── …
├── frontend/             # React 18 + Vite → layout/frontend.md
├── docs/                 # api/openapi.yml, contributing/, plugins/, noizu/
├── conf/                 # example.yaml, plugins.example.yaml
├── deploy/               # install.sh, MIGRATION.md
├── plugins/              # static plugin catalog
├── grammar/              # ANTLR grammars
├── templates/            # alertmanager + email templates
├── scripts/              # generators (clickhouse, grammar, semconv)
├── tests/                # Python integration/e2e + fixtures + seeder
├── .github/workflows/    # CI, releases, image publishing
├── Makefile              # build/test/run/docker
├── go.mod                # go 1.25.7
└── README.md
```

Generated, do not hand-edit: `docs/api/openapi.yml`, `frontend/src/api`,
`frontend/src/schemas/generated`, `pkg/semconv`.

Setup files: `conf/example.yaml`, `conf/plugins.example.yaml`,
`frontend/example.env`, `.devenv/`.
