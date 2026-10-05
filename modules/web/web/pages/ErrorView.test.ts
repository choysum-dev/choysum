// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import * as VueRouter from 'vue-router';
const createFeStubRouter = (VueRouter as any).createFeStubRouter;
import { createPinia, setActivePinia } from 'pinia';
import { resetInstalledMenu } from '@/core/web/menu';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import { MODULE_BOARD_PATH } from '../router/resolveDefaultLandPath';
import ErrorView from './ErrorView.vue';
import AuthPanel from '@/auth/web/components/AuthPanel.vue';
import ChoyPage from '@/web/web/components/layout/ChoyPage.vue';
import ChoyCard from '@/web/web/components/layout/ChoyCard.vue';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';

describe('ErrorView', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    // Drop a leaked getInstalledMenu singleton from earlier files in this FE realm.
    resetInstalledMenu();
    stubSfc(ChoyPage as any, {
      props: { title: String, padding: Boolean, width: String, class: null },
      setup: ((_props: any, { slots }: any) => {
        return () => h('div', { 'data-test': 'page' }, slots.default?.());
      }) as any,
    } as any);
    stubSfc(AuthPanel as any, {
      setup: ((_props: any, { slots }: any) => {
        return () => h('div', { 'data-test': 'auth-panel' }, slots.default?.());
      }) as any,
    } as any);
    stubSfc(ChoyCard as any, {
      props: { title: String },
      setup: ((props: any, { slots }: any) => {
        return () =>
          h('section', { 'data-test': 'card', 'data-title': props.title || '' }, [
            slots.header?.(),
            slots.default?.(),
          ]);
      }) as any,
    } as any);
    stubSfc(ChoyButton as any, {
      props: { type: String, variant: String },
      emits: ['click'],
      setup: ((_props: any, { slots, attrs, emit }: any) => {
        return () =>
          h(
            'button',
            {
              'data-test': 'action',
              type: 'button',
              ...attrs,
              onClick: (e: MouseEvent) => emit('click', e),
            },
            slots.default?.(),
          );
      }) as any,
    } as any);
  });

  afterEach(() => {
    restoreSfc(ChoyPage);
    restoreSfc(AuthPanel);
    restoreSfc(ChoyCard);
    restoreSfc(ChoyButton);
  });

  function mountError(route: Record<string, unknown>) {
    const stub = createFeStubRouter({ route });
    return mountApp(ErrorView as any, {
      plugins: [stub.router],
    });
  }

  test('resolves 403 from static /error/403 path', async () => {
    const mounted = mountError({
      path: '/error/403',
      fullPath: '/error/403?reason=role&from=/meta/modules',
      params: {},
      query: { reason: 'role', from: '/meta/modules', message: 'need admin' },
    });
    await flushPromises();
    expect(mounted.text()).toMatch(/denied|Access/i);
    expect(mounted.text()).toContain('need admin');
    expect(mounted.qa('[data-test=action]').length).toBeGreaterThanOrEqual(2);
    mounted.unmount();
  });

  test('resolves 500 from static path and 404 by default', async () => {
    const five = mountError({
      path: '/error/500',
      fullPath: '/error/500',
      params: {},
      query: {},
    });
    await flushPromises();
    expect(five.text()).toMatch(/Server|error/i);
    five.unmount();

    const missing = mountError({
      path: '/error/unknown',
      fullPath: '/error/unknown',
      params: {},
      query: {},
    });
    await flushPromises();
    expect(missing.text()).toMatch(/not found|Page/i);
    missing.unmount();
  });

  test('prefers route.params.code over path segment', async () => {
    const mounted = mountError({
      path: '/error/404',
      fullPath: '/error/404',
      params: { code: '500' },
      query: {},
    });
    await flushPromises();
    expect(mounted.text()).toMatch(/Server|error/i);
    mounted.unmount();
  });

  test('prefers static path segment over query code', async () => {
    const mounted = mountError({
      path: '/error/403',
      fullPath: '/error/403?code=500',
      params: {},
      query: { code: '500' },
    });
    await flushPromises();
    expect(mounted.text()).toMatch(/denied|Access/i);
    mounted.unmount();
  });

  test('normalizes array params and query code to the first entry', async () => {
    const fromParams = mountError({
      path: '/error',
      fullPath: '/error',
      params: { code: ['500', '400'] },
      query: {},
    });
    await flushPromises();
    expect(fromParams.text()).toMatch(/Server|error/i);
    fromParams.unmount();

    const fromQuery = mountError({
      path: '/oops',
      fullPath: '/oops?code=403&code=404',
      params: {},
      query: { code: ['403', '404'] },
    });
    await flushPromises();
    expect(fromQuery.text()).toMatch(/denied|Access/i);
    fromQuery.unmount();
  });

  test('normalizes repeated reason/message/from query values', async () => {
    const stub = createFeStubRouter({
      route: {
        path: '/error/403',
        fullPath: '/error/403?reason=role&reason=permission&from=/a&from=/b&message=one&message=two',
        params: {},
        query: {
          reason: ['role', 'permission'],
          from: ['/a', '/b'],
          message: ['one', 'two'],
        },
      },
    });
    const push = fnRecorder((to: unknown) => Promise.resolve(to));
    stub.router.push = push;

    const mounted = mountApp(ErrorView as any, { plugins: [stub.router] });
    await flushPromises();
    expect(mounted.text()).toMatch(/missing the required role/i);
    expect(mounted.text()).toContain('one');
    const actions = mounted.qa('[data-test=action]');
    (actions[1] as HTMLElement).click();
    expect(push.calls.map(c => c[0])).toEqual(['/a']);
    mounted.unmount();
  });

  test('omits Go back when from is not a same-app absolute path', async () => {
    for (const from of [
      '//evil.example',
      '/\\evil.example',
      '/\tevil.example',
      '/foo\\bar',
      '/%5cevil.example',
      '/%2f%2fhost',
      '/%09evil.example',
      '/error/500',
      '/error',
    ]) {
      const mounted = mountError({
        path: '/error/403',
        fullPath: `/error/403?from=${encodeURIComponent(from)}`,
        params: {},
        query: { from },
      });
      await flushPromises();
      const actions = mounted.qa('[data-test=action]');
      // Home + Contact only; unsafe or error from must not add Go back.
      expect(actions.length).toBe(2);
      expect(mounted.text()).not.toMatch(/Go back/i);
      mounted.unmount();
    }
  });




  test('invokes navigation and window helpers from action buttons', async () => {
    const open = fnRecorder(() => null);
    const reload = fnRecorder(() => undefined);
    const prevOpen = window.open;
    const prevLoc = window.location;
    (window as any).open = open;
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { ...prevLoc, reload },
    });

    try {
      const deniedStub = createFeStubRouter({
        route: {
          path: '/error/403',
          fullPath: '/error/403?reason=permission&from=/auth/users',
          params: {},
          query: { reason: 'permission', from: '/auth/users' },
        },
      });
      const deniedPush = fnRecorder((to: unknown) => Promise.resolve(to));
      deniedStub.router.push = deniedPush;

      const denied = mountApp(ErrorView as any, { plugins: [deniedStub.router] });
      await flushPromises();
      expect(denied.text()).toMatch(/missing the required permission/i);
      const deniedActions = denied.qa('[data-test=action]');
      expect(deniedActions.length).toBe(3);
      denied.click('[data-test=action]');
      (deniedActions[1] as HTMLElement).click();
      (deniedActions[2] as HTMLElement).click();
      expect(deniedPush.calls.map(c => c[0])).toEqual([MODULE_BOARD_PATH, '/auth/users']);
      expect(open.calls[0]?.[0]).toBe('mailto:admin@example.com');
      expect(open.calls[0]?.[2]).toBe('noopener,noreferrer');
      denied.unmount();

      const fiveStub = createFeStubRouter({
        route: { path: '/error/500', fullPath: '/error/500', params: {}, query: {} },
      });
      const five = mountApp(ErrorView as any, { plugins: [fiveStub.router] });
      await flushPromises();
      const fiveActions = five.qa('[data-test=action]');
      (fiveActions[1] as HTMLElement).click();
      (fiveActions[2] as HTMLElement).click();
      expect(reload.calls.length).toBe(1);
      const supportCall = open.calls.find(c => String(c[0]).includes('example.com/support'));
      expect(supportCall?.[2]).toBe('noopener,noreferrer');
      five.unmount();

      const missingStub = createFeStubRouter({
        route: { path: '/missing', fullPath: '/missing', params: {}, query: {} },
      });
      const back = fnRecorder(() => undefined);
      missingStub.router.back = back;
      const missing = mountApp(ErrorView as any, { plugins: [missingStub.router] });
      await flushPromises();
      const missingActions = missing.qa('[data-test=action]');
      (missingActions[1] as HTMLElement).click();
      expect(back.calls.length).toBe(1);
      missing.unmount();
    } finally {
      (window as any).open = prevOpen;
      Object.defineProperty(window, 'location', { configurable: true, value: prevLoc });
    }
  });
});
