// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  collectRelationIdsFromGroup,
  extractRelationId,
  formatRelationPreviewValue,
  relationIdBatchSignature,
  relationLabelCacheKey,
} from './filterRelationLabels';

test('extractRelationId supports string, object, and empty', () => {
  expect(extractRelationId('  c1  ')).toBe('c1');
  expect(extractRelationId({ Id: 'p1' })).toBe('p1');
  expect(extractRelationId({ Id: '' })).toBeUndefined();
  expect(extractRelationId(null)).toBeUndefined();
  expect(extractRelationId(['x'])).toBeUndefined();
});

test('collectRelationIdsFromGroup gathers manytoone and manytooneref ids', () => {
  const meta = {
    Name: { type: 'varchar' },
    PartnerId: { type: 'ManyToOne', relationModel: 'base.Partner' },
    CompanyId: { type: 'manytooneref', relationModel: 'base.Company' },
    LanguageId: { type: 'ManyToOneRef', relationModel: 'base.Language' },
  };
  const root = {
    id: 'g',
    logic: 'And' as const,
    children: [
      { id: 'c1', field: 'Name', operator: '=', value: 'Alice' },
      { id: 'c2', field: 'PartnerId', operator: '=', value: { Id: 'p1' } },
      { id: 'c3', field: 'CompanyId', operator: 'in', value: ['c1', { Id: 'c2' }, { Id: '' }] },
      { id: 'c4', field: 'LanguageId', operator: 'is', value: null },
      {
        id: 'g2',
        logic: 'Or' as const,
        children: [{ id: 'c5', field: 'LanguageId', operator: '=', value: 'lang-1' }],
      },
    ],
  };

  const batch = collectRelationIdsFromGroup(root, meta);
  expect(Array.from(batch.get('base.Partner') || [])).toEqual(['p1']);
  expect(new Set(batch.get('base.Company'))).toEqual(new Set(['c1', 'c2']));
  expect(Array.from(batch.get('base.Language') || [])).toEqual(['lang-1']);
});

test('collectRelationIdsFromGroup skips fields without relationModel', () => {
  const batch = collectRelationIdsFromGroup(
    {
      id: 'g',
      logic: 'And',
      children: [{ id: 'c1', field: 'PartnerId', operator: '=', value: 'p1' }],
    },
    { PartnerId: { type: 'manytoone' } }
  );
  expect(batch.size).toBe(0);
});

test('relationIdBatchSignature is stable regardless of set insertion order', () => {
  const a = new Map<string, Set<string>>([
    ['base.B', new Set(['2', '1'])],
    ['base.A', new Set(['z', 'a'])],
  ]);
  const b = new Map<string, Set<string>>([
    ['base.A', new Set(['a', 'z'])],
    ['base.B', new Set(['1', '2'])],
  ]);
  expect(relationIdBatchSignature(a)).toBe(relationIdBatchSignature(b));
  expect(relationIdBatchSignature(a)).toBe('base.A=a,z|base.B=1,2');
});

test('formatRelationPreviewValue maps ids through resolveLabel', () => {
  const resolve = (id: string) => (id === 'c1' ? 'Acme' : id);
  expect(formatRelationPreviewValue('c1', resolve)).toBe('Acme');
  expect(formatRelationPreviewValue({ Id: 'c1' }, resolve)).toBe('Acme');
  expect(formatRelationPreviewValue(['c1', { Id: 'c2' }], resolve)).toEqual(['Acme', 'c2']);
  expect(formatRelationPreviewValue({ Id: '' }, resolve)).toBe(null);
  expect(formatRelationPreviewValue(42, resolve)).toBe(42);
  expect(formatRelationPreviewValue(undefined, resolve)).toBe(undefined);
  expect(collectRelationIdsFromGroup(null, {})).toEqual(new Map());
  expect(relationLabelCacheKey('base.Company', 'c1')).toBe('base.Company::c1');
});
