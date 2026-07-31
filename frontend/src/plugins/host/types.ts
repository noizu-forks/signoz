/**
 * Dashboard plugin host contract (Noizu / Lacrosse fork).
 *
 * Security rule: dashboard JSON stores only pluginId (+ optional version, element,
 * attributes, properties). Free-form script URLs are never allowed in panel specs.
 * Bundles load only from the allowlisted registry (builtin or catalog entry).
 */

export type PluginSource = 'builtin' | 'catalog' | 'remote';

/** Manifest describing a loadable Web Component dashboard plugin. */
export interface PluginManifest {
	/** Stable id referenced from WebComponentPanelSpec.pluginId */
	id: string;
	/** Semver-ish string; empty means "any / latest known" */
	version: string;
	/** Display name for picker / debug UI */
	displayName: string;
	/** Custom element tag the bundle must define (e.g. lax-hello-wc) */
	element: string;
	/**
	 * How the host obtains the custom element.
	 * - builtin: define() runs in the main bundle (no network)
	 * - catalog: load script from allowlisted bundlePath / API
	 */
	source: PluginSource;
	/**
	 * Absolute-from-origin path to the JS module/script that defines `element`.
	 * Required when source === 'catalog'. Never a free-form external URL from
	 * dashboard JSON — only registry-authored paths.
	 */
	bundlePath?: string;
	/** Optional human description */
	description?: string;
}

export interface LoadPluginOptions {
	/** Prefer this version when multiple are registered for the same id */
	version?: string;
	/** Soft timeout for customElements.whenDefined (ms). Default 15000. */
	timeoutMs?: number;
}

export interface LoadedPlugin {
	manifest: PluginManifest;
	/** Resolved custom element tag that is defined and ready to create */
	element: string;
}

export type BuiltinRegistrar = () => void;
