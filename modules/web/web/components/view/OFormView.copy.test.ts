// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, test, vi } from 'vitest';
import { config, flushPromises, shallowMount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { reactive } from 'vue';

config.global.renderStubDefaultSlot = true;

const beginCreate = vi.fn(async (_seed?: any) => {});
const beginDisplay = vi.fn(async () => ({ Id: '1', Name: 'orig' }));
const onchangeReset = vi.fn();
const onchangeAggReset = vi.fn();

const controllerVm = reactive({
  mode: 'display' as string,
  draft: { Id: '1', Name: 'orig' } as Record<string, unknown> | null,
  original: { Id: '1', Name: 'orig' } as Record<string, unknown> | null,
  loading: false,
  error: null as unknown,
  result: null as unknown,
});

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: vi.fn(),
    replace: vi.fn(),
    currentRoute: { value: { path: '/demo/widget/1', params: { recordId: '1' }, query: {}, meta: {} } },
  }),
  useRoute: () => ({ path: '/demo/widget/1', params: { recordId: '1' }, query: {}, meta: {} }),
}));

vi.mock('@/web/web/controllers/formController', () => ({
  createFormController: () => ({
    vm: controllerVm,
    beginCreate,
    beginDisplay,
    beginEdit: vi.fn(),
    reset: vi.fn(),
    validate: vi.fn(async () => ({ valid: true, errors: [] })),
    submit: vi.fn(async () => null),
    delete: vi.fn(async () => null),
    provideToChildren: vi.fn(),
  }),
}));

vi.mock('@/web/web/composables/useOnchange', () => ({
  provideOnchange: () => ({
    reset: onchangeReset,
    resume: vi.fn(),
    pause: vi.fn(),
    registerAfterFlush: vi.fn(),
    unregisterAfterFlush: vi.fn(),
  }),
}));

vi.mock('@/web/web/composables/useOnchangeAggregation', () => ({
  useOnchangeAggregation: () => ({
    lastOnchangeResult: { value: null },
    fieldErrors: { value: {} },
    afterFlushHandler: vi.fn(),
    reset: onchangeAggReset,
  }),
}));

vi.mock('@/web/web/composables/useCancelableEmit', () => ({
  useCancelableEmit: () => ({
    emitCancelable: vi.fn(async () => true),
  }),
}));

vi.mock('@/web/web/stores/breadcrumbStore', () => ({
  useBreadcrumbStore: () => ({
    breadcrumbStack: [],
  }),
}));

vi.mock('@/web/web/i18n', () => ({
  createTranslate: () => ({
    _t: (msg: string) => msg,
    _lt: (msg: string) => ({ src: msg }),
  }),
}));

import OFormView from './OFormView.vue';

function fakeStore() {
  return {
    fullModelName: 'demo.Widget',
    storeId: 'demo.Widget',
    fieldsMetadata: { Name: { type: 'varchar' } },
    state: { queryState: {}, result: undefined, selection: [], planCache: new Map() },
    setContext: vi.fn(),
    getContext: vi.fn(() => ({})),
    withContext: vi.fn(async (_ctx: any, fn: any) => fn()),
    DefaultGet: vi.fn(async (seed: any) => ({ ...(seed || {}), Code: 'server' })),
  } as any;
}

describe('OFormView handleCopy awaits beginCreate (FD-4)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setActivePinia(createPinia());
    controllerVm.mode = 'display';
    controllerVm.draft = { Id: '1', Name: 'orig' };
    controllerVm.original = { Id: '1', Name: 'orig' };
    controllerVm.loading = false;
    controllerVm.error = null;
    controllerVm.result = null;
    beginCreate.mockImplementation(async (seed?: any) => {
      controllerVm.mode = 'create';
      controllerVm.original = null;
      controllerVm.draft = { ...(seed || {}) };
    });
  });

  function mountForm() {
    return shallowMount(OFormView as any, {
      props: {
        store: fakeStore(),
        recordId: '1',
        showHeader: true,
        showActions: true,
        showMessages: false,
        resolveRecordIdFromRoute: false,
      },
      global: {
        stubs: {
          OViewContainer: true,
          OPage: true,
          OBreadcrumb: true,
          'el-button': true,
          'el-icon': true,
          'el-form': true,
        },
        directives: {
          loading: {
            mounted() {},
            updated() {},
          },
        },
      },
    });
  }

  test('Copy action awaits beginCreate with Id stripped from seed', async () => {
    const wrapper = mountForm();
    await flushPromises();

    expect(typeof (wrapper.vm as any).copy).toBe('function');
    await (wrapper.vm as any).copy();
    await flushPromises();

    expect(beginCreate).toHaveBeenCalledTimes(1);
    const seed = beginCreate.mock.calls[0]?.[0] as Record<string, unknown>;
    expect(seed).toMatchObject({ Name: 'orig' });
    expect(seed).not.toHaveProperty('Id');
    expect(controllerVm.mode).toBe('create');
    expect(onchangeAggReset).toHaveBeenCalled();
    expect(onchangeReset).toHaveBeenCalled();
    expect(wrapper.emitted('copy')?.length).toBe(1);
    expect(wrapper.emitted('mode-change')?.at(-1)).toEqual([{ mode: 'create' }]);
    expect((wrapper.vm as any).getFormData()).toMatchObject({ Name: 'orig' });
    expect((wrapper.vm as any).getViewMode()).toBe('create');
    expect((wrapper.vm as any).isLoading()).toBe(false);

    wrapper.unmount();
  });

  test('Copy is a no-op when there is no original record', async () => {
    controllerVm.original = null;
    controllerVm.draft = null;
    const wrapper = mountForm();
    await flushPromises();

    await (wrapper.vm as any).copy();
    await flushPromises();

    expect(beginCreate).not.toHaveBeenCalled();
    expect(wrapper.emitted('copy')).toBeUndefined();
    wrapper.unmount();
  });
});
