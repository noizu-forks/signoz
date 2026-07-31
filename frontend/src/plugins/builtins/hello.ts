import { registerBuiltin, registerPluginManifest } from '../host/registry';

/**
 * Phase-0 demo custom element. Kept as a builtin so dashboards with
 * pluginId=hello work without network / catalog.
 */
const TAG = 'lax-hello-wc';

class LaxHelloWc extends HTMLElement {
	connectedCallback(): void {
		if (this.shadowRoot) {
			return;
		}
		const root = this.attachShadow({ mode: 'open' });
		const pluginId = this.getAttribute('plugin-id') || 'hello';
		const title = this.getAttribute('title') || 'Hello Web Component';
		root.innerHTML = `
			<style>
				:host {
					display: block;
					height: 100%;
					width: 100%;
					box-sizing: border-box;
					font-family: inherit;
				}
				.wrap {
					height: 100%;
					display: flex;
					flex-direction: column;
					align-items: flex-start;
					justify-content: center;
					gap: 0.35rem;
					padding: 0.75rem 1rem;
					border-radius: 6px;
					background: rgba(78, 116, 248, 0.08);
					border: 1px solid rgba(78, 116, 248, 0.35);
					color: inherit;
				}
				.title {
					font-size: 0.95rem;
					font-weight: 600;
				}
				.meta {
					font-size: 0.75rem;
					opacity: 0.75;
				}
			</style>
			<div class="wrap" part="root">
				<div class="title">${title}</div>
				<div class="meta">plugin: ${pluginId} · tag: ${TAG}</div>
				<div class="meta">SigNoz WebComponentPanel (builtin hello)</div>
			</div>
		`;
	}
}

export function ensureHelloWcDefined(): string {
	if (typeof customElements !== 'undefined' && !customElements.get(TAG)) {
		customElements.define(TAG, LaxHelloWc);
	}
	return TAG;
}

export const HELLO_WC_TAG = TAG;

export function registerHelloBuiltin(): void {
	registerPluginManifest({
		id: 'hello',
		version: '1.0.0',
		displayName: 'Hello Web Component',
		element: TAG,
		source: 'builtin',
		description: 'Built-in spike plugin for WebComponentPanel',
	});
	registerBuiltin('hello', () => {
		ensureHelloWcDefined();
	});
}
