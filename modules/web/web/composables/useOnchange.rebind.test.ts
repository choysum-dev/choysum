// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref } from 'vue';
import { describe, expect, it, vi } from 'vitest';
import { disposeOnchange, getOnchangeController } from '@/web/web/composables/useOnchange';

describe('getOnchangeController rebindOptions', () => {
  it('rebinds getRoot when the same store/session controller is reused', async () => {
    const onchange = vi.fn(async () => ({ value: {}, messages: [] }));
    const store = {
      fieldsMetadata: { Name: { id: '1', type: 'varchar', typeAnnotation: '' } },
      state: {},
      Onchange: onchange,
    } as any;

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
    const store = {
      fieldsMetadata: {
        Name: { id: '1', type: 'varchar', typeAnnotation: '' },
      },
      state: {},
      Onchange: onchange,
    } as any;

    const draft1 = ref<Record<string, any>>({ Id: '1', Name: 'A' });
    const ctrl = getOnchangeController(store, 'ListView', {
      getRoot: () => draft1.value,
      debounceMs: 0,
    });

    await ctrl.markChanged('Name');
    expect(ctrl.hasPending()).toBe(true);

    const draft2 = ref<Record<string, any>>({ Id: '2', Name: 'B' });
    getOnchangeController(store, 'ListView', {
      getRoot: () => draft2.value,
      debounceMs: 0,
    });
    expect(ctrl.hasPending()).toBe(false);

    draft2.value = { Id: '2', Name: 'B2' };
    await ctrl.markChanged('Name', { flush: true });
    expect(onchange.mock.calls.at(-1)?.[0]).toEqual(expect.objectContaining({ Id: '2', Name: 'B2' }));

    disposeOnchange(store);
  });
});
