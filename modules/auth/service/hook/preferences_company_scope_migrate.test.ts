// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { migratePreferencesCompanyScopeToColumns } from './preferences_company_scope_migrate';

test('migratePreferencesCompanyScopeToColumns: copies Preferences scope into empty columns', async () => {
  const executes: Array<{ sql: string; params: string }> = [];
  const prev = (globalThis as any).$choysum;
  (globalThis as any).$choysum = {
    ...(prev || {}),
    db: {
      query: async () =>
        JSON.stringify([
          {
            id: 'u1',
            preferences: JSON.stringify({
              activeCompanyId: 'c_active',
              enabledCompanyIds: ['c_active', 'c2'],
            }),
            active_company_id: null,
            enabled_company_ids: null,
          },
        ]),
      execute: async (sql: string, params: string) => {
        executes.push({ sql, params });
      },
    },
  };
  try {
    await migratePreferencesCompanyScopeToColumns();
  } finally {
    (globalThis as any).$choysum = prev;
  }
  expect(executes).toHaveLength(1);
  expect(executes[0].sql).toContain('active_company_id');
  expect(executes[0].sql).toContain('enabled_company_ids');
  expect(JSON.parse(executes[0].params)).toEqual([
    'c_active',
    JSON.stringify(['c_active', 'c2']),
    'u1',
  ]);
});

test('migratePreferencesCompanyScopeToColumns: keeps existing EnabledCompanyIds', async () => {
  const executes: Array<{ sql: string; params: string }> = [];
  const prev = (globalThis as any).$choysum;
  (globalThis as any).$choysum = {
    ...(prev || {}),
    db: {
      query: async () =>
        JSON.stringify([
          {
            id: 'u2',
            preferences: JSON.stringify({
              activeCompanyId: 'c_new',
              enabledCompanyIds: ['c_new'],
            }),
            active_company_id: '',
            enabled_company_ids: JSON.stringify(['c_keep']),
          },
        ]),
      execute: async (sql: string, params: string) => {
        executes.push({ sql, params });
      },
    },
  };
  try {
    await migratePreferencesCompanyScopeToColumns();
  } finally {
    (globalThis as any).$choysum = prev;
  }
  expect(executes).toHaveLength(1);
  expect(executes[0].sql).toContain('active_company_id');
  expect(executes[0].sql).not.toContain('enabled_company_ids');
  expect(JSON.parse(executes[0].params)[0]).toBe('c_new');
});

test('migratePreferencesCompanyScopeToColumns: no-ops without db or preferences column', async () => {
  const prev = (globalThis as any).$choysum;
  (globalThis as any).$choysum = { ...(prev || {}), db: undefined };
  try {
    await migratePreferencesCompanyScopeToColumns();
  } finally {
    (globalThis as any).$choysum = prev;
  }

  (globalThis as any).$choysum = {
    ...(prev || {}),
    db: {
      query: async () => {
        throw new Error('no such column: preferences');
      },
      execute: async () => {
        throw new Error('should not run');
      },
    },
  };
  try {
    await migratePreferencesCompanyScopeToColumns();
  } finally {
    (globalThis as any).$choysum = prev;
  }
});
