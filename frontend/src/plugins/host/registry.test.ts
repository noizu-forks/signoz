import {
	__resetPluginRegistryForTests,
	getPluginManifest,
	listPluginManifests,
	registerPluginManifest,
} from './registry';

describe('plugin registry', () => {
	beforeEach(() => {
		__resetPluginRegistryForTests();
	});

	it('registers and resolves by id', () => {
		registerPluginManifest({
			id: 'hello',
			version: '1.0.0',
			displayName: 'Hello',
			element: 'lax-hello-wc',
			source: 'builtin',
		});
		const m = getPluginManifest('hello');
		expect(m?.element).toBe('lax-hello-wc');
		expect(listPluginManifests()).toHaveLength(1);
	});

	it('prefers exact version when provided', () => {
		registerPluginManifest({
			id: 'p',
			version: '1.0.0',
			displayName: 'P1',
			element: 'p-one',
			source: 'catalog',
			bundlePath: '/plugins/p/1.0.0/plugin.js',
		});
		registerPluginManifest({
			id: 'p',
			version: '2.0.0',
			displayName: 'P2',
			element: 'p-two',
			source: 'catalog',
			bundlePath: '/plugins/p/2.0.0/plugin.js',
		});
		expect(getPluginManifest('p', '1.0.0')?.element).toBe('p-one');
		expect(getPluginManifest('p', '2.0.0')?.element).toBe('p-two');
	});

	it('rejects incomplete manifests', () => {
		expect(() =>
			registerPluginManifest({
				id: '',
				version: '1',
				displayName: 'x',
				element: 'x-y',
				source: 'builtin',
			}),
		).toThrow(/id and element/);
	});
});
