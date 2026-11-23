// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/**
 * FE unit stub for vue-router injection used by product pages under QJS.
 * Real vue-router is not required when this module is aliased.
 * Supports createRouter({ routes }) enough for named push / isReady / resolve.
 */
import { inject, defineComponent, h, reactive } from 'vue';

var ROUTER_KEY = 'choysumFeStubRouter';
var ROUTE_KEY = 'choysumFeStubRoute';

function stringifyQueryValue(k, v) {
  var key = encodeURIComponent(k);
  // Mirror vue-router: null → bare key; undefined → omit.
  if (v === undefined) return null;
  if (v === null) return key;
  return key + '=' + encodeURIComponent(String(v));
}

function stringifyQuery(query) {
  if (!query || typeof query !== 'object') return '';
  var keys = Object.keys(query);
  if (!keys.length) return '';
  var parts = [];
  for (var i = 0; i < keys.length; i++) {
    var k = keys[i];
    var v = query[k];
    if (Array.isArray(v)) {
      for (var j = 0; j < v.length; j++) {
        var arrPart = stringifyQueryValue(k, v[j]);
        if (arrPart != null) parts.push(arrPart);
      }
    } else {
      var part = stringifyQueryValue(k, v);
      if (part != null) parts.push(part);
    }
  }
  return parts.length ? '?' + parts.join('&') : '';
}

function parseQueryString(search) {
  var query = {};
  if (!search) return query;
  var raw = String(search).charAt(0) === '?' ? String(search).slice(1) : String(search);
  if (!raw) return query;
  var pairs = raw.split('&');
  for (var i = 0; i < pairs.length; i++) {
    if (!pairs[i]) continue;
    var eq = pairs[i].indexOf('=');
    var k = decodeURIComponent(eq >= 0 ? pairs[i].slice(0, eq) : pairs[i]);
    // Bare `?flag` → null (vue-router); `?flag=` → empty string.
    var v = eq < 0 ? null : decodeURIComponent(pairs[i].slice(eq + 1));
    if (Object.prototype.hasOwnProperty.call(query, k)) {
      var cur = query[k];
      query[k] = Array.isArray(cur) ? cur.concat([v]) : [cur, v];
    } else {
      query[k] = v;
    }
  }
  return query;
}

function splitHash(raw) {
  var text = String(raw || '');
  var hIdx = text.indexOf('#');
  if (hIdx < 0) return { base: text, hash: '' };
  return { base: text.slice(0, hIdx), hash: text.slice(hIdx) };
}

function normalizeHash(hash) {
  if (hash == null || hash === '') return '';
  var text = String(hash);
  return text.charAt(0) === '#' ? text : '#' + text;
}

function normalizeTo(to) {
  if (to == null) {
    return { path: '/', name: undefined, query: {}, params: {}, hash: '', fullPath: '/' };
  }
  if (typeof to === 'string') {
    var hashParts = splitHash(to);
    var hash = hashParts.hash;
    var withoutHash = hashParts.base;
    var qIdx = withoutHash.indexOf('?');
    if (qIdx >= 0) {
      var pathOnly = withoutHash.slice(0, qIdx) || '/';
      var queryFromString = parseQueryString(withoutHash.slice(qIdx));
      return {
        path: pathOnly,
        name: undefined,
        query: queryFromString,
        params: {},
        hash: hash,
        fullPath: pathOnly + stringifyQuery(queryFromString) + hash,
      };
    }
    var pathNoQuery = withoutHash || '/';
    return {
      path: pathNoQuery,
      name: undefined,
      query: {},
      params: {},
      hash: hash,
      fullPath: pathNoQuery + hash,
    };
  }
  var path = to.path;
  if (!path && to.name) path = '/' + String(to.name);
  if (!path) path = '/';
  var hash = normalizeHash(to.hash);
  var pathHash = splitHash(path);
  if (pathHash.hash) {
    if (!to.hash) hash = pathHash.hash;
    path = pathHash.base || '/';
  }
  // Strip any inline query from path; prefer explicit query object.
  var pathQ = path.indexOf('?');
  var query = to.query || {};
  if (pathQ >= 0) {
    if (!to.query) query = parseQueryString(path.slice(pathQ));
    path = path.slice(0, pathQ) || '/';
  }
  var fullPath =
    typeof to.fullPath === 'string' && to.fullPath
      ? to.fullPath
      : path + stringifyQuery(query) + hash;
  return {
    path: path,
    name: to.name,
    query: query,
    params: to.params || {},
    hash: hash,
    fullPath: fullPath,
    meta: to.meta || {},
    matched: to.matched || [],
  };
}

