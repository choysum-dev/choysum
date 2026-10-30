// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { AuthTerminologyGrantCleanupHooks, AuthUserLanguageHooks } from './post_upgrade';

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

test('AuthTerminologyGrantCleanupHooks: deletes obsolete xml_id rows', async () => {
  const calls: string[] = [];
  const prev = (globalThis as any).$choysum;
  (globalThis as any).$choysum = {
    ...(prev || {}),
    db: {
      execute: async (sql: string) => {
        calls.push(sql);
      },
    },
  };
  try {
    await AuthTerminologyGrantCleanupHooks.revokeObsoleteTerminologyEditorGrants();
  } finally {
    (globalThis as any).$choysum = prev;
  }
  expect(calls.length).toBe(3);
  expect(calls[0]).toContain('auth_role_method_access');
  expect(calls[0]).toContain('rma_terminology_editor_tt_logical');
  expect(calls[1]).toContain('auth_role_field_rule');
  expect(calls[1]).toContain('rfr_terminology_editor_tt_logical');
  expect(calls[2]).toContain('meta_model_data');
});

test('AuthTerminologyGrantCleanupHooks: no-ops without db', async () => {
  const prev = (globalThis as any).$choysum;
  (globalThis as any).$choysum = { ...(prev || {}), db: undefined };
  try {
    await AuthTerminologyGrantCleanupHooks.revokeObsoleteTerminologyEditorGrants();
  } finally {
    (globalThis as any).$choysum = prev;
  }
});

test('AuthTerminologyGrantCleanupHooks: continues after execute failures', async () => {
  let n = 0;
  const prev = (globalThis as any).$choysum;
  (globalThis as any).$choysum = {
    ...(prev || {}),
    db: {
      execute: async () => {
        n += 1;
        throw new Error('missing table');
      },
    },
  };
  try {
    await AuthTerminologyGrantCleanupHooks.revokeObsoleteTerminologyEditorGrants();
  } finally {
    (globalThis as any).$choysum = prev;
  }
  expect(n).toBe(3);
});
