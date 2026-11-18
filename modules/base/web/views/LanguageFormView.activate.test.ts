// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { mount, flushPromises } from '@choysum/test-utils';
import { useAuthStore } from '@/auth/web/stores/auth';
import { ChoyMessage } from '@/web/web/composables/useChoyMessage';
import { restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyFormView from '@/web/web/components/view/ChoyFormView.vue';
import LanguageFormView from './LanguageFormView.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

function seedLanguageWritePermission(actions: string[] = ['*']) {
  const auth = useAuthStore();
  auth.identity = {
    metadata: { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] },
  } as any;
  auth.permissionState = {
    permStateVersion: 1,
    byCompany: {
      '*': { ui: { routes: [], menus: [], actions } },
    },
  } as any;
}

function buildStore(row: { Id: string; Code: string; IsActive: boolean; Name?: string }, opts?: {
  recordsEntry?: unknown;
  rows?: unknown[];
  useEmptyRecords?: boolean;
  browse?: () => Promise<unknown>;
  activate?: (req: unknown) => Promise<unknown>;
  deactivate?: (req: unknown) => Promise<unknown>;
  reloadTerminology?: (req: unknown) => Promise<unknown>;
}) {
  const activateCalls: unknown[] = [];
  const deactivateCalls: unknown[] = [];
  const reloadTerminologyCalls: unknown[] = [];
  const updateByIdCalls: unknown[] = [];
  const browseCalls: unknown[] = [];
  const store = {
    $id: 'fe-stub-language-activate-store',
    state: {
      records: opts?.useEmptyRecords
        ? {}
        : opts?.recordsEntry !== undefined
          ? { [row.Id]: opts.recordsEntry }
          : { [row.Id]: { ...row } },
      result: { rows: opts?.rows ?? [] },
    },
    records: {},
    Activate: async (req: unknown) => {
      activateCalls.push(req);
      if (opts?.activate) return opts.activate(req);
      return { Id: row.Id, Code: row.Code, IsActive: true };
    },
    Deactivate: async (req: unknown) => {
      deactivateCalls.push(req);
      if (opts?.deactivate) return opts.deactivate(req);
      return { Id: row.Id, Code: row.Code, IsActive: false };
    },
    ReloadTerminology: async (req: unknown) => {
      reloadTerminologyCalls.push(req);
      if (opts?.reloadTerminology) return opts.reloadTerminology(req);
      return {
        Code: row.Code,
        ModulesScanned: 1,
        ModulesWithPo: 1,
        Upserted: 2,
        SkippedOverride: 0,
        RejectedNoCtxt: 0,
        SkippedObsolete: 0,
        PurgedRetired: 0,
      };
    },
    UpdateById: async (...args: unknown[]) => {
      updateByIdCalls.push(args);
      return row;
    },
    Browse: async (...args: unknown[]) => {
      browseCalls.push(args);
      if (opts?.browse) return opts.browse();
      return { ...row, IsActive: !row.IsActive };
    },
  };
  return { store, activateCalls, deactivateCalls, reloadTerminologyCalls, updateByIdCalls, browseCalls };
}

function mountLanguageForm(store: unknown, recordId: string) {
  const wrapper = mount(LanguageFormView as any, {
    props: { store, recordId, viewMode: 'display' },
    global: buildPageMountGlobal({
      route: { path: `/base/languages/${recordId}`, fullPath: `/base/languages/${recordId}` },
    }),
  });
  // Pinia auth store is ready after mount (same pattern as ModuleCatalogFormView.mount.test).
  seedLanguageWritePermission();
  return wrapper;
}

test('LanguageFormView: display shows Activate when inactive', async () => {
  const { store, activateCalls, updateByIdCalls, browseCalls } = buildStore({
    Id: 'lang-inactive',
    Code: 'zh_HK',
    IsActive: false,
    Name: 'Chinese (HK)',
  });
  const wrapper = mountLanguageForm(store, 'lang-inactive');
  await flushPromises();
  const btn = wrapper.find('[data-testid="language-activate"]');
  expect(btn.exists()).toBe(true);
  expect(wrapper.find('[data-testid="language-deactivate"]').exists()).toBe(false);
  await btn.trigger('click');
  await flushPromises();
  expect(activateCalls).toEqual([{ Id: 'lang-inactive' }]);
  expect(updateByIdCalls.length).toBe(0);
  expect(browseCalls.length).toBe(1);
  expect(wrapper.find('[data-testid="language-deactivate"]').exists()).toBe(true);
  wrapper.unmount();
});

