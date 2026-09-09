// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref } from 'vue';
import { useRecordImportScope } from './useRecordImportScope';

test('useRecordImportScope: companyId from request context', () => {
  const scope = useRecordImportScope({
    model: 'partner.Partner',
    getCurrentRequestContext: () => ({ activeCompanyId: 'cmp-1' }) as any,
  });
  expect(scope.model.value).toBe('partner.Partner');
  expect(scope.companyId.value).toBe('cmp-1');
  expect(scope.columnMapping.value).toEqual({});
  expect(scope.uploadHint.value).toBeUndefined();
});

test('useRecordImportScope: resolves hint and mapping from config', () => {
  const scope = useRecordImportScope({
    model: 'from.store',
    config: {
      import: {
        enabled: true,
        uploadHint: 'from-config',
        columnMapping: { Code: 'code' },
      },
    },
    getCurrentRequestContext: () => ({ activeCompanyId: 'cmp-1' }) as any,
  });
  expect(scope.model.value).toBe('from.store');
  expect(scope.companyId.value).toBe('cmp-1');
  expect(scope.uploadHint.value).toBe('from-config');
  expect(scope.columnMapping.value).toEqual({ Code: 'code' });
});

test('useRecordImportScope: falls back to companyId when activeCompanyId is absent', () => {
  const scope = useRecordImportScope({
    model: 'partner.Partner',
    getCurrentRequestContext: () => ({ companyId: 'cmp-scope' }) as any,
  });
  expect(scope.companyId.value).toBe('cmp-scope');
});

test('useRecordImportScope: empty model and mapping when options are omitted', () => {
  const scope = useRecordImportScope({
    getCurrentRequestContext: () => ({}) as any,
  });
  expect(scope.model.value).toBe('');
  expect(scope.companyId.value).toBe('');
  expect(scope.columnMapping.value).toEqual({});
});

test('useRecordImportScope: reactive model and config getters', () => {
  const model = ref('a.Model');
  const config = ref({
    import: { enabled: true, uploadHint: 'h1', columnMapping: { A: 'a' } },
  });
  const scope = useRecordImportScope({
    model: () => model.value,
    config: () => config.value,
    getCurrentRequestContext: () => ({ activeCompanyId: 'c1' }) as any,
  });
  expect(scope.model.value).toBe('a.Model');
  expect(scope.uploadHint.value).toBe('h1');
  model.value = 'b.Model';
  config.value = { import: { enabled: true, uploadHint: 'h2', columnMapping: { B: 'b' } } };
  expect(scope.model.value).toBe('b.Model');
  expect(scope.uploadHint.value).toBe('h2');
  expect(scope.columnMapping.value).toEqual({ B: 'b' });
});
