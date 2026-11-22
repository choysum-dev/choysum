// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import { applyIdToPathTemplate, canOpenDetailRow, resolveDetailPath } from './detailPath';
import { openRecordFromCollection } from './openRecordFromCollection';
import { snapshotFromListStore } from './snapshotFromListStore';
import { useRecordSetStore } from '@/web/web/stores/recordSetStore';

test('applyIdToPathTemplate replaces :id and appends bare prefixes', () => {
  expect(applyIdToPathTemplate('/base/countries/:id', 'c1')).toBe('/base/countries/c1');
  expect(applyIdToPathTemplate('/base/countries/{id}', 'c1')).toBe('/base/countries/c1');
  expect(applyIdToPathTemplate('/base/countries', 'c1')).toBe('/base/countries/c1');
  expect(applyIdToPathTemplate('/base/countries/c1', 'c1')).toBe('/base/countries/c1');
  expect(applyIdToPathTemplate('/base/countries?tab=details', 'c1')).toBe(
    '/base/countries/c1?tab=details'
  );
  expect(applyIdToPathTemplate('/base/countries#sec', 'c1')).toBe('/base/countries/c1#sec');
});

test('resolveDetailPath supports templates and factories', () => {
  expect(resolveDetailPath('/base/countries/:id', { Id: 'x' })).toBe('/base/countries/x');
  expect(
    resolveDetailPath(row => `/web/app-settings/${row.App}/${row.Id}`, { Id: '1', App: 'web' })
  ).toBe('/web/app-settings/web/1');
  expect(resolveDetailPath(() => null, { Id: '1' })).toBeNull();
  expect(
    resolveDetailPath(() => ({ name: 'MetaModelDataDetail', params: { id: '1' } }), { Id: '1' })
  ).toEqual({ name: 'MetaModelDataDetail', params: { id: '1' } });
});

test('canOpenDetailRow gates navigation', () => {
  expect(canOpenDetailRow(undefined, { Id: '1' })).toBe(true);
  expect(canOpenDetailRow(false, { Id: '1' })).toBe(false);
  expect(canOpenDetailRow(row => String(row.Id) === 'ok', { Id: 'ok' })).toBe(true);
  expect(canOpenDetailRow(row => String(row.Id) === 'ok', { Id: 'no' })).toBe(false);
});

test('snapshotFromListStore builds global index and page ids', () => {
  const store = {
    modelName: 'base.Country',
    state: {
      queryState: {
        appliedFilters: [],
        pagination: { offset: 100, limit: 100 },
        orderBy: [{ field: 'Name', direction: 'asc' }],
        forcedCondition: { IsActive: true },
      },
      result: {
        total: 249,
        rows: [{ Id: 'a' }, { Id: 'b' }, { Id: 'c' }],
      },
    },
    metadata: { fields: {} },
  };
  const entry = snapshotFromListStore(store, {
    model: 'base.Country',
    key: 'base.Country|list',
    rowIndex: 1,
    recordId: 'b',
  });
  expect(entry?.total).toBe(249);
  expect(entry?.index).toBe(101);
  expect(entry?.ids).toEqual(['a', 'b', 'c']);
  expect(entry?.windowStart).toBe(100);
  expect(entry?.condition).toEqual({ IsActive: true });

  // Missing Search total must still span offset + page before clamping index.
  const noTotal = snapshotFromListStore(
    {
      ...store,
      state: {
        ...store.state,
        result: { rows: [{ Id: 'a' }, { Id: 'b' }, { Id: 'c' }] },
      },
    },
    { model: 'base.Country', key: 'k', rowIndex: 2, recordId: 'c' }
  );
  expect(noTotal?.total).toBe(103);
  expect(noTotal?.index).toBe(102);
});

test('openRecordFromCollection seeds store and pushes route', () => {
  setActivePinia(createPinia());
  const pushed: unknown[] = [];
  openRecordFromCollection({
    router: { push: to => pushed.push(to) },
    to: '/base/countries/b',
    entry: {
      key: 'k',
      model: 'base.Country',
      condition: {},
      total: 3,
      index: 1,
      ids: ['a', 'b', 'c'],
      windowStart: 0,
    },
  });
  expect(useRecordSetStore().active?.model).toBe('base.Country');
  expect(pushed).toEqual(['/base/countries/b']);
});
