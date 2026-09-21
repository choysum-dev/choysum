// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { AuthUserLanguageHooks } from './post_upgrade';

test('AuthUserLanguageHooks.backfillUserLanguageId: sqlite SQL filters active trimmed codes', async () => {
  const calls: Array<{ sql: string; params: string }> = [];
  const prev = (globalThis as any).$choysum;
  (globalThis as any).$choysum = {
    ...(prev || {}),
    db: {
      dialectName: 'sqlite',
      execute: async (sql: string, params: string) => {
        calls.push({ sql, params });
      },
    },
  };
  try {
    await AuthUserLanguageHooks.backfillUserLanguageId();
  } finally {
    (globalThis as any).$choysum = prev;
  }
  expect(calls.length).toBe(1);
  expect(calls[0].params).toBe('[]');
  expect(calls[0].sql).toContain('is_active = 1');
  expect(calls[0].sql).toContain('trim(auth_user.language)');
});

test('AuthUserLanguageHooks.backfillUserLanguageId: postgres SQL filters active trimmed codes', async () => {
  const calls: Array<{ sql: string; params: string }> = [];
  const prev = (globalThis as any).$choysum;
  (globalThis as any).$choysum = {
    ...(prev || {}),
    db: {
      dialectName: 'postgres',
      execute: async (sql: string, params: string) => {
        calls.push({ sql, params });
      },
    },
  };
  try {
    await AuthUserLanguageHooks.backfillUserLanguageId();
  } finally {
    (globalThis as any).$choysum = prev;
  }
  expect(calls.length).toBe(1);
  expect(calls[0].sql).toContain('is_active = true');
  expect(calls[0].sql).toContain('trim(auth_user.language)');
});

test('AuthUserLanguageHooks.backfillUserLanguageId: ignores execute failures', async () => {
  const prev = (globalThis as any).$choysum;
  (globalThis as any).$choysum = {
    ...(prev || {}),
    db: {
      dialectName: 'sqlite',
      execute: async () => {
        throw new Error('no language column');
      },
    },
  };
  try {
    await AuthUserLanguageHooks.backfillUserLanguageId();
  } finally {
    (globalThis as any).$choysum = prev;
  }
});

test('AuthUserLanguageHooks.backfillUserLanguageId: no-ops without db', async () => {
  const prev = (globalThis as any).$choysum;
  (globalThis as any).$choysum = { ...(prev || {}), db: undefined };
  try {
    await AuthUserLanguageHooks.backfillUserLanguageId();
  } finally {
    (globalThis as any).$choysum = prev;
  }
});
