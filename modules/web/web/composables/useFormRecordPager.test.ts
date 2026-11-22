// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, nextTick, ref } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import { useFormRecordPager } from './useFormRecordPager';
import { useBreadcrumbStore } from '@/web/web/stores/breadcrumbStore';
import { useRecordSetStore } from '@/web/web/stores/recordSetStore';

test('goNext marks breadcrumb replace before pushing the sibling route', async () => {
  setActivePinia(createPinia());
  const recordSet = useRecordSetStore();
  recordSet.seed({
    key: 'k',
    model: 'base.Country',
    condition: {},
    total: 2,
    index: 0,
    ids: ['a', 'b'],
    windowStart: 0,
  });

  const breadcrumb = useBreadcrumbStore();
  const modes: unknown[] = [];
  const original = breadcrumb.setPendingPageMode.bind(breadcrumb);
  breadcrumb.setPendingPageMode = (mode: 'push' | 'replace' | null) => {
    modes.push(mode);
    original(mode);
  };

  const pushed: unknown[] = [];
  const router = {
    currentRoute: {
      value: {
        path: '/base/countries/a',
        name: 'CountryDetail',
        params: { id: 'a' },
        query: {},
      },
    },
    push: (to: unknown) => {
      pushed.push(to);
      return Promise.resolve();
    },
  };

  const pager = useFormRecordPager({
    model: computed(() => 'base.Country'),
    recordId: computed(() => 'a'),
    isCreate: computed(() => false),
    router,
    store: {},
    confirmDiscardIfDirty: async () => true,
  });

  await pager.goNext();
  expect(modes).toEqual(['replace']);
  expect(pushed).toEqual([
    { name: 'CountryDetail', params: { id: 'b' }, query: {} },
  ]);
  expect(recordSet.active?.index).toBe(1);
});

test('goTo jumps by 1-based position and marks breadcrumb replace', async () => {
  setActivePinia(createPinia());
  const recordSet = useRecordSetStore();
  recordSet.seed({
    key: 'k',
    model: 'base.Country',
    condition: {},
    total: 3,
    index: 0,
    ids: ['a', 'b', 'c'],
    windowStart: 0,
  });

  const breadcrumb = useBreadcrumbStore();
  const modes: unknown[] = [];
  const original = breadcrumb.setPendingPageMode.bind(breadcrumb);
  breadcrumb.setPendingPageMode = (mode: 'push' | 'replace' | null) => {
    modes.push(mode);
    original(mode);
  };

  const pushed: unknown[] = [];
  const router = {
    currentRoute: {
      value: {
        path: '/base/countries/a',
        name: 'CountryDetail',
        params: { id: 'a' },
        query: {},
      },
    },
    push: (to: unknown) => {
      pushed.push(to);
      return Promise.resolve();
    },
  };

  const pager = useFormRecordPager({
    model: computed(() => 'base.Country'),
    recordId: computed(() => 'a'),
    isCreate: computed(() => false),
    router,
    store: {},
    confirmDiscardIfDirty: async () => true,
  });

  await pager.goTo(3);
  expect(modes).toEqual(['replace']);
  expect(pushed).toEqual([
    { name: 'CountryDetail', params: { id: 'c' }, query: {} },
  ]);
  expect(recordSet.active?.index).toBe(2);

  pushed.length = 0;
  modes.length = 0;
  await pager.goTo(3);
  expect(pushed).toEqual([]);
  expect(modes).toEqual([]);
});

test('goNext restores recordSet when router.push fails', async () => {
  setActivePinia(createPinia());
  const recordSet = useRecordSetStore();
  recordSet.seed({
    key: 'k',
    model: 'base.Country',
    condition: {},
    total: 2,
    index: 0,
    ids: ['a', 'b'],
    windowStart: 0,
  });
  const breadcrumb = useBreadcrumbStore();
  const modes: unknown[] = [];
  const original = breadcrumb.setPendingPageMode.bind(breadcrumb);
  breadcrumb.setPendingPageMode = (mode: 'push' | 'replace' | null) => {
    modes.push(mode);
    original(mode);
  };

  const pager = useFormRecordPager({
    model: computed(() => 'base.Country'),
    recordId: computed(() => 'a'),
    isCreate: computed(() => false),
    router: {
      currentRoute: {
        value: { path: '/base/countries/a', name: 'CountryDetail', params: { id: 'a' }, query: {} },
      },
      push: () => Promise.reject(new Error('aborted')),
    },
    store: {},
    confirmDiscardIfDirty: async () => true,
  });

  await pager.goNext();
  expect(modes).toEqual(['replace', null]);
  expect(recordSet.active?.index).toBe(0);
  expect(recordSet.active?.ids).toEqual(['a', 'b']);
});

