// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoySearchFit from './ChoySearchFit.vue';
import { searchFieldFits } from './searchFit';

function withWidth(width: number, run: () => void) {
  const prev = (globalThis as { ResizeObserver?: unknown }).ResizeObserver;
  class FakeObserver {
    private cb: (entries: Array<{ contentRect: { width: number } }>) => void;
    constructor(cb: (entries: Array<{ contentRect: { width: number } }>) => void) {
      this.cb = cb;
    }
    observe() {
      this.cb([{ contentRect: { width } }]);
    }
    disconnect() {}
  }
  (globalThis as { ResizeObserver?: unknown }).ResizeObserver = FakeObserver;
  try {
    run();
  } finally {
    (globalThis as { ResizeObserver?: unknown }).ResizeObserver = prev;
  }
}

function mountFit() {
  const Host = defineComponent({
    name: 'SearchFitHost',
    setup() {
      return () =>
        h('div', { 'data-slot': 'view-action-bar' }, [
          h(ChoySearchFit, null, { default: () => h('input', { 'data-testid': 'search-field' }) }),
        ]);
    },
  });
  return mountApp(Host);
}

test('ChoySearchFit collapses the field to an icon below the wide threshold', async () => {
  let mounted: ReturnType<typeof mountFit> | null = null;
  withWidth(480, () => {
    mounted = mountFit();
  });
  await flushPromises();
  const toggle = mounted!.q('[data-testid=choy-search-fit-toggle]') as HTMLElement;
  const field = mounted!.q('[data-testid=search-field]') as HTMLElement;
  expect(toggle.style.display).not.toBe('none');
  expect(field.parentElement?.style.display).toBe('none');
  mounted!.click('[data-testid=choy-search-fit-toggle]');
  await flushPromises();
  expect(field.parentElement?.style.display).not.toBe('none');
  expect(field.closest('[data-slot=view-action-bar]')).toBeNull();
  expect(field.closest('[data-slot=search-fit-row]')).not.toBeNull();
  mounted!.click('[data-testid=choy-search-fit-toggle]');
  await flushPromises();
  expect(field.parentElement?.style.display).toBe('none');
  mounted!.unmount();
});

test('ChoySearchFit keeps the field when the action bar is wide', async () => {
  let mounted: ReturnType<typeof mountFit> | null = null;
  withWidth(1280, () => {
    mounted = mountFit();
  });
  await flushPromises();
  const toggle = mounted!.q('[data-testid=choy-search-fit-toggle]') as HTMLElement;
  const field = mounted!.q('[data-testid=search-field]') as HTMLElement;
  expect(toggle.style.display).toBe('none');
  expect(field.parentElement?.style.display).not.toBe('none');
  mounted!.unmount();
});

test('ChoySearchFit width threshold is 640px of the action bar', () => {
  expect(searchFieldFits(640)).toBe(true);
  expect(searchFieldFits(639)).toBe(false);
});
