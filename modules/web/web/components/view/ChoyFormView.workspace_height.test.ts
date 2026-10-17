// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Workspace height: form body scrolls inside the shell; ActionBar matches
 * List/Kanban chrome (thirds, no bottom border, title in the bar).
 */
import { computed, defineComponent, h, shallowRef } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import * as VueRouter from 'vue-router';

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyPage from '@/web/web/components/layout/ChoyPage.vue';
import { PageContextKey, type PageActionTarget } from '@/web/web/composables/usePageContext';
import { createLocalFormStore } from '@/web/web/stores/localFormStore';
import ChoyFormView from './ChoyFormView.vue';

const createFeStubRouter = (VueRouter as any).createFeStubRouter;

function loginStore() {
  return createLocalFormStore({
    fields: [{ name: 'Username', label: 'Username', type: 'varchar' }],
    initialValues: { Username: 'admin' },
  });
}

function mountForm(props: Record<string, unknown> = {}, extra: Record<string, unknown> = {}) {
  const { router } = createFeStubRouter({
    route: { name: 'login', path: '/web/login', fullPath: '/web/login', params: {}, query: {}, meta: {} },
  });
  return mountApp(ChoyFormView as any, {
    props: {
      store: loginStore(),
      viewMode: 'display',
      resolveRecordIdFromRoute: false,
      showMessages: false,
      ...props,
    },
    plugins: [createPinia(), router],
    ...extra,
  });
}

/** Action-tray chrome order for the minimal DOM (no compound querySelector). */
function actionTrayOrder(tray: Element): string[] {
  const out: string[] = [];
  const walk = (node: Element) => {
    const testId = node.getAttribute('data-testid');
    if (testId === 'form-primary-new' || testId === 'form-secondary-edit' || testId === 'form-secondary-reset') {
      out.push(testId);
      return;
    }
    if (node.getAttribute('data-slot') === 'page-title') {
      out.push('page-title');
      return;
    }
    if (node.tagName === 'BUTTON' || node.tagName === 'button') {
      const label = (node.textContent || '').replace(/\s+/g, ' ').trim();
      if (label) out.push(label);
      return;
    }
    for (const child of Array.from(node.children)) {
      walk(child as Element);
    }
  };
  walk(tray);
  return out;
}

