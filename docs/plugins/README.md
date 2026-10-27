# SigNoz Dashboard Plugins (Web Components)

<!-- 〚📁:26642385-25c2-5ec4-9d2a-68b04d0a1b97〛 Plugin Catalog Docs :: Web Component plugin allowlist and security model. -->

Noizu / Lacrosse fork extension. Dashboards can host **Web Component** panels that load only from an **allowlisted plugin catalog**. Free-form script URLs are never taken from dashboard JSON.

## Security model

| Allowed | Forbidden |
|--------|-----------|
| `pluginId` (+ optional `pluginVersion`) in panel spec | Arbitrary `scriptUrl` / CDN URLs in panel JSON |
| Registry-authored `bundlePath` under `/plugins/…` or `/api/v1/plugins/…` | Path traversal (`..`), `//host`, `https://…` in catalog paths |
| Builtin plugins defined in the main frontend bundle | Defining elements without a registry entry |

Panel JSON shape:

```json
{
  "kind": "signoz/WebComponentPanel",
  "spec": {
    "pluginId": "devops-status",
    "pluginVersion": "1.0.0",
    "element": "lax-devops-status",
    "attributes": { "env": "prod" },
    "properties": {}
  }
}
```

## Frontend host

| Path | Role |
|------|------|
| `frontend/src/plugins/host/` | Registry + loader |
| `frontend/src/plugins/builtins/` | In-bundle plugins (`hello`) |
| `frontend/public/plugins/` | Static catalog + sample bundles (dev/prod static) |
| `WebComponentPanel` renderer | Resolves `pluginId` → load → mount custom element |

Boot order:

1. Register builtins (`hello`)
2. Hydrate `public/plugins/manifest.json` (optional)
3. Hydrate `GET /api/v1/plugins` (optional)
4. Load allowlisted bundle if source is `catalog`
5. `customElements.whenDefined(tag)` then `document.createElement(tag)`

## Backend catalog API

Env:

| Variable | Default | Meaning |
|----------|---------|---------|
| `SIGNOZ_PLUGINS_DIR` | `./plugins` | Root with `manifest.json` and bundles |
| `SIGNOZ_PLUGINS_ENABLED` | enabled | Set `false` / `0` to disable |

Disk layout:

```
plugins/
  manifest.json
  devops-status/
    1.0.0/
      plugin.js    # or bundle.js
```

HTTP:

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| GET | `/api/v1/plugins` | View | List manifests |
| GET | `/api/v1/plugins/{id}?version=` | View | One manifest |
| GET | `/api/v1/plugins/{id}/{version}/bundle.js` | Open | Serve allowlisted JS |

## Authoring a plugin

1. Create a custom element (tag must contain a hyphen).
2. Ship a single JS file that calls `customElements.define('your-tag', …)` (classic script / IIFE is fine).
3. Add a catalog entry:

```json
{
  "id": "my-plugin",
  "version": "1.0.0",
  "displayName": "My Plugin",
  "element": "my-plugin-tile",
  "source": "catalog",
  "bundlePath": "/plugins/my-plugin/1.0.0/plugin.js",
  "description": "…"
}
```

4. Place the file at `frontend/public/plugins/my-plugin/1.0.0/plugin.js` **and/or** `plugins/my-plugin/1.0.0/plugin.js` for the API server.
5. Add a dashboard panel of type **Web Component** with `pluginId: "my-plugin"`.

### Lifecycle tips

- Prefer **Shadow DOM** so host styles do not leak.
- Read instance config from **attributes** / **properties** set by the host (`plugin-id`, `title`, plus `spec.attributes` / `spec.properties`).
- Do not fetch untrusted remote code from inside the plugin.
- Keep bundles small; they load on every dashboard view that uses them.

## Builtin: `hello`

- **id:** `hello`
- **element:** `lax-hello-wc`
- Always available without catalog/network (frontend builtin).

## Sample: `devops-status`

- **id:** `devops-status`
- **element:** `lax-devops-status`
- Static path: `/plugins/devops-status/1.0.0/plugin.js`
- API path: `/api/v1/plugins/devops-status/1.0.0/bundle.js`

## Roadmap (later phases)

- Plugin picker UI in the panel editor
- Signed / hashed bundle verification
- Full-page plugins (nav routes), not only dashboard panels
- Server-side CSP nonces for plugin scripts
