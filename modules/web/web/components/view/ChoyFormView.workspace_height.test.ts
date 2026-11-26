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
import { createUiModelStore } from '@/web/web/stores/uiModelStore';
import ChoyFormView from './ChoyFormView.vue';

const createFeStubRouter = (VueRouter as any).createFeStubRouter;

function loginStore() {
  return createUiModelStore({
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

/** Leading-column chrome order for the minimal DOM (no compound querySelector). */
function actionBarLeadingOrder(root: Element): string[] {
  const out: string[] = [];
  const walk = (node: Element) => {
    const testId = node.getAttribute('data-testid');
    if (
      testId === 'form-primary-edit' ||
      testId === 'form-primary-new' ||
      testId === 'form-secondary-edit' ||
      testId === 'form-secondary-reset'
    ) {
      out.push(testId);
      return;
    }
    if (node.getAttribute('data-slot') === 'page-title') {
      out.push('page-title');
      return;
    }
    // Title-as-menu trigger: record page-title, skip the whole-button label blob.
    if (testId === 'page-io-menu-trigger') {
      const title = node.querySelector('[data-slot="page-title"]');
      if (title) out.push('page-title');
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
  walk(root);
  return out;
}

describe('ChoyFormView workspace height', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  test('body scrolls inside the shell and the container uses gap-5', async () => {
    const mounted = mountForm({ showHeader: false, showActions: false });
    await flushPromises();

    const root = mounted.q('[data-slot="view-container"]') as HTMLElement | null;
    expect(root).not.toBeNull();
    expect(root!.className).toContain('gap-5');

    const shell = mounted.q('[data-slot="view-body"]') as HTMLElement | null;
    expect(shell).not.toBeNull();
    expect(shell!.className).toContain('overflow-hidden');
    expect(shell!.className.includes('bg-card')).toBe(false);
    expect(shell!.getAttribute('data-anchor')).toBe('choy.view.sheet');

    const body = mounted.q('[data-anchor="choy.form-view"]') as HTMLElement | null;
    expect(body).not.toBeNull();
    expect(body!.className).toContain('min-h-0');
    expect(body!.className).toContain('flex-1');
    expect(body!.className).toContain('overflow-y-auto');
    expect(body!.className.split(/\s+/)).not.toContain('p-5');
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

  test('display left chrome is title → Edit primary (New in ⋮)', async () => {
    const store = loginStore();
    const { router } = createFeStubRouter({
      route: { name: 'login', path: '/web/login', fullPath: '/web/login', params: {}, query: {}, meta: {} },
    });
    const Host = defineComponent({
      setup() {
        return () =>
          h(
            ChoyPage as any,
            { store, title: 'Token Details' },
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
                  actionIds: {
                    create: 'auth.action.token_create',
                    edit: 'auth.action.token_edit',
                  },
                  hasAction: () => true,
                }),
            },
          );
      },
    });
    const mounted = mountApp(Host, { plugins: [createPinia(), router] });
    await flushPromises();

    const leading = mounted.q('[data-slot=view-action-bar-leading]') as HTMLElement | null;
    expect(leading).not.toBeNull();
    const order = actionBarLeadingOrder(leading!);
    const titleIdx = order.indexOf('page-title');
    const editIdx = order.indexOf('form-primary-edit');
    expect(titleIdx).toBeGreaterThanOrEqual(0);
    expect(editIdx).toBeGreaterThan(titleIdx);
    expect(order.indexOf('form-primary-new')).toBe(-1);
    expect(mounted.q('[data-testid=form-overflow-new]')).not.toBeNull();
    expect(mounted.q('[data-slot=view-action-bar-overflow]')).not.toBeNull();

    mounted.unmount();
  });

  test('display with domain primary keeps Edit secondary and lifecycle in ⋮', async () => {
    const store = loginStore();
    const { router } = createFeStubRouter({
      route: { name: 'login', path: '/web/login', fullPath: '/web/login', params: {}, query: {}, meta: {} },
    });
    const Host = defineComponent({
      setup() {
        return () =>
          h(
            ChoyPage as any,
            { store, title: 'Module' },
            {
              default: () =>
                h(
                  ChoyFormView as any,
                  {
                    store,
                    viewMode: 'display',
                    recordId: 'mod-1',
                    resolveRecordIdFromRoute: false,
                    showHeader: true,
                    showActions: true,
                    showMessages: false,
                    createAction: '/modules/new',
                    actionIds: {
                      create: 'meta.action.create',
                      edit: 'meta.action.edit',
                      copy: 'meta.action.copy',
                      delete: 'meta.action.delete',
                    },
                    hasAction: () => true,
                  },
                  {
                    'system-actions': () =>
                      h('button', { type: 'button', 'data-testid': 'domain-primary' }, 'Install'),
                  },
                ),
            },
          );
      },
    });
    const mounted = mountApp(Host, { plugins: [createPinia(), router] });
    await flushPromises();

    expect(mounted.q('[data-testid=domain-primary]')?.textContent).toBe('Install');
    expect(mounted.q('[data-testid=form-primary-edit]')).toBeNull();
    expect(mounted.q('[data-testid=form-secondary-edit]')).not.toBeNull();
    expect(mounted.q('[data-testid=form-overflow-new]')).not.toBeNull();
    expect(mounted.q('[data-testid=form-overflow-copy]')).not.toBeNull();
    expect(mounted.q('[data-testid=form-overflow-delete]')).not.toBeNull();

    mounted.unmount();
  });

  test('create left chrome is title → Save/Cancel → ⋮ (Reset in overflow)', async () => {
    const store = loginStore();
    const { router } = createFeStubRouter({
      route: { name: 'login', path: '/web/login', fullPath: '/web/login', params: {}, query: {}, meta: {} },
    });
    const Host = defineComponent({
      setup() {
        return () =>
          h(
            ChoyPage as any,
            { store, title: 'New Token' },
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

    const leading = mounted.q('[data-slot=view-action-bar-leading]') as HTMLElement | null;
    expect(leading).not.toBeNull();
    const order = actionBarLeadingOrder(leading!);
    const saveIdx = order.findIndex(t => t.includes('Save') || t.includes('保存'));
    const cancelIdx = order.findIndex(t => t.includes('Cancel') || t.includes('取消'));
    const titleIdx = order.indexOf('page-title');
    expect(titleIdx).toBeGreaterThanOrEqual(0);
    expect(saveIdx).toBeGreaterThan(titleIdx);
    expect(cancelIdx).toBeGreaterThan(saveIdx);
    expect(order.includes('form-secondary-reset')).toBe(false);
    expect(mounted.q('[data-testid=page-io-menu-trigger]')).not.toBeNull();
    expect(mounted.q('[data-slot=view-action-bar-overflow]')).not.toBeNull();

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
            { store, title: 'Partner' },
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
    // Shell content owns gutters; page default padding is off.
    expect(pageRoot?.className || '').not.toContain('choy-inset-pad');

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
            { store },
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
    expect((mounted.q('[data-anchor="choy.page"]') as HTMLElement | null)?.className || '').not.toContain(
      'choy-inset-pad',
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