test('LanguageFormView: resolves Activate from route :id when prop omitted', async () => {
  seedLanguageWritePermission();
  const row = { Id: 'lang-route', Code: 'ca_ES', IsActive: false, Name: 'Catalan' };
  const { store } = buildStore(row);
  const wrapper = mount(LanguageFormView as any, {
    props: { store, viewMode: 'display' },
    global: buildPageMountGlobal({
      route: {
        path: '/base/languages/lang-route',
        fullPath: '/base/languages/lang-route',
        params: { id: 'lang-route' },
      },
    }),
  });
  seedLanguageWritePermission();
  await flushPromises();
  expect(wrapper.find('[data-testid="language-activate"]').exists()).toBe(true);
  wrapper.unmount();
});

test('LanguageFormView: display shows Deactivate when active', async () => {
  const { store, deactivateCalls, updateByIdCalls } = buildStore({
    Id: 'lang-active',
    Code: 'zh_CN',
    IsActive: true,
    Name: 'Chinese',
  });
  const wrapper = mountLanguageForm(store, 'lang-active');
  await flushPromises();
  const btn = wrapper.find('[data-testid="language-deactivate"]');
  expect(btn.exists()).toBe(true);
  expect(wrapper.find('[data-testid="language-activate"]').exists()).toBe(false);
  await btn.trigger('click');
  await flushPromises();
  expect(deactivateCalls).toEqual([{ Id: 'lang-active' }]);
  expect(updateByIdCalls.length).toBe(0);
  wrapper.unmount();
});

test('LanguageFormView: unwraps RecordRow payload in store.records', async () => {
  const row = { Id: 'lang-payload', Code: 'fr_FR', IsActive: false, Name: 'French' };
  const { store } = buildStore(row, {
    recordsEntry: { kind: 'record', key: 'lang-payload', payload: row },
  });
  const wrapper = mountLanguageForm(store, 'lang-payload');
  await flushPromises();
  expect(wrapper.find('[data-testid="language-activate"]').exists()).toBe(true);
  wrapper.unmount();
});

test('LanguageFormView: resolves row from result.rows when records empty', async () => {
  const row = { Id: 'lang-rows', Code: 'de_DE', IsActive: true, Name: 'German' };
  const { store } = buildStore(row, {
    useEmptyRecords: true,
    rows: [{ kind: 'record', key: 'lang-rows', payload: row }],
  });
  const wrapper = mountLanguageForm(store, 'lang-rows');
  await flushPromises();
  expect(wrapper.find('[data-testid="language-deactivate"]').exists()).toBe(true);
  wrapper.unmount();
});

test('LanguageFormView: hides activate actions without write permission', async () => {
  const { store } = buildStore({ Id: 'lang-noperm', Code: 'zh_HK', IsActive: false });
  const wrapper = mount(LanguageFormView as any, {
    props: { store, recordId: 'lang-noperm', viewMode: 'display' },
    global: buildPageMountGlobal({
      route: { path: '/base/languages/lang-noperm', fullPath: '/base/languages/lang-noperm' },
    }),
  });
  seedLanguageWritePermission([]);
  await flushPromises();
  expect(wrapper.find('[data-testid="language-activate"]').exists()).toBe(false);
  expect(wrapper.find('[data-testid="language-deactivate"]').exists()).toBe(false);
  expect(wrapper.find('[data-testid="language-reload-terminology"]').exists()).toBe(false);
  wrapper.unmount();
});

test('LanguageFormView: ignores empty Browse payload after Activate', async () => {
  const { store, activateCalls } = buildStore(
    { Id: 'lang-browse-empty', Code: 'zh_HK', IsActive: false },
    {
      browse: async () => ({}),
    }
  );
  const wrapper = mountLanguageForm(store, 'lang-browse-empty');
  await flushPromises();
  await wrapper.find('[data-testid="language-activate"]').trigger('click');
  await flushPromises();
  expect(activateCalls).toEqual([{ Id: 'lang-browse-empty' }]);
  // Empty Browse payload does not override; Activate button remains.
  expect(wrapper.find('[data-testid="language-activate"]').exists()).toBe(true);
  wrapper.unmount();
});

