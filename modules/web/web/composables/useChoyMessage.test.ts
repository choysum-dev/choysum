// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoyMessage, useChoyMessage } from '@/web/web/composables/useChoyMessage';
import { toast } from 'vue-sonner';
// Pull a .vue import so the FE unit runner enables the Vue host bundle (resolves `vue`).
import Toaster from '@/web/web/components/vendor/ui/sonner/Sonner.vue';

void Toaster;

function clearToasts() {
  (toast as any)._clear?.();
}

function entries(): Array<{
  id: number
  level: string
  title: string
  description?: string
  duration?: number
}> {
  return (toast as any)._entries || [];
}

describe('useChoyMessage', () => {
  test('exposes ChoyMessage facade levels onto vue-sonner', () => {
    clearToasts();
    const api = useChoyMessage();

    expect(api).toBe(ChoyMessage);

    const successId = api.success('Saved');
    const warningId = api.warning('Check fields', { description: 'Name is empty' });
    const errorId = api.error('Failed');
    const infoId = api.info('Hint', { duration: 0 });

    const list = entries();
    expect(list.length).toBe(4);
    expect(list.map((t) => t.id)).toEqual([successId, warningId, errorId, infoId]);
    expect(list[0].title).toBe('Success: Saved');
    expect(list[0].level).toBe('success');
    expect(list[1].title).toBe('Warning: Check fields');
    expect(list[1].description).toBe('Name is empty');
    expect(list[2].title).toBe('Error: Failed');
    expect(list[3].title).toBe('Info: Hint');
    expect(list[3].duration).toBe(Number.POSITIVE_INFINITY);

    clearToasts();
    expect(entries().length).toBe(0);
  });

  test('falls back to the bare level label for an empty title', () => {
    clearToasts();
    const api = useChoyMessage();
    api.info('');
    expect(entries()[0].title).toBe('Info');
    clearToasts();
    api.info('   ');
    expect(entries()[0].title).toBe('Info');
    clearToasts();
  });

  test('toasts created without options omit duration override', () => {
    clearToasts();
    const api = useChoyMessage();
    api.success('Saved');
    expect(entries()[0].duration).toBeUndefined();
    expect(entries()[0].description).toBeUndefined();
    clearToasts();
  });

  test('ignores invalid durations and floors positive ones', () => {
    clearToasts();
    const api = useChoyMessage();
    api.info('Bad', { duration: Number.NaN });
    expect(entries()[0].duration).toBeUndefined();
    clearToasts();
    api.info('Neg', { duration: -1 });
    expect(entries()[0].duration).toBeUndefined();
    clearToasts();
    api.info('Floor', { duration: 1500.9 });
    expect(entries()[0].duration).toBe(1500);
    clearToasts();
    api.info('Tiny', { duration: 0.5 });
    expect(entries()[0].duration).toBe(1);
    clearToasts();
    api.info('Huge', { duration: 1e12 });
    expect(entries()[0].duration).toBe(2_147_483_647);
    clearToasts();
    api.info('Sticky', { duration: 0 });
    expect(entries()[0].duration).toBe(Number.POSITIVE_INFINITY);
    clearToasts();
  });
});
