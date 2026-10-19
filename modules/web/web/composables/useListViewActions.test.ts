// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, defineComponent, h, ref } from 'vue';
import * as VueRouter from 'vue-router';

import { flushPromises, fnRecorder, mountApp } from '@/web/web/__tests__/mountApp';
import { resolveConfirmChoy } from '@/web/web/composables/confirmChoyAction';
import { ChoyMessage } from '@/web/web/composables/useChoyMessage';
import { useListViewActions } from '@/web/web/composables/useListViewActions';

const createFeStubRouter = (VueRouter as any).createFeStubRouter;

const origSuccess = ChoyMessage.success;
const origError = ChoyMessage.error;
const origWarning = ChoyMessage.warning;

function mountListActions(opts?: {
  createAction?: string;
  selectedItems?: any[];
  applyImpl?: () => Promise<void>;
  deleteImpl?: () => Promise<void>;
}) {
  const { router } = createFeStubRouter({
    route: { path: '/list', fullPath: '/list', params: {}, query: {}, meta: {} },
  });

  const selectedItems = ref<any[]>(opts?.selectedItems ?? [{ Id: 'a' }]);
  const items = computed(() => [{ Id: 'a' }, { Id: 'b' }]);
  const store = {
    state: { result: { total: 2 } },
    Delete: fnRecorder(opts?.deleteImpl ?? (async () => {})),
  };
  const controller = {
    apply: fnRecorder(opts?.applyImpl ?? (async () => {})),
  };
  const selection = { clear: fnRecorder(() => {}) };
  const loadData = fnRecorder(async () => {});
  const afterLayoutRecompute = fnRecorder(async () => {});
  const emitCancelable = fnRecorder(async () => true);
  const emit = fnRecorder((_e?: string, _p?: any) => {});

  let api: ReturnType<typeof useListViewActions<any>> | null = null;

  const Host = defineComponent({
    setup() {
      api = useListViewActions({
        store: store as any,
        controller: controller as any,
        router,
        resolvedCreateAction: computed(() => opts?.createAction),
        selectedItems,
        items,
        selection,
        emitCancelable,
        emit,
        loadData,
        afterLayoutRecompute,
        _t: (k: string, ...args: any[]) => (args.length ? `${k}:${args.join(',')}` : k),
      });
      return () => h('div');
    },
  });

  const mounted = mountApp(Host);
  return {
    ...mounted,
    api: () => api!,
    store,
    controller,
    selection,
    loadData,
    emitCancelable,
    emit,
    router,
    selectedItems,
  };
}

