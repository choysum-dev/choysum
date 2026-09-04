// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, test, vi } from 'vitest';
import { config, flushPromises, mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { defineComponent, h, reactive } from 'vue';

const { emitCancelableMock } = vi.hoisted(() => ({
  emitCancelableMock: vi.fn(async () => true),
}));

const ElFormStub = defineComponent({
  name: 'ElFormStub',
  props: {
    model: { type: Object, required: false },
    hideRequiredAsterisk: { type: Boolean, required: false },
  },
  setup(_, { slots, expose }) {
    expose({
      clearValidate: () => {},
      validate: async () => true,
    });
    return () => h('form', { class: 'el-form-stub' }, slots.default?.());
  },
});

config.global.renderStubDefaultSlot = true;

const beginCreate = vi.fn(async (_seed?: any) => {});
const beginDisplay = vi.fn(async (id?: string) => ({ Id: id ?? '1', Name: 'row' }));
const beginEdit = vi.fn();
let displayLoadSeq = 0;

const controllerVm = reactive({
  mode: 'display' as string,
  draft: { Id: '1', Name: 'row' } as Record<string, unknown> | null,
  original: { Id: '1', Name: 'row' } as Record<string, unknown> | null,
  loading: false,
  error: null as unknown,
  result: null as unknown,
});

async function applyDisplay(id?: string) {
  const seq = ++displayLoadSeq;
  const row = { Id: id ?? '1', Name: 'row' };
  // Match formController: only commit when this load is still current.
  await Promise.resolve();
  if (seq !== displayLoadSeq) return row;
  controllerVm.mode = 'display';
  controllerVm.original = row;
  controllerVm.draft = { ...row };
  return row;
}

const routeState = reactive({
  name: 'WidgetDetail' as string | undefined,
  path: '/demo/widgets/1',
  fullPath: '/demo/widgets/1',
  params: { id: '1' } as Record<string, string>,
  query: {} as Record<string, string>,
  meta: {},
});

const routerPush = vi.fn(async () => undefined);
const routerBack = vi.fn();

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: routerPush,
    replace: vi.fn(),
    back: routerBack,
    resolve: vi.fn((loc: { name?: string }) => ({
      name: loc?.name,
      matched: loc?.name === 'WidgetCreate' ? [{ path: '/demo/widgets/new' }] : [],
    })),
    currentRoute: { value: routeState },
  }),
  useRoute: () => routeState,
}));

vi.mock('@/web/web/controllers/formController', () => ({
  createFormController: () => ({
    vm: controllerVm,
    beginCreate,
    beginDisplay,
    beginEdit,
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
    fieldErrors: { value: new Map() },
    afterFlushHandler: vi.fn(),
    reset: vi.fn(),
  }),
}));

