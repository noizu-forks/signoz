import { registerHelloBuiltin } from '../builtins/hello';

import {
	__resetPluginRegistryForTests,
	getPluginManifest,
	listPluginManifests,
	registerBuiltin,
	registerPluginManifest,
} from './registry';
import {
	__resetLoaderForTests,
	hydratePluginCatalogFromApi,
	hydratePluginCatalogFromPublicManifest,
	loadPlugin,
} from './loader';

export type {
	LoadedPlugin,
	LoadPluginOptions,
	PluginManifest,
	PluginSource,
} from './types';

export {
	registerPluginManifest,
	registerBuiltin,
	getPluginManifest,
	listPluginManifests,
	loadPlugin,
	hydratePluginCatalogFromApi,
	hydratePluginCatalogFromPublicManifest,
	__resetPluginRegistryForTests,
	__resetLoaderForTests,
};

let builtinsRegistered = false;

/** Idempotent: register bundled plugins once at app/panel boot. */
export function ensureBuiltinPluginsRegistered(): void {
	if (builtinsRegistered) {
		return;
	}
	registerHelloBuiltin();
	builtinsRegistered = true;
}

/** Test helper */
export function __resetBuiltinsFlagForTests(): void {
	builtinsRegistered = false;
}
