// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/**
 * Choysum FE unit mount host — VTU subset (frozen for PR-unit-vue-host).
 * In: mount, shallowMount, stubs, flushPromises, wrapper.find/trigger/unmount.
 * Out: findComponent, setProps family, happy-dom.
 *
 * Bundle this module with esbuild (imports vue). Do not eval as a bare global
 * unless Vue is already on globalThis.Vue.
 */
import { createApp, h, nextTick } from 'vue';

function normalizeStubs(stubs) {
  if (!stubs) return {};
  if (stubs === true) return { __all: true };
  var out = Object.create(null);
  Object.keys(stubs).forEach(function (name) {
    var val = stubs[name];
    if (val === true) {
      out[name] = {
        name: name,
        render: function () {
          return h(name);
        },
      };
    } else if (val && typeof val === 'object') {
      out[name] = val;
    }
  });
  return out;
}

function installStubs(app, stubs) {
  var map = normalizeStubs(stubs);
  Object.keys(map).forEach(function (name) {
    if (name === '__all') return;
    app.component(name, map[name]);
  });
}

function makeWrapper(app, el, vm) {
  return {
    element: el,
    vm: vm,
    find: function (sel) {
      var node = el.querySelector(sel);
      return {
        exists: function () {
          return !!node;
        },
        element: node,
        text: function () {
          return node ? String(node.textContent || '') : '';
        },
        trigger: function (eventName) {
          if (!node) return Promise.resolve();
          var Evt = typeof Event === 'function' ? Event : null;
          var evt = Evt ? new Evt(String(eventName), { bubbles: true } ) : { type: String(eventName) };
          if (typeof node.dispatchEvent === 'function') {
            node.dispatchEvent(evt);
          }
          return flushPromises();
        },
      };
    },
    trigger: function (eventName) {
      var Evt = typeof Event === 'function' ? Event : null;
      var evt = Evt ? new Evt(String(eventName), { bubbles: true }) : { type: String(eventName) };
      if (el && typeof el.dispatchEvent === 'function') {
        el.dispatchEvent(evt);
      }
      return flushPromises();
    },
    text: function () {
      return el ? String(el.textContent || '') : '';
    },
    html: function () {
      return el ? String(el.innerHTML || el.textContent || '') : '';
    },
    unmount: function () {
      try {
        app.unmount();
      } catch (_) {}
      if (el && el.parentNode) {
        try {
          el.parentNode.removeChild(el);
        } catch (_) {}
      }
    },
  };
}

function autoStubComponents(components) {
  var auto = Object.create(null);
  Object.keys(components || {}).forEach(function (name) {
    auto[name] = {
      name: name,
      render: function () {
        return h('div', { class: 'stub-' + name }, name);
      },
    };
  });
  return auto;
}

function withComponentStubs(component, stubs) {
  var map = normalizeStubs(stubs);
  var base = component && typeof component === 'object' ? component : {};
  if (map.__all) {
    // shallowMount default: stub every options-API child registered on `components`.
    map = Object.assign(autoStubComponents(base.components), map);
    delete map.__all;
  }
  var keys = Object.keys(map);
  if (!keys.length) {
    return component;
  }
  // Options-API / defineComponent: merge into components so template lookups hit stubs.
  // script-setup local imports are closed over and are not stubbed (use defineComponent fixtures).
  var merged = Object.assign({}, base);
  merged.components = Object.assign({}, base.components || {}, map);
  return merged;
}

export function mount(component, options) {
  options = options || {};
  var doc = globalThis.document;
  if (!doc || typeof doc.createElement !== 'function') {
    throw new Error('choysumMount: minimal DOM not installed (call InstallMinimalDOM first)');
  }
  var el = doc.createElement('div');
  if (doc.body && typeof doc.body.appendChild === 'function') {
    doc.body.appendChild(el);
  }
  var root = withComponentStubs(component, options.stubs);
  var app = createApp(root, options.props || {});
  installStubs(app, options.stubs);
  if (typeof options.shallow === 'boolean' && options.shallow) {
    if (options.stubs === true) {
      app.config.warnHandler = function () {};
    }
  }
  var vm = app.mount(el);
  return makeWrapper(app, el, vm);
}

export function shallowMount(component, options) {
  options = options || {};
  if (options.stubs == null) {
    options = Object.assign({}, options, { stubs: true, shallow: true });
  } else {
    options = Object.assign({}, options, { shallow: true });
  }
  return mount(component, options);
}

export function flushPromises() {
  // QuickJS unit host may lack setTimeout until BootstrapTimers; prefer microtasks + nextTick.
  var chain = Promise.resolve();
  if (typeof nextTick === 'function') {
    chain = chain.then(function () {
      return nextTick();
    });
  }
  return chain
    .then(function () {
      return new Promise(function (resolve) {
        if (typeof queueMicrotask === 'function') {
          queueMicrotask(resolve);
        } else {
          Promise.resolve().then(resolve);
        }
      });
    })
    .then(function () {
      return new Promise(function (resolve) {
        if (typeof queueMicrotask === 'function') {
          queueMicrotask(resolve);
        } else {
          Promise.resolve().then(resolve);
        }
      });
    });
}

const api = { mount: mount, shallowMount: shallowMount, flushPromises: flushPromises };
if (typeof globalThis !== 'undefined') {
  globalThis.choysumMount = api;
}
export default api;
