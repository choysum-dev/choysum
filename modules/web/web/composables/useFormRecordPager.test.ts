// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed } from 'vue';
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
  breadcrumb.setPendingPageMode = (mode: 'push' | 'replace') => {
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
  breadcrumb.setPendingPageMode = (mode: 'push' | 'replace') => {
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