test('LanguageFormView: missing Activate method is handled without UpdateById', async () => {
  const { store, updateByIdCalls } = buildStore({ Id: 'lang-missing', Code: 'zh_HK', IsActive: false });
  delete (store as any).Activate;
  const wrapper = mountLanguageForm(store, 'lang-missing');
  await flushPromises();
  await wrapper.find('[data-testid="language-activate"]').trigger('click');
  await flushPromises();
  expect(updateByIdCalls.length).toBe(0);
  expect(wrapper.find('[data-testid="language-activate"]').exists()).toBe(true);
  wrapper.unmount();
});

test('LanguageFormView: Activate rejection surfaces error and clears busy', async () => {
  const toasts: string[] = [];
  const origError = ChoyMessage.error;
  ChoyMessage.error = ((msg: string) => {
    toasts.push(String(msg));
    return 0;
  }) as typeof ChoyMessage.error;
  try {
    let rejectActivate!: (err: Error) => void;
    const activateGate = new Promise<never>((_, reject) => {
      rejectActivate = reject;
    });
    const { store, activateCalls, browseCalls } = buildStore(
      { Id: 'lang-act-fail', Code: 'zh_HK', IsActive: false },
      {
        activate: async () => {
          await activateGate;
        },
      }
    );
    const wrapper = mountLanguageForm(store, 'lang-act-fail');
    await flushPromises();
    const setupState = (wrapper.vm as any).$.setupState;
    const pending = setupState.onActivate('lang-act-fail');
    await flushPromises();
    expect(setupState.activeBusy).toBe(true);
    rejectActivate(new Error('activate denied'));
    let settled: 'ok' | 'err' = 'ok';
    await pending.then(
      () => {
        settled = 'ok';
      },
      () => {
        settled = 'err';
      }
    );
    await flushPromises();
    expect(settled).toBe('ok');
    expect(activateCalls).toEqual([{ Id: 'lang-act-fail' }]);
    expect(browseCalls.length).toBe(0);
    expect(toasts).toEqual(['activate denied']);
    expect(setupState.activeBusy).toBe(false);
    expect(wrapper.find('[data-testid="language-activate"]').exists()).toBe(true);
    wrapper.unmount();
  } finally {
    ChoyMessage.error = origError;
  }
});

test('LanguageFormView: missing Deactivate method is handled', async () => {
  const { store, updateByIdCalls } = buildStore({ Id: 'lang-missing-deact', Code: 'zh_CN', IsActive: true });
  delete (store as any).Deactivate;
  const wrapper = mountLanguageForm(store, 'lang-missing-deact');
  await flushPromises();
  await wrapper.find('[data-testid="language-deactivate"]').trigger('click');
  await flushPromises();
  expect(updateByIdCalls.length).toBe(0);
  expect(wrapper.find('[data-testid="language-deactivate"]').exists()).toBe(true);
  wrapper.unmount();
});

test('LanguageFormView: Deactivate rejection surfaces error and clears busy', async () => {
  const toasts: string[] = [];
  const origError = ChoyMessage.error;
  ChoyMessage.error = ((msg: string) => {
    toasts.push(String(msg));
    return 0;
  }) as typeof ChoyMessage.error;
  try {
    let rejectDeactivate!: (err: Error) => void;
    const deactivateGate = new Promise<never>((_, reject) => {
      rejectDeactivate = reject;
    });
    const { store, deactivateCalls, browseCalls } = buildStore(
      { Id: 'lang-deact-fail', Code: 'zh_CN', IsActive: true },
      {
        deactivate: async () => {
          await deactivateGate;
        },
      }
    );
    const wrapper = mountLanguageForm(store, 'lang-deact-fail');
    await flushPromises();
    const setupState = (wrapper.vm as any).$.setupState;
    const pending = setupState.onDeactivate('lang-deact-fail');
    await flushPromises();
    expect(setupState.activeBusy).toBe(true);
    rejectDeactivate(new Error('deactivate denied'));
    let settled: 'ok' | 'err' = 'ok';
    await pending.then(
      () => {
        settled = 'ok';
      },
      () => {
        settled = 'err';
      }
    );
    await flushPromises();
    expect(settled).toBe('ok');
    expect(deactivateCalls).toEqual([{ Id: 'lang-deact-fail' }]);
    expect(browseCalls.length).toBe(0);
    expect(toasts).toEqual(['deactivate denied']);
    expect(setupState.activeBusy).toBe(false);
    expect(wrapper.find('[data-testid="language-deactivate"]').exists()).toBe(true);
    wrapper.unmount();
  } finally {
    ChoyMessage.error = origError;
  }
});

