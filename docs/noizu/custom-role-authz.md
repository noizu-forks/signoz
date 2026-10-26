# Custom-Role Authorization — Design

Status: draft for review · Date: 2026-10-14 · Base: `develop` @ `e036ce387` (noizu-forks/signoz)

User direction (verbatim):

> "Impleent the check/control/acl etc mechanism but do not lock down all resources yet. do things incrementally to avoid breaks and easy stop/resume points. flesh out how they're assigned, checked, audited, granality etc and dril down in detail."

**Clean room.** This document was designed exclusively from the MIT tree of this fork. `ee/` and `cmd/enterprise/` are deleted from the fork (PR #1) and licensed under the SigNoz Enterprise License; they were never opened, diffed, grepped, or read at any commit, nor was any upstream history under those paths, nor any SigNoz web/documentation page describing enterprise RBAC internals. Every citation below is to files present in the MIT tree at `e036ce387`.

---

## 1. Current state (what exists today)

The fork inherits a complete *vocabulary* for fine-grained authorization and a *coarse* decision engine. Understanding the split is the whole game:

**What is fine-grained already (the vocabulary):**

- A permission unit called a **transaction**: `{relation (verb), object (resource ref + selector)}` (`pkg/types/authtypes/transaction.go:24-28`). A **transaction group** is `{relation, objectGroup{resource, selectors[]}}` (`pkg/types/authtypes/transaction.go:30-35`). A role's permission set is a `TransactionGroups` JSON column on the `role` table (`pkg/types/authtypes/role.go:58-68`).
- Eight verbs: `create/read/update/delete/list/assignee/attach/detach` (`pkg/types/coretypes/verb.go:20-36`).
- A resource taxonomy: types `role`, `user`, `serviceaccount`, `organization`, `metaresource` (kinds: dashboard, rule, pipeline, saved-view, trace-funnel, planned-maintenance, notification-channel, route-policy, ingestion-key, ingestion-limit, auth-domain, cloud-integration, integration, license, subscription, org-preference, public-dashboard, session, user-preference, apdex/quick-filter/ttl settings, deployment-host, logs-field, traces-field) and `telemetryresource` (kinds: logs, traces, metrics, audit-logs, meter-metrics) (`pkg/types/coretypes/registry_managed_role.go:19-378`).
- Selectors, including the wildcard `*` and concrete ids, with object ids that embed the org: `<type>:organization/<orgID>/<kind>/<selector>` (`pkg/types/coretypes/object.go:69-82`, example at `pkg/types/coretypes/resource_metaresource.go:27-35`).
- The **managed-role permission matrix**: `coretypes.ManagedRoleToTransactions` maps `signoz-admin`, `signoz-editor`, `signoz-viewer`, `signoz-anonymous` to explicit transaction lists (`pkg/types/coretypes/registry_managed_role.go:19-378`, names at `:12-17`). This is a fourth managed role people forget: `signoz-anonymous` holds `read:public-dashboard` for the public-dashboard flow (`:374-377`).
- Route declarations: every apiserver route wraps its handler in `authzMiddleware.CheckResources(handler, <managed role names>...)` (115 call sites, e.g. `pkg/apiserver/signozapiserver/dashboard.go:18-393`) and declares `handler.WithResourceDefs(handler.BasicResourceDef{Resource, Verb, Category, ID, Selector})` (e.g. `pkg/apiserver/signozapiserver/role.go:30-36`). The `Resource` middleware resolves those defs into the request context (`pkg/http/middleware/resource.go:24-48`), where both the authz gate and the audit middleware read them.
- Role CRUD API: `/api/v1/roles` + `/api/v1/roles/{id}`, admin-gated (`pkg/apiserver/signozapiserver/role.go:13-149`), backed by handlers that persist `TransactionGroups` verbatim (`pkg/authz/signozauthzapi/handler.go:25-156`).
- An explain endpoint that already exists: `POST /api/v1/authz/check` takes transactions and returns per-transaction `authorized` (`pkg/apiserver/signozapiserver/authz.go:11-30`, handler at `pkg/authz/signozauthzapi/handler.go:158-186`).
- The frontend already consumes it: `useAuthZ` batches `authzCheck` calls into a permission map keyed `verb:kind`, with devtools overrides (`frontend/src/lib/authz/hooks/useAuthZ/useAuthZ.tsx:1-70`), and a derived permission catalogue (`frontend/src/storybook/access/access.ts:19-40`).
- An audit middleware in the apiserver chain that emits one audit event per resolved resource per request (`pkg/http/middleware/audit.go:98-188`), chain order Recovery → Otel → IdentN → Timeout → Resource → Audit → Comment (`pkg/apiserver/signozapiserver/provider.go:314-324`), shipped by the buffered `auditor` exporter (`pkg/auditor/auditorserver/`, metrics at `telemetry.go:9-27`).
- Assignment storage: `user_role` table (`pkg/types/authtypes/user_role.go:20-31`, store iface `:112-127`), service accounts carry a single `role_id` (`pkg/types/serviceaccounttypes/service_acccount.go:57-59`). Grants are written as OpenFGA tuples keyed by **role name**: `(subject, assignee, role:organization/<org>/role/<name>)` (`pkg/authz/openfgaauthz/provider.go:113-127`), driven from user invite/update (`pkg/modules/user/impluser/setter.go:63` et al.) and SA create/update (`pkg/modules/serviceaccount/implserviceaccount/module.go:390`).

**What is coarse (the gaps this design closes):**

1. **The OpenFGA model is minimal.** `pkg/authz/openfgaschema/base.fga` defines only `role.assignee: [user, serviceaccount]` and `organization` with four CRUD relations over `[role#assignee]`. There are no per-resource types or relations, so no tuple can express "editor may update dashboards", let alone "this user may update dashboard X".
2. **The main check discards the resource entirely.** `CheckWithTupleCreation` accepts `relation, resource, selectors` and ignores all three (parameters are `_` at `pkg/authz/openfgaserver/server.go:145`); it only checks whether the subject is an assignee of one of the *route-declared role names*. Authorization today == "does the subject hold one of the managed roles this route lists".
3. **`CheckTransactions` expands only managed roles.** It looks up `registry.ManagedRolesByTransaction()[verb:type:kind]` and checks assignee tuples for those role names (`pkg/authz/openfgaauthz/provider.go:235-255`, expansion at `pkg/types/authtypes/tuple.go:175-217`). A transaction no managed role holds is denied without any check (`preResolved=false`, `pkg/types/authtypes/tuple.go:200-204`). **Custom roles are persisted and assignable but invisible to every decision.**
4. **Role Create/Update never writes permission tuples.** `provider.Create/Update` only write the `role` table (`pkg/authz/openfgaauthz/provider.go:182-188`). The helper that would write them exists and is unused at runtime: `NewTuplesFromTransactionGroups` produces `(role:<org>/<name>#assignee, verb, resource-object)` tuples (`pkg/types/authtypes/tuple.go:51-70`); `NewTuplesFromTransactionsWithCorrelations` produces per-transaction object checks with specific+wildcard selectors (`pkg/types/authtypes/tuple.go:135-173`); `DiffTuples` computes set-difference rewrites (`pkg/types/authtypes/tuple.go:72-102`). (Migration 099 read legacy tuples back into `transaction_groups` — `pkg/sqlmigration/099_add_role_transaction_groups.go:95-130` — so the column is authoritative by construction.)
5. **No decision audit, no shadow mode, no enforcement switch.** Denials log a `::AUTHZ-DENIED::` warning (`pkg/http/middleware/authz.go:19-21,64`) but nothing records what was decided, why, or whether a future evaluation would disagree.

Also relevant: a legacy surface of ~211 `ViewAccess/EditAccess/AdminAccess` wrappings (role-name gates, `pkg/http/middleware/authz.go:39-151`), concentrated in the old query-service handler (`pkg/query-service/app/http_handler.go`), which do not go through `CheckResources` at all.

---

## 2. Goals and non-goals

**Goals**

1. Custom roles affect decisions: a role's `transactionGroups` are the permissions its assignees get, at org-wide granularity per resource kind, with optional per-object selectors later.
2. One evaluation path for managed and custom roles (same tuples, same check code), with managed-role behavior provably unchanged until an explicit per-resource switch is flipped.
3. Incremental rollout: per-resource-kind mode `off → shadow → enforce`, default `off`, every phase a mergeable PR set that leaves the system working, with explicit stop/resume points.
4. Full auditability of decisions and admin changes.
5. API + provider surface for permission sets, effective permissions, mode toggles, and dry-run checks.

**Non-goals (this design)**

- No deny rules, no role hierarchy/inheritance (justified in §4.4).
- No changes to authentication, session, or org membership.
- No enforcement on the legacy query-service gates beyond what falls out naturally (§6.1).
- No new resources in the permission vocabulary beyond what `coretypes` already defines.

---

## 3. Model

### 3.1 Permission naming

A permission is a **transaction**: `verb:type:kind[:selector]`, rendered `verb:kind` when the selector is the wildcard (matching `Scope()` at `pkg/types/coretypes/resource_metaresource.go:35-37` and the frontend's `relation:kind` catalogue at `frontend/src/storybook/access/access.ts:19-40`). Examples: `read:dashboard`, `update:rule`, `delete:pipeline`, `attach:ingestion-key`, `read:logs` (telemetry kinds are selector-capable: `pkg/types/telemetrytypes` grant selectors validated in `pkg/types/authtypes/transaction.go:191-196`).

Granularity levels:

| Level | Shape | Status |
|---|---|---|
| Org-wide per kind | `{verb, metaresource:dashboard, *}` | P1-P5 (core deliverable) |
| Per-object | `{verb, metaresource:dashboard, <uuid>}` | P6 (types already accept it: `pkg/types/coretypes/selector.go:29-41`) |
| Telemetry-scoped | `{read, telemetryresource:logs, <grant-selector>}` | vocabulary exists; enforcement deferred (open question Q6) |

### 3.2 OpenFGA model v2

Extend `base.fga` (append types; the existing four stay byte-identical):

```fga
module base

type user

type serviceaccount

type role
  relations
    define assignee: [user, serviceaccount]

type organization
  relations
    define create: [role#assignee]
    define read: [role#assignee]
    define update: [role#assignee]
    define delete: [role#assignee]

type metaresource
  relations
    define create: [role#assignee, user, serviceaccount]
    define read:   [role#assignee, user, serviceaccount]
    define update: [role#assignee, user, serviceaccount]
    define delete: [role#assignee, user, serviceaccount]
    define list:   [role#assignee, user, serviceaccount]
    define attach: [role#assignee, user, serviceaccount]
    define detach: [role#assignee, user, serviceaccount]

type telemetryresource
  relations
    define read: [role#assignee, user, serviceaccount]
    define update: [role#assignee, user, serviceaccount]

type user
  relations
    define create: [role#assignee]
    define read: [role#assignee]
    define update: [role#assignee]
    define delete: [role#assignee]
    define list: [role#assignee]
    define attach: [role#assignee]
    define detach: [role#assignee]

type serviceaccount
  relations
    define create: [role#assignee]
    define read: [role#assignee]
    define update: [role#assignee]
    define delete: [role#assignee]
    define list: [role#assignee]
    define attach: [role#assignee]
    define detach: [role#assignee]
```

Notes:

- Object ids reuse the existing encoding verbatim (`<type>:organization/<orgID>/<kind>/<selector>`), so org isolation is structural: an object id from another org simply never matches (`pkg/types/coretypes/object.go:69-82`).
- Relations per type are exactly the verbs `AllowedVerbs()` already advertises for that type (e.g. `pkg/types/coretypes/resource_metaresource.go:39-41`); the model is generated from `coretypes` in a unit test that fails when the two drift (see §7.2 P1).
- `[role#assignee, user, serviceaccount]` on `metaresource`/`telemetryresource` leaves direct per-object user grants representable from day one, even though P6 only ships roles-with-selectors.
- `role` and `organization` keep their current relations: `role.assignee` drives all role-management checks (`ViewAccess`/role routes), and `organization` remains for the anonymous/public flows (`CheckWithTupleCreationWithoutClaims`, `pkg/authz/openfgaserver/server.go:186-212`).

### 3.3 Permission tuples

For every role (managed **and** custom) write one tuple per grant:

```
user     = role:organization/<orgID>/role/<roleName>#assignee
relation = <verb>
object   = <type>:organization/<orgID>/<kind>/<selector>
```

This is exactly what `NewTuplesFromTransactionGroups` produces (`pkg/types/authtypes/tuple.go:51-70`). Wildcard grants are concrete `.../<kind>/*` tuples — no wildcard logic inside FGA, evaluation stays O(1) per tuple.

### 3.4 Managed roles in the same scheme

Managed roles are not special-cased in evaluation. Their `transactionGroups` are generated from the same static registry (`NewManagedRoles` → `coretypes.ManagedRoleToTransactions`, `pkg/types/authtypes/role.go:123-130`), so their permission tuples are literally derived from the matrix that drives today's decisions. One evaluation path: `subject → assignee tuples → roles → permission tuples → allow`. The registry remains the single source of truth for managed roles (code-defined, seeded into the `role` table at org bootstrap, backfilled by `pkg/sqlmigration/059_add_managed_roles.go` / `099`), and the tuple writer treats them read-only — `ErrIfManaged` already blocks edits (`pkg/types/authtypes/role.go:158-164`) and the `signoz-` name prefix is reserved (`:29-32,185-187`).

---

## 4. Assignment

### 4.1 Subjects and roles

- Users: many-to-many via `user_role` (`pkg/types/authtypes/user_role.go:20-31`). Unchanged.
- Service accounts: single `role_id` (`pkg/types/serviceaccounttypes/service_acccount.go:57-59`). Unchanged. (Multi-role SAs would be an additive schema change later; not needed now.)
- The SQL tables stay the **system of record**; OpenFGA tuples (assignee + permissions) are a derived index. This is the fork's existing shape for assignee tuples (grants flow through `Grant/Revoke`, `pkg/authz/openfgaauthz/provider.go:113-157`) extended to permission tuples.

### 4.2 Tuple lifecycle

- **Write path:** role `Create`/`Update`/`Delete` compute the tuple diff (`DiffTuples`, `pkg/types/authtypes/tuple.go:72-102`) between the role table's old and new `transactionGroups` and write it in the same logical operation (`provider.Create/Update/Delete`, extending `pkg/authz/openfgaauthz/provider.go:182-211`). Delete additionally revokes assignee tuples — already implemented via `revokeRoleTuples` + `onBeforeRoleDelete` guard hooks (`:190-233`; hooks registered at `pkg/signoz/signoz.go:441-444`, blocking deletion while users/SAs/auth-domain mappings still reference the role).
- **Reconciler:** on startup, and opportunistically on role reads, compare desired tuples (from the `role` table) against actual (via `ReadTuples`, `pkg/authz/openfgaserver/server.go:287-313`) and heal drift. Heals lost writes, migration gaps, and manual DB surgery.
- **Bootstrap:** managed-role permission tuples are written by the reconciler from the registry, so a fresh org or a tuple wipe self-heals without a migration.

### 4.3 Per-object grants

A grant on one dashboard is a transaction group with a concrete selector: `{"relation":"update","objectGroup":{"resource":{"type":"metaresource","kind":"dashboard"},"selectors":["<uuid>"]}}`. Nothing new is stored; `transactionGroups` already accepts and validates concrete selectors (`newTransactionGroup`, `pkg/types/authtypes/transaction.go:189-203`). P6 adds the UX/API sugar and list filtering (§7.2).

### 4.4 Deny semantics and inheritance: allow-only, flat

- **Allow-only.** OpenFGA has no first-class deny; adding one means exceptions-model complexity, non-monotone decisions, harder audit reasoning ("allowed by X, denied by Y — which wins?"), and cache invalidation headaches. Restriction needs (e.g. "viewer but not on billing dashboards") are expressed in P6 by simply not granting, at object granularity. Revisit only with a concrete blocker use case.
- **No role-to-role inheritance.** Flat roles keep `ManagedRolesByTransaction`-style reasoning trivial and the reconciler linear. Composition is achieved by assigning multiple roles (users already support that).
- **Managed-role floor.** `signoz-admin` holds every transaction in the registry (`registry_managed_role.go:20-217`) and its tuples are generated from it, so an admin can always administer roles, users, and modes — the structural anti-lockout guarantee (see §8.1).

### 4.5 Org isolation

Every object id embeds the org (`object.go:69-82`); every grant/permission tuple is written with the acting org id; checks construct object ids from the *claims* org (`CheckWithTupleCreation`, `server.go:145-176`). Roles are org-scoped rows (unique name per org, enforced by the store: `pkg/authz/authzstore/sqlauthzstore/store.go:21-33`). A subject from org A evaluating an object in org B is not expressible in the tuple space. Anonymous subjects are pinned to `anonymous:organization/<org>/anonymous` + the `signoz-anonymous` role (`server.go:186-192`, `pkg/sqlmigration/061_migrate_rbac_to_authz.go:113-118`).

---

## 5. Checking

### 5.1 Enforcement points

| Surface | Mechanism | Change |
|---|---|---|
| apiserver routes (115 sites) | `CheckResources(handler, roles...)` wraps the handler; `Resource` middleware pre-resolves defs | Mode-aware: in `off`/`shadow` keep the role-list check verbatim; in `enforce` evaluate the resolved transactions (§5.2) |
| Legacy query-service (~211 sites) | `View/Edit/AdminAccess` role-name gates | Unchanged (managed-role UI gates). Migrate opportunistically when routes move to apiserver |
| Frontend | `useAuthZ` → `/api/v1/authz/check` | No change; lights up automatically once `CheckTransactions` consults custom groups |
| Background/internal calls | none today | Out of scope; nothing to enforce |

The mode check lives in one place — the `AuthZ` middleware's `checkResource` (`pkg/http/middleware/authz.go:207-267`) and the `CheckTransactions` entry (`pkg/authz/openfgaauthz/provider.go:235`) — so call sites do not multiply.

### 5.2 Decision algorithm

For a request with claims `(subject, orgID)` and resolved resources `[(verb, resource, ids)]`:

1. Look up the mode for `(resource type, kind)`. `off` → run today's role-list check, return. (`off` is byte-for-byte today: role names → assignee tuple check, `server.go:164-183`.)
2. `shadow` → run today's check **and** the new evaluation; record both outcomes + divergence flag (§6). Never block on the new one.
3. `enforce` → new evaluation decides:
   a. For each resolved id, build the specific-selector and wildcard tuples (`IDSelector` already returns `[id, *]`, `pkg/types/coretypes/selector.go:29-41`; the expansion helper exists: `newCheckSelectors`, `pkg/types/authtypes/tuple.go:263-279`).
   b. `BatchCheck` all tuples in one call (`pkg/authz/openfgaserver/server.go:103-143`).
   c. Allow iff every resolved resource has at least one allowing tuple (OR across selectors/roles, AND across resources — same shape as today's loop at `pkg/http/middleware/authz.go:221-264`).

Role-management routes keep their direct `role:assignee` checks (the `TypeRole`+`VerbAssignee` special case in `CheckTransactions`, `pkg/types/authtypes/tuple.go:188-198`) — that is what stops a custom "role-admin" from editing roles unless explicitly granted, and it is also how "who holds role X" is evaluated today.

### 5.3 Caching and invalidation

- The OpenFGA server is embedded in-process (`pkg/authz/openfgaserver/server.go:45-69`) with a SQL datastore, so a check is one indexed SQL query; `BatchCheck` amortizes per-route tuples into one round trip.
- Add a small in-process decision cache keyed `(epoch, subject, verb, type, kind, selector)`; `epoch` is an atomic counter bumped on every authz `Write` (roles, grants, modes) via the provider, which invalidates the whole cache in O(1) without TTL guessing. Bound entries with a 60s TTL as a belt-and-braces.
- Budget: p99 added latency ≤ 2ms per request warm, ≤ 15ms cold; zero added DB round trips for `off` mode; cache hit ratio ≥ 99% steady-state. Track with metrics on the existing meter (`serverMetrics` pattern, `pkg/auditor/auditorserver/telemetry.go:9-27` is the model to copy).

### 5.4 Failure modes

- **Provider error in `off`/`shadow`:** identical to today (`authz_unavailable` 500s, `pkg/types/authtypes/tuple.go:12`); shadow evaluation errors are logged, never block.
- **Provider error in `enforce`:** writes **fail closed** (403 with `authz_forbidden`-style error plus `degraded` marker in the decision audit; 5xx would invite retry storms and blur the signal). Reads fail **open** with an audit record and a fired metric — rationale: SigNoz is the observability plane; blinding on-call users during an authz hiccup is the worse outage. Both behaviors config-gated (`signoz.authz.enforcement.reads_fail_open`, default true). This split is deliberate and is open question Q1 for confirmation.
- **Tuple drift:** reconciler heals (§4.2); enforcement reads tuples, so healing restores correctness without code action.

---

## 6. Rollout without breaking (the core requirement)

### 6.1 Per-resource enforcement mode

- Modes: `off` (default) → `shadow` → `enforce`, per `(type, kind)` pair (e.g. `metaresource/dashboard`), i.e. per row of the §1 taxonomy.
- Storage: new table `authz_enforcement_mode(type, kind, mode, updated_by, created_at, updated_at)` — next migration number is 136 (`pkg/sqlmigration/135_add_ingestion_keys.go` is the current last). Config file provides the bootstrap default (`signoz.authz.enforcement.default: off`) and a hard override.
- **Kill switch:** env `SIGNOZ_AUTHZ_ENFORCEMENT_DEFAULT=off` forces every kind to `off` at boot regardless of the table — the break-glass lever that needs no DB access.
- Mode changes are admin-only API calls (admin floor §4.4) and audit-logged (§6.3).

### 6.2 Managed roles keep today's behavior byte-for-byte — proof

Two mechanisms, both required before any kind leaves `off`:

1. **Structural parity (CI, gate for P2):** the managed-role permission tuples are *generated from* `ManagedRoleToTransactions`, and the legacy path *reads the same map* (`tuple.go:200-212`). A table-driven test asserts, for every managed role × every transaction key in the registry (the full `verb:type:kind` cross-product, ~8 verbs × ~30 kinds), that `legacy CheckTransactions(subject with role R)` == `new evaluation(subject with role R)` against an embedded openfga server seeded only by the P1 writer. Legacy semantics are reproduced exactly, including the "no managed role holds it ⇒ deny" case (`tuple.go:200-204`) — in the new model that transaction simply has no allow tuple.
2. **Runtime shadow (gate for P4+):** in `shadow` mode every decision computes both outcomes and logs divergences with full context. A kind may move to `enforce` only after a burn-in window (default 7 days) with **zero** shadow divergences for that kind; the divergence counter is exported as a metric and queried in the rollout checklist.

Selector nuance covered by the same test: the legacy `TransactionKey` ignores selectors (`pkg/types/authtypes/transaction.go:164-166`), so legacy semantics are org-wide-only; the new path with all-wildcard tuples is provably identical, and concrete selectors only ever *add* allows in `enforce` mode (P6).

### 6.3 Audit

**Decision audit (new):** one record per shadow/enforce evaluation that is *interesting*: every `enforce` deny, every shadow divergence, every degraded/fail-open read, plus a 1-in-N sample of allows (N configurable, default 100) for coverage evidence. Schema (fields on the existing audit event shape, `pkg/types/audittypes/event.go`):

```
who:      principal type + id, org id
what:     verb, type, kind, selector(s), route template
decision: allow | deny | shadow-allow | shadow-deny | diverged | degraded-open
reason:   matched role name(s) / "no matching grant" / "provider unavailable"
context:  request id, trace id, enforcement mode, latency bucket
```

Storage: the existing auditor pipeline (buffered exporter → its sink; the admin-only `read:audit-logs` telemetry kind already exists in the registry at `registry_managed_role.go:204-208` as the consumption path). Volume is bounded by the "interesting only" filter — a full per-request decision table would be the largest table in the system. Retention follows the audit sink's existing retention; redaction: subject ids only (no emails), resource ids only (no titles/bodies), no permission-set contents.

**Admin-change audit (mostly exists):** role CRUD already emits audit events via `WithResourceDefs` + `ActionCategoryAccessControl` (`role.go:30-36`, `pkg/http/middleware/audit.go:120-187`). Extend the same pattern to: role permission-set changes (before/after group diff in the event), grant/revoke (already resource-def'd on user/SA routes), and mode flips (new routes with resource defs). `GET /api/v1/authz/audit` (admin) queries decision records with subject/resource/decision filters.

---

## 7. API surface, provider, and phased plan

### 7.1 API surface

| Endpoint | Method | Purpose | Phase |
|---|---|---|---|
| `/api/v1/authz/check` | POST | exists (`signozapiserver/authz.go:12`); extend response with `reason` (matched roles) when evaluating new-path | P2 |
| `/api/v1/authz/permissions` | GET | catalogue: types, kinds, allowed verbs, selector support — single source for UI + TF validation (frontend has only a static mock today, `frontend/src/lib/authz/hooks/useAuthZ/permissions.type.ts`) | P1 |
| `/api/v1/authz/modes` | GET | current mode per kind + shadow burn-in/divergence stats | P3 |
| `/api/v1/authz/modes/{type}/{kind}` | PUT | flip mode (`off→shadow→enforce`, one step at a time; backwards always allowed) | P3 |
| `/api/v1/authz/effective` | GET | "what can X do?" — subject's roles flattened to transaction groups (from the `role` table, joined via `user_role` / SA `role_id`) | P3 |
| `/api/v1/roles...` | CRUD | exists; no shape change — permission sets *are* `transactionGroups` | — |
| `/api/v1/authz/audit` | GET | decision-audit query API (admin) | P2 |

All admin-gated (`CheckResources(..., SigNozAdminRoleName)`), all OpenAPI-registered per repo convention (separate commit for generated specs, per `.claude/rules/go-contrib.md`).

**Terraform provider (later, noizu-forks/terraform-provider-signoz):** `signoz_role` gains a `permissions` block validated against the catalogue data source (`signoz_permissions`); `signoz_user_role` / `signoz_service_account_role` already exist and need no change; new data source `signoz_effective_permissions` for drift checks. No provider resource for modes — enforcement posture is operator state, not infrastructure.

### 7.2 Phased plan (each phase = one squash-merged PR set, system working at every merge)

| Phase | Content | Size | Acceptance tests | Rollback |
|---|---|---|---|---|
| **P1** Permission tuples + model v2 | Append model v2 types; write/diff permission tuples in role Create/Update/Delete; startup reconciler incl. managed roles; `/api/v1/authz/permissions`; model-vs-coretypes drift test | M (~600-900 LoC) | Structural parity matrix green; role CRUD round-trips tuples (the itest harness from `/tmp/trl-infra-signoz-itest-results.md` re-run: 9/9); no decision changes (checks still legacy) | Revert: schema file reverts, `getOrCreateModel` re-matches the old model (`server.go:351-383`), permission tuples sit inert (no relations reference them) |
| **P2** Shadow evaluation + decision audit | Mode table (all `off`); dual evaluation in `CheckTransactions` + `checkResource` when mode ≥ shadow; divergence logging/metrics; `/api/v1/authz/check` reason trace; `/api/v1/authz/audit` | M | Flip a kind to shadow in itest: decisions unchanged, audit rows appear, synthetic custom-role cases diverge as expected | Mode table to `off`; dual-eval code inert |
| **P3** Assignment API + effective permissions | `/authz/modes` GET/PUT with step-gating; `/authz/effective`; grant/revoke audit enrichment | S/M (~300-500 LoC) | Mode API admin-gated + audited; effective-permissions matches hand-computed unions | Modes stay `off`; API is additive |
| **P4** Enforce one low-risk kind | `saved-view`: flip default to `shadow` at deploy, then `enforce` after burn-in; custom role with only `saved-view` grants tested end-to-end through the frontend (`useAuthZ` shows/hides) | S | Viewer-minus-saved-view custom role: read denied in enforce, allowed in off; admin unaffected; zero divergence during burn-in | Mode → `off` (one API call or kill switch) |
| **P5** Resource-by-resource enforcement | Dashboards, rules, channels, pipelines, planned-maintenance, route-policies, ingestion keys/limits, settings kinds — each its own shadow→enforce cycle; milestone: custom roles carry real production permissions | M spread over weeks | Per-kind burn-in with zero divergences; lockout drill (kill switch) rehearsed once | Per-kind mode flip back |
| **P6** Per-object grants | Concrete-selector groups first-class in role UI/API; list-endpoint filtering via `ListObjects` (`server.go:315-329`) or post-filter; per-object audit attributes | M/L (~800-1200 LoC) | Single-dashboard grant: allows that dashboard only, denies others, list shows subset | Drop selector groups from roles (reconciler cleans tuples) |
| **P7** Provider support | `signoz_role.permissions` + catalogue validation, `signoz_effective_permissions` data source | S/M | TF plan/apply round-trip against catalogue; drift detection demo | Provider-only |

**STOP/RESUME points.** Every phase boundary is a stop point: merged state is always shippable, enforcement is always `off` unless an operator has explicitly flipped a kind, and the kill switch returns to today's behavior without a deploy. Resume = start the next phase's PR; no phase depends on unmerged work. The natural long stops are after P3 (everything shipped, zero enforcement) and after P5 (enforcement complete, per-object deferred).

---

## 8. Risks

1. **Lockout.** An admin could flip a kind to `enforce` and lose the UI path to flip it back. Mitigations: admin floor is structural (`signoz-admin` tuples are registry-generated and cover role/mode administration, §4.4); mode API is admin-gated but the kill switch (`SIGNOZ_AUTHZ_ENFORCEMENT_DEFAULT=off`) is env-only; direct SQL (`UPDATE authz_enforcement_mode SET mode='off'`) is documented as the last resort. Recovery drill is part of P4 acceptance.
2. **OpenFGA model migration.** Models are append-only; `getOrCreateModel` pins the id at startup (`server.go:351-383`). New types make existing tuples newly meaningful — that is the mechanism, and it is also the risk (a bad tuple wave). Mitigations: P1 writes tuples only through the validated writer; reconciler diffs before writing; rollback leaves tuples inert. The `MaxTuplesPerWrite` batching already exists (`server.go:214-236`, config at `pkg/authz/config.go:15-18`).
3. **Performance.** Worst case is a dashboard list with N resolved ids × selectors; `BatchCheck` batches, the epoch cache absorbs repeats. The budget in §5.3 is enforced with metrics and a load test in P2 acceptance.
4. **Frontend impact.** `useAuthZ` makes custom roles effective UI-wide the moment P2 lands the new-path `CheckTransactions` (in shadow the endpoint still answers legacy — flip is per-mode). Residual hardcodes: `user.role === USER_ROLES.ADMIN` checks (e.g. `frontend/src/hooks/metricsExplorer/useVolumeControlFeatureGate.ts:25`) and `useComponentPermission` role-name maps remain managed-role-only until migrated; inventory them in P5.
5. **Two route surfaces.** The legacy query-service gates (211 sites) never see custom roles. Accepted: those routes are admin/editor/viewer by design; they migrate as routes move to apiserver. Tracked, not blocking.
6. **Tuple drift / DB surgery.** Reconciler heals on boot; a `signoz.authz.reconcile` admin endpoint can force one.
7. **Shared-cluster migration safety.** New tables are additive; no destructive migration in any phase (repo policy: destructive needs explicit user approval). Liquibase is not involved — this repo's schema path is `pkg/sqlmigration`.

---

## 9. Open questions for the user

1. **Reads on provider failure in enforce mode:** fail-open proposed (writes fail-closed). Confirm, or prefer strict fail-closed everywhere?
2. **Legacy query-service gates:** leave as managed-role-only (proposed), or budget a sweep migrating them to `CheckResources` so custom roles apply there too?
3. **Per-object grants v1 shape:** roles-with-selectors only (proposed — one mechanism, one audit trail), or also direct user/SA object grants (model already permits)?
4. **Shadow burn-in:** 7 days zero-divergence before enforce (proposed) — right window for this fleet, or shorter/longer? And who flips modes in production — you only, or agents too?
5. **Decision audit retention + sampling:** interesting-only + 1% allows proposed; what retention does the audit sink run today, and is a per-request decision table ever wanted (cost warning)?
6. **Telemetry gating:** `read:logs/traces/metrics` and selector-scoped telemetry grants exist in the vocabulary — keep out of enforcement scope for now (proposed) or include in P5?

---

## Appendix A — Tuple shapes after P1

```
# assignment (exists today)
(user:organization/<org>/user/<uuid>, assignee, role:organization/<org>/role/<name>)
(serviceaccount:organization/<org>/serviceaccount/<uuid>, assignee, role:organization/<org>/role/<name>)
(anonymous:organization/<org>/anonymous, assignee, role:organization/<org>/role/signoz-anonymous)

# permissions (new, written for every role from transactionGroups)
(role:organization/<org>/role/<name>#assignee, update, metaresource:organization/<org>/dashboard/*)
(role:organization/<org>/role/<name>#assignee, read,   telemetryresource:organization/<org>/logs/*)
(role:organization/<org>/role/<name>#assignee, update, metaresource:organization/<org>/dashboard/<uuid>)   # P6
```

## Appendix B — Key file index (MIT tree @ e036ce387)

- Interface: `pkg/authz/authz.go:14-94`
- OpenFGA model: `pkg/authz/openfgaschema/base.fga`; server: `pkg/authz/openfgaserver/server.go`
- Provider (decisions, tuples, role CRUD): `pkg/authz/openfgaauthz/provider.go`
- Expansion helpers: `pkg/types/authtypes/tuple.go`
- Transactions/groups: `pkg/types/authtypes/transaction.go`
- Registry/matrix: `pkg/types/authtypes/registry.go`, `pkg/types/coretypes/registry_managed_role.go`
- Role/user/SA types: `pkg/types/authtypes/role.go`, `user_role.go`, `pkg/types/serviceaccounttypes/service_acccount.go`
- Gates: `pkg/http/middleware/authz.go`; resolution: `pkg/http/middleware/resource.go`; audit: `pkg/http/middleware/audit.go`
- Routes: `pkg/apiserver/signozapiserver/role.go`, `authz.go`, `dashboard.go` (representative), chain at `provider.go:314-324`
- API handlers: `pkg/authz/signozauthzapi/handler.go`
- Migrations (pattern + history): `pkg/sqlmigration/059,061,071-072,075,078,083,096,099,105,115,132-135_*.go`
- Frontend: `frontend/src/lib/authz/hooks/useAuthZ/`, `frontend/src/api/generated/services/{authz,role}/`
