// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Minimal vue stub for the P0 coverage spike: createApp(...).mount() must run setup.
 * Not a production Vue runtime — see vue_cov_spike_migration.md for host roadmap.
 */
export function ref(value) {
  return { value };
}

export function computed(getter) {
  return {
    get value() {
      return typeof getter === 'function' ? getter() : getter;
    },
  };
}

export function onMounted(fn) {
  if (typeof fn === 'function') {
    queueMicrotask(fn);
  }
}

export function defineComponent(options) {
  return options;
}

export function h() {
  return null;
}

export function createApp(rootComponent) {
  const comp = rootComponent && rootComponent.default ? rootComponent.default : rootComponent;
  return {
    mount(_el) {
      if (!comp) return {};
      const setup = typeof comp.setup === 'function' ? comp.setup : null;
      if (setup) {
        setup({}, { attrs: {}, slots: {}, emit() {}, expose() {} });
      }
      if (typeof comp.render === 'function') {
        try {
          comp.render(h, {});
        } catch {
          // Spike stub: render may reference DOM helpers we do not implement.
        }
      }
      return { unmount() {} };
    },
  };
}

export default { createApp, ref, computed, onMounted, defineComponent, h };