test('LanguageFormView: Activate without Browse still succeeds', async () => {
  const { store, activateCalls } = buildStore({ Id: 'lang-nobrowse', Code: 'zh_HK', IsActive: false });
  delete (store as any).Browse;
  const wrapper = mountLanguageForm(store, 'lang-nobrowse');
  await flushPromises();
  await wrapper.find('[data-testid="language-activate"]').trigger('click');
  await flushPromises();
  expect(activateCalls).toEqual([{ Id: 'lang-nobrowse' }]);
  wrapper.unmount();
});

test('LanguageFormView: edit-mode reload prefers Browse over reloadDisplay', async () => {
  const { store, activateCalls, browseCalls } = buildStore({
    Id: 'lang-edit-reload',
    Code: 'zh_HK',
    IsActive: false,
  });
  let reloadDisplayCalls = 0;
  const wrapper = mountLanguageForm(store, 'lang-edit-reload');
  await flushPromises();
  const setupState = (wrapper.vm as any).$.setupState;
  setupState.formViewRef = {
    getViewMode: () => 'edit',
    reloadDisplay: async () => {
      reloadDisplayCalls += 1;
    },
  };
  await wrapper.find('[data-testid="language-activate"]').trigger('click');
  await flushPromises();
  expect(activateCalls).toEqual([{ Id: 'lang-edit-reload' }]);
  expect(reloadDisplayCalls).toBe(0);
  expect(browseCalls.length).toBe(1);
  wrapper.unmount();
});

test('LanguageFormView: empty slot recordId does not toggle busy', async () => {
  const { store, activateCalls } = buildStore({ Id: 'lang-empty-id', Code: 'zh_HK', IsActive: false });
  const wrapper = mountLanguageForm(store, 'lang-empty-id');
  await flushPromises();
  const setupState = (wrapper.vm as any).$.setupState;
  await setupState.onActivate('  ');
  await flushPromises();
  expect(activateCalls.length).toBe(0);
  expect(setupState.activeBusy).toBe(false);
  await setupState.onDeactivate('');
  await flushPromises();
  expect(setupState.activeBusy).toBe(false);
  wrapper.unmount();
});

test('LanguageFormView: reloadDisplay success clears override', async () => {
  const { store, activateCalls, browseCalls } = buildStore({
    Id: 'lang-reload-display',
    Code: 'zh_HK',
    IsActive: false,
  });
  let reloadDisplayCalls = 0;
  const wrapper = mountLanguageForm(store, 'lang-reload-display');
  await flushPromises();
  const setupState = (wrapper.vm as any).$.setupState;
  setupState.formViewRef = {
    getViewMode: () => 'display',
    reloadDisplay: async () => {
      reloadDisplayCalls += 1;
    },
  };
  setupState.rowOverride = {
    id: 'lang-reload-display',
    fields: { Id: 'lang-reload-display', Code: 'zh_HK', IsActive: false },
  };
  await wrapper.find('[data-testid="language-activate"]').trigger('click');
  await flushPromises();
  expect(activateCalls).toEqual([{ Id: 'lang-reload-display' }]);
  expect(reloadDisplayCalls).toBe(1);
  expect(browseCalls.length).toBe(0);
  expect(setupState.rowOverride).toBeNull();
  wrapper.unmount();
});

test('LanguageFormView: refresh path clears override when reloadDisplay absent', async () => {
  const { store, activateCalls, browseCalls } = buildStore({
    Id: 'lang-refresh',
    Code: 'zh_HK',
    IsActive: false,
  });
  let refreshCalls = 0;
  const wrapper = mountLanguageForm(store, 'lang-refresh');
  await flushPromises();
  const setupState = (wrapper.vm as any).$.setupState;
  setupState.formViewRef = {
    getViewMode: () => 'display',
    refresh: async () => {
      refreshCalls += 1;
    },
  };
  await wrapper.find('[data-testid="language-activate"]').trigger('click');
  await flushPromises();
  expect(activateCalls).toEqual([{ Id: 'lang-refresh' }]);
  expect(refreshCalls).toBe(1);
  expect(browseCalls.length).toBe(0);
  expect(setupState.rowOverride).toBeNull();
  wrapper.unmount();
});

