# SigNoz — Threat Model (summary)

- **Crown jewels**: telemetry data, identity/authorization tuples, and ingestion
  keys (bearer credentials for telemetry writes).
- **Trust boundaries**: internet ↔ UI/API (:8080); instrumented apps ↔ OTel
  collector; backend ↔ metastore and ClickHouse; plugin catalog ↔ browser;
  CI ↔ container registries. No licensing surface exists in this fork.
- **Register**: 2 mitigated (T-001 panel script injection, T-004 impersonation),
  1 open (T-002 ingestion keys stored plaintext and returned by the API),
  6 partial (manifest `bundlePath`, header-derived identity, public images,
  timeout-excluded long-lived routes, SQL filter construction, audit coverage).
- **Controls**: tuple-based authz (OpenFGA) on every route, cross-org identity
  rejection, plugin id/version regex + directory containment for bundles,
  timeout and recovery middleware.
- **Residual risk**: long-lived ingestion keys (no rotation path but
  delete-and-recreate; `expires_at` only), no field-level encryption of
  telemetry, frontend-primary enforcement of the plugin allowlist, and
  header-derived identity that is only safe behind a trusted proxy.
