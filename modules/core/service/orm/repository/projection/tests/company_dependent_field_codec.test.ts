// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Decimal from 'decimal.js';
import { withContext } from '../../../../runtime/context';
import {
  applyCompanyDependentFieldsForWrite,
  applyFieldCompanyValuesPatch,
  assertCompanyDependentCompanyKey,
  decodeCompanyDependentFieldValue,
  deleteCompanyKey,
  encodeCompanyDependentMapForDb,
  fieldIsCompanyDependent,
  getCompanyDependentWriteReplace,
  getPrefetchCompanies,
  isCompanyDependentScalarEnvelope,
  mergeCompanyDependentWrite,
  normalizeCompanyDependentScalarValue,
  parseCompanyDependentStoredMap,
  payloadHasCompanyDependentFieldWrite,
  resolveCompanyDependentCompanyId,
  unwrapCompanyDependentValue,
} from '../company_dependent_field_codec';

test('companyDependent codec unwrap is F0 (no fallback)', () => {
  const map = { C1: 12.5, C2: 11 };
  expect(unwrapCompanyDependentValue(map, 'C1')).toBe(12.5);
  expect(unwrapCompanyDependentValue(map, 'MISSING')).toBeNull();
  expect(unwrapCompanyDependentValue(null, 'C1')).toBeNull();
});

test('companyDependent merge: scalar write, null deletes current key only', () => {
  const current = { C1: 1, C2: 2 };
  expect(
    mergeCompanyDependentWrite({
      fieldName: 'Cost',
      value: 9,
      companyId: 'C1',
      currentMap: current,
      mode: 'update',
    })
  ).toEqual({ C1: 9, C2: 2 });

  expect(
    mergeCompanyDependentWrite({
      fieldName: 'Cost',
      value: null,
      companyId: 'C1',
      currentMap: current,
      mode: 'update',
    })
  ).toEqual({ C2: 2 });

  expect(
    mergeCompanyDependentWrite({
      fieldName: 'Cost',
      value: null,
      companyId: 'C2',
      currentMap: { C2: 2 },
      mode: 'update',
    })
  ).toBeNull();
});

test('companyDependent merge: replace null clears whole column', () => {
  expect(
    mergeCompanyDependentWrite({
      fieldName: 'Cost',
      value: null,
      companyId: 'C1',
      currentMap: { C1: 1, C2: 2 },
      mode: 'update',
      replace: true,
    })
  ).toBeNull();
});

test('companyDependent merge: create writes only active company key', () => {
  expect(
    mergeCompanyDependentWrite({
      fieldName: 'Cost',
      value: 3,
      companyId: 'C1',
      currentMap: null,
      mode: 'create',
    })
  ).toEqual({ C1: 3 });
});

test('companyDependent merge: map object merges; replace replaces', () => {
  expect(
    mergeCompanyDependentWrite({
      fieldName: 'Cost',
      value: { C2: 8 },
      companyId: 'C1',
      currentMap: { C1: 1 },
      mode: 'update',
    })
  ).toEqual({ C1: 1, C2: 8 });

  expect(
    mergeCompanyDependentWrite({
      fieldName: 'Cost',
      value: { C2: 8 },
      companyId: 'C1',
      currentMap: { C1: 1 },
      mode: 'update',
      replace: true,
    })
  ).toEqual({ C2: 8 });
});

test('companyDependent patch false deletes key; empty map null', () => {
  expect(
    applyFieldCompanyValuesPatch({
      fieldName: 'Cost',
      currentMap: { C1: 1, C2: 2 },
      values: { C1: false },
    })
  ).toEqual({ C2: 2 });

  expect(
    applyFieldCompanyValuesPatch({
      fieldName: 'Cost',
      currentMap: { C1: 1 },
      values: { C1: false },
    })
  ).toBeNull();
});

test('companyDependent decode respects prefetch_companies', async () => {
  const stored = JSON.stringify({ C1: 'a', C2: 'b' });
  await withContext({ activeCompanyId: 'C1', prefetch_companies: true }, async () => {
    expect(decodeCompanyDependentFieldValue(stored)).toEqual({ C1: 'a', C2: 'b' });
  });
  await withContext({ activeCompanyId: 'C1' }, async () => {
    expect(decodeCompanyDependentFieldValue(stored)).toBe('a');
  });
  await withContext({ activeCompanyId: 'MISSING' }, async () => {
    expect(decodeCompanyDependentFieldValue(stored)).toBeNull();
  });
});