describe('ChoyFormView workspace height', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  test('body scrolls inside the shell and the container uses gap-4', async () => {
    const mounted = mountForm({ showHeader: false, showActions: false });
    await flushPromises();

    const root = mounted.q('[data-slot="view-container"]') as HTMLElement | null;
    expect(root).not.toBeNull();
    expect(root!.className).toContain('gap-4');

    const shell = mounted.q('[data-slot="view-body"]') as HTMLElement | null;
    expect(shell).not.toBeNull();
    expect(shell!.className).toContain('overflow-hidden');

    const body = mounted.q('[data-anchor="choy.form-view"]') as HTMLElement | null;
    expect(body).not.toBeNull();
    expect(body!.className).toContain('min-h-0');
    expect(body!.className).toContain('flex-1');
    expect(body!.className).toContain('overflow-y-auto');
    expect(body!.className.split(/\s+/)).not.toContain('pt-2');

    mounted.unmount();
  });

  test('ActionBar drops the bottom border like ListView', async () => {
    const mounted = mountForm({ showHeader: true, showActions: true, recordId: 'rec-1' });
    await flushPromises();

    const bar = mounted.q('[data-slot="view-action-bar"]') as HTMLElement | null;
    expect(bar).not.toBeNull();
    expect(bar!.getAttribute('data-layout')).toBe('thirds');
    const cls = bar!.className || '';
    expect(cls).toContain('border-b-0');
    expect(cls).toContain('[border-bottom-width:0px]');
    expect(mounted.q('[data-testid="page-io-menu-trigger"]')).not.toBeNull();

    mounted.unmount();
  });

  test('display left chrome is primary → title → secondary', async () => {
    const store = loginStore();
    const { router } = createFeStubRouter({
      route: { name: 'login', path: '/web/login', fullPath: '/web/login', params: {}, query: {}, meta: {} },
    });
    const Host = defineComponent({
      setup() {
        return () =>
          h(
            ChoyPage as any,
            { store, title: 'Token Details', padding: true },
            {
              default: () =>
                h(ChoyFormView as any, {
                  store,
                  viewMode: 'display',
                  recordId: 'tok-1',
                  resolveRecordIdFromRoute: false,
                  showHeader: true,
                  showActions: true,
                  showMessages: false,
                  createAction: '/tokens/new',
                }),
            },
          );
      },
    });
    const mounted = mountApp(Host, { plugins: [createPinia(), router] });
    await flushPromises();

    const tray = mounted.q('[data-testid=choy-action-tray]') as HTMLElement | null;
    expect(tray).not.toBeNull();
    const order = actionTrayOrder(tray!);
    const newIdx = order.indexOf('form-primary-new');
    const titleIdx = order.indexOf('page-title');
    const editIdx = order.indexOf('form-secondary-edit');
    expect(newIdx).toBeGreaterThanOrEqual(0);
    expect(titleIdx).toBeGreaterThan(newIdx);
    expect(editIdx).toBeGreaterThan(titleIdx);

    mounted.unmount();
  });

  test('create left chrome is Save/Cancel → title → ⋮ (Reset in overflow)', async () => {
    const store = loginStore();
    const { router } = createFeStubRouter({
      route: { name: 'login', path: '/web/login', fullPath: '/web/login', params: {}, query: {}, meta: {} },
    });
    const Host = defineComponent({
      setup() {
        return () =>
          h(
            ChoyPage as any,
            { store, title: 'New Token', padding: true },
            {
              default: () =>
                h(ChoyFormView as any, {
                  store,
                  viewMode: 'create',
                  resolveRecordIdFromRoute: false,
                  showHeader: true,
                  showActions: true,
                  showMessages: false,
                }),
            },
          );
      },
    });
    const mounted = mountApp(Host, { plugins: [createPinia(), router] });
    await flushPromises();

    const tray = mounted.q('[data-testid=choy-action-tray]') as HTMLElement | null;
    expect(tray).not.toBeNull();
    const order = actionTrayOrder(tray!);
    const saveIdx = order.findIndex(t => t.includes('Save') || t.includes('保存'));
    const cancelIdx = order.findIndex(t => t.includes('Cancel') || t.includes('取消'));
    const titleIdx = order.indexOf('page-title');
    expect(saveIdx).toBeGreaterThanOrEqual(0);
    expect(cancelIdx).toBeGreaterThan(saveIdx);
    expect(titleIdx).toBeGreaterThan(cancelIdx);
    expect(order.includes('form-secondary-reset')).toBe(false);
    expect(mounted.q('[data-testid=page-io-menu-trigger]')).not.toBeNull();

    mounted.unmount();
  });

  test('hosts page title in ActionBar when mounted under ChoyPage', async () => {
    const store = loginStore();
    const { router } = createFeStubRouter({
      route: { name: 'login', path: '/web/login', fullPath: '/web/login', params: {}, query: {}, meta: {} },
    });
    const Host = defineComponent({
      setup() {
        return () =>
          h(
            ChoyPage as any,
            { store, title: 'Partner', padding: true },
            {
              default: () =>
                h(ChoyFormView as any, {
                  store,
                  viewMode: 'display',
                  resolveRecordIdFromRoute: false,
                  showHeader: true,
                  showActions: true,
                  showMessages: false,
                }),
            },
          );
      },
    });

    const mounted = mountApp(Host, { plugins: [createPinia(), router] });
    await flushPromises();

    const pageRoot = mounted.q('[data-anchor="choy.page"]') as HTMLElement | null;
    expect(pageRoot?.className || '').toContain('choy-inset-pad-no-top');

    const titles = mounted.qa('[data-slot=page-title]');
    expect(titles.length).toBe(1);
    expect(titles[0]?.textContent?.trim()).toBe('Partner');
    expect(titles[0]?.closest('[data-anchor="choy.form.view-chrome"]')).not.toBeNull();

    mounted.unmount();
  });

  test('inherits route meta title into ActionBar when ChoyPage omits title', async () => {
    const store = loginStore();
    const { router } = createFeStubRouter({
      route: {
        name: 'CompanyDetail',
        path: '/base/companies/1',
        fullPath: '/base/companies/1',
        params: { id: '1' },
        query: {},
        meta: { pageTitle: 'Company Details' },
      },
    });
    const Host = defineComponent({
      setup() {
        return () =>
          h(
            ChoyPage as any,
            { store, padding: true },
            {
              default: () =>
                h(ChoyFormView as any, {
                  store,
                  viewMode: 'display',
                  recordId: '1',
                  resolveRecordIdFromRoute: false,
                  showHeader: true,
                  showActions: true,
                  showMessages: false,
                }),
            },
          );
      },
    });

    const mounted = mountApp(Host, { plugins: [createPinia(), router] });
    await flushPromises();

    const titles = mounted.qa('[data-slot=page-title]');
    expect(titles.length).toBe(1);
    expect(titles[0]?.textContent?.trim()).toBe('Company Details');
    expect(titles[0]?.closest('[data-anchor="choy.form.view-chrome"]')).not.toBeNull();
    expect((mounted.q('[data-anchor="choy.page"]') as HTMLElement | null)?.className || '').toContain(
      'choy-inset-pad-no-top',
    );

    mounted.unmount();
  });

  test('registers the current record as the page action target unless embedded', async () => {
    const store = loginStore();
    const holder: { target: PageActionTarget | null } = { target: null };
    const pageCtx = {
      store: computed(() => store),
      actionTarget: shallowRef(null),
      registerActionTarget(next: PageActionTarget) {
        holder.target = next;
      },
      unregisterActionTarget() {},
      title: computed(() => ''),
      titleId: 'page-title',
      titleActionsIo: computed(() => null),
      titleHostedByView: shallowRef(false),
      claimPageTitle() {},
      releasePageTitle() {},
    };
    const { router } = createFeStubRouter({
      route: { name: 'login', path: '/web/login', fullPath: '/web/login', params: {}, query: {}, meta: {} },
    });
    const mounted = mountApp(ChoyFormView as any, {
      props: {
        viewMode: 'display',
        resolveRecordIdFromRoute: false,
        showMessages: false,
        showHeader: true,
        showActions: false,
        recordId: 'rec-9',
      },
      plugins: [createPinia(), router],
      provide: { [PageContextKey as symbol]: pageCtx },
    });
    await flushPromises();
    expect(holder.target?.selectedItems).toEqual([{ Id: 'rec-9' }]);
    expect(typeof holder.target?.refresh).toBe('function');
    mounted.unmount();

    holder.target = null;
    const { router: embeddedRouter } = createFeStubRouter({
      route: { name: 'login', path: '/web/login', fullPath: '/web/login', params: {}, query: {}, meta: {} },
    });
    const embedded = mountApp(ChoyFormView as any, {
      props: {
        store,
        embedded: true,
        viewMode: 'display',
        resolveRecordIdFromRoute: false,
        showMessages: false,
        showHeader: false,
        showActions: false,
        recordId: 'rec-9',
        registerActionTarget: true,
      },
      plugins: [createPinia(), embeddedRouter],
      provide: { [PageContextKey as symbol]: pageCtx },
    });
    await flushPromises();
    expect(holder.target).toBeNull();
    embedded.unmount();
  });
});
