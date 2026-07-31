import { type SectionConfig } from '../../types/sections';

/**
 * Web Component panels have no query-driven viz sections.
 * Instance config lives on WebComponentPanelSpec (pluginId, attributes, …).
 * A dedicated config section/picker can land in a later phase.
 */
export const sections: SectionConfig[] = [];
