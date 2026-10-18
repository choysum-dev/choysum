// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoySearchFit from './ChoySearchFit.vue';
import ChoyViewActionBar from './ChoyViewActionBar.vue';
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

function mountFit(slotDefault?: () => unknown) {
  const Host = defineComponent({
    name: 'SearchFitHost',
    setup() {
      return () =>
        h(
          ChoyViewActionBar,
          { layout: 'thirds' },
          {
            center: () =>
              h(ChoySearchFit, null, {
                default: slotDefault || (() => h('input', { 'data-testid': 'search-field' })),
              }),
          },
        );
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
  expect(mounted!.q('[data-slot=search-fit-row]')).not.toBeNull();
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
  // Wide: field stays under the toolbar center, not the below-bar row.
  expect(field.closest('[data-slot=view-action-bar]')).not.toBeNull();
  mounted!.unmount();
});

test('ChoySearchFit width threshold is 640px of the action bar', () => {
  expect(searchFieldFits(640)).toBe(true);
  expect(searchFieldFits(639)).toBe(false);
});

test('ChoySearchFit shows the active dot from a teleported search node', async () => {
  const prevMo = (globalThis as { MutationObserver?: unknown }).MutationObserver;
  let lastObserver: { trigger: () => void } | null = null;
  class FakeMutationObserver {
    private cb: () => void;
    constructor(cb: () => void) {
      this.cb = cb;
      lastObserver = this;
    }
    observe() {}
    disconnect() {}
    trigger() {
      this.cb();
    }
  }
  (globalThis as { MutationObserver?: unknown }).MutationObserver = FakeMutationObserver;

  let mounted: ReturnType<typeof mountFit> | null = null;
  try {
    withWidth(480, () => {
      mounted = mountFit(() =>
        h('div', {
          'data-anchor': 'choy.search',
          'data-search-active': 'true',
          'data-testid': 'search-anchor',
        }),
      );
    });
    await flushPromises();
    const anchor = mounted!.q('[data-testid=search-anchor]') as HTMLElement;
    expect(anchor.closest('[data-slot=search-fit-row]')).not.toBeNull();
    expect(mounted!.q('[data-testid=choy-search-fit-dot]')).not.toBeNull();
    expect(mounted!.q('[data-testid=choy-search-fit-toggle]')?.getAttribute('data-search-active')).toBe(
      'true',
    );
    anchor.setAttribute('data-search-active', 'false');
    lastObserver!.trigger();
    await flushPromises();
    expect(mounted!.q('[data-testid=choy-search-fit-dot]')).toBeNull();
    expect(mounted!.q('[data-testid=choy-search-fit-toggle]')?.getAttribute('data-search-active')).toBe(
      'false',
    );
  } finally {
    (globalThis as { MutationObserver?: unknown }).MutationObserver = prevMo;
    mounted!.unmount();
  }
});

test('ChoySearchFit hides the teleported field when fieldVisible is false', async () => {
  const Host = defineComponent({
    name: 'SearchFitVisibleHost',
    setup() {
      return () =>
        h(
          ChoyViewActionBar,
          { layout: 'thirds' },
          {
            center: () =>
              h(
                ChoySearchFit,
                { fieldVisible: false },
                {
                  default: () => h('input', { 'data-testid': 'search-field' }),
                },
              ),
          },
        );
    },
  });
  let mounted: ReturnType<typeof mountApp> | null = null;
  withWidth(480, () => {
    mounted = mountApp(Host);
  });
  await flushPromises();
  mounted!.click('[data-testid=choy-search-fit-toggle]');
  await flushPromises();
  const field = mounted!.q('[data-testid=search-field]') as HTMLElement;
  expect(field.closest('[data-slot=search-fit-row]')).not.toBeNull();
  expect(field.parentElement?.style.display).toBe('none');
  mounted!.unmount();
});

test('ChoySearchFit focuses the input when the collapsed field opens', async () => {
  let mounted: ReturnType<typeof mountFit> | null = null;
  withWidth(480, () => {
    mounted = mountFit();
  });
  await flushPromises();
  const field = mounted!.q('[data-testid=search-field]') as HTMLInputElement;
  let focused = false;
  field.focus = () => {
    focused = true;
  };
  mounted!.click('[data-testid=choy-search-fit-toggle]');
  await flushPromises();
  expect(focused).toBe(true);
  mounted!.unmount();
});
