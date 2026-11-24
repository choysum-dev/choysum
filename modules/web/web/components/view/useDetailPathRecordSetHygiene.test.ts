// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import { defineComponent, h, nextTick } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { useRecordSetStore, type RecordSetEntry } from '@/web/web/stores/recordSetStore';
import { useDetailPathRecordSetHygiene } from './useDetailPathRecordSetHygiene';

function sampleEntry(over: Partial<RecordSetEntry> = {}): RecordSetEntry {
  return {
    key: 'partner.Partner|list',
    model: 'partner.Partner',
    condition: {},
    orderBy: [],
    total: 1,
    index: 0,
    ids: ['1'],
    windowStart: 0,
    ...over,
  };
}

describe('useDetailPathRecordSetHygiene', () => {
  test('clears active session when detailPath host matches model', async () => {
    setActivePinia(createPinia());
    const rs = useRecordSetStore();
    rs.seed(sampleEntry());
    const Host = defineComponent({
      name: 'HygieneHost',
      setup() {
        useDetailPathRecordSetHygiene({
          detailPath: '/partner/:id',
          modelName: 'partner.Partner',
        });
        return () => h('div');
      },
    });
    const mounted = mountApp(Host as any);
    await flushPromises();
    await nextTick();
    expect(rs.active).toBeNull();
    mounted.unmount();
  });

  test('no-ops when detailPath is null', async () => {
    setActivePinia(createPinia());
    const rs = useRecordSetStore();
    rs.seed(sampleEntry());
    const Host = defineComponent({
      setup() {
        useDetailPathRecordSetHygiene({
          detailPath: null,
          modelName: 'partner.Partner',
        });
        return () => h('div');
      },
    });
    const mounted = mountApp(Host as any);
    await flushPromises();
    expect(rs.active?.model).toBe('partner.Partner');
    mounted.unmount();
  });

  test('preserves active session for a different model', async () => {
    setActivePinia(createPinia());
    const rs = useRecordSetStore();
    rs.seed(sampleEntry({ model: 'base.Company', key: 'base.Company|list' }));
    const Host = defineComponent({
      setup() {
        useDetailPathRecordSetHygiene({
          detailPath: '/partner/:id',
          modelName: 'partner.Partner',
        });
        return () => h('div');
      },
    });
    const mounted = mountApp(Host as any);
    await flushPromises();
    expect(rs.active?.model).toBe('base.Company');
    mounted.unmount();
  });

  test('no-ops when modelName is empty so another model session survives', async () => {
    setActivePinia(createPinia());
    const rs = useRecordSetStore();
    rs.seed(sampleEntry({ model: 'base.Company', key: 'base.Company|list' }));
    const Host = defineComponent({
      setup() {
        useDetailPathRecordSetHygiene({
          detailPath: '/partner/:id',
          modelName: '',
        });
        return () => h('div');
      },
    });
    const mounted = mountApp(Host as any);
    await flushPromises();
    expect(rs.active?.model).toBe('base.Company');
    mounted.unmount();
  });

  test('no-ops when active.model is empty', async () => {
    setActivePinia(createPinia());
    const rs = useRecordSetStore();
    rs.seed(sampleEntry({ model: '', key: 'unknown|list' }));
    const Host = defineComponent({
      setup() {
        useDetailPathRecordSetHygiene({
          detailPath: '/partner/:id',
          modelName: 'partner.Partner',
        });
        return () => h('div');
      },
    });
    const mounted = mountApp(Host as any);
    await flushPromises();
    expect(rs.active).not.toBeNull();
    expect(rs.active?.key).toBe('unknown|list');
    mounted.unmount();
  });

  test('clearRecordSetSessionIfDetailHost swallows store errors', async () => {
    setActivePinia(createPinia());
    let clearFn: (() => void) | null = null;
    const Host = defineComponent({
      setup() {
        const api = useDetailPathRecordSetHygiene({
          detailPath: '/partner/:id',
          modelName: 'partner.Partner',
        });
        clearFn = api.clearRecordSetSessionIfDetailHost;
        return () => h('div');
      },
    });
    const mounted = mountApp(Host as any);
    await flushPromises();
    // Drop active Pinia so useRecordSetStore throws into the catch.
    setActivePinia(undefined as unknown as ReturnType<typeof createPinia>);
    expect(clearFn).not.toBeNull();
    expect(() => clearFn!()).not.toThrow();
    mounted.unmount();
  });
});
