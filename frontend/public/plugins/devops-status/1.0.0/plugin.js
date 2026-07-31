/**
 * Sample catalog plugin: DevOps Status tile.
 * Loaded only via the allowlisted registry (pluginId=devops-status).
 * IIFE so it works as a classic script without bundler ESM assumptions.
 */
(function registerDevopsStatusPlugin(global) {
	var TAG = 'lax-devops-status';
	if (global.customElements && global.customElements.get(TAG)) {
		return;
	}

	function DevopsStatusElement() {
		return Reflect.construct(HTMLElement, [], DevopsStatusElement);
	}
	DevopsStatusElement.prototype = Object.create(HTMLElement.prototype);
	DevopsStatusElement.prototype.constructor = DevopsStatusElement;
	Object.setPrototypeOf(DevopsStatusElement, HTMLElement);

	DevopsStatusElement.prototype.connectedCallback = function connectedCallback() {
		if (this.shadowRoot) {
			return;
		}
		var root = this.attachShadow({ mode: 'open' });
		var title = this.getAttribute('title') || 'DevOps Status';
		var pluginId = this.getAttribute('plugin-id') || 'devops-status';
		var env = this.getAttribute('env') || 'prod';
		root.innerHTML =
			'<style>' +
			':host{display:block;height:100%;width:100%;box-sizing:border-box;font-family:inherit;}' +
			'.wrap{height:100%;display:flex;flex-direction:column;justify-content:center;gap:.4rem;' +
			'padding:.75rem 1rem;border-radius:6px;background:rgba(34,197,94,.08);' +
			'border:1px solid rgba(34,197,94,.4);color:inherit;}' +
			'.title{font-size:.95rem;font-weight:600;}' +
			'.row{display:flex;gap:.5rem;align-items:center;font-size:.8rem;}' +
			'.dot{width:.55rem;height:.55rem;border-radius:50%;background:#22c55e;}' +
			'.meta{font-size:.75rem;opacity:.75;}' +
			'</style>' +
			'<div class="wrap" part="root">' +
			'<div class="title"></div>' +
			'<div class="row"><span class="dot"></span><span>All systems nominal</span></div>' +
			'<div class="meta"></div>' +
			'</div>';
		root.querySelector('.title').textContent = title;
		root.querySelector('.meta').textContent =
			'plugin: ' + pluginId + ' · env: ' + env + ' · catalog sample';
	};

	if (global.customElements) {
		global.customElements.define(TAG, DevopsStatusElement);
	}
})(typeof window !== 'undefined' ? window : globalThis);