describe('useListViewActions', () => {
  beforeEach(() => {
    ChoyMessage.success = fnRecorder(() => {}) as any;
    ChoyMessage.error = fnRecorder(() => {}) as any;
    ChoyMessage.warning = fnRecorder(() => {}) as any;
  });

  afterEach(() => {
    ChoyMessage.success = origSuccess;
    ChoyMessage.error = origError;
    ChoyMessage.warning = origWarning;
  });

  test('handleRefresh respects before-refresh and succeeds', async () => {
    const m = mountListActions();
    m.emitCancelable.mockImplementation(async () => false);
    await m.api().handleRefresh();
    expect(m.controller.apply.calls.length).toBe(0);

    m.emitCancelable.mockImplementation(async () => true);
    await m.api().handleRefresh();
    await flushPromises();
    expect(m.controller.apply.calls.length).toBe(1);
    expect(m.emit.calls.some(c => c[0] === 'refresh-success')).toBe(true);
    expect((ChoyMessage.success as any).calls.length).toBe(1);
    m.unmount();
  });

  test('handleRefresh failure emits action-error', async () => {
    const m = mountListActions({
      applyImpl: async () => {
        throw 'refresh boom';
      },
    });
    await m.api().handleRefresh();
    await flushPromises();
    expect(m.emit.calls.some(c => c[0] === 'action-error' && (c[1] as any)?.action === 'refresh')).toBe(true);
    expect((ChoyMessage.error as any).calls.length).toBe(1);
    m.unmount();
  });

  test('handleCreate no-op and navigation error', async () => {
    const m = mountListActions({ createAction: undefined });
    await m.api().handleCreate();

    const m2 = mountListActions({ createAction: '/new' });
    m2.router.push = fnRecorder(async () => {
      throw new Error('nav');
    }) as any;
    await m2.api().handleCreate();
    expect(m2.emit.calls.some(c => c[0] === 'action-error' && (c[1] as any)?.action === 'create')).toBe(true);
    m.unmount();
    m2.unmount();
  });

  test('handleCreate pushes resolved create action', async () => {
    const m = mountListActions({ createAction: '/widgets/new' });
    const push = fnRecorder(async () => {});
    m.router.push = push as any;
    await m.api().handleCreate();
    expect(push.calls[0]?.[0]).toBe('/widgets/new');
    m.unmount();
  });

  test('handleDelete empty selection is no-op', async () => {
    const m = mountListActions({ selectedItems: [] });
    await m.api().handleDelete();
    expect(m.store.Delete.calls.length).toBe(0);
    m.unmount();
  });

  test('handleDelete cancel/dismiss and missing ids warning', async () => {
    const m = mountListActions({ selectedItems: [{ Name: 'no-id' }] });
    const pending = m.api().handleDelete();
    resolveConfirmChoy('dismiss');
    await pending;

    const pending2 = m.api().handleDelete();
    resolveConfirmChoy('cancel');
    await pending2;

    const pending3 = m.api().handleDelete();
    resolveConfirmChoy('confirm');
    await pending3;
    expect((ChoyMessage.warning as any).calls.length).toBe(1);
    m.unmount();
  });

  test('handleDelete refuses mixed selection with and without Ids', async () => {
    const m = mountListActions({
      selectedItems: [{ Id: 'a' }, { Name: 'no-id' }],
    });
    const pending = m.api().handleDelete();
    resolveConfirmChoy('confirm');
    await pending;
    expect(m.store.Delete.calls.length).toBe(0);
    expect((ChoyMessage.warning as any).calls.length).toBe(1);
    m.unmount();
  });

  test('handleDelete maps selection shapes and succeeds', async () => {
    const m = mountListActions({
      selectedItems: [
        { type: 'record', record: { Id: 'r1' } },
        { kind: 'record', payload: { Id: 'r2' } },
        { payload: { Id: 'r3' } },
        { record: { Id: 'r4' } },
        { Id: 'r5' },
      ],
    });
    m.emitCancelable.mockImplementation(async () => false);
    let pending = m.api().handleDelete();
    resolveConfirmChoy('confirm');
    await pending;
    expect(m.store.Delete.calls.length).toBe(0);

    m.emitCancelable.mockImplementation(async () => true);
    pending = m.api().handleDelete();
    resolveConfirmChoy('confirm');
    await pending;
    expect(m.store.Delete.calls.length).toBe(1);
    expect(m.selection.clear.calls.length).toBe(1);
    expect(m.loadData.calls.length).toBe(1);
    expect(m.emit.calls.some(c => c[0] === 'delete-success')).toBe(true);
    m.unmount();
  });

  test('handleDelete failure emits action-error', async () => {
    const m = mountListActions({
      selectedItems: [{ Id: 'x' }],
      deleteImpl: async () => {
        throw new Error('del');
      },
    });
    const pending = m.api().handleDelete();
    resolveConfirmChoy('confirm');
    await pending;
    expect(m.emit.calls.some(c => c[0] === 'action-error' && (c[1] as any)?.action === 'delete')).toBe(true);
    expect((ChoyMessage.error as any).calls.length).toBe(1);
    expect(m.api().deleteLoading.value).toBe(false);
    m.unmount();
  });
});