vi.mock('@/web/web/composables/useCancelableEmit', () => ({
  useCancelableEmit: () => ({
    emitCancelable: emitCancelableMock,
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
    emitCancelableMock.mockResolvedValue(true);
    setActivePinia(createPinia());
    routeState.name = 'WidgetDetail';
    routeState.path = '/demo/widgets/1';
    routeState.fullPath = '/demo/widgets/1';
    routeState.params = { id: '1' };
    routeState.query = {};
    controllerVm.mode = 'display';
    displayLoadSeq = 0;
    routerPush.mockImplementation(async () => undefined);
    beginDisplay.mockImplementation(async (id?: string) => applyDisplay(id));
    beginCreate.mockImplementation(async (seed?: any) => {
      displayLoadSeq += 1;
      controllerVm.mode = 'create';
      controllerVm.original = null;
      controllerVm.draft = { ...(seed || {}) };
    });
  });

  function mountForm(props: Record<string, unknown> = {}) {
    return mount(OFormView as any, {
      props: {
        store: fakeStore(),
        showHeader: true,
        showActions: true,
        showMessages: false,
        ...props,
      },
      global: {
        stubs: {
          OViewContainer: { template: '<div><slot name="header" /><slot /></div>' },
          OPage: true,
          OBreadcrumb: true,
          'el-button': { template: '<button type="button" v-bind="$attrs"><slot /></button>' },
          'el-icon': true,
          'el-form': ElFormStub,
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

  test('discards stale initializeForm results after rapid route changes', async () => {
    let releaseFirst!: () => void;
    const firstGate = new Promise<void>(resolve => {
      releaseFirst = resolve;
    });
    beginDisplay.mockImplementationOnce(async (id?: string) => {
      const seq = ++displayLoadSeq;
      await firstGate;
      const row = { Id: id ?? '1', Name: 'stale' };
      if (seq !== displayLoadSeq) return row;
      controllerVm.original = row;
      controllerVm.draft = { ...row };
      return row;
    });

    const wrapper = mountForm();
    await flushPromises();
    expect(beginDisplay).toHaveBeenCalledTimes(1);

    routeState.params = { id: '2' };
    routeState.fullPath = '/demo/widgets/2';
    await flushPromises();
    expect(beginDisplay).toHaveBeenCalledWith('2');

    releaseFirst();
    await flushPromises();

    expect(controllerVm.original).toMatchObject({ Id: '2' });
    // Stale run must not emit a second load-success for id 1 after id 2 won.
    const loadSuccessIds = (wrapper.emitted('load-success') || []).map(
      ([payload]: any[]) => payload?.record?.Id ?? null
    );
    expect(loadSuccessIds.filter(id => id === '1').length).toBeLessThanOrEqual(1);
    expect(loadSuccessIds[loadSuccessIds.length - 1]).toBe('2');
    wrapper.unmount();
  });

  test('invalidates in-flight init before nextTick when route changes same turn', async () => {
    let releaseFirst!: () => void;
    const firstGate = new Promise<void>(resolve => {
      releaseFirst = resolve;
    });
    beginDisplay.mockImplementationOnce(async (id?: string) => {
      const seq = ++displayLoadSeq;
      await firstGate;
      const row = { Id: id ?? '1', Name: 'stale-pre-tick' };
      if (seq !== displayLoadSeq) return row;
      controllerVm.original = row;
      controllerVm.draft = { ...row };
      return row;
    });

    const wrapper = mountForm();
    // Change route in the same turn as mount, before the first initializeForm
    // finishes — watcher must bump initializeSeq before awaiting nextTick.
    routeState.params = { id: '2' };
    routeState.fullPath = '/demo/widgets/2';
    releaseFirst();
    await flushPromises();

    expect(beginDisplay.mock.calls.map(c => c[0])).toContain('2');
    expect(controllerVm.original).toMatchObject({ Id: '2' });
    const loadSuccessIds = (wrapper.emitted('load-success') || []).map(
      ([payload]: any[]) => payload?.record?.Id ?? null
    );
    expect(loadSuccessIds[loadSuccessIds.length - 1]).toBe('2');
    wrapper.unmount();
  });

  test('shows New and refreshes using route-resolved effective record id', async () => {
    const wrapper = mountForm({ createAction: '/demo/widgets/new' });
    await flushPromises();

    const newBtn = wrapper.findAll('button').find(b => b.text().includes('New'));
    expect(newBtn).toBeTruthy();

    beginDisplay.mockClear();
    await (wrapper.vm as any).refresh();
    await flushPromises();
    expect(beginDisplay).toHaveBeenCalledWith('1');

    routerPush.mockClear();
    await newBtn!.trigger('click');
    await flushPromises();
    expect(routerPush).toHaveBeenCalledWith('/demo/widgets/new');
    wrapper.unmount();
  });

  test('create navigation swallows push failures', async () => {
    routerPush.mockRejectedValueOnce(new Error('nav failed'));
    const wrapper = mountForm({ createAction: '/demo/widgets/new' });
    await flushPromises();

    const newBtn = wrapper.findAll('button').find(b => b.text().includes('New'));
    expect(newBtn).toBeTruthy();
    await newBtn!.trigger('click');
    await flushPromises();
    expect(routerPush).toHaveBeenCalled();
    wrapper.unmount();
  });

  test('handleCreate no-ops when create action is disabled', async () => {
    const wrapper = mountForm({ createAction: '' });
    await flushPromises();
    routerPush.mockClear();
    await (wrapper.vm as any).$.setupState.handleCreate();
    expect(routerPush).not.toHaveBeenCalled();
    wrapper.unmount();
  });

  test('cancel from edit reloads effective route record id', async () => {
    const wrapper = mountForm();
    await flushPromises();

    const editBtn = wrapper.findAll('button').find(b => b.text().includes('Edit'));
    expect(editBtn).toBeTruthy();
    await editBtn!.trigger('click');
    controllerVm.mode = 'edit';
    await flushPromises();

    beginDisplay.mockClear();
    const cancelBtn = wrapper.findAll('button').find(b => b.text().includes('Cancel'));
    expect(cancelBtn).toBeTruthy();
    await cancelBtn!.trigger('click');
    await flushPromises();
    expect(beginDisplay).toHaveBeenCalledWith('1');
    wrapper.unmount();
  });

  test('cancel from edit without effective id skips display reload', async () => {
    routeState.params = {};
    const wrapper = mountForm({ resolveRecordIdFromRoute: false });
    await flushPromises();
    controllerVm.mode = 'edit';

    beginDisplay.mockClear();
    const cancelBtn = wrapper.findAll('button').find(b => b.text().includes('Cancel'));
    expect(cancelBtn).toBeTruthy();
    await cancelBtn!.trigger('click');
    await flushPromises();
    expect(beginDisplay).not.toHaveBeenCalled();
    wrapper.unmount();
  });

  test('create-mode cancel goes back', async () => {
    routeState.name = 'WidgetCreate';
    routeState.params = {};
    routeState.path = '/demo/widgets/new';
    routeState.fullPath = '/demo/widgets/new';
    const wrapper = mountForm();
    await flushPromises();
    expect(controllerVm.mode).toBe('create');

    const cancelBtn = wrapper.findAll('button').find(b => b.text().includes('Cancel'));
    expect(cancelBtn).toBeTruthy();
    await cancelBtn!.trigger('click');
    expect(routerBack).toHaveBeenCalled();
    wrapper.unmount();
  });

  test('refresh without effective id emits null record', async () => {
    routeState.params = {};
    routeState.name = 'WidgetCreate';
    const wrapper = mountForm({ resolveRecordIdFromRoute: false });
    await flushPromises();

    await (wrapper.vm as any).refresh();
    await flushPromises();
    expect(wrapper.emitted('refresh-success')?.[0]?.[0]).toEqual({ record: null });
    wrapper.unmount();
  });

  test('normalizeRecordId ignores blank sentinel strings from route', async () => {
    routeState.params = { id: 'null' };
    const wrapper = mountForm();
    await flushPromises();
    expect(beginCreate).toHaveBeenCalled();
    wrapper.unmount();
  });

  test('normalizeRecordId ignores undefined sentinel from route', async () => {
    routeState.params = { id: 'undefined' };
    const wrapper = mountForm();
    await flushPromises();
    expect(beginCreate).toHaveBeenCalled();
    wrapper.unmount();
  });

  test('skips initializeForm when before-load is cancelled', async () => {
    emitCancelableMock.mockResolvedValueOnce(false);
    const wrapper = mountForm();
    await flushPromises();
    expect(beginDisplay).not.toHaveBeenCalled();
    expect(beginCreate).not.toHaveBeenCalled();
    wrapper.unmount();
  });

  test('keeps display mode for initialValues-only preview', async () => {
    routeState.params = {};
    const wrapper = mountForm({
      viewMode: 'display',
      initialValues: { Name: 'preview' },
    });
    await flushPromises();
    expect(beginCreate).toHaveBeenCalled();
    expect(controllerVm.mode).toBe('display');
    wrapper.unmount();
  });

  test('emits action-error when display load fails', async () => {
    beginDisplay.mockRejectedValueOnce(new Error('load fail'));
    const wrapper = mountForm();
    await flushPromises();
    expect(wrapper.emitted('action-error')?.[0]?.[0]).toMatchObject({
      action: 'load',
      error: expect.objectContaining({ message: 'load fail' }),
    });
    wrapper.unmount();
  });

  test('wraps non-Error load failures in action-error', async () => {
    beginDisplay.mockRejectedValueOnce('load boom');
    const wrapper = mountForm();
    await flushPromises();
    expect(wrapper.emitted('action-error')?.[0]?.[0]).toMatchObject({
      action: 'load',
      error: expect.objectContaining({ message: 'load boom' }),
    });
    wrapper.unmount();
  });

  test('suppresses load action-error when init becomes stale before failure', async () => {
    let release!: () => void;
    const gate = new Promise<void>(resolve => {
      release = resolve;
    });
    beginDisplay.mockImplementationOnce(async () => {
      await gate;
      throw new Error('late fail');
    });

    const wrapper = mountForm();
    await flushPromises();

    routeState.params = { id: '2' };
    routeState.fullPath = '/demo/widgets/2';
    await flushPromises();

    release();
    await flushPromises();

    expect(wrapper.emitted('action-error')).toBeUndefined();
    wrapper.unmount();
  });

  test('discards stale beginCreate completion after route gains id', async () => {
    let releaseCreate!: () => void;
    const createGate = new Promise<void>(resolve => {
      releaseCreate = resolve;
    });
    routeState.params = {};
    beginCreate.mockImplementationOnce(async (seed?: any) => {
      await createGate;
      controllerVm.mode = 'create';
      controllerVm.original = null;
      controllerVm.draft = { ...(seed || {}) };
    });

    const wrapper = mountForm();
    await flushPromises();
    routeState.params = { id: '9' };
    routeState.fullPath = '/demo/widgets/9';
    await flushPromises();
    releaseCreate();
    await flushPromises();

    expect(beginDisplay).toHaveBeenCalledWith('9');
    expect(wrapper.emitted('load-success')?.at(-1)?.[0]).toMatchObject({ record: { Id: '9' } });
    wrapper.unmount();
  });

  test('resolves effective id from route.params.recordId', async () => {
    routeState.params = { recordId: 'rec-5' };
    const wrapper = mountForm();
    await flushPromises();
    expect(beginDisplay).toHaveBeenCalledWith('rec-5');
    wrapper.unmount();
  });

  test('resolves effective id from route.query.id', async () => {
    routeState.params = {};
    routeState.query = { id: 'q-7' };
    const wrapper = mountForm();
    await flushPromises();
    expect(beginDisplay).toHaveBeenCalledWith('q-7');
    wrapper.unmount();
  });

  test('enters edit mode when viewMode prop requests edit', async () => {
    const wrapper = mountForm({ recordId: '1', viewMode: 'edit' });
    await flushPromises();
    expect(beginEdit).toHaveBeenCalled();
    wrapper.unmount();
  });

  test('emits null load-success when display record payload is empty', async () => {
    beginDisplay.mockImplementationOnce(async () => {
      controllerVm.mode = 'display';
      controllerVm.original = null;
      controllerVm.draft = null;
      return null;
    });
    const wrapper = mountForm();
    await flushPromises();
    expect(wrapper.emitted('load-success')?.at(-1)?.[0]).toEqual({ record: null });
    wrapper.unmount();
  });

  test('skips change emit when route identity changes during load-success', async () => {
    let rerouted = false;
    const wrapper = mount(OFormView as any, {
      props: {
        store: fakeStore(),
        showHeader: true,
        showActions: true,
        showMessages: false,
      },
      attrs: {
        onLoadSuccess: () => {
          if (rerouted) return;
          rerouted = true;
          routeState.params = { id: '2' };
          routeState.fullPath = '/demo/widgets/2';
        },
      },
      global: {
        stubs: {
          OViewContainer: { template: '<div><slot name="header" /><slot /></div>' },
          OPage: true,
          OBreadcrumb: true,
          'el-button': { template: '<button type="button" v-bind="$attrs"><slot /></button>' },
          'el-icon': true,
          'el-form': ElFormStub,
        },
        directives: {
          loading: {
            mounted() {},
            updated() {},
          },
        },
      },
    });
    await flushPromises();
    expect(beginDisplay.mock.calls.map(c => c[0])).toEqual(['1', '2']);
    expect(wrapper.emitted('change')?.length ?? 0).toBeGreaterThan(0);
    wrapper.unmount();
  });

  test('reloads when route name changes', async () => {
    routeState.name = 'WidgetDetail';
    const wrapper = mountForm();
    await flushPromises();
    beginDisplay.mockClear();

    routeState.name = 'WidgetCreate';
    routeState.params = {};
    routeState.path = '/demo/widgets/new';
    routeState.fullPath = '/demo/widgets/new';
    await flushPromises();

    expect(beginCreate).toHaveBeenCalled();
    wrapper.unmount();
  });

  test('treats undefined route name as empty watcher key', async () => {
    routeState.name = undefined;
    const wrapper = mountForm();
    await flushPromises();
    expect(beginDisplay).toHaveBeenCalledWith('1');
    wrapper.unmount();
  });
});
