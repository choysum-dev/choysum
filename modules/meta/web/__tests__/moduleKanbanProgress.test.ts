// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, shallowMount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const onTips = vi.fn(async () => undefined);
const subscribeModuleOp = vi.fn(() => ({}));

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
}));

vi.mock('@/auth/web/composables/usePermission', () => ({
  usePermission: () => ({
    canRoute: () => true,
    hasAction: () => true,
  }),
}));

vi.mock('@/web/web/composables/usePageContext', () => ({
  resolvePageStore: (store: unknown) => store,
}));

vi.mock('@/core/web/tip', () => ({
  onTips: (...args: unknown[]) => onTips(...args),
  subscribeModuleOp: (...args: unknown[]) => subscribeModuleOp(...args),
}));

vi.mock('element-plus', async () => {
  const actual = await vi.importActual<typeof import('element-plus')>('element-plus');
  return {
    ...actual,
    ElMessage: {
      warning: vi.fn(),
      error: vi.fn(),
      success: vi.fn(),
    },
  };
});

import { ElMessage } from 'element-plus';
import ModuleKanbanView from '../views/ModuleKanbanView.vue';

const stubs = {
  OKanbanView: { template: '<div />' },
  OVirtualField: { template: '<div />' },
  OSearchView: { template: '<div />' },
  ElDialog: { template: '<div><slot /><slot name="footer" /></div>' },
  ElButton: { template: '<button><slot /></button>' },
  ElTooltip: { template: '<div><slot /></div>' },
  ElButtonGroup: { template: '<div><slot /></div>' },
  ElTag: { template: '<span />' },
  ElAlert: { template: '<div />' },
  ElDivider: { template: '<div />' },
  ElCheckbox: { template: '<input type="checkbox" />' },
  ElSkeleton: { template: '<div />' },
};

describe('ModuleKanbanView C1 progress integration', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    onTips.mockReset();
    onTips.mockResolvedValue(undefined);
    subscribeModuleOp.mockReset();
    subscribeModuleOp.mockReturnValue({});
    vi.mocked(ElMessage.warning).mockClear();
    vi.mocked(ElMessage.error).mockClear();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  function mountView(moduleStore?: Record<string, unknown>) {
    const store = {
      state: { queryState: {}, result: undefined, selection: [], planCache: new Map() },
      setContext: vi.fn(),
      getContext: vi.fn(),
      withContext: vi.fn(),
      RequestSync: vi.fn(async () => 'sync-job'),
    };
    const modStore = {
      PlanOperation: vi.fn(async () => ({
        baseRevision: '1',
        affectedModules: [{ moduleName: 'demo' }],
        risks: [],
        blockers: [],
      })),
      RequestInstall: vi.fn(async () => 'job-install'),
      RequestUninstall: vi.fn(async () => 'job-uninstall'),
      RequestUpgrade: vi.fn(async () => 'job-upgrade'),
      GetOpStatus: vi.fn(async () => ({ status: 'succeeded', resultStatus: 'SUCCEEDED' })),
      ...(moduleStore || {}),
    };
    const wrapper = shallowMount(ModuleKanbanView, {
      props: { store: store as any, moduleStore: modStore as any },
      global: { stubs },
    });
    return { wrapper, modStore, vm: wrapper.vm as any };
  }

  it('opens an action, watches install progress, and stops on close/unmount', async () => {
    const { wrapper, modStore, vm } = mountView();
    await flushPromises();

    await vm.onActionClick('install', { ModuleName: 'demo', InstalledStatus: 'uninstalled', Available: true });
    await flushPromises();
    expect(vm.dialogVisible).toBe(true);
    expect(modStore.PlanOperation).toHaveBeenCalled();

    await vm.submitOperation();
    await flushPromises();
    expect(modStore.RequestInstall).toHaveBeenCalledWith('demo', false);
    expect(modStore.GetOpStatus).toHaveBeenCalledWith('job-install');
    expect(vm.dialogStep).toBe('result');

    vm.onDialogClose();
    vm.resetDialog();
    expect(vm.dialogStep).toBe('plan');

    wrapper.unmount();
  });

  it('covers uninstall/upgrade submit paths and request failure', async () => {
    const { vm, modStore } = mountView({
      GetOpStatus: vi.fn(async () => ({ status: 'queued' })),
    });
    await flushPromises();

    await vm.onActionClick('uninstall', { ModuleName: 'demo', InstalledStatus: 'installed' });
    await flushPromises();
    await vm.submitOperation();
    await flushPromises();
    expect(modStore.RequestUninstall).toHaveBeenCalled();

    await vm.onActionClick('upgrade', { ModuleName: 'demo', InstalledStatus: 'installed' });
    await flushPromises();
    await vm.submitOperation();
    await flushPromises();
    expect(modStore.RequestUpgrade).toHaveBeenCalled();

    modStore.RequestInstall.mockRejectedValueOnce(new Error('enqueue failed'));
    await vm.onActionClick('install', { ModuleName: 'demo', InstalledStatus: 'uninstalled', Available: true });
    await flushPromises();
    await vm.submitOperation();
    await flushPromises();
    expect(vm.dialogStep).toBe('result');
    expect(vm.opStatus?.errorCode).toBe('REQUEST_FAILED');
    expect(ElMessage.error).toHaveBeenCalled();
  });

  it('invokes timeout and transient hooks through a long-running watch', async () => {
    const { vm, modStore } = mountView({
      GetOpStatus: vi.fn(async () => ({ status: 'queued' })),
    });
    onTips.mockImplementation(async (_stream, _cb, signal?: AbortSignal) => {
      await new Promise<void>((resolve) => {
        signal?.addEventListener('abort', () => resolve(), { once: true });
      });
    });
    await vm.onActionClick('install', { ModuleName: 'demo', InstalledStatus: 'uninstalled', Available: true });
    await flushPromises();
    const pending = vm.submitOperation();
    await Promise.resolve();
    await Promise.resolve();
    await vi.advanceTimersByTimeAsync(10 * 60 * 1000);
    await pending;
    expect(vm.dialogStep).toBe('result');
    expect(ElMessage.warning).toHaveBeenCalled();
    expect(modStore.GetOpStatus).toHaveBeenCalled();
  });

  it('routes hard GetOpStatus failures through ElMessage.error', async () => {
    const { vm } = mountView({
      GetOpStatus: vi.fn(async () => {
        throw new Error('status hard fail');
      }),
    });
    onTips.mockResolvedValue(undefined);
    await vm.onActionClick('install', { ModuleName: 'demo', InstalledStatus: 'uninstalled', Available: true });
    await flushPromises();
    await vm.submitOperation();
    await flushPromises();
    expect(ElMessage.error).toHaveBeenCalledWith('status hard fail');
  });
});