test('companyDependent apply write rewrites payload field to map', () => {
  const meta = {
    fields: new Map([['Cost', { name: 'Cost', type: 'number', companyDependent: true, column: {} }]]),
  } as any;

  const created = applyCompanyDependentFieldsForWrite(meta, { Cost: 4 } as any, {
    mode: 'create',
    companyId: 'C1',
  });
  expect(created).toEqual({ Cost: { C1: 4 } });

  const updated = applyCompanyDependentFieldsForWrite(
    meta,
    { Cost: null } as any,
    { mode: 'update', companyId: 'C1', current: { Cost: { C1: 4, C2: 5 } } }
  );
  expect(updated).toEqual({ Cost: { C2: 5 } });

  expect(encodeCompanyDependentMapForDb({ C1: 1 })).toBe(JSON.stringify({ C1: 1 }));
  expect(encodeCompanyDependentMapForDb(null)).toBeNull();
  expect(parseCompanyDependentStoredMap('{"C1":1}')).toEqual({ C1: 1 });
});

test('companyDependent merge treats ManyToOne Id / Date envelopes as scalars', () => {
  expect(
    mergeCompanyDependentWrite({
      fieldName: 'PartnerId',
      value: { Id: 'p1' },
      companyId: 'C1',
      currentMap: { C2: 'p2' },
      mode: 'update',
    })
  ).toEqual({ C2: 'p2', C1: 'p1' });

  const when = new Date('2024-01-02T00:00:00.000Z');
  expect(
    mergeCompanyDependentWrite({
      fieldName: 'DueDate',
      value: when,
      companyId: 'C1',
      currentMap: null,
      mode: 'create',
    })
  ).toEqual({ C1: '2024-01-02T00:00:00.000Z' });
});

test('companyDependent encodeForDb prefers company map over ManyToOne Id strip', async () => {
  const { encodeForDb } = await import('../row_codec');
  const meta = {
    fields: new Map([
      [
        'PartnerId',
        {
          name: 'PartnerId',
          type: 'ManyToOne',
          companyDependent: true,
          column: {},
          relation: { model: 'base.Partner' },
        },
      ],
    ]),
  } as any;
  const encoded = encodeForDb(meta, { PartnerId: { C1: 'p1', C2: 'p2' } } as any);
  expect(encoded).toEqual({ PartnerId: JSON.stringify({ C1: 'p1', C2: 'p2' }) });
});

