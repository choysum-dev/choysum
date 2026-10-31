// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { revokeObsoleteTerminologyEditorGrants } from './terminology_grant_cleanup';

test('revokeObsoleteTerminologyEditorGrants: deletes obsolete xml_id rows', async () => {
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
    await revokeObsoleteTerminologyEditorGrants();
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

test('revokeObsoleteTerminologyEditorGrants: no-ops without db', async () => {
  const prev = (globalThis as any).$choysum;
  (globalThis as any).$choysum = { ...(prev || {}), db: undefined };
  try {
    await revokeObsoleteTerminologyEditorGrants();
  } finally {
    (globalThis as any).$choysum = prev;
  }
});

test('revokeObsoleteTerminologyEditorGrants: continues grant deletes after failures', async () => {
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
    await revokeObsoleteTerminologyEditorGrants();
  } finally {
    (globalThis as any).$choysum = prev;
  }
  // Both grant deletes run; meta_model_data cleanup is skipped when either fails.
  expect(n).toBe(2);
});

test('revokeObsoleteTerminologyEditorGrants: skips mapping cleanup when a grant delete fails', async () => {
  const calls: string[] = [];
  const prev = (globalThis as any).$choysum;
  (globalThis as any).$choysum = {
    ...(prev || {}),
    db: {
      execute: async (sql: string) => {
        calls.push(sql);
        if (sql.includes('auth_role_field_rule')) {
          throw new Error('transient');
        }
      },
    },
  };
  try {
    await revokeObsoleteTerminologyEditorGrants();
  } finally {
    (globalThis as any).$choysum = prev;
  }
  expect(calls.length).toBe(2);
  expect(calls.some(s => s.includes('meta_model_data') && s.includes('DELETE FROM meta_model_data'))).toBe(false);
});
