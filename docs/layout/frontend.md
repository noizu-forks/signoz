# Frontend (`frontend/`)

React 18 + TypeScript SPA built with Vite, package-managed by pnpm
(`preinstall` enforces pnpm). Tests run on Jest, with Storybook for components.

```
frontend/
├── src/
│   ├── api/                # API clients generated from docs/api/openapi.yml by orval
│   ├── app/                # App shell and providers
│   ├── AppRoutes/          # Route table
│   ├── components/         # Shared UI components
│   ├── container/          # Page-level containers
│   ├── hooks/              # Shared React hooks
│   ├── lib/                # Utilities, authz helpers, fixtures
│   ├── modules/            # Feature modules (Servicemap, Usage)
│   ├── pages/              # Route pages
│   ├── parser/             # ANTLR-generated filter query parser
│   ├── periscope/          # Query builder UI components
│   ├── plugins/            # Web Component plugin host + builtins
│   ├── providers/          # Context providers
│   ├── schemas/generated/  # Generated schemas
│   ├── store/              # Redux store (legacy) + global time state
│   ├── styles/             # SCSS and CSS modules
│   ├── ReactI18/           # i18n bundles
│   └── utils/, types/, constants/
├── public/                 # Static assets, plugin bundles manifest
├── storybook/              # Storybook configuration
├── __mocks__/              # Jest mocks
├── orval.config.ts         # Generates src/api from ../docs/api/openapi.yml
├── vite.config.ts          # Build and dev-server configuration
├── jest.config.ts          # Test configuration
├── pnpm-workspace.yaml     # Workspace definition
├── example.env             # Environment template
└── CLAUDE.md / AGENTS.md   # UI agent conventions
```

## Build and test

| Command | Effect |
|---------|--------|
| `pnpm dev` | Vite dev server |
| `pnpm build` | Production bundle |
| `pnpm test` | Jest suite |
| `pnpm lint` / `pnpm lint:fix` | ESLint over the source tree |
| `pnpm storybook` | Component workshop on port 6006 |

Generated directories (`src/api`, `src/schemas/generated`, `src/parser` ANTLR
output) are produced by codegen, not edited by hand.