test('companyDependent parse/normalize/helpers cover remaining branches', () => {
  expect(parseCompanyDependentStoredMap(null)).toBeNull();
  expect(parseCompanyDependentStoredMap('')).toBeNull();
  expect(parseCompanyDependentStoredMap('   ')).toBeNull();
  expect(parseCompanyDependentStoredMap('not-json')).toBeNull();
  expect(parseCompanyDependentStoredMap('{"C1":1}')).toEqual({ C1: 1 });
  expect(parseCompanyDependentStoredMap('{}')).toBeNull();
  expect(parseCompanyDependentStoredMap('{"":1," C2 ":2}')).toEqual({ C2: 2 });
  expect(parseCompanyDependentStoredMap([])).toBeNull();
  expect(parseCompanyDependentStoredMap({ Id: 'p1' })).toBeNull();
  expect(parseCompanyDependentStoredMap('true')).toBeNull();
  expect(parseCompanyDependentStoredMap('[1]')).toBeNull();
  expect(parseCompanyDependentStoredMap('"x"')).toBeNull();
  expect(parseCompanyDependentStoredMap('-1')).toBeNull();
  expect(parseCompanyDependentStoredMap('12')).toBeNull();
  expect(parseCompanyDependentStoredMap('false')).toBeNull();
  expect(parseCompanyDependentStoredMap('null')).toBeNull();
  expect(parseCompanyDependentStoredMap('{bad')).toBeNull();

  expect(isCompanyDependentScalarEnvelope(null)).toBe(false);
  expect(isCompanyDependentScalarEnvelope([])).toBe(false);
  expect(isCompanyDependentScalarEnvelope({ Id: 'p1', DisplayName: 'P' })).toBe(true);
  expect(isCompanyDependentScalarEnvelope({ Id: 'p1', Name: 'P' })).toBe(true);
  expect(isCompanyDependentScalarEnvelope({ Id: 'p1', Extra: 1 })).toBe(false);
  expect(isCompanyDependentScalarEnvelope({ $bigdecimal: '1.2' })).toBe(true);

  expect(normalizeCompanyDependentScalarValue(null)).toBeNull();
  expect(normalizeCompanyDependentScalarValue({ Id: null })).toBeNull();
  expect(normalizeCompanyDependentScalarValue({ $bigdecimal: '1.25' })).toBe('1.25');
  expect(
    normalizeCompanyDependentScalarValue(
      { $bigdecimal: '1.234' },
      { type: 'decimal', column: { precision: 10, scale: 2 } } as any
    )
  ).toBe('1.23');
  expect(normalizeCompanyDependentScalarValue(true)).toBe(true);
  expect(normalizeCompanyDependentScalarValue('1.5', { type: 'decimal', column: { scale: 1 } } as any)).toBe('1.5');
  expect(normalizeCompanyDependentScalarValue(2, { type: 'monetary', column: { scale: 0 } } as any)).toBe('2');

  const originalChoysum = (globalThis as any).$choysum;
  try {
    (globalThis as any).$choysum = {
      html: {
        sanitize: (s: string) => (s === '<script>x</script><p>ok</p>' ? '<p>ok</p>' : s),
      },
    };
    expect(
      normalizeCompanyDependentScalarValue('<script>x</script><p>ok</p>', { type: 'html' } as any)
    ).toBe('<p>ok</p>');
    expect(encodeCompanyDependentMapForDb({ C1: '<script>x</script><p>ok</p>' }, { type: 'html' } as any)).toBe(
      JSON.stringify({ C1: '<p>ok</p>' })
    );
  } finally {
    (globalThis as any).$choysum = originalChoysum;
  }

  const dec = new Decimal('3.1415');
  expect(normalizeCompanyDependentScalarValue(dec)).toBe('3.1415');
  expect(normalizeCompanyDependentScalarValue(dec, { type: 'decimal', column: { scale: 2 } } as any)).toBe('3.14');

  expect(() => assertCompanyDependentCompanyKey('', 'Cost')).toThrow(/non-empty company id/);
  expect(() => assertCompanyDependentCompanyKey('  ', 'Cost')).toThrow(/non-empty company id/);
  expect(assertCompanyDependentCompanyKey(' C1 ', 'Cost')).toBe('C1');

  expect(deleteCompanyKey({ C1: 1 }, 'MISSING', 'Cost')).toEqual({ C1: 1 });
  expect(deleteCompanyKey({}, 'C1', 'Cost')).toBeNull();

  expect(fieldIsCompanyDependent({ companyDependent: true } as any)).toBe(true);
  expect(fieldIsCompanyDependent({ companyDependent: false } as any)).toBe(false);
  expect(fieldIsCompanyDependent(undefined)).toBe(false);

  const meta = {
    fields: new Map([
      ['Cost', { companyDependent: true }],
      ['Name', { companyDependent: false }],
    ]),
  } as any;
  expect(payloadHasCompanyDependentFieldWrite(meta, null as any)).toBe(false);
  expect(payloadHasCompanyDependentFieldWrite(meta, { Name: 'x' } as any)).toBe(false);
  expect(payloadHasCompanyDependentFieldWrite(meta, { Cost: undefined } as any)).toBe(false);
  expect(payloadHasCompanyDependentFieldWrite(meta, { Cost: 1 } as any)).toBe(true);

  expect(withContext({ prefetchCompanies: true }, () => getPrefetchCompanies())).toBe(true);
  expect(withContext({ companyWriteReplace: true }, () => getCompanyDependentWriteReplace())).toBe(true);
});

