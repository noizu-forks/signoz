/**
 * Re-export builtin hello for any remaining direct imports.
 * Prefer `plugins/host` + `plugins/builtins/hello` for new code.
 */
export {
	ensureHelloWcDefined,
	HELLO_WC_TAG,
} from 'plugins/builtins/hello';
