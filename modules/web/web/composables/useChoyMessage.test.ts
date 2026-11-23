// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import { ChoyMessage, useChoyMessage } from '@/web/web/composables/useChoyMessage';
import { toast } from 'vue-sonner';
// Pull a .vue import so the FE unit runner enables the Vue host bundle (resolves `vue`).
import Toaster from '@/web/web/components/vendor/ui/sonner/Sonner.vue';
import { useConnectivityStore } from '../stores/connectivityStore';

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
  beforeEach(() => {
    setActivePinia(createPinia());
    useConnectivityStore().resetForTests();
  });

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
    expect(list[0].title).toBe('Saved');
    expect(list[0].level).toBe('success');
    expect(list[1].title).toBe('Check fields');
    expect(list[1].description).toBe('Name is empty');
    expect(list[2].title).toBe('Failed');
    expect(list[3].title).toBe('Hint');
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
    api.success('');
    expect(entries()[0].title).toBe('Success');
    clearToasts();
    api.warning('  ');
    expect(entries()[0].title).toBe('Warning');
    clearToasts();
    api.error(null as unknown as string);
    expect(entries()[0].title).toBe('Error');
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

  test('suppresses transport-like errors while unreachable', () => {
    clearToasts();
    useConnectivityStore().markUnreachable();
    expect(ChoyMessage.error('Failed to fetch')).toBe(0);
    expect(entries().length).toBe(0);
    expect(ChoyMessage.error('Name is required')).not.toBe(0);
    expect(entries().length).toBe(1);
    clearToasts();
  });

  test('keeps ambiguous Load failed visible while unreachable', () => {
    clearToasts();
    useConnectivityStore().markUnreachable();
    expect(ChoyMessage.error('Load failed', { description: 'Invalid file format' })).not.toBe(0);
    expect(entries().length).toBe(1);
    clearToasts();
  });

  test('transportOutage option forces suppress or keep while unreachable', () => {
    clearToasts();
    useConnectivityStore().markUnreachable();
    expect(ChoyMessage.error('Business boom', { transportOutage: true })).toBe(0);
    expect(ChoyMessage.error('Failed to fetch', { transportOutage: false })).not.toBe(0);
    expect(entries().length).toBe(1);
    clearToasts();
  });

  test('error suppress path tolerates a broken active Pinia', () => {
    clearToasts();
    setActivePinia({} as any);
    expect(ChoyMessage.error('Failed to fetch')).not.toBe(0);
    expect(entries().length).toBe(1);
    clearToasts();
    setActivePinia(createPinia());
  });
});
