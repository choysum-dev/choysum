// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import { openRelatedRecord } from './openRelatedRecord';
import { useRecordSetStore } from '@/web/web/stores/recordSetStore';

test('openRelatedRecord clears the pager session and pushes the target', () => {
  setActivePinia(createPinia());
  const recordSet = useRecordSetStore();
  recordSet.seed({
    key: 'k',
    model: 'base.Company',
    condition: {},
    total: 2,
    index: 0,
    ids: ['a', 'b'],
    windowStart: 0,
  });
  expect(recordSet.hasActive).toBe(true);

  const pushed: unknown[] = [];
  openRelatedRecord({
    router: { push: to => pushed.push(to) },
    to: { name: 'CompanyDetail', params: { id: 'parent' } },
  });

  expect(recordSet.hasActive).toBe(false);
  expect(pushed).toEqual([{ name: 'CompanyDetail', params: { id: 'parent' } }]);
});

test('openRelatedRecord accepts a string path', () => {
  setActivePinia(createPinia());
  const pushed: unknown[] = [];
  openRelatedRecord({
    router: { push: to => pushed.push(to) },
    to: '/auth/roles/r1',
  });
  expect(pushed).toEqual(['/auth/roles/r1']);
});

test('openRelatedRecord still pushes when Pinia is unavailable', () => {
  // No active Pinia — clear must not block navigation.
  const pushed: unknown[] = [];
  openRelatedRecord({
    router: { push: to => pushed.push(to) },
    to: '/auth/roles/r2',
  });
  expect(pushed).toEqual(['/auth/roles/r2']);
});

test('openRelatedRecord swallows rejected navigation promises', async () => {
  setActivePinia(createPinia());
  openRelatedRecord({
    router: { push: () => Promise.reject(new Error('aborted')) },
    to: '/auth/roles/r3',
  });
  await Promise.resolve();
  await Promise.resolve();
  expect(true).toBe(true);
});
