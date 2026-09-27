// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoyMessage, useChoyMessage } from '@/web/web/composables/useChoyMessage';
import {
  clearToasts,
  useToastStore,
} from '@/web/web/components/vendor/ui/toast/useToast';
// Pull a .vue import so the FE unit runner enables the Vue host bundle (resolves `vue`).
import Toaster from '@/web/web/components/vendor/ui/toast/Toaster.vue';

void Toaster;

describe('useChoyMessage', () => {
  test('exposes ChoyMessage facade levels onto the toast store', () => {
    clearToasts();
    const store = useToastStore();
    const api = useChoyMessage();

    expect(api).toBe(ChoyMessage);

    const successId = api.success('Saved');
    const warningId = api.warning('Check fields', { description: 'Name is empty' });
    const errorId = api.error('Failed');
    const infoId = api.info('Hint', { duration: 0 });

    expect(store.value.length).toBe(4);
    expect(store.value.map((t) => t.id)).toEqual([successId, warningId, errorId, infoId]);
    expect(store.value[0].title).toBe('Success: Saved');
    expect(store.value[1].title).toBe('Warning: Check fields');
    expect(store.value[1].description).toBe('Name is empty');
    expect(store.value[2].title).toBe('Error: Failed');
    expect(store.value[3].title).toBe('Info: Hint');
    expect(store.value[3].duration).toBe(Number.POSITIVE_INFINITY);

    clearToasts();
    expect(store.value.length).toBe(0);
  });

  test('falls back to the bare level label for an empty title', () => {
    clearToasts();
    const api = useChoyMessage();
    api.info('');
    const store = useToastStore();
    expect(store.value[0].title).toBe('Info');
    clearToasts();
    api.info('   ');
    expect(store.value[0].title).toBe('Info');
    clearToasts();
  });

  test('toasts created without options keep the store defaults', () => {
    clearToasts();
    const api = useChoyMessage();
    api.success('Saved');
    const store = useToastStore();
    expect(store.value[0].duration).not.toBeUndefined();
    expect(store.value[0].duration).toBe(5000);
    expect(store.value[0].description).toBeUndefined();
    clearToasts();
  });

  test('ignores invalid durations and floors positive ones', () => {
    clearToasts();
    const api = useChoyMessage();
    const store = useToastStore();
    api.info('Bad', { duration: Number.NaN });
    expect(store.value[0].duration).toBe(5000);
    clearToasts();
    api.info('Neg', { duration: -1 });
    expect(store.value[0].duration).toBe(5000);
    clearToasts();
    api.info('Floor', { duration: 1500.9 });
    expect(store.value[0].duration).toBe(1500);
    clearToasts();
    api.info('Tiny', { duration: 0.5 });
    expect(store.value[0].duration).toBe(1);
    clearToasts();
    api.info('Huge', { duration: 1e12 });
    expect(store.value[0].duration).toBe(2_147_483_647);
    clearToasts();
    api.info('Sticky', { duration: 0 });
    expect(store.value[0].duration).toBe(Number.POSITIVE_INFINITY);
    clearToasts();
  });
});