test('companyDependent merge/patch/encode remaining error and decimal paths', () => {
  expect(() =>
    mergeCompanyDependentWrite({
      fieldName: 'Cost',
      value: { C1: false },
      companyId: 'C1',
      currentMap: null,
      mode: 'update',
    })
  ).toThrow(/does not accept false/);

  // Arrays are objects but not company maps → reject.
  expect(() =>
    mergeCompanyDependentWrite({
      fieldName: 'Cost',
      value: [1, 2],
      companyId: 'C1',
      currentMap: null,
      mode: 'update',
    })
  ).toThrow(/expects scalar/);

  expect(
    mergeCompanyDependentWrite({
      fieldName: 'Cost',
      value: {},
      companyId: 'C1',
      currentMap: { C1: 1 },
      mode: 'update',
      replace: true,
    })
  ).toBeNull();

  expect(
    mergeCompanyDependentWrite({
      fieldName: 'Active',
      value: false,
      companyId: 'C1',
      currentMap: null,
      mode: 'create',
    })
  ).toEqual({ C1: false });

  expect(
    mergeCompanyDependentWrite({
      fieldName: 'Qty',
      value: 7n,
      companyId: 'C1',
      currentMap: null,
      mode: 'create',
    })
  ).toEqual({ C1: 7n });

  expect(() =>
    applyFieldCompanyValuesPatch({
      fieldName: 'Cost',
      currentMap: null,
      values: null as any,
    })
  ).toThrow(/must be an object map/);

  expect(
    applyFieldCompanyValuesPatch({
      fieldName: 'Cost',
      currentMap: { C1: 1 },
      values: { C3: 9 },
    })
  ).toEqual({ C1: 1, C3: 9 });

  expect(encodeCompanyDependentMapForDb({ C1: '1.234' }, { type: 'decimal', column: { scale: 2 } } as any)).toBe(
    JSON.stringify({ C1: '1.23' })
  );

  expect(decodeCompanyDependentFieldValue(JSON.stringify({ C1: 1 }))).toBeNull();
  expect(decodeCompanyDependentFieldValue(JSON.stringify({ C1: 1 }), { companyId: 'C1' })).toBe(1);
  expect(decodeCompanyDependentFieldValue(JSON.stringify({ C1: 1 }), { prefetchCompanies: true })).toEqual({ C1: 1 });
});

test('companyDependent apply write covers replace ctx, missing company, skips', () => {
  const meta = {
    fields: new Map([
      ['Cost', { name: 'Cost', type: 'number', companyDependent: true, column: {} }],
      ['Name', { name: 'Name', type: 'char', column: {} }],
    ]),
  } as any;

  expect(applyCompanyDependentFieldsForWrite(meta, null as any, { mode: 'create', companyId: 'C1' })).toBeNull();
  expect(applyCompanyDependentFieldsForWrite(meta, { Name: 'x' } as any, { mode: 'create', companyId: 'C1' })).toEqual({
    Name: 'x',
  });
  expect(applyCompanyDependentFieldsForWrite(meta, { Cost: undefined } as any, { mode: 'create', companyId: 'C1' })).toEqual({
    Cost: undefined,
  });

  expect(() => applyCompanyDependentFieldsForWrite(meta, { Cost: 1 } as any, { mode: 'create' })).toThrow(
    /requires an active company id/
  );

  const replaced = withContext({ company_write_replace: true }, () =>
    applyCompanyDependentFieldsForWrite(
      meta,
      { Cost: { C9: 9 } } as any,
      { mode: 'update', companyId: 'C1', current: { Cost: { C1: 1, C2: 2 } } }
    )
  );
  expect(replaced).toEqual({ Cost: { C9: 9 } });
});

test('companyDependent normalize hits invalid-decimal fallbacks and empty merge null', () => {
  expect(parseCompanyDependentStoredMap('hello')).toBeNull(); // !maybeJsonFast → bare string
  expect(parseCompanyDependentStoredMap('world')).toBeNull();

  const decimalFm = { type: 'decimal', column: { scale: 2 } } as any;
  expect(normalizeCompanyDependentScalarValue({ $bigdecimal: 'not-a-number' }, decimalFm)).toBe('not-a-number');
  expect(normalizeCompanyDependentScalarValue('nope', decimalFm)).toBe('nope');
  // Decimal instance that normalizes to undefined → toString fallback.
  const badDec = new Decimal(NaN);
  expect(normalizeCompanyDependentScalarValue(badDec, decimalFm)).toBe(badDec.toString());

  expect(
    mergeCompanyDependentWrite({
      fieldName: 'Cost',
      value: {},
      companyId: 'C1',
      currentMap: null,
      mode: 'update',
    })
  ).toBeNull();

  expect(withContext({ activeCompanyId: '' }, () => resolveCompanyDependentCompanyId())).toBeUndefined();
});

test('companyDependent null delete uses empty map when currentMap is null', () => {
  expect(
    mergeCompanyDependentWrite({
      fieldName: 'Cost',
      value: null,
      companyId: 'C1',
      currentMap: null,
      mode: 'update',
      replace: false,
    })
  ).toBeNull();

  // applyFieldCompanyValuesPatch: falsy currentMap takes `|| {}` (line 198 branch).
  expect(
    applyFieldCompanyValuesPatch({
      fieldName: 'Cost',
      currentMap: null,
      values: { C1: 3 },
    })
  ).toEqual({ C1: 3 });
});
