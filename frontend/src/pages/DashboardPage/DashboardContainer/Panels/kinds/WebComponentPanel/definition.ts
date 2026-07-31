import { LayoutGrid } from '@signozhq/icons';

import type { PanelDefinition } from '../../types/panelDefinition';
import Renderer from './Renderer';
import WebComponentEditorPane from './components/WebComponentEditorPane/WebComponentEditorPane';
import { sections } from './sections';

export const definition: PanelDefinition<'signoz/WebComponentPanel'> = {
	kind: 'signoz/WebComponentPanel',
	displayName: 'Web Component',
	icon: LayoutGrid,
	mode: 'static',
	Renderer,
	EditorPane: WebComponentEditorPane,
	sections,
	actions: {
		view: true,
		edit: true,
		clone: true,
		download: { csv: false, png: false, svg: false },
		createAlert: false,
		search: false,
		drilldown: false,
	},
};
