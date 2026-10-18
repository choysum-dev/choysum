// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, defineComponent, h, reactive, ref } from 'vue';
import * as VueRouter from 'vue-router';
import { createPinia, setActivePinia } from 'pinia';

import { flushPromises, fnRecorder, mountApp } from '@/web/web/__tests__/mountApp';
import { resolveConfirmChoy, useConfirmChoyStore } from '@/web/web/composables/confirmChoyAction';
import { ChoyMessage } from '@/web/web/composables/useChoyMessage';
import { useFormViewLifecycle } from '@/web/web/composables/useFormViewLifecycle';
import type { ViewMode } from '@/web/web/components/view/viewScopeKeys';
import { useBreadcrumbStore } from '@/web/web/stores/breadcrumbStore';

const createFeStubRouter = (VueRouter as any).createFeStubRouter;

const origSuccess = ChoyMessage.success;
const origError = ChoyMessage.error;
const origWarn = console.warn;

function makeController() {
  const ctrl = {
    vm: reactive({ mode: 'display' as ViewMode, original: null as any, draft: null as any }),
    beginDisplay: fnRecorder(async (_id: string) => {}),
    beginCreate: fnRecorder(async (_seed: any) => {}),
    beginEdit: fnRecorder(() => {
      ctrl.vm.mode = 'edit';
    }),
    reset: fnRecorder(() => {}),
    delete: fnRecorder(async () => {}),
  };
  return ctrl;
}

type MutableIdentity = {
  recordId?: string;
  viewMode?: ViewMode;
  initialValues?: Record<string, unknown>;
  onchangeSessionId?: string;
};

type MountOpts = {
  identity?: Partial<MutableIdentity>;
  route?: Record<string, unknown>;
  resolveRecordIdFromRoute?: boolean;
  createAction?: string;
  showMessages?: boolean;
  viewMode?: ViewMode;
  onAfterInit?: () => void;
  watchIdentity?: boolean;
};

function mountLifecycle(opts: MountOpts = {}) {
  const { router, route } = createFeStubRouter({
    route: {
      name: 'Detail',
      path: '/app/items/rec-1',
      fullPath: '/app/items/rec-1',
      params: { id: 'rec-1' },
      query: {},
      meta: {},
      ...(opts.route || {}),
    },
  });

  const identity = reactive<MutableIdentity>({
    recordId: undefined,
    viewMode: opts.viewMode ?? 'display',
    initialValues: { Name: 'seed' },
    onchangeSessionId: undefined,
    ...(opts.identity || {}),
  });

  const controller = makeController();
  const onchangeCtrl = {
    reset: fnRecorder(() => {}),
    pause: fnRecorder(() => {}),
    resume: fnRecorder(() => {}),
  };
  const resetOnchangeAgg = fnRecorder(() => {});
  const emitCancelable = fnRecorder(async () => true);
  const emit = fnRecorder((_event?: string, _payload?: any) => {});
  const exposedFormData = computed(() => ({ Name: 'draft' }));

  let api: ReturnType<typeof useFormViewLifecycle<any>> | null = null;

  const Host = defineComponent({
    setup() {
      api = useFormViewLifecycle({
        identity,
        route,
        router,
        controller: controller as any,
        resolvedResolveRecordIdFromRoute: computed(() => opts.resolveRecordIdFromRoute ?? true),
        resolvedShowMessages: computed(() => opts.showMessages ?? false),
        resolvedCreateAction: computed(() => opts.createAction),
        viewMode: computed(() => identity.viewMode ?? 'display'),
        exposedFormData: exposedFormData as any,
        onchangeCtrl: onchangeCtrl as any,
        resetOnchangeAgg,
        emitCancelable,
        emit,
        _t: (k: string) => k,
      });
      if (opts.watchIdentity !== false) {
        api.watchFormIdentity(opts.onAfterInit);
      }
      return () => h('div');
    },
  });

  const mounted = mountApp(Host, { plugins: [createPinia(), router] });

  return {
    ...mounted,
    api: () => api!,
    identity,
    route,
    router,
    controller,
    onchangeCtrl,
    resetOnchangeAgg,
    emitCancelable,
    emit,
  };
}

