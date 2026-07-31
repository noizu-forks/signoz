import type { BuiltinRegistrar, PluginManifest } from './types';

const manifestsByKey = new Map<string, PluginManifest>();
const builtinsById = new Map<string, BuiltinRegistrar>();
let catalogHydrated = false;

function keyOf(id: string, version: string): string {
	return `${id}@${version || 'latest'}`;
}

/** Register a plugin manifest. Later registrations for the same id@version replace earlier ones. */
export function registerPluginManifest(manifest: PluginManifest): void {
	if (!manifest.id || !manifest.element) {
		throw new Error('plugin manifest requires id and element');
	}
	manifestsByKey.set(keyOf(manifest.id, manifest.version), {
		...manifest,
		version: manifest.version || 'latest',
	});
}

/** Register an in-bundle define() function for a builtin plugin id. */
export function registerBuiltin(id: string, define: BuiltinRegistrar): void {
	builtinsById.set(id, define);
}

export function getBuiltinRegistrar(id: string): BuiltinRegistrar | undefined {
	return builtinsById.get(id);
}

/**
 * Resolve a manifest by id (+ optional version).
 * Preference: exact version match → highest registered version for id → undefined.
 */
export function getPluginManifest(
	id: string,
	version?: string,
): PluginManifest | undefined {
	if (version) {
		const exact = manifestsByKey.get(keyOf(id, version));
		if (exact) {
			return exact;
		}
	}
	// Prefer non-latest explicit versions last-wins order is map insertion;
	// scan for any matching id and return the first (registration order).
	let fallback: PluginManifest | undefined;
	manifestsByKey.forEach((m) => {
		if (m.id === id) {
			if (!fallback || (version && m.version === version)) {
				fallback = m;
			}
			// Prefer exact 'latest' or last registered when no version pin
			if (!version) {
				fallback = m;
			}
		}
	});
	return fallback;
}

export function listPluginManifests(): PluginManifest[] {
	return Array.from(manifestsByKey.values());
}

export function isCatalogHydrated(): boolean {
	return catalogHydrated;
}

export function markCatalogHydrated(): void {
	catalogHydrated = true;
}

/** Test helper — clears registry state. */
export function __resetPluginRegistryForTests(): void {
	manifestsByKey.clear();
	builtinsById.clear();
	catalogHydrated = false;
}
