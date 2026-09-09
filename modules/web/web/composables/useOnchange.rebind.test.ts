// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { nextTick, ref, defineComponent, provide } from 'vue';

import { disposeOnchange, getOnchangeController } from '@/web/web/composables/useOnchange';
import { fnRecorder, flushPromises, mountApp } from '@/web/web/__tests__/mountApp';

function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function waitUntil(pred: () => boolean, timeoutMs = 800): Promise<void> {
  const start = Date.now();
  while (!pred()) {
    if (Date.now() - start > timeoutMs) throw new Error('waitUntil timeout');
    await sleep(10);
  }
}

function makeStore(onchange = fnRecorder(async () => ({ value: {}, messages: [] }))) {
  return {
    fieldsMetadata: { Name: { id: '1', type: 'varchar', typeAnnotation: '' } },
    state: {} as Record<string, any>,
    Onchange: onchange,
  } as any;
}

function lastDraft(onchange: ReturnType<typeof fnRecorder>): Record<string, any> | undefined {
  const args = onchange.calls.at(-1);
  return args?.[0] as Record<string, any> | undefined;
}

describe('useOnchange.rebind getOnchangeController rebindOptions', () => {
  test('rebinds getRoot when the same store/session controller is reused', async () => {
    const onchange = fnRecorder(async () => ({ value: {}, messages: [] }));
    const store = makeStore(onchange);

    const draft1 = ref<Record<string, any> | null>({ Id: '1', Name: 'A' });
    const ctrl = getOnchangeController(store, 'ListView', {
      getRoot: () => draft1.value ?? undefined,
    });

    draft1.value = null;

    const draft2 = ref<Record<string, any> | null>({ Id: '2', Name: 'B' });
    const again = getOnchangeController(store, 'ListView', {
      getRoot: () => draft2.value ?? undefined,
    });
    expect(again).toBe(ctrl);

    await ctrl.markChanged('Name', { flush: true });
    expect(onchange.calls.length).toBeGreaterThan(0);
    expect(lastDraft(onchange)?.Id).toBe('2');
    expect(lastDraft(onchange)?.Name).toBe('B');

    disposeOnchange(store);
  });

  test('clears pending from the previous root on rebind', async () => {
    const onchange = fnRecorder(async () => ({ value: {}, messages: [] }));
    const store = makeStore(onchange);

    const draft1 = ref<Record<string, any>>({ Id: '1', Name: 'A' });
    const ctrl = getOnchangeController(store, 'ListView', {
      getRoot: () => draft1.value,
    });

    // Pause so markChanged keeps pending; debounceMs: 0 would flush synchronously and clear it.
    ctrl.pause();
    await ctrl.markChanged('Name');
    expect(ctrl.hasPending()).toBe(true);

    const draft2 = ref<Record<string, any>>({ Id: '2', Name: 'B' });
    getOnchangeController(store, 'ListView', {
      getRoot: () => draft2.value,
    });
    expect(ctrl.hasPending()).toBe(false);

    draft2.value = { Id: '2', Name: 'B2' };
    await ctrl.markChanged('Name', { flush: true });
    expect(lastDraft(onchange)?.Id).toBe('2');
    expect(lastDraft(onchange)?.Name).toBe('B2');

    disposeOnchange(store);
  });

  test('rebinds singleton controller when sessionId is omitted', async () => {
    const onchange = fnRecorder(async () => ({ value: {}, messages: [] }));
    const store = makeStore(onchange);

    const draft1 = ref({ Id: '1', Name: 'A' });
    const ctrl = getOnchangeController(store, undefined, {
      getRoot: () => draft1.value,
    });
    const draft2 = ref({ Id: '2', Name: 'B' });
    const again = getOnchangeController(store, undefined, {
      getRoot: () => draft2.value,
    });
    expect(again).toBe(ctrl);

    await ctrl.markChanged('Name', { flush: true });
    expect(lastDraft(onchange)?.Id).toBe('2');
    expect(lastDraft(onchange)?.Name).toBe('B');

    // Omitting opts must not rebind away from the current getRoot.
    getOnchangeController(store);
    draft2.value = { Id: '2', Name: 'B2' };
    await ctrl.markChanged('Name', { flush: true });
    expect(lastDraft(onchange)?.Id).toBe('2');
    expect(lastDraft(onchange)?.Name).toBe('B2');

    disposeOnchange(store);
  });

  test('rebindOptions(undefined) falls back to store record root', async () => {
    const onchange = fnRecorder(async () => ({ value: {}, messages: [] }));
    const store = makeStore(onchange);
    store.state.record = { Id: 'store', Name: 'S' };

    const draft = ref({ Id: '1', Name: 'A' });
    const ctrl = getOnchangeController(store, 'ListViewFallback', {
      getRoot: () => draft.value,
    });
    ctrl.rebindOptions(undefined);
    await ctrl.markChanged('Name', { flush: true });
    expect(lastDraft(onchange)?.Id).toBe('store');
    expect(lastDraft(onchange)?.Name).toBe('S');

    disposeOnchange(store);
  });

  test('routes patch through rebound onPatch when root disappears mid-flush', async () => {
    let root: Record<string, any> | null = { Id: '1', Name: 'A' };
    const onPatch = fnRecorder();
    const onchange = fnRecorder(async () => {
      root = null;
      return { value: { Name: 'Patched' }, messages: [] };
    });
    const store = makeStore(onchange);
    const ctrl = getOnchangeController(store, 'ListViewPatch', {
      getRoot: () => root ?? undefined,
      onPatch,
    });

    await ctrl.markChanged('Name', { flush: true });
    expect(onPatch.calls.length).toBe(1);
    expect(onPatch.calls[0]?.[0]).toEqual({ Name: 'Patched' });

    const onPatch2 = fnRecorder();
    root = { Id: '2', Name: 'B' };
    getOnchangeController(store, 'ListViewPatch', {
      getRoot: () => root ?? undefined,
      onPatch: onPatch2,
    });
    await ctrl.markChanged('Name', { flush: true });
    expect(onPatch2.calls.length).toBeGreaterThan(0);
    expect(onPatch.calls.length).toBe(1);

    disposeOnchange(store);
  });

  test('honors rebound immediateFirst on the first watched change', async () => {
    const onchange = fnRecorder(async () => ({ value: {}, messages: [] }));
    const store = makeStore(onchange);
    const draft = ref<Record<string, any> | null>(null);
    getOnchangeController(store, 'ListViewImmediate', {
      getRoot: () => draft.value ?? undefined,
      immediateFirst: true,
      debounceMs: 50,
    });

    // First non-null root only seeds baseline; second mutation schedules immediate flush.
    draft.value = { Id: '1', Name: 'A' };
    await nextTick();
    draft.value = { Id: '1', Name: 'B' };
    await nextTick();
    await waitUntil(() => onchange.calls.length > 0);
    expect(lastDraft(onchange)?.Name).toBe('B');

    disposeOnchange(store);
  });

  test('schedules debounced flush when immediateFirst is unset', async () => {
    const onchange = fnRecorder(async () => ({ value: {}, messages: [] }));
    const store = makeStore(onchange);
    const draft = ref<Record<string, any> | null>(null);
    getOnchangeController(store, 'ListViewDebounced', {
      getRoot: () => draft.value ?? undefined,
      debounceMs: 30,
    });
    // Touch session again without opts so the else-if(opts) false branch is hit.
    getOnchangeController(store, 'ListViewDebounced');

    draft.value = { Id: '1', Name: 'A' };
    await nextTick();
    draft.value = { Id: '1', Name: 'B' };
    await nextTick();
    expect(onchange.calls.length).toBe(0);
    await sleep(50);
    await waitUntil(() => onchange.calls.length > 0);
    disposeOnchange(store);
  });

  test('rebindOptions clears pause and applies a new debounceMs', async () => {
    const onchange = fnRecorder(async () => ({ value: {}, messages: [] }));
    const store = makeStore(onchange);
    const draft = ref<Record<string, any> | null>(null);
    const ctrl = getOnchangeController(store, 'ListViewRebindDebounce', {
      getRoot: () => draft.value ?? undefined,
      debounceMs: 200,
    });
    ctrl.pause();
    ctrl.rebindOptions({
      getRoot: () => draft.value ?? undefined,
      debounceMs: 20,
    });

    draft.value = { Id: '1', Name: 'A' };
    await nextTick();
    draft.value = { Id: '1', Name: 'B' };
    await nextTick();
    expect(onchange.calls.length).toBe(0);
    await sleep(40);
    await waitUntil(() => onchange.calls.length > 0);
    disposeOnchange(store);
  });
});