test('LanguageFormView: reloadDisplay rejection falls back to Browse', async () => {
  const { store, activateCalls, browseCalls } = buildStore({
    Id: 'lang-reload-fail',
    Code: 'zh_HK',
    IsActive: false,
  });
  const wrapper = mountLanguageForm(store, 'lang-reload-fail');
  await flushPromises();
  const setupState = (wrapper.vm as any).$.setupState;
  setupState.formViewRef = {
    getViewMode: () => 'display',
    reloadDisplay: async () => {
      throw new Error('reload failed');
    },
  };
  await wrapper.find('[data-testid="language-activate"]').trigger('click');
  await flushPromises();
  expect(activateCalls).toEqual([{ Id: 'lang-reload-fail' }]);
  expect(browseCalls.length).toBe(1);
  wrapper.unmount();
});

test('LanguageFormView: activeBusy blocks re-entrant Activate', async () => {
  let release!: () => void;
  const gate = new Promise<void>(resolve => {
    release = resolve;
  });
  const { store, activateCalls } = buildStore(
    { Id: 'lang-busy', Code: 'zh_HK', IsActive: false },
    {
      activate: async () => {
        await gate;
        return { Id: 'lang-busy', Code: 'zh_HK', IsActive: true };
      },
    }
  );
  const wrapper = mountLanguageForm(store, 'lang-busy');
  await flushPromises();
  const setupState = (wrapper.vm as any).$.setupState;
  const first = setupState.onActivate('lang-busy');
  await flushPromises();
  expect(setupState.activeBusy).toBe(true);
  await setupState.onActivate('lang-busy');
  expect(activateCalls.length).toBe(1);
  release();
  await first;
  await flushPromises();
  expect(setupState.activeBusy).toBe(false);
  wrapper.unmount();
});

test('LanguageFormView: empty Browse keeps override for a different record', async () => {
  const { store } = buildStore(
    { Id: 'lang-keep-override', Code: 'zh_HK', IsActive: false },
    { browse: async () => ({}) }
  );
  const wrapper = mountLanguageForm(store, 'lang-keep-override');
  await flushPromises();
  const setupState = (wrapper.vm as any).$.setupState;
  setupState.rowOverride = { id: 'other-lang', fields: { Id: 'other-lang', IsActive: true } };
  await setupState.reloadRecordViaBrowse('lang-keep-override');
  await flushPromises();
  expect(setupState.rowOverride).toEqual({ id: 'other-lang', fields: { Id: 'other-lang', IsActive: true } });
  wrapper.unmount();
});

test('LanguageFormView: shows Update translations for active and inactive rows', async () => {
  const inactive = buildStore({ Id: 'lang-reload-off', Code: 'zh_HK', IsActive: false });
  const inactiveWrap = mountLanguageForm(inactive.store, 'lang-reload-off');
  await flushPromises();
  expect(inactiveWrap.find('[data-testid="language-reload-terminology"]').exists()).toBe(true);
  await inactiveWrap.find('[data-testid="language-reload-terminology"]').trigger('click');
  await flushPromises();
  expect(inactive.reloadTerminologyCalls).toEqual([
    { Code: 'zh_HK', LanguageId: 'lang-reload-off' },
  ]);
  inactiveWrap.unmount();

  const active = buildStore({ Id: 'lang-reload-on', Code: 'zh_CN', IsActive: true });
  const activeWrap = mountLanguageForm(active.store, 'lang-reload-on');
  await flushPromises();
  expect(activeWrap.find('[data-testid="language-reload-terminology"]').exists()).toBe(true);
  activeWrap.unmount();
});