test('path siblingLocation preserves query and syncs index from recordId', async () => {
  setActivePinia(createPinia());
  const recordSet = useRecordSetStore();
  recordSet.seed({
    key: 'k',
    model: 'base.Country',
    condition: {},
    total: 2,
    index: 0,
    ids: ['a', 'b'],
    windowStart: 0,
  });

  const recordId = ref('a');
  const pushed: unknown[] = [];
  const pager = useFormRecordPager({
    model: computed(() => 'base.Country'),
    recordId,
    isCreate: computed(() => false),
    router: {
      currentRoute: {
        value: {
          path: '/base/countries/a',
          query: { tab: 'details' },
        },
      },
      push: (to: unknown) => {
        pushed.push(to);
        return Promise.resolve();
      },
    },
    store: {},
    confirmDiscardIfDirty: async () => true,
  });

  await pager.goNext();
  expect(pushed).toEqual([{ path: '/base/countries/b', query: { tab: 'details' } }]);
  expect(recordSet.active?.index).toBe(1);

  // Simulate route settling on the moved id, then browser Back to the prior id.
  recordId.value = 'b';
  await nextTick();
  recordId.value = 'a';
  await nextTick();
  expect(recordSet.active?.index).toBe(0);
});

test('create mode clears the active record set; discard cancel skips move', async () => {
  setActivePinia(createPinia());
  const recordSet = useRecordSetStore();
  recordSet.seed({
    key: 'k',
    model: 'base.Country',
    condition: {},
    total: 2,
    index: 0,
    ids: ['a', 'b'],
    windowStart: 0,
  });
  const isCreate = ref(false);
  const pager = useFormRecordPager({
    model: computed(() => 'base.Country'),
    recordId: computed(() => 'a'),
    isCreate,
    router: {
      currentRoute: { value: { path: '/base/countries/a', params: { id: 'a' }, name: 'CountryDetail', query: {} } },
      push: () => Promise.resolve(),
    },
    store: {},
    confirmDiscardIfDirty: async () => false,
  });
  expect(pager.visible.value).toBe(true);
  await pager.goNext();
  expect(recordSet.active?.index).toBe(0);

  isCreate.value = true;
  await nextTick();
  expect(recordSet.active).toBeNull();
});

test('goPrev and Search-backed moveTo cover buildSearchIds', async () => {
  setActivePinia(createPinia());
  const recordSet = useRecordSetStore();
  recordSet.seed({
    key: 'k',
    model: 'base.Country',
    condition: {},
    total: 5,
    index: 2,
    ids: ['a', 'b', 'c'],
    windowStart: 0,
  });
  const searches: unknown[] = [];
  const pager = useFormRecordPager({
    model: computed(() => 'base.Country'),
    recordId: computed(() => 'c'),
    isCreate: computed(() => false),
    router: {
      currentRoute: {
        value: { path: '/base/countries/c', name: 'CountryDetail', params: { id: 'c' }, query: {} },
      },
      push: () => Promise.resolve(),
    },
    store: {
      Search: async (_c?: unknown, opts?: Record<string, unknown>) => {
        searches.push(opts);
        return [{ Id: 'd' }, { Id: 'e' }];
      },
    },
    confirmDiscardIfDirty: async () => true,
  });
  await pager.goPrev();
  expect(recordSet.active?.index).toBe(1);
  await pager.goTo(5);
  expect(searches.length).toBeGreaterThan(0);
  expect(recordSet.active?.index).toBe(4);
});