function applyRoute(route, next) {
  route.path = next.path;
  route.fullPath = next.fullPath || next.path;
  route.name = next.name;
  route.query = next.query || {};
  route.params = next.params || {};
  route.hash = next.hash || '';
  route.meta = next.meta || {};
  route.matched = next.matched || route.matched || [];
}

export function createFeStubRouter(overrides) {
  overrides = overrides || {};
  var route = reactive(
    Object.assign(
      {
        path: '/',
        fullPath: '/',
        query: {},
        params: {},
        hash: '',
        name: undefined,
        meta: {},
        matched: [],
        redirectedFrom: undefined,
      },
      overrides.route || {}
    )
  );
  var afterHooks = [];
  var beforeHooks = [];
  var errorHandlers = [];
  var router = Object.assign(
    {
      currentRoute: { value: route },
      push: function (to) {
        var fromSnapshot = {
          path: route.path,
          fullPath: route.fullPath,
          name: route.name,
          query: route.query,
          params: route.params,
          hash: route.hash,
          meta: route.meta,
          matched: route.matched,
        };
        var next = normalizeTo(to);
        // Mirror vue-router: navigating to the current location is duplicated —
        // beforeEach does not run, afterEach receives NavigationFailureType.duplicated.
        if (String(next.fullPath || next.path) === String(route.fullPath || route.path)) {
          var dupFailure = Object.assign(new Error('Avoided redundant navigation'), {
            type: 16,
            from: fromSnapshot,
            to: next,
          });
          var dupChain = Promise.resolve();
          afterHooks.forEach(function (fn) {
            dupChain = dupChain.then(function () {
              return fn(route, fromSnapshot, dupFailure);
            });
          });
          return dupChain.then(function () {
            return dupFailure;
          });
        }
        applyRoute(route, next);
        router.currentRoute.value = route;
        var chain = Promise.resolve();
        var aborted = false;
        var failure = undefined;
        var failedTo = undefined;
        beforeHooks.forEach(function (fn) {
          chain = chain.then(function () {
            if (aborted) return;
            return Promise.resolve(fn(route, fromSnapshot)).then(function (result) {
              if (result === false) {
                aborted = true;
                // Shape matches vue-router NavigationFailure enough for isNavigationFailure.
                failure = Object.assign(new Error('Navigation aborted'), {
                  type: 4,
                  from: fromSnapshot,
                  to: null,
                });
                // Keep the intended destination (incl. guard-mutated meta) for afterEach,
                // matching vue-router's failure semantics.
                failedTo = {
                  path: route.path,
                  fullPath: route.fullPath,
                  name: route.name,
                  query: route.query,
                  params: route.params,
                  meta: Object.assign({}, route.meta),
                  matched: route.matched,
                };
                failure.to = failedTo;
                applyRoute(route, fromSnapshot);
                router.currentRoute.value = route;
              }
            });
          });
        });
        afterHooks.forEach(function (fn) {
          chain = chain.then(function () {
            return fn(failure ? failedTo : route, fromSnapshot, failure);
          });
        });
        // Real vue-router resolves undefined on success, NavigationFailure on abort.
        return chain.then(function () {
          return failure;
        });
      },
      replace: function (to) {
        return router.push(to);
      },
      go: function () {},
      back: function () {},
      forward: function () {},
      isReady: function () {
        return Promise.resolve();
      },
      resolve: function (to) {
        var n = normalizeTo(to);
        return {
          name: n.name,
          path: n.path,
          fullPath: n.fullPath,
          href: n.fullPath,
          matched: n.matched && n.matched.length ? n.matched : [{ path: n.path }],
          query: n.query,
          params: n.params,
          meta: n.meta || {},
        };
      },
      beforeEach: function (fn) {
        if (typeof fn !== 'function') return function () {};
        beforeHooks.push(fn);
        return function () {
          var i = beforeHooks.indexOf(fn);
          if (i >= 0) beforeHooks.splice(i, 1);
        };
      },
      afterEach: function (fn) {
        if (typeof fn !== 'function') return function () {};
        afterHooks.push(fn);
        return function () {
          var i = afterHooks.indexOf(fn);
          if (i >= 0) afterHooks.splice(i, 1);
        };
      },
      onError: function (fn) {
        if (typeof fn !== 'function') return function () {};
        errorHandlers.push(fn);
        return function () {
          var i = errorHandlers.indexOf(fn);
          if (i >= 0) errorHandlers.splice(i, 1);
        };
      },
      /** Test helper: fire registered onError listeners (optional failed `to`). */
      triggerError: function (err, to) {
        errorHandlers.forEach(function (fn) {
          fn(err, to);
        });
      },
      install: function (app) {
        app.config.globalProperties.$router = router;
        app.config.globalProperties.$route = route;
        app.provide(ROUTER_KEY, router);
        app.provide(ROUTE_KEY, route);
        app.provide('router', router);
        app.provide('route', route);
      },
    },
    overrides.router || {}
  );
  // Keep hook-aware push when overrides only customize resolve/lookup helpers.
  if (overrides.router && typeof overrides.router.push === 'function') {
    var overridePush = overrides.router.push;
    router.push = function (to) {
      var self = this;
      var fromSnapshot = {
        path: route.path,
        fullPath: route.fullPath,
        name: route.name,
        query: route.query,
        params: route.params,
        hash: route.hash,
        meta: route.meta,
        matched: route.matched,
      };
      return Promise.resolve(overridePush.call(self, to)).then(function (result) {
        // createRouter's override applies the destination before hooks. Same-path
        // navigations are duplicated: skip beforeEach and report NAVIGATION_DUPLICATED.
        var samePlace =
          String(route.fullPath || route.path) === String(fromSnapshot.fullPath || fromSnapshot.path);
        if (samePlace) {
          var dupFailure = Object.assign(new Error('Avoided redundant navigation'), {
            type: 16,
            from: fromSnapshot,
            to: {
              path: route.path,
              fullPath: route.fullPath,
              name: route.name,
              query: route.query,
              params: route.params,
              meta: route.meta,
              matched: route.matched,
            },
          });
          var dupChain = Promise.resolve();
          afterHooks.forEach(function (fn) {
            dupChain = dupChain.then(function () {
              return fn(route, fromSnapshot, dupFailure);
            });
          });
          return dupChain.then(function () {
            return dupFailure;
          });
        }
        var chain = Promise.resolve();
        var aborted = false;
        var failure = undefined;
        var failedTo = undefined;
        beforeHooks.forEach(function (fn) {
          chain = chain.then(function () {
            if (aborted) return;
            return Promise.resolve(fn(route, fromSnapshot)).then(function (guardResult) {
              if (guardResult === false) {
                aborted = true;
                failure = Object.assign(new Error('Navigation aborted'), {
                  type: 4,
                  from: fromSnapshot,
                  to: null,
                });
                failedTo = {
                  path: route.path,
                  fullPath: route.fullPath,
                  name: route.name,
                  query: route.query,
                  params: route.params,
                  meta: Object.assign({}, route.meta),
                  matched: route.matched,
                };
                applyRoute(route, fromSnapshot);
                router.currentRoute.value = route;
              }
            });
          });
        });
        afterHooks.forEach(function (fn) {
          chain = chain.then(function () {
            return fn(failure ? failedTo : route, fromSnapshot, failure);
          });
        });
        return chain.then(function () {
          return result;
        });
      });
    };
  }
  router.currentRoute.value = route;
  return { router: router, route: route };
}