test('LanguageFormView: ReloadTerminology toast branches by packaged PO result', async () => {
  const levels: string[] = [];
  const titles: string[] = [];
  const orig = {
    success: ChoyMessage.success,
    warning: ChoyMessage.warning,
  };
  ChoyMessage.success = ((msg: string) => {
    levels.push('success');
    titles.push(String(msg));
    return 0;
  }) as typeof ChoyMessage.success;
  ChoyMessage.warning = ((msg: string) => {
    levels.push('warning');
    titles.push(String(msg));
    return 0;
  }) as typeof ChoyMessage.warning;
  try {
    const ok = buildStore(
      { Id: 'lang-toast-ok', Code: 'zh_CN', IsActive: true },
      {
        reloadTerminology: async () => ({
          Code: 'zh_CN',
          ModulesScanned: 8,
          ModulesWithPo: 8,
          Upserted: 12,
        }),
      }
    );
    const okWrap = mountLanguageForm(ok.store, 'lang-toast-ok');
    await flushPromises();
    await (okWrap.vm as any).$.setupState.onReloadTerminology('lang-toast-ok', {
      Id: 'lang-toast-ok',
      Code: 'zh_CN',
      IsActive: true,
    });
    await flushPromises();
    okWrap.unmount();

    const missing = buildStore(
      { Id: 'lang-toast-tl', Code: 'tl_PH', IsActive: false },
      {
        reloadTerminology: async () => ({
          Code: 'tl_PH',
          ModulesScanned: 8,
          ModulesWithPo: 0,
          Upserted: 0,
        }),
      }
    );
    const tlWrap = mountLanguageForm(missing.store, 'lang-toast-tl');
    await flushPromises();
    await (tlWrap.vm as any).$.setupState.onReloadTerminology('lang-toast-tl', {
      Id: 'lang-toast-tl',
      Code: 'tl_PH',
      IsActive: false,
    });
    await flushPromises();
    tlWrap.unmount();

    expect(levels).toEqual(['success', 'warning']);
    expect(titles[0]).toMatch(/Synced packaged translations|zh_CN/i);
    expect(titles[1]).toMatch(/tl_PH/i);
  } finally {
    ChoyMessage.success = orig.success;
    ChoyMessage.warning = orig.warning;
  }
});

test('LanguageFormView: missing ReloadTerminology method is handled', async () => {
  const { store } = buildStore({ Id: 'lang-reload-missing', Code: 'zh_HK', IsActive: false });
  delete (store as any).ReloadTerminology;
  const wrapper = mountLanguageForm(store, 'lang-reload-missing');
  await flushPromises();
  await wrapper.find('[data-testid="language-reload-terminology"]').trigger('click');
  await flushPromises();
  expect(wrapper.find('[data-testid="language-reload-terminology"]').exists()).toBe(true);
  wrapper.unmount();
});

test('LanguageFormView: ReloadTerminology rejection surfaces error and clears busy', async () => {
  const toasts: string[] = [];
  const origError = ChoyMessage.error;
  ChoyMessage.error = ((msg: string) => {
    toasts.push(String(msg));
    return 0;
  }) as typeof ChoyMessage.error;
  try {
    let rejectReload!: (err: Error) => void;
    const gate = new Promise<never>((_, reject) => {
      rejectReload = reject;
    });
    const { store, reloadTerminologyCalls } = buildStore(
      { Id: 'lang-reload-fail', Code: 'zh_HK', IsActive: false },
      {
        reloadTerminology: async () => {
          await gate;
        },
      }
    );
    const wrapper = mountLanguageForm(store, 'lang-reload-fail');
    await flushPromises();
    const setupState = (wrapper.vm as any).$.setupState;
    const pending = setupState.onReloadTerminology('lang-reload-fail', {
      Id: 'lang-reload-fail',
      Code: 'zh_HK',
      IsActive: false,
    });
    await flushPromises();
    expect(setupState.reloadBusy).toBe(true);
    rejectReload(new Error('reload denied'));
    let settled: 'ok' | 'err' = 'ok';
    await pending.then(
      () => {
        settled = 'ok';
      },
      () => {
        settled = 'err';
      }
    );
    await flushPromises();
    expect(settled).toBe('ok');
    expect(reloadTerminologyCalls.length).toBe(1);
    expect(toasts).toEqual(['reload denied']);
    expect(setupState.reloadBusy).toBe(false);
    wrapper.unmount();
  } finally {
    ChoyMessage.error = origError;
  }
});

test('LanguageFormView: loading slot hides Activate/Deactivate', async () => {
  const { store } = buildStore({ Id: 'lang-loading', Code: 'zh_HK', IsActive: false });
  stubSfc(ChoyFormView as any, {
    name: 'ChoyFormView',
    setup(_: unknown, { slots }: any) {
      return () =>
        h('div', [
          slots['secondary-actions']?.({
            formData: { Id: 'lang-loading', Code: 'zh_HK', IsActive: false },
            recordId: 'lang-loading',
            loading: true,
          }),
        ]);
    },
  });
  try {
    const wrapper = mountLanguageForm(store, 'lang-loading');
    await flushPromises();
    expect(wrapper.find('[data-testid="language-activate"]').exists()).toBe(false);
    expect(wrapper.find('[data-testid="language-deactivate"]').exists()).toBe(false);
    expect(wrapper.find('[data-testid="language-reload-terminology"]').exists()).toBe(false);
    wrapper.unmount();
  } finally {
    restoreSfc(ChoyFormView as any);
  }
});
