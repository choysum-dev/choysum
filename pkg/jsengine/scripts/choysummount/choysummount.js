// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/**
 * Choysum FE unit mount host — VTU subset (PR-unit-vue-host + host-2).
 * In: mount, shallowMount, stubs, flushPromises, wrapper.find/trigger/unmount,
 *     options.global.plugins / provide / components.
 * Out: findComponent, setProps family, wrapper.html, happy-dom.
 *
 * Bundle this module with esbuild (imports vue). Do not eval as a bare global
 * unless Vue is already on globalThis.Vue.
 */
import { createApp, h, nextTick } from 'vue';

function makeDefaultStub(name) {
  var key = String(name);
  return {
    name: key,
    render: function () {
      return h('div', { class: 'stub-' + key }, key);
    },
  };
}

function normalizeStubs(stubs) {
  if (!stubs) return {};
  if (stubs === true) return { __all: true };
  var out = Object.create(null);
  if (Array.isArray(stubs)) {
    stubs.forEach(function (name) {
      out[String(name)] = makeDefaultStub(name);
    });
    return out;
  }
  Object.keys(stubs).forEach(function (name) {
    var val = stubs[name];
    if (val === true) {
      out[name] = makeDefaultStub(name);
    } else if (val && (typeof val === 'object' || typeof val === 'function')) {
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

function installGlobalOptions(app, globalOpts) {
  if (!globalOpts || typeof globalOpts !== 'object') return;
  var plugins = globalOpts.plugins;
  if (Array.isArray(plugins)) {
    for (var i = 0; i < plugins.length; i++) {
      var plugin = plugins[i];
      if (!plugin) continue;
      if (Array.isArray(plugin)) {
        app.use.apply(app, plugin);
      } else {
        app.use(plugin);
      }
    }
  }
  var provideMap = globalOpts.provide;
  if (provideMap && typeof provideMap === 'object') {
    var provideKeys =
      typeof Reflect !== 'undefined' && typeof Reflect.ownKeys === 'function'
        ? Reflect.ownKeys(provideMap)
        : Object.keys(provideMap);
    provideKeys.forEach(function (key) {
      app.provide(key, provideMap[key]);
    });
  }
  var components = globalOpts.components;
  if (components && typeof components === 'object') {
    Object.keys(components).forEach(function (name) {
      app.component(name, components[name]);
    });
  }
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
          var evt = Evt ? new Evt(String(eventName), { bubbles: true }) : { type: String(eventName) };
          return Promise.resolve()
            .then(function () {
              if (typeof node.dispatchEvent === 'function') {
                node.dispatchEvent(evt);
              }
            })
            .then(function () {
              return flushPromises();
            });
        },
      };
    },
    trigger: function (eventName) {
      var Evt = typeof Event === 'function' ? Event : null;
      var evt = Evt ? new Evt(String(eventName), { bubbles: true }) : { type: String(eventName) };
      return Promise.resolve()
        .then(function () {
          if (el && typeof el.dispatchEvent === 'function') {
            el.dispatchEvent(evt);
          }
        })
        .then(function () {
          return flushPromises();
        });
    },
    text: function () {
      return el ? String(el.textContent || '') : '';
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
    auto[name] = makeDefaultStub(name);
  });
  return auto;
}

function withComponentStubs(component, stubs, shallow) {
  var map = normalizeStubs(stubs);
  var base = component && typeof component === 'object' ? component : {};
  // shallowMount always auto-stubs options-API children; explicit stubs override.
  if (map.__all || shallow) {
    var explicit = Object.assign({}, map);
    delete explicit.__all;
    map = Object.assign(autoStubComponents(base.components), explicit);
  }
  var keys = Object.keys(map);
  if (!keys.length) {
    return component;
  }
  // Options-API / defineComponent: merge into components so template lookups hit stubs.
  // script-setup local imports are closed over and are not stubbed (use bundle aliases).
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
  var shallow = !!options.shallow;
  var root = withComponentStubs(component, options.stubs, shallow);
  var app = createApp(root, options.props || {});
  // Order: plugins → provide → components → stubs (VTU-like).
  installGlobalOptions(app, options.global);
  installStubs(app, options.stubs);
  if (shallow) {
    app.config.warnHandler = function () {};
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