describe('useOnchange.rebind view-mode inject gating', () => {
  test('skips inject outside setup and still creates a usable controller', async () => {
    const onchange = fnRecorder(async () => ({ value: {}, messages: [] }));
    const store = makeStore(onchange);
    const draft = ref({ Id: '1', Name: 'A' });
    // Call site is a plain test body (no currentInstance) — must not Vue-warn on inject.
    const ctrl = getOnchangeController(store, 'OutsideSetup', {
      getRoot: () => draft.value,
    });
    draft.value = { Id: '1', Name: 'B' };
    await ctrl.markChanged('Name', { flush: true });
    expect(onchange.calls.length).toBeGreaterThan(0);
    disposeOnchange(store);
  });

  test('honors injected view-mode when created inside setup', async () => {
    const onchange = fnRecorder(async () => ({ value: {}, messages: [] }));
    const store = makeStore(onchange);
    const draft = ref({ Id: '1', Name: 'A' });
    const viewMode = ref<'display' | 'edit'>('display');
    let ctrl: ReturnType<typeof getOnchangeController> | undefined;

    const Host = defineComponent({
      setup() {
        provide('view-mode', viewMode);
        ctrl = getOnchangeController(store, 'InsideSetup', {
          getRoot: () => draft.value,
        });
        return () => null;
      },
    });
    const mounted = mountApp(Host);
    try {
      expect(ctrl).toBeTruthy();
      // Immediate display watch pauses auto-flush.
      draft.value = { Id: '1', Name: 'B' };
      await ctrl!.markChanged('Name');
      expect(ctrl!.hasPending()).toBe(true);
      expect(onchange.calls.length).toBe(0);

      viewMode.value = 'edit';
      await nextTick();
      await flushPromises();
      await waitUntil(() => onchange.calls.length > 0);
    } finally {
      mounted.unmount();
      disposeOnchange(store);
    }
  });
});
