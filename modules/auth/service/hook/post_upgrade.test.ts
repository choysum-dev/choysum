// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { AuthRoleInheritanceHooks, AuthUserLanguageHooks } from './post_upgrade';

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

test('AuthUserLanguageHooks.backfillUserLanguageId: mysql SQL filters active trimmed codes', async () => {
  const calls: Array<{ sql: string; params: string }> = [];
  const prev = (globalThis as any).$choysum;
  (globalThis as any).$choysum = {
    ...(prev || {}),
    db: {
      dialectName: 'mysql',
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

test('AuthUserLanguageHooks.backfillUserLanguageId: dialect name is trimmed and lowercased', async () => {
  const calls: Array<{ sql: string; params: string }> = [];
  const prev = (globalThis as any).$choysum;
  (globalThis as any).$choysum = {
    ...(prev || {}),
    db: {
      dialectName: ' SQLite ',
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
  expect(calls[0].sql).toContain('is_active = 1');
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

test('AuthUserLanguageHooks.backfillUserLanguageId: unknown dialect executes no SQL', async () => {
  const calls: Array<{ sql: string; params: string }> = [];
  const prev = (globalThis as any).$choysum;
  (globalThis as any).$choysum = {
    ...(prev || {}),
    db: {
      dialectName: 'oracle',
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
  expect(calls.length).toBe(0);
});

test('AuthRoleInheritanceHooks.purgeSoftDeletedRoleInheritance: deletes soft-deleted rows', async () => {
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
    await AuthRoleInheritanceHooks.purgeSoftDeletedRoleInheritance();
  } finally {
    (globalThis as any).$choysum = prev;
  }
  expect(calls.length).toBe(1);
  expect(calls[0].params).toBe('[]');
  expect(calls[0].sql).toContain('DELETE FROM auth_role_inheritance');
  expect(calls[0].sql).toContain('deleted_at IS NOT NULL');
});

test('AuthRoleInheritanceHooks.purgeSoftDeletedRoleInheritance: no-ops without db', async () => {
  const prev = (globalThis as any).$choysum;
  (globalThis as any).$choysum = { ...(prev || {}), db: undefined };
  try {
    await AuthRoleInheritanceHooks.purgeSoftDeletedRoleInheritance();
  } finally {
    (globalThis as any).$choysum = prev;
  }
});

test('AuthRoleInheritanceHooks.purgeSoftDeletedRoleInheritance: rethrows execute failures', async () => {
  let executed = 0;
  const prev = (globalThis as any).$choysum;
  (globalThis as any).$choysum = {
    ...(prev || {}),
    db: {
      dialectName: 'postgres',
      execute: async () => {
        executed += 1;
        throw new Error('missing table');
      },
    },
  };
  try {
    let threw: unknown;
    try {
      await AuthRoleInheritanceHooks.purgeSoftDeletedRoleInheritance();
    } catch (err) {
      threw = err;
    }
    expect(executed).toBe(1);
    expect(threw instanceof Error).toBe(true);
    expect(String((threw as Error).message)).toContain('missing table');
  } finally {
    (globalThis as any).$choysum = prev;
  }
});

test('AuthRoleInheritanceHooks.purgeSoftDeletedRoleInheritance: unknown dialect executes nothing', async () => {
  const calls: Array<{ sql: string; params: string }> = [];
  const prev = (globalThis as any).$choysum;
  (globalThis as any).$choysum = {
    ...(prev || {}),
    db: {
      dialectName: 'oracle',
      execute: async (sql: string, params: string) => {
        calls.push({ sql, params });
      },
    },
  };
  try {
    await AuthRoleInheritanceHooks.purgeSoftDeletedRoleInheritance();
  } finally {
    (globalThis as any).$choysum = prev;
  }
  expect(calls.length).toBe(0);
});
