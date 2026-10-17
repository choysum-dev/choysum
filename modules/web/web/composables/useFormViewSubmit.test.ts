// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, defineComponent, h, reactive, ref } from 'vue';
import * as VueRouter from 'vue-router';
import { createPinia, setActivePinia } from 'pinia';

import { flushPromises, fnRecorder, mountApp } from '@/web/web/__tests__/mountApp';
import { ChoyMessage } from '@/web/web/composables/useChoyMessage';
import { useFormViewSubmit } from '@/web/web/composables/useFormViewSubmit';
import { useBreadcrumbStore } from '@/web/web/stores/breadcrumbStore';

const createFeStubRouter = (VueRouter as any).createFeStubRouter;

const origSuccess = ChoyMessage.success;
const origError = ChoyMessage.error;
const origWarn = console.warn;

function mountSubmit(opts?: {
  viewMode?: 'display' | 'edit' | 'create';
  loading?: boolean;
  showMessages?: boolean;
  submitHandler?: (...args: any[]) => any;
  routePath?: string;
  routeParams?: Record<string, string>;
  fieldErrors?: Map<string, string>;
  validators?: Map<string, () => Promise<string>>;
}) {
  const { router, route } = createFeStubRouter({
    route: {
      path: opts?.routePath ?? '/app/items/new',
      fullPath: opts?.routePath ?? '/app/items/new',
      params: opts?.routeParams ?? {},
      query: {},
      meta: {},
    },
  });

  const controller = {
    vm: reactive({ mode: opts?.viewMode ?? 'create', original: { Id: 'created-1', Name: 'N' } as any }),
    submit: fnRecorder(async () => {
      controller.vm.original = { Id: 'created-1', Name: 'N' };
    }),
  };

  const viewMode = computed(() => opts?.viewMode ?? 'create');
  const loading = computed(() => opts?.loading ?? false);
  const fieldErrors = ref(opts?.fieldErrors ?? new Map<string, string>());
  const fieldClientValidators = opts?.validators ?? new Map<string, () => Promise<string>>();
  const emitCancelable = fnRecorder(async () => true);
  const emit = fnRecorder((_e?: string, _p?: any) => {});
  const onchangeCtrl = {
    pause: fnRecorder(() => {}),
    resume: fnRecorder(() => {}),
    reset: fnRecorder(() => {}),
  };
  const resetOnchangeAgg = fnRecorder(() => {});

  let api: ReturnType<typeof useFormViewSubmit<any>> | null = null;

  const Host = defineComponent({
    setup() {
      api = useFormViewSubmit({
        controller: controller as any,
        viewMode,
        loading,
        exposedFormData: computed(() => ({ Name: 'draft' })),
        fieldClientValidators,
        fieldErrors,
        resolvedShowMessages: computed(() => opts?.showMessages ?? false),
        submitHandler: opts?.submitHandler,
        onchangeCtrl: onchangeCtrl as any,
        resetOnchangeAgg,
        emitCancelable,
        emit,
        router,
        detailsTitle: { src: 'Details title' },
        _t: (k: string) => k,
      });
      return () => h('div');
    },
  });

  const mounted = mountApp(Host, { plugins: [createPinia(), router] });
  return {
    ...mounted,
    api: () => api!,
    controller,
    emitCancelable,
    emit,
    route,
    router,
    onchangeCtrl,
    fieldErrors,
    fieldClientValidators,
  };
}

