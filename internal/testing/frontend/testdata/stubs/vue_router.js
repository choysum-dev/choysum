// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/**
 * FE unit stub for vue-router injection used by product pages under QJS.
 * Real vue-router is not required when this module is aliased.
 */
import { inject, defineComponent, h } from 'vue';

var ROUTER_KEY = 'choysumFeStubRouter';
var ROUTE_KEY = 'choysumFeStubRoute';

export function createFeStubRouter(overrides) {
  overrides = overrides || {};
  var route = Object.assign(
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
    overrides.route || {},
  );
  var router = Object.assign(
    {
      currentRoute: { value: route },
      push: function (to) {
        return Promise.resolve(to);
      },
      replace: function (to) {
        return Promise.resolve(to);
      },
      go: function () {},
      back: function () {},
      forward: function () {},
      beforeEach: function () {
        return function () {};
      },
      afterEach: function () {
        return function () {};
      },
      install: function (app) {
        app.config.globalProperties.$router = router;
        app.config.globalProperties.$route = route;
        app.provide(ROUTER_KEY, router);
        app.provide(ROUTE_KEY, route);
        // vue-router symbols are not public; also mirror common string keys.
        app.provide('router', router);
        app.provide('route', route);
      },
    },
    overrides.router || {},
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
      return h('a', { href: typeof props.to === 'string' ? props.to : '#', class: 'fe-stub-router-link' }, ctx.slots.default ? ctx.slots.default() : []);
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

export function createRouter() {
  return createFeStubRouter().router;
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
  createFeStubRouter: createFeStubRouter,
};
