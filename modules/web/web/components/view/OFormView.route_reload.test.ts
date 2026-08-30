// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, test, vi } from 'vitest';
import { config, flushPromises, shallowMount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { reactive } from 'vue';

config.global.renderStubDefaultSlot = true;

const beginCreate = vi.fn(async (_seed?: any) => {});
const beginDisplay = vi.fn(async (id?: string) => ({ Id: id ?? '1', Name: 'row' }));

const controllerVm = reactive({
  mode: 'display' as string,
  draft: { Id: '1', Name: 'row' } as Record<string, unknown> | null,
  original: { Id: '1', Name: 'row' } as Record<string, unknown> | null,
  loading: false,
  error: null as unknown,
  result: null as unknown,
});

const routeState = reactive({
  name: 'WidgetDetail' as string | undefined,
  path: '/demo/widgets/1',
  fullPath: '/demo/widgets/1',
  params: { id: '1' } as Record<string, string>,
  query: {} as Record<string, string>,
  meta: {},
});

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: vi.fn(),
    replace: vi.fn(),
    resolve: vi.fn((loc: { name?: string }) => ({ name: loc?.name, matched: [] })),
    currentRoute: { value: routeState },
  }),
  useRoute: () => routeState,
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
    reset: vi.fn(),
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
    reset: vi.fn(),
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
    DefaultGet: vi.fn(async (seed: any) => ({ ...(seed || {}) })),
  } as any;
}

describe('OFormView reloads when route identity changes', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setActivePinia(createPinia());
    routeState.name = 'WidgetDetail';
    routeState.path = '/demo/widgets/1';
    routeState.fullPath = '/demo/widgets/1';
    routeState.params = { id: '1' };
    routeState.query = {};
    controllerVm.mode = 'display';
    beginDisplay.mockImplementation(async (id?: string) => {
      const row = { Id: id ?? '1', Name: 'row' };
      controllerVm.original = row;
      controllerVm.draft = { ...row };
      return row;
    });
    beginCreate.mockImplementation(async (seed?: any) => {
      controllerVm.mode = 'create';
      controllerVm.original = null;
      controllerVm.draft = { ...(seed || {}) };
    });
  });

  function mountForm(props: Record<string, unknown> = {}) {
    return shallowMount(OFormView as any, {
      props: {
        store: fakeStore(),
        showHeader: true,
        showActions: true,
        showMessages: false,
        ...props,
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

  test('reloads display when route params.id changes', async () => {
    const wrapper = mountForm();
    await flushPromises();
    expect(beginDisplay).toHaveBeenCalledWith('1');

    routeState.params = { id: '2' };
    routeState.path = '/demo/widgets/2';
    routeState.fullPath = '/demo/widgets/2';
    await flushPromises();

    expect(beginDisplay.mock.calls.map(c => c[0])).toEqual(['1', '2']);
    wrapper.unmount();
  });

  test('switches to create when navigating to a Create route without id', async () => {
    const wrapper = mountForm();
    await flushPromises();
    expect(beginDisplay).toHaveBeenCalledTimes(1);

    routeState.name = 'WidgetCreate';
    routeState.params = {};
    routeState.path = '/demo/widgets/new';
    routeState.fullPath = '/demo/widgets/new';
    await flushPromises();

    expect(beginCreate).toHaveBeenCalled();
    expect(controllerVm.mode).toBe('create');
    wrapper.unmount();
  });

  test('does not follow route when resolveRecordIdFromRoute is false', async () => {
    const wrapper = mountForm({
      recordId: 'fixed-1',
      resolveRecordIdFromRoute: false,
    });
    await flushPromises();
    expect(beginDisplay).toHaveBeenCalledWith('fixed-1');
    beginDisplay.mockClear();

    routeState.params = { id: '9' };
    routeState.fullPath = '/demo/widgets/9';
    await flushPromises();

    expect(beginDisplay).not.toHaveBeenCalled();
    wrapper.unmount();
  });
});