describe('useFormViewSubmit', () => {
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

  test('returns not-editable in display mode', async () => {
    const m = mountSubmit({ viewMode: 'display' });
    const out = await m.api().handleSubmit();
    expect(out.reason).toBe('not-editable');
    m.unmount();
  });

  test('returns loading when loading or submitting', async () => {
    const m = mountSubmit({ viewMode: 'edit', loading: true });
    expect((await m.api().handleSubmit()).reason).toBe('loading');

    const m2 = mountSubmit({ viewMode: 'edit' });
    let resolveSubmit: (() => void) | undefined;
    m2.controller.submit.mockImplementation(
      () =>
        new Promise<void>(resolve => {
          resolveSubmit = resolve;
        })
    );
    const pending = m2.api().handleSubmit();
    await flushPromises();
    const blocked = await m2.api().handleSubmit();
    expect(blocked.reason).toBe('loading');
    resolveSubmit!();
    await pending;
    m.unmount();
    m2.unmount();
  });

  test('validateForm fails on fieldErrors and async validators', async () => {
    const m = mountSubmit({
      viewMode: 'edit',
      showMessages: true,
      fieldErrors: new Map([['Name', 'bad']]),
    });
    expect(await m.api().validateForm()).toBe(false);
    expect((ChoyMessage.error as any).calls.length).toBe(1);

    const validators = new Map<string, () => Promise<string>>();
    validators.set('Name', async () => 'required');
    const m2 = mountSubmit({ viewMode: 'edit', showMessages: true, validators });
    expect(await m2.api().validateForm()).toBe(false);
    m.unmount();
    m2.unmount();
  });

  test('before-submit canceled and validate-failed paths', async () => {
    const m = mountSubmit({ viewMode: 'edit' });
    m.emitCancelable.mockImplementation(async () => false);
    expect((await m.api().handleSubmit()).reason).toBe('before-submit-canceled');

    const validators = new Map<string, () => Promise<string>>();
    validators.set('X', async () => 'err');
    const m2 = mountSubmit({ viewMode: 'edit', validators, showMessages: true });
    expect((await m2.api().handleSubmit()).reason).toBe('validate-failed');
    m.unmount();
    m2.unmount();
  });

  test('default submit create navigates from /new and updates breadcrumb', async () => {
    const m = mountSubmit({ viewMode: 'create', showMessages: true, routePath: '/app/items/new' });
    const replace = fnRecorder(async (_to: any) => {});
    m.router.replace = replace as any;
    const store = useBreadcrumbStore();
    store.breadcrumbStack.push({
      title: 'New',
      path: '/app/items/new',
      clickable: true,
      timestamp: 1,
      kind: 'page',
    });
    const out = await m.api().handleSubmit();
    expect(out.ok).toBe(true);
    expect(replace.calls.length).toBe(1);
    expect(m.emit.calls.some(c => c[0] === 'create-success')).toBe(true);
    m.unmount();
  });

  test('copy save replaces the source record id and returns to display', async () => {
    const m = mountSubmit({
      viewMode: 'create',
      routePath: '/app/items/rec-1',
      routeParams: { id: 'rec-1' },
    });
    const replace = fnRecorder(async (_to: any) => {});
    m.router.replace = replace as any;
    const store = useBreadcrumbStore();
    store.breadcrumbStack.push({
      title: 'Source',
      path: '/app/items/rec-1',
      clickable: true,
      timestamp: 1,
      kind: 'page',
    });
    const out = await m.api().handleSubmit();
    expect(out.ok).toBe(true);
    expect(replace.calls[0]?.[0]).toBe('/app/items/created-1');
    expect(m.controller.vm.mode).toBe('display');
    expect(store.breadcrumbStack[0]?.path).toBe('/app/items/created-1');
    m.unmount();
  });

  test('create-success preventDefault skips navigation', async () => {
    const m = mountSubmit({ viewMode: 'create', routePath: '/app/items/new' });
    const replace = fnRecorder(async (_to: any) => {});
    m.router.replace = replace as any;
    m.emit.mockImplementation((event: string, payload?: any) => {
      if (event === 'create-success') payload?.preventDefault?.();
    });
    await m.api().handleSubmit();
    expect(replace.calls.length).toBe(0);
    m.unmount();
  });

  test('default submit edit emits update-success', async () => {
    const m = mountSubmit({ viewMode: 'edit', showMessages: true, routePath: '/app/items/1' });
    const out = await m.api().handleSubmit();
    expect(out.ok).toBe(true);
    expect(m.emit.calls.some(c => c[0] === 'update-success')).toBe(true);
    m.unmount();
  });

  test('submitHandler boolean and object shapes', async () => {
    const m = mountSubmit({
      viewMode: 'edit',
      submitHandler: async () => true,
    });
    const handled = await m.api().handleSubmit();
    expect(handled.handledByHandler).toBe(true);
    expect(m.controller.submit.calls.length).toBe(0);

    const m2 = mountSubmit({
      viewMode: 'create',
      showMessages: true,
      submitHandler: fnRecorder(async () => ({
        handled: true,
        record: { Id: 'h-1' },
        successMessage: 'Custom ok',
        skipSuccessMessage: false,
      })),
    });
    const out = await m2.api().handleSubmit();
    expect(out.handledByHandler).toBe(true);
    expect(m2.emit.calls.some(c => c[0] === 'create-success')).toBe(true);

    const m3 = mountSubmit({
      viewMode: 'edit',
      showMessages: true,
      submitHandler: fnRecorder(async () => ({
        handled: true,
        skipSuccessMessage: true,
      })),
    });
    (ChoyMessage.success as any).mockClear();
    await m3.api().handleSubmit();
    expect((ChoyMessage.success as any).calls.length).toBe(0);
    m.unmount();
    m2.unmount();
    m3.unmount();
  });

  test('submitHandler false runs default submit', async () => {
    const m = mountSubmit({
      viewMode: 'edit',
      submitHandler: fnRecorder(async () => false),
    });
    await m.api().handleSubmit();
    expect(m.controller.submit.calls.length).toBe(1);
    m.unmount();
  });

  test('submitHandler that calls defaultSubmit then returns false does not double-save', async () => {
    const m = mountSubmit({
      viewMode: 'edit',
      submitHandler: async ({ defaultSubmit }) => {
        await defaultSubmit();
        return false;
      },
    });
    await m.api().handleSubmit();
    expect(m.controller.submit.calls.length).toBe(1);
    m.unmount();
  });

  test('submitHandler void/undefined treated as not handled', async () => {
    const m = mountSubmit({
      viewMode: 'edit',
      submitHandler: fnRecorder(async () => undefined),
    });
    await m.api().handleSubmit();
    expect(m.controller.submit.calls.length).toBe(1);
    m.unmount();
  });

  test('create success without new Id skips router.replace', async () => {
    const m = mountSubmit({ viewMode: 'create', routePath: '/app/items/new' });
    m.controller.submit.mockImplementation(async () => {
      m.controller.vm.original = { Name: 'no-id' };
    });
    const replace = fnRecorder(async () => {});
    m.router.replace = replace as any;
    await m.api().handleSubmit();
    expect(replace.calls.length).toBe(0);
    m.unmount();
  });

  test('create success with null original skips router.replace without throwing', async () => {
    const m = mountSubmit({ viewMode: 'create', routePath: '/app/items/new' });
    m.controller.submit.mockImplementation(async () => {
      m.controller.vm.original = null;
    });
    const replace = fnRecorder(async () => {});
    m.router.replace = replace as any;
    const out = await m.api().handleSubmit();
    expect(out.ok).toBe(true);
    expect(replace.calls.length).toBe(0);
    expect(m.emit.calls.some(c => c[0] === 'action-error')).toBe(false);
    m.unmount();
  });

  test('submit error emits action-error and shows message', async () => {
    const m = mountSubmit({ viewMode: 'edit', showMessages: true });
    m.controller.submit.mockImplementation(async () => {
      throw new Error('save fail');
    });
    const out = await m.api().handleSubmit();
    expect(out.reason).toBe('error');
    expect(m.emit.calls.some(c => c[0] === 'action-error' && (c[1] as any)?.action === 'update')).toBe(true);
    expect((ChoyMessage.error as any).calls.length).toBe(1);
    expect(m.onchangeCtrl.resume.calls.length).toBe(1);
    m.unmount();
  });

  test('create submit error uses create action label', async () => {
    const m = mountSubmit({ viewMode: 'create', showMessages: true });
    m.controller.submit.mockImplementation(async () => {
      throw 'create fail';
    });
    const out = await m.api().handleSubmit();
    expect(out.reason).toBe('error');
    expect(m.emit.calls.some(c => c[0] === 'action-error' && (c[1] as any)?.action === 'create')).toBe(true);
    m.unmount();
  });

  test('create breadcrumb failure warns; onNativeSubmit delegates', async () => {
    setActivePinia(undefined as any);
    const { router } = createFeStubRouter({
      route: { path: '/app/items/new', fullPath: '/app/items/new', params: {}, query: {}, meta: {} },
    });
    const controller = {
      vm: reactive({ mode: 'create', original: { Id: 'created-1' } as any }),
      submit: fnRecorder(async () => {}),
    };
    let api: ReturnType<typeof useFormViewSubmit<any>> | null = null;
    const Host = defineComponent({
      setup() {
        api = useFormViewSubmit({
          controller: controller as any,
          viewMode: computed(() => 'create'),
          loading: computed(() => false),
          exposedFormData: computed(() => ({})),
          fieldClientValidators: new Map(),
          fieldErrors: ref(new Map()),
          resolvedShowMessages: computed(() => false),
          onchangeCtrl: { pause: () => {}, resume: () => {}, reset: () => {} } as any,
          resetOnchangeAgg: () => {},
          emitCancelable: fnRecorder(async () => true),
          emit: fnRecorder(() => {}),
          router,
          detailsTitle: { src: 'T' },
          _t: (k: string) => k,
        });
        return () => h('div');
      },
    });
    mountApp(Host, { plugins: [router] });
    await api!.handleSubmit();
    expect((console.warn as any).calls.length).toBeGreaterThan(0);

    api!.onNativeSubmit();
    await flushPromises();
  });
});
