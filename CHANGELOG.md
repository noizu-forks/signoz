## 2026-10-13
- Remove ee/ and cmd/enterprise (SigNoz Enterprise License) — MIT-only fork.
- Clean-room MIT ingestion-key gateway (sqlgateway): replace noopgateway with a sqlstore-backed implementation supporting create/list/search/get/update/delete for ingestion keys and their limits (#1).
- Community-correct integration suite: add_license is a no-op (licensing is enterprise-only), SSO/licence suites skipped via a shared marker, CI matrix is sqlite-only (#2).
- CI: noizu-image workflow builds the community image (Go binary + frontend) and publishes to ops.noizu.com/signoz, ghcr.io/noizu-forks/signoz and docker.io/noizu/signoz on develop/lacrosse.main pushes and noizu-v* tags; registries with missing creds are skipped, GHCR made public best-effort via API (#5, #6).
