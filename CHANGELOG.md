## 2026-10-26
- Doc sweep: mint `.meta/pointers.yaml` with 10 seed file pointers (#14); add layout, architecture, schema, and threat-model docs under `docs/` with their summaries and `docs/layout/{pkg,frontend}.md`.
## 2026-10-14
- Custom-role authorization design doc: model v2 tuples, per-kind off/shadow/enforce rollout, audit and phased plan (docs/noizu/custom-role-authz.md).
- User delete via service account keys: drop users-table deleter lookup that 404'd the service account id before the delete ran (#12).
- Role CRUD in the openfga authz provider: create/update/delete persist through the role store, delete revokes assignee tuples and runs guard hooks, missing-row updates/deletes now 404 (#9).
- Delete contracts made observable and precise: deleted service accounts and users 404 on GET, planned-maintenance delete blocks with 409 only when rules reference it (#10).
- Pipelines batch save no longer regenerates ids: posted-with-id pipelines update in place (org-scoped, audit columns preserved) (#11).
## 2026-10-13
- Remove ee/ and cmd/enterprise (SigNoz Enterprise License) — MIT-only fork.
- Clean-room MIT ingestion-key gateway (sqlgateway): replace noopgateway with a sqlstore-backed implementation supporting create/list/search/get/update/delete for ingestion keys and their limits (#1).
- Community-correct integration suite: add_license is a no-op (licensing is enterprise-only), SSO/licence suites skipped via a shared marker, CI matrix is sqlite-only (#2).
- CI: noizu-image workflow builds the community image (Go binary + frontend) and publishes to ops.noizu.com/signoz, ghcr.io/noizu-forks/signoz and docker.io/noizu/signoz on develop/lacrosse.main pushes and noizu-v* tags; registries with missing creds are skipped, GHCR made public best-effort via API (#5, #6).