describe('useFormViewLifecycle', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    ChoyMessage.success = fnRecorder(() => {}) as any;
    ChoyMessage.error = fnRecorder(() => {}) as any;
    console.warn = fnRecorder(() => {}) as any;
  });

  afterEach(() => {
    ChoyMessage.success = origSuccess;
    ChoyMessage.error = origError;
    console.warn = origWarn;
  });

  test('effectiveRecordId normalizes blank and literal null/undefined strings', () => {
    const m = mountLifecycle({
      watchIdentity: false,
      resolveRecordIdFromRoute: false,
      identity: { recordId: ' undefined ' },
    });
    expect(m.api().effectiveRecordId.value).toBeUndefined();
    m.identity.recordId = 'null';
    expect(m.api().effectiveRecordId.value).toBeUndefined();
    m.unmount();
  });

  test('effectiveRecordId prefers identity then route params', async () => {
    const m = mountLifecycle({ resolveRecordIdFromRoute: true, watchIdentity: false });
    expect(m.api().effectiveRecordId.value).toBe('rec-1');
    m.identity.recordId = 'prop-id';
    expect(m.api().effectiveRecordId.value).toBe('prop-id');
    m.unmount();
  });

  test('effectiveRecordId reads route.params.recordId and query fallbacks', () => {
    const m = mountLifecycle({
      resolveRecordIdFromRoute: true,
      watchIdentity: false,
      route: { params: { recordId: 'from-param' }, query: {} },
    });
    expect(m.api().effectiveRecordId.value).toBe('from-param');
    m.unmount();

    const m2 = mountLifecycle({
      resolveRecordIdFromRoute: true,
      watchIdentity: false,
      route: { params: {}, query: { Id: 'from-query' } },
    });
    expect(m2.api().effectiveRecordId.value).toBe('from-query');
    m2.unmount();
  });

  test('handleRefresh without id emits refresh-success null', async () => {
    const m = mountLifecycle({
      watchIdentity: false,
      resolveRecordIdFromRoute: false,
      identity: { recordId: undefined },
    });
    await m.api().handleRefresh();
    expect(m.emit.calls.some(c => c[0] === 'refresh-success' && (c[1] as any)?.record == null)).toBe(true);
    m.unmount();
  });

  test('handleCreate no-op without action; pushes when set', async () => {
    const m = mountLifecycle({ watchIdentity: false, createAction: undefined });
    await m.api().handleCreate();
    m.unmount();

    const m2 = mountLifecycle({ watchIdentity: false, createAction: '/new' });
    const push = fnRecorder(async () => {});
    m2.router.push = push as any;
    await m2.api().handleCreate();
    expect(push.calls[0]?.[0]).toBe('/new');
    m2.unmount();
  });

  test('handleDelete with empty path segments skips navigation', async () => {
    const m = mountLifecycle({
      watchIdentity: false,
      route: { path: '/', fullPath: '/', params: {}, query: {} },
    });
    m.controller.vm.original = { Id: 'x1' };
    const replace = fnRecorder(async () => {});
    m.router.replace = replace as any;
    const pending = m.api().handleDelete();
    resolveConfirmChoy('confirm');
    await pending;
    await flushPromises();
    expect(replace.calls.length).toBe(0);
    m.unmount();
  });

  test('initializeForm loads record when id present', async () => {
    const m = mountLifecycle();
    await flushPromises();
    expect(m.controller.beginDisplay.calls.length).toBeGreaterThan(0);
    expect(m.emit.calls.some(c => c[0] === 'load-success')).toBe(true);
    m.unmount();
  });

  test('initializeForm create path and display viewMode on create', async () => {
    const m = mountLifecycle({
      identity: { recordId: undefined, viewMode: 'display' },
      route: { params: {}, path: '/app/items/new', fullPath: '/app/items/new' },
    });
    await flushPromises();
    expect(m.controller.beginCreate.calls.length).toBe(1);
    expect(m.controller.vm.mode).toBe('display');
    m.unmount();
  });

  test('initializeForm skips when before-load canceled', async () => {
    const m = mountLifecycle();
    m.emitCancelable.mockImplementation(async () => false);
    await m.api().initializeForm();
    await flushPromises();
    const displayAfter = m.controller.beginDisplay.calls.length;
    await m.api().initializeForm();
    await flushPromises();
    expect(m.controller.beginDisplay.calls.length).toBe(displayAfter);
    m.unmount();
  });

  test('initializeForm ignores stale completion after bumpInitializeSeq', async () => {
    const m = mountLifecycle({ watchIdentity: false });
    let release: (() => void) | undefined;
    m.controller.beginDisplay.mockImplementation(
      () =>
        new Promise<void>(resolve => {
          release = resolve;
        })
    );
    const pending = m.api().initializeForm();
    await flushPromises();
    m.api().bumpInitializeSeq();
    release?.();
    await pending;
    await flushPromises();
    expect(m.emit.calls.some(c => c[0] === 'change')).toBe(false);
    m.unmount();
  });

  test('initializeForm emits action-error on load failure', async () => {
    const m = mountLifecycle({ watchIdentity: false });
    m.controller.beginDisplay.mockImplementation(async () => {
      throw 'boom';
    });
    await m.api().initializeForm();
    await flushPromises();
    const errCall = m.emit.calls.find(c => c[0] === 'action-error');
    expect((errCall?.[1] as any)?.action).toBe('load');
    expect((errCall?.[1] as any)?.error?.message).toBe('boom');
    m.unmount();
  });

  test('watchFormIdentity re-inits and beginEdit when viewMode is edit', async () => {
    const afterInit = fnRecorder(() => {});
    const m = mountLifecycle({ onAfterInit: afterInit, identity: { viewMode: 'edit' } });
    await flushPromises();
    expect(m.controller.beginEdit.calls.length).toBeGreaterThan(0);
    expect(afterInit.calls.length).toBeGreaterThan(0);
    const before = m.controller.beginDisplay.calls.length;
    m.route.params = { id: 'rec-2' };
    m.route.path = '/app/items/rec-2';
    await flushPromises();
    expect(m.controller.beginDisplay.calls.length).toBeGreaterThan(before);
    m.unmount();
  });

  test('handleEdit resets onchange and emits mode-change', () => {
    const m = mountLifecycle({ watchIdentity: false });
    m.api().handleEdit();
    expect(m.onchangeCtrl.reset.calls.length).toBe(1);
    expect(m.emit.calls.some(c => c[0] === 'mode-change' && (c[1] as any)?.mode === 'edit')).toBe(true);
    m.unmount();
  });

  test('handleCancel in create mode calls router.back', async () => {
    const m = mountLifecycle({ viewMode: 'create', watchIdentity: false });
    const back = fnRecorder(() => {});
    m.router.back = back;
    await m.api().handleCancel();
    expect(back.calls.length).toBe(1);
    m.unmount();
  });

  test('handleCancel reloads display and handles beginDisplay error', async () => {
    const m = mountLifecycle({ watchIdentity: false });
    m.controller.beginDisplay.mockImplementation(async () => {
      throw new Error('reload fail');
    });
    await m.api().handleCancel();
    await flushPromises();
    expect(m.emit.calls.some(c => c[0] === 'action-error' && (c[1] as any)?.action === 'load')).toBe(true);
    expect(m.emit.calls.some(c => c[0] === 'mode-change' && (c[1] as any)?.mode === 'display')).toBe(true);
    m.unmount();
  });

  test('handleReset emits reset with form data', () => {
    const m = mountLifecycle({ watchIdentity: false });
    m.api().handleReset();
    expect(m.controller.reset.calls.length).toBe(1);
    expect(m.emit.calls.some(c => c[0] === 'reset')).toBe(true);
    m.unmount();
  });

  test('handleCreate no-op without create action; push emits action-error on failure', async () => {
    const m = mountLifecycle({ watchIdentity: false, createAction: undefined });
    await m.api().handleCreate();
    expect(m.router.push).toBeDefined();

    const m2 = mountLifecycle({ watchIdentity: false, createAction: '/new' });
    const push = fnRecorder(async (_to: any) => {});
    m2.router.push = push as any;
    await m2.api().handleCreate();
    expect(push.calls.length).toBe(1);

    m2.router.push = fnRecorder(async () => {
      throw new Error('nav fail');
    }) as any;
    await m2.api().handleCreate();
    expect(m2.emit.calls.some(c => c[0] === 'action-error' && (c[1] as any)?.action === 'create')).toBe(true);
    m.unmount();
    m2.unmount();
  });

  test('handleCancel without record id still emits display mode', async () => {
    const m = mountLifecycle({
      watchIdentity: false,
      resolveRecordIdFromRoute: false,
      identity: { recordId: undefined },
    });
    await m.api().handleCancel();
    expect(m.controller.beginDisplay.calls.length).toBe(0);
    expect(m.emit.calls.some(c => c[0] === 'mode-change' && (c[1] as any)?.mode === 'display')).toBe(true);
    m.unmount();
  });

  test('handleRefresh respects before-refresh and no-id success', async () => {
    const m = mountLifecycle({
      watchIdentity: false,
      identity: { recordId: undefined },
      route: { params: {}, path: '/x', fullPath: '/x' },
    });
    m.emitCancelable.mockImplementation(async () => false);
    await m.api().handleRefresh();
    expect(m.controller.beginDisplay.calls.length).toBe(0);

    m.emitCancelable.mockImplementation(async () => true);
    await m.api().handleRefresh();
    expect(m.emit.calls.some(c => c[0] === 'refresh-success' && (c[1] as any)?.record === null)).toBe(true);
    m.unmount();
  });

  test('handleRefresh success and failure messages', async () => {
    const m = mountLifecycle({ watchIdentity: false, showMessages: true });
    m.controller.vm.original = { Id: 'rec-1' };
    m.controller.beginDisplay.mockImplementation(async () => {
      m.controller.vm.original = { Id: 'rec-1', Name: 'ok' };
    });
    await m.api().handleRefresh();
    await flushPromises();
    expect((ChoyMessage.success as any).calls.length).toBe(1);

    m.controller.beginDisplay.mockImplementation(async () => {
      throw new Error('refresh err');
    });
    await m.api().handleRefresh();
    await flushPromises();
    expect((ChoyMessage.error as any).calls.length).toBe(1);
    expect(m.emit.calls.some(c => c[0] === 'action-error' && (c[1] as any)?.action === 'refresh')).toBe(true);
    m.unmount();
  });

  test('handleCopy no-op without original; otherwise beginCreate', async () => {
    const m = mountLifecycle({ watchIdentity: false });
    await m.api().handleCopy();
    expect(m.controller.beginCreate.calls.length).toBe(0);

    m.controller.vm.original = { Id: '1', Name: 'A' };
    await m.api().handleCopy();
    await flushPromises();
    expect(m.controller.beginCreate.calls.length).toBe(1);
    expect(m.emit.calls.some(c => c[0] === 'copy')).toBe(true);
    m.unmount();
  });

  test('handleDelete confirm cancel/dismiss, before-delete, success navigation', async () => {
    const m = mountLifecycle({ watchIdentity: false });
    m.controller.vm.original = null;
    m.api().handleDelete();
    await flushPromises();

    m.controller.vm.original = { Id: 'del-1' };
    m.api().handleDelete();
    resolveConfirmChoy('dismiss');
    await flushPromises();

    m.api().handleDelete();
    resolveConfirmChoy('cancel');
    await flushPromises();

    m.emitCancelable.mockImplementation(async () => false);
    m.api().handleDelete();
    resolveConfirmChoy('confirm');
    await flushPromises();
    expect(m.controller.delete.calls.length).toBe(0);

    m.emitCancelable.mockImplementation(async () => true);
    const replace = fnRecorder(async (_to: any) => {});
    m.router.replace = replace as any;
    const store = useBreadcrumbStore();
    store.breadcrumbStack.push({
      title: 'Item',
      path: '/app/items/del-1',
      clickable: true,
      timestamp: 1,
      kind: 'page',
    });
    m.api().handleDelete();
    resolveConfirmChoy('confirm');
    await flushPromises();
    expect(m.controller.delete.calls.length).toBe(1);
    expect(replace.calls.length).toBe(1);

    m.unmount();
  });

  test('handleDelete delete-success preventDefault skips list navigation', async () => {
    const m = mountLifecycle({ watchIdentity: false });
    m.controller.vm.original = { Id: 'del-pd' };
    const replace = fnRecorder(async (_to: any) => {});
    m.router.replace = replace as any;
    m.emit.mockImplementation((event: string, payload?: any) => {
      if (event === 'delete-success') payload?.preventDefault?.();
    });
    m.api().handleDelete();
    resolveConfirmChoy('confirm');
    await flushPromises();
    expect(replace.calls.length).toBe(0);
    m.unmount();
  });

  test('handleDelete failure emits action-error', async () => {
    const m = mountLifecycle({ watchIdentity: false });
    m.controller.vm.original = { Id: 'x' };
    m.controller.delete.mockImplementation(async () => {
      throw new Error('del fail');
    });
    m.api().handleDelete();
    resolveConfirmChoy('confirm');
    await flushPromises();
    expect(m.emit.calls.some(c => c[0] === 'action-error' && (c[1] as any)?.action === 'delete')).toBe(true);
    expect((ChoyMessage.error as any).calls.length).toBeGreaterThan(0);
    m.unmount();
  });

  test('handleDelete pops matching breadcrumb entry', async () => {
    const m = mountLifecycle({
      watchIdentity: false,
      resolveRecordIdFromRoute: false,
      identity: { recordId: 'rec-1' },
    });
    const breadcrumbStore = useBreadcrumbStore();
    const stack = breadcrumbStore.breadcrumbStack as any[];
    stack.length = 0;
    stack.push({ path: m.route.path, title: 'Detail' });
    m.controller.vm.original = { Id: 'rec-1' };
    const pending = m.api().handleDelete();
    resolveConfirmChoy('confirm');
    await pending;
    await flushPromises();
    expect(stack.some((e: any) => e.path === m.route.path)).toBe(false);
    m.unmount();
  });

  test('handleDelete breadcrumb update failure is warned', async () => {
    setActivePinia(undefined as any);
    const { router, route } = createFeStubRouter({
      route: {
        name: 'Detail',
        path: '/app/items/warn-1',
        fullPath: '/app/items/warn-1',
        params: { id: 'warn-1' },
        query: {},
        meta: {},
      },
    });
    const identity = reactive<MutableIdentity>({ viewMode: 'display' });
    const controller = makeController();
    controller.vm.original = { Id: 'warn-1' };
    const emitCancelable = fnRecorder(async () => true);
    const emit = fnRecorder(() => {});
    let api: ReturnType<typeof useFormViewLifecycle<any>> | null = null;
    const Host = defineComponent({
      setup() {
        api = useFormViewLifecycle({
          identity,
          route,
          router,
          controller: controller as any,
          resolvedResolveRecordIdFromRoute: computed(() => false),
          resolvedShowMessages: computed(() => false),
          resolvedCreateAction: computed(() => undefined),
          viewMode: computed(() => 'display'),
          exposedFormData: computed(() => ({})),
          onchangeCtrl: { reset: () => {} } as any,
          resetOnchangeAgg: () => {},
          emitCancelable,
          emit,
          _t: (k: string) => k,
        });
        return () => h('div');
      },
    });
    mountApp(Host, { plugins: [router] });
    api!.handleDelete();
    resolveConfirmChoy('confirm');
    await flushPromises();
    expect((console.warn as any).calls.some((c: any[]) => String(c[0]).includes('breadcrumb'))).toBe(true);
  });

  test('dirty edit cancel and refresh wait for discard confirmation', async () => {
    const m = mountLifecycle({ watchIdentity: false, viewMode: 'edit' });
    m.controller.vm.original = { Id: 'rec-1', Name: 'a' };
    m.controller.vm.draft = { Id: 'rec-1', Name: 'b' };
    expect(m.api().isFormDirty()).toBe(true);

    const pendingCancel = m.api().handleCancel();
    await flushPromises();
    resolveConfirmChoy('cancel');
    await pendingCancel;
    expect(m.controller.beginDisplay.calls.length).toBe(0);

    const pendingRefresh = m.api().handleRefresh();
    await flushPromises();
    resolveConfirmChoy('confirm');
    await pendingRefresh;
    expect(m.controller.beginDisplay.calls.length).toBe(1);
    m.unmount();
  });

  test('route leave asks before discarding a dirty edit', async () => {
    const m = mountLifecycle({ watchIdentity: false, viewMode: 'edit' });
    m.controller.vm.original = { Id: 'rec-1', Name: 'a' };
    m.controller.vm.draft = { Id: 'rec-1', Name: 'b' };
    const pending = m.router.push('/elsewhere');
    await flushPromises();
    expect(useConfirmChoyStore().open).toBe(true);
    resolveConfirmChoy('dismiss');
    await pending;
    m.unmount();
  });

  test('route leave skips discard prompt when navigating to login', async () => {
    const m = mountLifecycle({ watchIdentity: false, viewMode: 'edit' });
    m.controller.vm.original = { Id: 'rec-1', Name: 'a' };
    m.controller.vm.draft = { Id: 'rec-1', Name: 'b' };
    const pending = m.router.push({ name: 'login', path: '/login' });
    await flushPromises();
    expect(useConfirmChoyStore().open).toBe(false);
    await pending;
    m.unmount();
  });
});
