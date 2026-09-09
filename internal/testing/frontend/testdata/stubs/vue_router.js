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

function normalizeTo(to) {
  if (to == null) return { path: '/', name: undefined, query: {}, params: {}, fullPath: '/' };
  if (typeof to === 'string') {
    return { path: to, name: undefined, query: {}, params: {}, fullPath: to };
  }
  var path = to.path;
  if (!path && to.name) path = '/' + String(to.name);
  if (!path) path = '/';
  return {
    path: path,
    name: to.name,
    query: to.query || {},
    params: to.params || {},
    fullPath: path,
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
        name: undefined,
        meta: {},
        matched: [],
        redirectedFrom: undefined,
      },
      overrides.route || {}
    )
  );
  var router = Object.assign(
    {
      currentRoute: { value: route },
      push: function (to) {
        applyRoute(route, normalizeTo(to));
        router.currentRoute.value = route;
        return Promise.resolve(to);
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
      beforeEach: function () {
        return function () {};
      },
      afterEach: function () {
        return function () {};
      },
      onError: function () {
        return function () {};
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
        fullPath: n.fullPath,
        meta: n.meta || {},
        matched: [],
      };
    }
    return {
      path: found.path || n.path,
      name: found.name != null ? found.name : n.name,
      query: n.query,
      params: n.params,
      fullPath: found.path || n.fullPath,
      meta: found.meta || {},
      matched: [{ path: found.path || n.path, name: found.name, meta: found.meta || {} }],
    };
  }

  return createFeStubRouter({
    router: {
      push: function (to) {
        var next = lookup(to);
        applyRoute(this.currentRoute.value, next);
        return Promise.resolve(to);
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

export default {
  useRouter: useRouter,
  useRoute: useRoute,
  RouterLink: RouterLink,
  RouterView: RouterView,
  createRouter: createRouter,
  createMemoryHistory: createMemoryHistory,
  createWebHistory: createWebHistory,
  createFeStubRouter: createFeStubRouter,
};
