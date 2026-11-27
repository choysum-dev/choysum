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

  test('body scrolls inside the shell; page Form uses tight gap and in-paper body pad', async () => {
    const mounted = mountForm({ showHeader: false, showActions: false });
    await flushPromises();

    const root = mounted.q('[data-slot="view-container"]') as HTMLElement | null;
    expect(root).not.toBeNull();
    // Form overrides ViewBase gap; gap-2 matches sheet gutter (section-gap).
    expect(root!.className).toContain('gap-2');
    expect(root!.className.split(/\s+/)).not.toContain('gap-5');
    expect(root!.className.split(/\s+/)).not.toContain('gap-1');

    const shell = mounted.q('[data-slot="view-body"]') as HTMLElement | null;
    expect(shell).not.toBeNull();
    expect(shell!.className).toContain('overflow-hidden');
    // Paper shell on view-body; in-paper density on host (sheetPad=form).
    expect(shell!.getAttribute('data-sheet-surface')).toBe('true');
    expect(shell!.className.includes('bg-card')).toBe(true);
    expect(shell!.className).toContain('rounded-md');
    expect(shell!.className).toContain('border-border');
    expect(shell!.className).toContain('shadow-xs');
    expect(shell!.className).toContain('mx-[var(--choy-view-sheet-gutter)]');
    expect(shell!.getAttribute('data-anchor')).toBe('choy.view.sheet');

    const body = mounted.q('[data-anchor="choy.form-view"]') as HTMLElement | null;
    expect(body).not.toBeNull();
    expect(body!.className).toContain('min-h-0');
    expect(body!.className).toContain('flex-1');
    expect(body!.className).toContain('overflow-y-auto');
    expect(body!.className).toContain('choy-form-view__sheet');
    expect(body!.className).toContain('p-[var(--choy-view-sheet-pad)]');
    expect(body!.className).not.toContain('border-t');
    expect(body!.getAttribute('data-testid')).toBe('form-sheet');
    expect(body!.getAttribute('data-sheet-pad')).toBe('form');
    expect(body!.className.split(/\s+/)).not.toContain('p-5');

    mounted.unmount();
  });

  test('embedded Form skips in-paper body chrome', async () => {
    const mounted = mountForm({ embedded: true, showHeader: false, showActions: false });
    await flushPromises();
    const root = mounted.q('[data-slot="view-container"]') as HTMLElement | null;
    expect(root!.className).toContain('gap-5');
    const sheet = mounted.q('[data-slot="view-body"]') as HTMLElement | null;
    expect(sheet!.getAttribute('data-sheet-surface')).toBe('false');
    expect(sheet!.className.includes('bg-card')).toBe(false);
    expect(sheet!.className).not.toContain('border-border');
    expect(sheet!.className).not.toContain('mx-[var(--choy-view-sheet-gutter)]');
    const body = mounted.q('[data-anchor="choy.form-view"]') as HTMLElement | null;
    expect(body).not.toBeNull();
    expect(body!.className).not.toContain('choy-form-view__sheet');
    expect(body!.className).not.toContain('p-[var(--choy-view-sheet-pad)]');
    expect(body!.getAttribute('data-sheet-pad')).toBe('none');
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

  test('display with statusbar-buttons under Action Bar; platform Edit soft-primary; lifecycle in ⋮', async () => {
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
                    'statusbar-buttons': () =>
                      h('button', { type: 'button', 'data-testid': 'domain-primary' }, 'Install'),
                    'statusbar-status': () =>
                      h('div', { 'data-testid': 'domain-status' }, 'Installed'),
                  },
                ),
            },
          );
      },
    });
    const mounted = mountApp(Host, { plugins: [createPinia(), router] });
    await flushPromises();

    expect(mounted.q('[data-testid=domain-primary]')?.textContent).toBe('Install');
    const statusbar = mounted.q('[data-testid=form-statusbar]') as HTMLElement | null;
    expect(statusbar).not.toBeNull();
    expect(statusbar!.querySelector('[data-testid=form-statusbar-buttons]')).not.toBeNull();
    expect(statusbar!.querySelector('[data-testid=domain-primary]')).not.toBeNull();
    expect(statusbar!.querySelector('[data-testid=form-statusbar-status]')).not.toBeNull();
    expect(statusbar!.querySelector('[data-testid=domain-status]')).not.toBeNull();
    // Statusbar stays in view-header (control ribbon on muted canvas).
    // view-header gap owns Action Bar ↔ statusbar; header-extra has no vertical pad.
    const header = mounted.q('[data-slot=view-header]') as HTMLElement | null;
    expect(header).not.toBeNull();
    expect(header!.className).toContain('gap-2');
    expect(header!.querySelector('[data-testid=form-statusbar]')).not.toBeNull();
    expect(mounted.q('[data-slot=view-body]')?.querySelector('[data-testid=form-statusbar]')).toBeNull();
    const headerExtra = mounted.q('[data-slot=view-header-extra]') as HTMLElement | null;
    expect(headerExtra).not.toBeNull();
    expect(headerExtra!.className.split(/\s+/)).not.toContain('pt-1');
    expect(headerExtra!.className.split(/\s+/)).not.toContain('pb-2');
    expect(headerExtra!.className.split(/\s+/)).not.toContain('pb-1.5');
    // Domain CTAs leave Action Bar center empty (List selection still owns center).
    expect(mounted.q('[data-testid=form-contextual-actions]')).toBeNull();
    const center = mounted.q('[data-slot=view-action-bar-center]') as HTMLElement | null;
    expect(center?.querySelector('[data-testid=domain-primary]') ?? null).toBeNull();
    // Platform Edit stays in the left system tray (not inside the statusbar).
    expect(mounted.q('[data-testid=form-primary-edit]')).not.toBeNull();
    expect(statusbar!.querySelector('[data-testid=form-primary-edit]')).toBeNull();
    expect(mounted.q('[data-testid=form-secondary-edit]')).toBeNull();
    expect(mounted.q('[data-testid=form-overflow-new]')).not.toBeNull();
    expect(mounted.q('[data-testid=form-overflow-copy]')).not.toBeNull();
    expect(mounted.q('[data-testid=form-overflow-delete]')).not.toBeNull();

    mounted.unmount();
  });

  test('create/edit keeps statusbar-status ribbon and hides statusbar-buttons', async () => {
    // viewMode create reliably enters isEditing (edit+recordId depends on Browse).
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
                    viewMode: 'create',
                    resolveRecordIdFromRoute: false,
                    showHeader: true,
                    showActions: true,
                    showMessages: false,
                    hasAction: () => true,
                  },
                  {
                    'statusbar-buttons': () =>
                      h('button', { type: 'button', 'data-testid': 'domain-primary' }, 'Install'),
                    'statusbar-status': () =>
                      h('div', { 'data-testid': 'domain-status' }, 'Installed'),
                  },
                ),
            },
          );
      },
    });
    const mounted = mountApp(Host, { plugins: [createPinia(), router] });
    await flushPromises();

    const leading = mounted.q('[data-slot=view-action-bar-leading]') as HTMLElement | null;
    expect(leading).not.toBeNull();
    const order = actionBarLeadingOrder(leading!);
    expect(order.some((t) => t.includes('Save') || t.includes('保存'))).toBe(true);
    const statusbar = mounted.q('[data-testid=form-statusbar]') as HTMLElement | null;
    expect(statusbar).not.toBeNull();
    expect(statusbar!.style.display).not.toBe('none');
    expect(statusbar!.querySelector('[data-testid=form-statusbar-buttons]')).toBeNull();
    expect(mounted.q('[data-testid=domain-primary]')).toBeNull();
    expect(statusbar!.querySelector('[data-testid=form-statusbar-status]')).not.toBeNull();
    expect(mounted.q('[data-testid=domain-status]')?.textContent).toBe('Installed');

    mounted.unmount();
  });

  test('create/edit with only statusbar-buttons omits header-extra (no empty gap)', async () => {
    const store = loginStore();
    const { router } = createFeStubRouter({
      route: { name: 'login', path: '/web/login', fullPath: '/web/login', params: {}, query: {}, meta: {} },
    });
    const Host = defineComponent({
      setup() {
        return () =>
          h(
            ChoyPage as any,
            { store, title: 'Language' },
            {
              default: () =>
                h(
                  ChoyFormView as any,
                  {
                    store,
                    viewMode: 'create',
                    resolveRecordIdFromRoute: false,
                    showHeader: true,
                    showActions: true,
                    showMessages: false,
                    hasAction: () => true,
                  },
                  {
                    'statusbar-buttons': () =>
                      h('button', { type: 'button', 'data-testid': 'domain-activate' }, 'Activate'),
                  },
                ),
            },
          );
      },
    });
    const mounted = mountApp(Host, { plugins: [createPinia(), router] });
    await flushPromises();

    // Slot omitted entirely so ChoyViewBase does not keep an empty header-extra gap.
    expect(mounted.q('[data-testid=form-statusbar]')).toBeNull();
    expect(mounted.q('[data-slot=view-header-extra]')).toBeNull();
    expect(mounted.q('[data-testid=domain-activate]')).toBeNull();

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
