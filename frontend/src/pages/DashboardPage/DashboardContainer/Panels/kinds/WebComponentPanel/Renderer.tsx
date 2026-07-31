import { useEffect, useMemo, useRef, useState } from 'react';
import type { DashboardtypesWebComponentPanelSpecDTO } from 'api/generated/services/sigNoz.schemas';
import {
	ensureBuiltinPluginsRegistered,
	loadPlugin,
} from 'plugins/host';

import type { StaticRendererProps } from '../../types/rendererProps';
import styles from './Renderer.module.scss';

/**
 * Host for dashboard Web Components.
 * Resolves pluginId via the allowlisted registry, loads the bundle (or builtin),
 * then mounts the custom element. Free-form script URLs from panel JSON are ignored.
 */
function WebComponentPanelRenderer({
	panel,
}: StaticRendererProps<'signoz/WebComponentPanel'>): JSX.Element {
	const hostRef = useRef<HTMLDivElement>(null);
	const [error, setError] = useState<string | null>(null);
	const [loading, setLoading] = useState(true);

	const spec = useMemo<DashboardtypesWebComponentPanelSpecDTO>(
		() => panel.spec.plugin.spec ?? {},
		[panel.spec.plugin.spec],
	);

	useEffect(() => {
		const host = hostRef.current;
		if (!host) {
			return undefined;
		}

		let disposed = false;
		let el: HTMLElement | null = null;

		const run = async (): Promise<void> => {
			try {
				ensureBuiltinPluginsRegistered();
				const pluginId = spec.pluginId || 'hello';
				const loaded = await loadPlugin(pluginId, {
					version: spec.pluginVersion || undefined,
				});

				// Prefer registry element; allow panel.element only when it matches
				// the loaded plugin (or is empty). Prevents tag spoofing.
				const requested = spec.element?.trim();
				const tag =
					requested && requested === loaded.element
						? requested
						: loaded.element;

				if (typeof customElements !== 'undefined' && !customElements.get(tag)) {
					throw new Error(`custom element <${tag}> is not defined after load`);
				}

				el = document.createElement(tag);
				el.setAttribute('plugin-id', pluginId);
				el.setAttribute(
					'title',
					panel.spec.display?.name || loaded.manifest.displayName || tag,
				);

				if (spec.attributes) {
					Object.entries(spec.attributes).forEach(([key, value]) => {
						el?.setAttribute(key, value);
					});
				}

				if (spec.properties) {
					Object.entries(spec.properties).forEach(([key, value]) => {
						// Custom elements often accept config as properties.
						// eslint-disable-next-line no-param-reassign, @typescript-eslint/no-explicit-any
						(el as any)[key] = value;
					});
				}

				if (disposed || !hostRef.current) {
					return;
				}
				hostRef.current.replaceChildren(el);
				setError(null);
				setLoading(false);
			} catch (err) {
				if (!disposed) {
					setError(
						err instanceof Error
							? err.message
							: 'Failed to mount web component plugin',
					);
					setLoading(false);
				}
			}
		};

		setLoading(true);
		void run();

		return (): void => {
			disposed = true;
			host.replaceChildren();
		};
	}, [panel.spec.display?.name, spec]);

	if (error) {
		return (
			<div className={styles.error} data-testid="wc-panel-error">
				{error}
			</div>
		);
	}

	return (
		<div
			ref={hostRef}
			className={styles.host}
			data-testid="wc-panel-host"
			data-loading={loading ? 'true' : 'false'}
		/>
	);
}

export default WebComponentPanelRenderer;
