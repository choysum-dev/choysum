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

    // First mount: controller captures draft1.
    const draft1 = ref<Record<string, any> | null>({ Id: '1', Name: 'A' });
    const ctrl = getOnchangeController(store, 'ListView', {
      getRoot: () => draft1.value ?? undefined,
    });

    // Simulate unmount: old draft is cleared.
    draft1.value = null;

    // Remount with a new draft; without rebind, flush would see a null root and skip Onchange.
    const draft2 = ref<Record<string, any> | null>({ Id: '2', Name: 'B' });
    const again = getOnchangeController(store, 'ListView', {
      getRoot: () => draft2.value ?? undefined,
    });
    expect(again).toBe(ctrl);

    await ctrl.markChanged('Name', { flush: true });
    expect(onchange).toHaveBeenCalled();

    disposeOnchange(store);
  });
});
