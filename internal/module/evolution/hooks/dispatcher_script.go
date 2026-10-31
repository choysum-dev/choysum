// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hooks

// HookDispatcherScriptContent is the always-on JS that defines
// globalThis.__choysum_hook__. Bundle it into application init so ready
// (and other phases) can invoke registered hooks without Reload.
//
// Keep in sync with buildHookWrapperScript's dispatcher body (no CHOYSUM_* seed).
const HookDispatcherScriptContent = `(() => {
	const resolveModuleRoot = (app, moduleName) => {
		const stable = globalThis.__choysum_hook_module_roots__;
		const fromStable = stable && stable[app] && stable[app][moduleName];
		if (fromStable) return fromStable;
		const root = globalThis[app];
		return root && root[moduleName];
	};

	globalThis.__choysum_hook__ = async function (app, moduleName, phase) {
		if (!app || !moduleName || !phase) {
			return;
		}
		globalThis.CHOYSUM_APP_NAME = app;
		globalThis.CHOYSUM_MODULE_NAME = moduleName;
		const moduleRoot = resolveModuleRoot(app, moduleName);
		const registryRoot = moduleRoot && moduleRoot.__hookRegistry__;
		const registry = registryRoot && registryRoot[phase];
		const hook = moduleRoot && moduleRoot.hook;
		if (!registry || registry.length === 0) {
			return;
		}
		for (const name of registry) {
			const fn = hook && hook[name];
			if (typeof fn !== 'function') {
				throw new Error('HOOK_UNSUPPORTED: hook not found ' + app + '.' + moduleName + '.hook.' + name);
			}
			await fn();
		}
	};
})();
`

// HookDispatcherFileName is the synthetic filename for the dispatcher script.
const HookDispatcherFileName = "__choysum_hook_dispatcher.js"
