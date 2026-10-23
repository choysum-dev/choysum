// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, nextTick } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import RelationCombobox from './RelationCombobox.vue';

describe('RelationCombobox mount', () => {
  test('clear control is an icon button with translated aria-label inside the input', async () => {
    const m = mountApp(RelationCombobox as any, {
      props: {
        modelValue: 'p1',
        selectedOption: { id: 'p1', label: 'Pinned', raw: { Id: 'p1', DisplayName: 'Pinned' } },
        search: async () => [],
        searchKey: 'partner.Partner',
        clearable: true,
        debounce: 0,
      },
    });
    await nextTick();

    const clear = m.q('[data-testid="choy-relation-clear"]') as HTMLElement | null;
    expect(clear).toBeTruthy();
    expect(clear!.getAttribute('aria-label')).toBe('Clear');
    // Visible label must not be the hardcoded English word (icon-only).
    expect((clear!.textContent || '').trim()).toBe('');
    // Clear is absolutely positioned inside the relative ComboboxAnchor (not a sibling outside).
    const parent = clear!.parentElement as HTMLElement | null;
    expect(parent).toBeTruthy();
    expect(String(parent!.className || '')).toContain('relative');
    expect(String(clear!.className || '')).toContain('absolute');
    m.unmount();
  });

  test('focus opens the panel and runs NameSearch', async () => {
    const calls: string[] = [];
    const m = mountApp(RelationCombobox as any, {
      props: {
        modelValue: null,
        search: async (q: string) => {
          calls.push(q);
          return [];
        },
        searchKey: 'partner.Partner',
        debounce: 0,
      },
    });
    await nextTick();
    const input = m.q('input') as HTMLInputElement | null;
    expect(input).toBeTruthy();
    input!.dispatchEvent(new Event('focus', { bubbles: true }));
    await nextTick();
    await flushPromises();
    const state = m.setupState() as { open?: boolean } | undefined;
    expect(state?.open).toBe(true);
    expect(calls.length).toBeGreaterThan(0);
    m.unmount();
  });

  test('open panel shows translated empty, searching, and search-more copy', async () => {
    let resolveSearch: ((rows: any[]) => void) | null = null;
    const search = () =>
      new Promise<any[]>((resolve) => {
        resolveSearch = resolve;
      });

    const m = mountApp(RelationCombobox as any, {
      props: {
        modelValue: null,
        search,
        searchKey: 'partner.Partner',
        searchMore: true,
        debounce: 0,
      },
      slots: {
        footer: () => h('button', { type: 'button', 'data-testid': 'footer-slot' }, 'Create "x"'),
      },
    });
    await nextTick();

    // setupState unwraps refs; assign the boolean to open the panel.
    const state = m.setupState() as { open?: boolean } | undefined;
    expect(state).toBeTruthy();
    state!.open = true;
    await nextTick();
    await flushPromises();

    const bodyText = () => String(document.body.textContent || '');
    expect(bodyText()).toContain('Searching…');

    resolveSearch?.([]);
    await flushPromises();
    await nextTick();

    expect(bodyText()).toContain('No matches');
    expect(bodyText()).toContain('Search more…');

    expect(document.body.querySelector('[data-testid="footer-slot"]')).toBeTruthy();
    const more = document.body.querySelector('[data-testid="choy-relation-search-more"]') as HTMLElement | null;
    expect(more).toBeTruthy();
    // Action affordance: primary text like Create, not plain foreground like data rows.
    expect(more!.className).toContain('text-primary');

    m.unmount();
  });

  test('reopen keeps prior options and skips a second NameSearch', async () => {
    let resolveSearch: ((rows: any[]) => void) | null = null;
    let calls = 0;
    const search = () => {
      calls += 1;
      return new Promise<any[]>((resolve) => {
        resolveSearch = resolve;
      });
    };

    const m = mountApp(RelationCombobox as any, {
      props: {
        modelValue: null,
        search,
        searchKey: 'partner.Partner',
        debounce: 0,
      },
    });
    await nextTick();
    const state = m.setupState() as { open?: boolean } | undefined;
    state!.open = true;
    await flushPromises();
    resolveSearch?.([{ Id: 'c1', DisplayName: 'Cached' }]);
    await flushPromises();
    await nextTick();
    expect(String(document.body.textContent || '')).toContain('Cached');
    expect(calls).toBe(1);

    state!.open = false;
    await nextTick();
    resolveSearch = null;
    state!.open = true;
    await nextTick();
    await flushPromises();

    const bodyText = String(document.body.textContent || '');
    expect(bodyText).toContain('Cached');
    expect(bodyText).not.toContain('Searching…');
    // Identical rpcQuery + searchKey → memoized, no second RPC.
    expect(calls).toBe(1);
    m.unmount();
  });

  test('focus with a selected value NameSearches empty string (full page)', async () => {
    const calls: string[] = [];
    const m = mountApp(RelationCombobox as any, {
      props: {
        modelValue: 'c1',
        selectedOption: { id: 'c1', label: '启用分支', raw: { Id: 'c1', DisplayName: '启用分支' } },
        search: async (q: string) => {
          calls.push(q);
          return [
            { Id: 'c1', DisplayName: '启用分支' },
            { Id: 'c2', DisplayName: '根分类' },
          ];
        },
        searchKey: 'partner.Partner',
        debounce: 0,
      },
    });
    await nextTick();
    const input = m.q('input') as HTMLInputElement | null;
    expect(input).toBeTruthy();
    // Simulate Combobox syncing the display label into the filter without a user edit.
    input!.value = '启用分支';
    input!.dispatchEvent(new Event('focus', { bubbles: true }));
    await nextTick();
    await flushPromises();

    expect(calls.length).toBeGreaterThan(0);
    expect(calls[0]).toBe('');
    const bodyText = String(document.body.textContent || '');
    expect(bodyText).toContain('启用分支');
    expect(bodyText).toContain('根分类');
    m.unmount();
  });

  test('editing after focus NameSearches the typed keyword', async () => {
    const calls: string[] = [];
    const m = mountApp(RelationCombobox as any, {
      props: {
        modelValue: 'c1',
        selectedOption: { id: 'c1', label: '启用分支', raw: { Id: 'c1', DisplayName: '启用分支' } },
        search: async (q: string) => {
          calls.push(q);
          return q ? [{ Id: 'c2', DisplayName: '根分类' }] : [{ Id: 'c1', DisplayName: '启用分支' }];
        },
        searchKey: 'partner.Partner',
        debounce: 0,
      },
    });
    await nextTick();
    const input = m.q('input') as HTMLInputElement | null;
    expect(input).toBeTruthy();
    input!.dispatchEvent(new Event('focus', { bubbles: true }));
    await nextTick();
    await flushPromises();
    expect(calls[0]).toBe('');

    // Simulate a user edit (native @input → userEdited; v-model updates query).
    const state = m.setupState() as { query?: string; userEdited?: boolean } | undefined;
    state!.userEdited = true;
    state!.query = '根';
    await nextTick();
    await flushPromises();

    expect(calls.some((q) => q === '根')).toBe(true);
    m.unmount();
  });

  test('search failure surfaces translated inline error', async () => {
    const m = mountApp(RelationCombobox as any, {
      props: {
        modelValue: null,
        search: async () => {
          throw new Error('boom');
        },
        searchKey: 'partner.Partner',
        debounce: 0,
      },
    });
    await nextTick();
    const state = m.setupState() as { open?: boolean } | undefined;
    state!.open = true;
    await flushPromises();
    await nextTick();
    expect(String(document.body.textContent || '')).toContain('Search failed. Please retry.');
    m.unmount();
  });
});
