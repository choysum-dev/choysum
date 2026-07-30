// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { nextTick, ref } from 'vue';
import { describe, expect, it, vi } from 'vitest';
import { disposeOnchange, getOnchangeController } from '@/web/web/composables/useOnchange';

function makeStore(onchange = vi.fn(async () => ({ value: {}, messages: [] }))) {
  return {
    fieldsMetadata: { Name: { id: '1', type: 'varchar', typeAnnotation: '' } },
    state: {} as Record<string, any>,
    Onchange: onchange,
  } as any;
}

describe('getOnchangeController rebindOptions', () => {
  it('rebinds getRoot when the same store/session controller is reused', async () => {
    const onchange = vi.fn(async () => ({ value: {}, messages: [] }));
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
    expect(onchange).toHaveBeenCalled();
    expect(onchange.mock.calls.at(-1)?.[0]).toEqual(expect.objectContaining({ Id: '2', Name: 'B' }));

    disposeOnchange(store);
  });

  it('clears pending from the previous root on rebind', async () => {
    const onchange = vi.fn(async () => ({ value: {}, messages: [] }));
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
    expect(onchange.mock.calls.at(-1)?.[0]).toEqual(expect.objectContaining({ Id: '2', Name: 'B2' }));

    disposeOnchange(store);
  });

  it('rebinds singleton controller when sessionId is omitted', async () => {
    const onchange = vi.fn(async () => ({ value: {}, messages: [] }));
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
    expect(onchange.mock.calls.at(-1)?.[0]).toEqual(expect.objectContaining({ Id: '2', Name: 'B' }));

    // Omitting opts must not rebind away from the current getRoot.
    getOnchangeController(store);
    draft2.value = { Id: '2', Name: 'B2' };
    await ctrl.markChanged('Name', { flush: true });
    expect(onchange.mock.calls.at(-1)?.[0]).toEqual(expect.objectContaining({ Id: '2', Name: 'B2' }));

    disposeOnchange(store);
  });

  it('rebindOptions(undefined) falls back to store record root', async () => {
    const onchange = vi.fn(async () => ({ value: {}, messages: [] }));
    const store = makeStore(onchange);
    store.state.record = { Id: 'store', Name: 'S' };

    const draft = ref({ Id: '1', Name: 'A' });
    const ctrl = getOnchangeController(store, 'ListViewFallback', {
      getRoot: () => draft.value,
    });
    ctrl.rebindOptions(undefined);
    await ctrl.markChanged('Name', { flush: true });
    expect(onchange.mock.calls.at(-1)?.[0]).toEqual(expect.objectContaining({ Id: 'store', Name: 'S' }));

    disposeOnchange(store);
  });

  it('routes patch through rebound onPatch when root disappears mid-flush', async () => {
    let root: Record<string, any> | null = { Id: '1', Name: 'A' };
    const onPatch = vi.fn();
    const onchange = vi.fn(async () => {
      root = null;
      return { value: { Name: 'Patched' }, messages: [] };
    });
    const store = makeStore(onchange);
    const ctrl = getOnchangeController(store, 'ListViewPatch', {
      getRoot: () => root ?? undefined,
      onPatch,
    });

    await ctrl.markChanged('Name', { flush: true });
    expect(onPatch).toHaveBeenCalledWith({ Name: 'Patched' }, expect.anything());

    const onPatch2 = vi.fn();
    root = { Id: '2', Name: 'B' };
    getOnchangeController(store, 'ListViewPatch', {
      getRoot: () => root ?? undefined,
      onPatch: onPatch2,
    });
    await ctrl.markChanged('Name', { flush: true });
    expect(onPatch2).toHaveBeenCalled();
    expect(onPatch).toHaveBeenCalledTimes(1);

    disposeOnchange(store);
  });

  it('honors rebound immediateFirst on the first watched change', async () => {
    const onchange = vi.fn(async () => ({ value: {}, messages: [] }));
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
    await vi.waitFor(() => expect(onchange).toHaveBeenCalled());
    expect(onchange.mock.calls.at(-1)?.[0]).toEqual(expect.objectContaining({ Name: 'B' }));

    disposeOnchange(store);
  });

  it('schedules debounced flush when immediateFirst is unset', async () => {
    vi.useFakeTimers();
    try {
      const onchange = vi.fn(async () => ({ value: {}, messages: [] }));
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
      expect(onchange).not.toHaveBeenCalled();
      await vi.advanceTimersByTimeAsync(40);
      await vi.waitFor(() => expect(onchange).toHaveBeenCalled());
      disposeOnchange(store);
    } finally {
      vi.useRealTimers();
    }
  });
});
