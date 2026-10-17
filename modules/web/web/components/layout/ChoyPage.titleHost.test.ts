// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, onMounted } from 'vue';
import * as VueRouter from 'vue-router';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyPage from './ChoyPage.vue';
import {
  useHostPageTitleInView,
  usePageContext,
} from '@/web/web/composables/usePageContext';

const createFeStubRouter = (VueRouter as any).createFeStubRouter;

const fakeStore = {
  modelName: 'auth.User',
  meta: { fields: {} },
} as any;

function mountWithRouter(Host: any, route?: Record<string, unknown>) {
  const { router } = createFeStubRouter({
    route: {
      name: 'page',
      path: '/',
      fullPath: '/',
      params: {},
      query: {},
      meta: {},
      ...(route || {}),
    },
  });
  return mountApp(Host, { plugins: [router] });
}

describe('ChoyPage title hosting', () => {
  test('keeps custom title-actions when a view hosts the title (without duplicating IO)', async () => {
    const Claimer = defineComponent({
      setup() {
        const ctx = usePageContext();
        onMounted(() => {
          ctx?.claimPageTitle();
        });
        return () => h('div', { 'data-test': 'claimer' });
      },
    });

    const Host = defineComponent({
      setup() {
        return () =>
          h(
            ChoyPage as any,
            {
              store: fakeStore,
              title: 'Company list',
              actionImport: true,
              actionExport: true,
            },
            {
              default: () => h(Claimer),
              'title-actions': () => h('button', { 'data-test': 'custom-title-action' }, 'Custom'),
            },
          );
      },
    });

    const wrapper = mountWithRouter(Host);
    await flushPromises();
    // Title text is hosted away from the page band.
    expect(wrapper.q('[data-slot=page-title]')).toBeNull();
    // Custom title-actions remain visible.
    expect(wrapper.q('[data-test=custom-title-action]')).not.toBeNull();
    // IO menu should not duplicate in the page header while hosted.
    expect(wrapper.q('[data-testid=page-io-menu-trigger]')).toBeNull();
    wrapper.unmount();
  });

  test('hosted title without title-actions hides the page title-actions band', async () => {
    const Claimer = defineComponent({
      setup() {
        const ctx = usePageContext();
        onMounted(() => {
          ctx?.claimPageTitle();
        });
        return () => h('div', { 'data-test': 'claimer' });
      },
    });

    const Host = defineComponent({
      setup() {
        return () =>
          h(
            ChoyPage as any,
            {
              store: fakeStore,
              title: 'Sessions',
              actionImport: true,
              actionExport: true,
              padding: true,
            },
            {
              default: () => h(Claimer),
            },
          );
      },
    });

    const wrapper = mountWithRouter(Host);
    await flushPromises();
    expect(wrapper.q('[data-slot=page-title]')).toBeNull();
    expect(wrapper.q('[data-testid=page-io-menu-trigger]')).toBeNull();
    // No custom title-actions while hosted → showPageTitleActions is false.
    expect(wrapper.text()).not.toContain('Import');
    const root = wrapper.q('[data-anchor="choy.page"]') as HTMLElement | null;
    expect(root?.className || '').toContain('choy-inset-pad-no-top');
    wrapper.unmount();
  });

  test('unhosted title keeps page title band, IO menu, and top padding', async () => {
    const Host = defineComponent({
      setup() {
        return () =>
          h(ChoyPage as any, {
            store: fakeStore,
            title: 'Company list',
            actionImport: true,
            actionExport: false,
            padding: true,
            width: 'medium',
          });
      },
    });
    const wrapper = mountWithRouter(Host);
    await flushPromises();
    expect(wrapper.q('[data-slot=page-title]')?.textContent?.trim()).toBe('Company list');
    expect(wrapper.q('[data-testid=page-io-menu-trigger]')).not.toBeNull();
    const root = wrapper.q('[data-anchor="choy.page"]') as HTMLElement | null;
    expect(root?.className || '').toContain('choy-inset-pad');
    expect(root?.className || '').not.toContain('choy-inset-pad-no-top');
    expect(root?.className || '').toContain('max-w-4xl');
    wrapper.unmount();
  });

  test('useHostPageTitleInView reads titleActionsIo from the page context', async () => {
    let hostedIo: { actionImport?: boolean; actionExport?: boolean } | null | undefined;
    const HostChild = defineComponent({
      setup() {
        const host = useHostPageTitleInView(true);
        hostedIo = host.pageTitleActionsIo.value ?? null;
        return () => h('div', { 'data-test': 'host-child' });
      },
    });
    const Host = defineComponent({
      setup() {
        return () =>
          h(
            ChoyPage as any,
            {
              store: fakeStore,
              title: 'Tokens',
              actionImport: true,
              actionExport: true,
              actionImportUploadHint: 'csv',
              width: 'narrow',
            },
            { default: () => h(HostChild) },
          );
      },
    });
    const wrapper = mountWithRouter(Host);
    await flushPromises();
    expect(hostedIo?.actionImport).toBe(true);
    expect(hostedIo?.actionExport).toBe(true);
    expect(wrapper.q('[data-slot=page-title]')).toBeNull();
    wrapper.unmount();
  });

  test('workspace pages without title prop inherit route meta pageTitle', async () => {
    const Host = defineComponent({
      setup() {
        return () =>
          h(ChoyPage as any, {
            store: fakeStore,
            padding: true,
          });
      },
    });
    const wrapper = mountWithRouter(Host, {
      name: 'CompanyDetail',
      path: '/base/companies/1',
      fullPath: '/base/companies/1',
      params: { id: '1' },
      meta: { pageTitle: 'Company Details' },
    });
    await flushPromises();
    expect(wrapper.q('[data-slot=page-title]')?.textContent?.trim()).toBe('Company Details');
    wrapper.unmount();
  });

  test('width variants apply max-width utility classes', async () => {
    for (const [width, cls] of [
      ['wide', 'max-w-6xl'],
      ['full', 'max-w-none'],
    ] as const) {
      const Host = defineComponent({
        setup() {
          return () =>
            h(ChoyPage as any, {
              store: fakeStore,
              title: 'W',
              width,
              padding: false,
            });
        },
      });
      const wrapper = mountWithRouter(Host);
      await flushPromises();
      const root = wrapper.q('[data-anchor="choy.page"]') as HTMLElement | null;
      expect(root?.className || '').toContain(cls);
      wrapper.unmount();
    }
  });
});
