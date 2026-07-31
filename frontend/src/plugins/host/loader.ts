import { withBasePath } from 'utils/basePath';

import {
	getBuiltinRegistrar,
	getPluginManifest,
	listPluginManifests,
	markCatalogHydrated,
	registerPluginManifest,
	isCatalogHydrated,
} from './registry';
import type { LoadPluginOptions, LoadedPlugin, PluginManifest } from './types';

const scriptPromises = new Map<string, Promise<void>>();

function isAllowlistedBundlePath(path: string): boolean {
	// Absolute-from-origin paths only; no scheme, no //host.
	if (!path.startsWith('/') || path.startsWith('//')) {
		return false;
	}
	if (path.includes('://') || path.includes('\\') || path.includes('..')) {
		return false;
	}
	// Allow only known plugin URL prefixes.
	return (
		path.startsWith('/plugins/') ||
		path.startsWith('/api/v1/plugins/') ||
		path.startsWith('/public/plugins/')
	);
}

function loadScriptOnce(bundlePath: string): Promise<void> {
	const existing = scriptPromises.get(bundlePath);
	if (existing) {
		return existing;
	}

	const promise = new Promise<void>((resolve, reject) => {
		const src = withBasePath(bundlePath);
		// Reuse a tag if HMR / prior panel already injected it.
		const prior = document.querySelector<HTMLScriptElement>(
			`script[data-signoz-plugin-src="${src}"]`,
		);
		if (prior) {
			if (prior.dataset.loaded === '1') {
				resolve();
				return;
			}
			prior.addEventListener('load', () => resolve(), { once: true });
			prior.addEventListener(
				'error',
				() => reject(new Error(`failed to load plugin script: ${src}`)),
				{ once: true },
			);
			return;
		}

		const script = document.createElement('script');
		script.src = src;
		script.async = true;
		script.dataset.signozPluginSrc = src;
		script.onload = (): void => {
			script.dataset.loaded = '1';
			resolve();
		};
		script.onerror = (): void => {
			scriptPromises.delete(bundlePath);
			reject(new Error(`failed to load plugin script: ${src}`));
		};
		document.head.appendChild(script);
	});

	scriptPromises.set(bundlePath, promise);
	return promise;
}

function whenDefined(tag: string, timeoutMs: number): Promise<void> {
	if (typeof customElements === 'undefined') {
		return Promise.reject(new Error('customElements is not available'));
	}
	if (customElements.get(tag)) {
		return Promise.resolve();
	}
	const defined = customElements.whenDefined(tag).then(() => undefined);
	const timeout = new Promise<void>((_, reject) => {
		window.setTimeout(() => {
			reject(
				new Error(
					`timed out waiting for custom element <${tag}> (${timeoutMs}ms)`,
				),
			);
		}, timeoutMs);
	});
	return Promise.race([defined, timeout]);
}

/**
 * Pull remote catalog from the API when available. Failures are non-fatal —
 * builtins and public/manifest still work offline.
 */
export async function hydratePluginCatalogFromApi(): Promise<void> {
	if (isCatalogHydrated()) {
		return;
	}
	try {
		const res = await fetch(withBasePath('/api/v1/plugins'), {
			credentials: 'same-origin',
		});
		if (!res.ok) {
			markCatalogHydrated();
			return;
		}
		const body = (await res.json()) as {
			data?: PluginManifest[] | { plugins?: PluginManifest[] };
			status?: string;
		};
		const raw = body.data;
		const list: PluginManifest[] = Array.isArray(raw)
			? raw
			: raw && typeof raw === 'object' && Array.isArray(raw.plugins)
				? raw.plugins
				: [];
		list.forEach((m) => {
			if (m?.id && m?.element) {
				registerPluginManifest({
					...m,
					source: m.source || 'catalog',
					version: m.version || 'latest',
				});
			}
		});
	} catch {
		// offline / API not yet deployed — ignore
	} finally {
		markCatalogHydrated();
	}
}

/**
 * Optionally merge manifests from a static file under public/plugins/manifest.json.
 */
export async function hydratePluginCatalogFromPublicManifest(): Promise<void> {
	try {
		const res = await fetch(withBasePath('/plugins/manifest.json'), {
			credentials: 'same-origin',
		});
		if (!res.ok) {
			return;
		}
		const body = (await res.json()) as {
			plugins?: PluginManifest[];
		};
		(body.plugins || []).forEach((m) => {
			if (m?.id && m?.element) {
				registerPluginManifest({
					...m,
					source: m.source || 'catalog',
					version: m.version || 'latest',
				});
			}
		});
	} catch {
		// optional file
	}
}

/**
 * Load (or resolve builtin) a plugin by id. Only allowlisted registry paths
 * are fetched — dashboard-supplied URLs are never used.
 */
export async function loadPlugin(
	pluginId: string,
	options: LoadPluginOptions = {},
): Promise<LoadedPlugin> {
	if (!pluginId) {
		throw new Error('pluginId is required');
	}

	// Best-effort catalog hydrate once per session.
	if (!isCatalogHydrated()) {
		await Promise.all([
			hydratePluginCatalogFromPublicManifest(),
			hydratePluginCatalogFromApi(),
		]);
	}

	const manifest = getPluginManifest(pluginId, options.version);
	if (!manifest) {
		const known = listPluginManifests()
			.map((m) => m.id)
			.join(', ');
		throw new Error(
			`unknown plugin "${pluginId}"${
				options.version ? `@${options.version}` : ''
			}. Known: ${known || '(none)'}`,
		);
	}

	const timeoutMs = options.timeoutMs ?? 15_000;

	if (manifest.source === 'builtin') {
		const define = getBuiltinRegistrar(manifest.id);
		if (define) {
			define();
		}
		await whenDefined(manifest.element, timeoutMs);
		return { manifest, element: manifest.element };
	}

	const bundlePath = manifest.bundlePath;
	if (!bundlePath) {
		throw new Error(
			`plugin "${manifest.id}" is catalog-sourced but has no bundlePath`,
		);
	}
	if (!isAllowlistedBundlePath(bundlePath)) {
		throw new Error(
			`plugin "${manifest.id}" bundlePath is not allowlisted: ${bundlePath}`,
		);
	}

	await loadScriptOnce(bundlePath);
	await whenDefined(manifest.element, timeoutMs);
	return { manifest, element: manifest.element };
}

/** Test helper */
export function __resetLoaderForTests(): void {
	scriptPromises.clear();
}
