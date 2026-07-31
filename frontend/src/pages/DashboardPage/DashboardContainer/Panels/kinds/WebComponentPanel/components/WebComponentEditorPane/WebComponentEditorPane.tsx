import type {
	DashboardtypesPanelSpecDTO,
	DashboardtypesWebComponentPanelSpecDTO,
} from 'api/generated/services/sigNoz.schemas';

import type { StaticEditorPaneProps } from '../../../../types/panelDefinition';

import styles from './WebComponentEditorPane.module.scss';

/**
 * The Web Component panel's authoring pane — plugin identity fields only
 * (pluginId / pluginVersion / element). Attributes and properties have no
 * generic editor yet; they are set via the dashboard JSON until a dedicated
 * config section lands.
 */
function WebComponentEditorPane({
	spec,
	onChangeSpec,
}: StaticEditorPaneProps): JSX.Element {
	const pluginSpec = spec.plugin.spec as DashboardtypesWebComponentPanelSpecDTO;

	const update = (
		patch: Partial<DashboardtypesWebComponentPanelSpecDTO>,
	): void => {
		const next: DashboardtypesPanelSpecDTO = {
			...spec,
			plugin: { ...spec.plugin, spec: { ...pluginSpec, ...patch } },
		};
		onChangeSpec(next);
	};

	return (
		<div className={styles.container}>
			<label className={styles.field}>
				<span>Plugin ID</span>
				<input
					type="text"
					value={pluginSpec.pluginId ?? ''}
					onChange={(e): void => update({ pluginId: e.target.value })}
					placeholder="hello"
					data-testid="wc-editor-plugin-id"
				/>
			</label>
			<label className={styles.field}>
				<span>Plugin Version</span>
				<input
					type="text"
					value={pluginSpec.pluginVersion ?? ''}
					onChange={(e): void => update({ pluginVersion: e.target.value })}
					placeholder="1.0.0"
					data-testid="wc-editor-plugin-version"
				/>
			</label>
			<label className={styles.field}>
				<span>Element</span>
				<input
					type="text"
					value={pluginSpec.element ?? ''}
					onChange={(e): void => update({ element: e.target.value })}
					placeholder="lax-hello-wc"
					data-testid="wc-editor-element"
				/>
			</label>
		</div>
	);
}

export default WebComponentEditorPane;