export function useRouter() {
  var r = inject(ROUTER_KEY, null) || inject('router', null);
  if (!r) {
    throw new Error('fe stub vue-router: useRouter() missing plugin (pass createFeStubRouter().router)');
  }
  return r;
}

export function useRoute() {
  var r = inject(ROUTE_KEY, null) || inject('route', null);
  if (!r) {
    var router = inject(ROUTER_KEY, null) || inject('router', null);
    if (router && router.currentRoute) return router.currentRoute.value;
    throw new Error('fe stub vue-router: useRoute() missing plugin');
  }
  return r;
}

export const RouterLink = defineComponent({
  name: 'RouterLink',
  props: { to: { type: [String, Object], default: '/' } },
  setup: function (props, ctx) {
    return function () {
      return h(
        'a',
        { href: typeof props.to === 'string' ? props.to : '#', class: 'fe-stub-router-link' },
        ctx.slots.default ? ctx.slots.default() : []
      );
    };
  },
});

export const RouterView = defineComponent({
  name: 'RouterView',
  setup: function (_p, ctx) {
    return function () {
      return h('div', { class: 'fe-stub-router-view' }, ctx.slots.default ? ctx.slots.default() : []);
    };
  },
});

export function createRouter(options) {
  options = options || {};
  var routes = options.routes || [];
  var byName = Object.create(null);
  var byPath = Object.create(null);
  for (var i = 0; i < routes.length; i++) {
    var r = routes[i] || {};
    if (r.name != null) byName[r.name] = r;
    if (r.path != null) byPath[r.path] = r;
  }

  function lookup(to) {
    var n = normalizeTo(to);
    var found = null;
    if (n.name != null && byName[n.name]) found = byName[n.name];
    else if (n.path && byPath[n.path]) found = byPath[n.path];
    if (!found) {
      return {
        path: n.path,
        name: n.name,
        query: n.query,
        params: n.params,
        hash: n.hash || '',
        fullPath: n.fullPath,
        meta: n.meta || {},
        matched: [],
      };
    }
    var resolvedPath = found.path || n.path;
    var hash = n.hash || '';
    return {
      path: resolvedPath,
      name: found.name != null ? found.name : n.name,
      query: n.query,
      params: n.params,
      hash: hash,
      // Keep query+hash in fullPath so query/hash-only changes are not duplicates.
      fullPath: resolvedPath + stringifyQuery(n.query) + hash,
      meta: found.meta || {},
      matched: [{ path: resolvedPath, name: found.name, meta: found.meta || {} }],
    };
  }

  return createFeStubRouter({
    router: {
      push: function (to) {
        var next = lookup(to);
        applyRoute(this.currentRoute.value, next);
        return Promise.resolve(undefined);
      },
      replace: function (to) {
        return this.push(to);
      },
      resolve: function (to) {
        var next = lookup(to);
        return {
          name: next.name,
          path: next.path,
          fullPath: next.fullPath,
          href: next.fullPath,
          matched: next.matched,
          query: next.query,
          params: next.params,
          meta: next.meta || {},
        };
      },
    },
  }).router;
}

export function createWebHistory() {
  return {};
}

export function createMemoryHistory() {
  return {};
}

/**
 * FE-stub stand-in for vue-router isNavigationFailure.
 * True for Error failures (incl. stub aborted pushes) with a numeric type.
 */
export function isNavigationFailure(error, type) {
  if (!(error instanceof Error) || typeof error.type !== 'number') return false;
  if (type == null) return true;
  return !!(error.type & type);
}

export var NavigationFailureType = {
  aborted: 4,
  cancelled: 8,
  duplicated: 16,
};

export default {
  useRouter: useRouter,
  useRoute: useRoute,
  RouterLink: RouterLink,
  RouterView: RouterView,
  createRouter: createRouter,
  createMemoryHistory: createMemoryHistory,
  createWebHistory: createWebHistory,
  createFeStubRouter: createFeStubRouter,
  isNavigationFailure: isNavigationFailure,
  NavigationFailureType: NavigationFailureType,
};
