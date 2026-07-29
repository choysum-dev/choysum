// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Decimal from '@/core/utils/decimal';
import { buildHiddenScaleAlias } from '../../hidden_scale_alias';
import {
  cleanupHiddenScaleKeys,
  decodeFromDb,
  encodeForDb,
  parseJsonObjectFieldValue,
  resolveDecimalScaleForWrite,
  resolveDecimalScaleFromRow,
} from '../row_codec';

test('repository row codec encodes relation, jsonobject, decimal and ignores internal fields', () => {
  const meta = {
    fields: new Map<string, any>([
      ['Owner', { type: 'ManyToOne', column: { name: 'Owner' } }],
      ['RemoteId', { type: 'ManyToOneRef', column: { name: 'RemoteId' } }],
      ['TagIds', { type: 'ManyToManyRef', column: { name: 'TagIds' } }],
      ['Payload', { type: 'jsonobject', column: { name: 'Payload' } }],
      ['Amount', { type: 'decimal', name: 'Amount', column: { name: 'Amount', precision: 6, scaleField: 'AmountScale' } }],
      ['AmountScale', { type: 'int', column: { name: 'AmountScale' } }],
    ]),
  } as any;

  const encoded = encodeForDb(meta, {
    Owner: { Id: 'user_1', Name: 'ignored' },
    RemoteId: { Id: 'remote_1' },
    TagIds: 'tag_a',
    Payload: { ok: true },
    Amount: '12.345',
    AmountScale: 2,
    __runtime: 'skip-me',
    Ignored: 'skip-me-too',
  } as any);

  expect(encoded).toEqual({
    Owner: 'user_1',
    RemoteId: 'remote_1',
    TagIds: ['tag_a'],
    Payload: '{"ok":true}',
    Amount: { $bigdecimal: '12.35' },
    AmountScale: 2,
  });
  expect(resolveDecimalScaleForWrite(meta.fields.get('Amount'), { AmountScale: 2 } as any)).toBe(2);
});

test('repository row codec decodes jsonobject, decimal hidden scale alias and many2many refs', () => {
  const meta = {
    fields: new Map<string, any>([
      ['Payload', { type: 'jsonobject', column: { name: 'Payload' } }],
      ['Amount', { type: 'decimal', column: { name: 'Amount', precision: 6, scaleField: 'AmountScale' } }],
      ['AmountScale', { type: 'int', column: { name: 'AmountScale' } }],
      ['TagIds', { type: 'ManyToManyRef', column: { name: 'TagIds' } }],
    ]),
  } as any;

  const hiddenScaleAlias = buildHiddenScaleAlias('Amount');
  const decoded = decodeFromDb(meta, {
    Payload: '{"ok":true}',
    Amount: { $bigdecimal: '7.891' },
    [hiddenScaleAlias]: 1,
    TagIds: '[1,"a"]',
  } as any) as any;

  expect(decoded.Payload).toEqual({ ok: true });
  expect(decoded.Amount.toString()).toBe('7.9');
  expect(decoded.TagIds).toEqual(['1', 'a']);
  expect(hiddenScaleAlias in decoded).toBe(false);
});

test('repository row codec parse helpers and scale resolvers cover fallback branches', () => {
  expect(parseJsonObjectFieldValue(null)).toEqual({});
  expect(parseJsonObjectFieldValue('   ')).toEqual({});
  expect(parseJsonObjectFieldValue('hello')).toBe('hello');
  expect(parseJsonObjectFieldValue('{broken')).toBe('{broken');
  expect(parseJsonObjectFieldValue('{"a":1}')).toEqual({ a: 1 });
  expect(parseJsonObjectFieldValue(123)).toBe(123);

  const nonDecimal = { type: 'varchar', name: 'X' } as any;
  expect(resolveDecimalScaleForWrite(nonDecimal, {} as any)).toBeUndefined();

  const fixedScale = { type: 'decimal', name: 'Amount', column: { scale: 3 } } as any;
  expect(resolveDecimalScaleForWrite(fixedScale, {} as any)).toBe(3);

  const noScaleField = { type: 'decimal', name: 'Amount', column: {} } as any;
  expect(resolveDecimalScaleForWrite(noScaleField, {} as any)).toBeUndefined();

  let missingErr = '';
  try {
    resolveDecimalScaleForWrite({ type: 'decimal', name: 'Amount', column: { scaleField: 'AmountScale' } } as any, {} as any);
  } catch (error) {
    missingErr = String((error as Error).message || error);
  }
  expect(missingErr.includes('requires "AmountScale" as scale')).toBe(true);

  let invalidErr = '';
  try {
    resolveDecimalScaleForWrite({ type: 'decimal', name: 'Amount', column: { scaleField: 'AmountScale' } } as any, { AmountScale: 99 } as any);
  } catch (error) {
    invalidErr = String((error as Error).message || error);
  }
  expect(invalidErr.includes('invalid scaleField')).toBe(true);

  const meta = { fields: new Map([['Amount', { type: 'decimal', column: { scaleField: 'AmountScale' } }]]) } as any;
  expect(resolveDecimalScaleFromRow(meta, undefined as any, 'Amount', {})).toBeUndefined();
  expect(resolveDecimalScaleFromRow(meta, { type: 'decimal', column: { scale: 4 } } as any, 'Amount', {})).toBe(4);
  expect(resolveDecimalScaleFromRow(meta, { type: 'decimal', column: {} } as any, 'Amount', {})).toBeUndefined();
  expect(resolveDecimalScaleFromRow(meta, { type: 'varchar' } as any, 'X', {})).toBeUndefined();
  expect(resolveDecimalScaleForWrite(undefined, {} as any)).toBeUndefined();
  expect(resolveDecimalScaleFromRow(meta, { type: 'decimal', column: { scaleField: 'AmountScale' } } as any, 'Amount', { AmountScale: '2' })).toBe(2);
  expect(
    resolveDecimalScaleFromRow(meta, { type: 'decimal', column: { scaleField: 'AmountScale' } } as any, 'Amount', { [buildHiddenScaleAlias('Amount')]: 1 })
  ).toBe(1);

  const row: any = { a: 1, [buildHiddenScaleAlias('Amount')]: 2 };
  cleanupHiddenScaleKeys(row);
  expect(row).toEqual({ a: 1 });
  expect(() => cleanupHiddenScaleKeys(null as any)).not.toThrow();
});

test('repository row codec encode/decode cover many2one-ref and many2many-ref fallback branches', () => {
  const meta = {
    fields: new Map<string, any>([
      ['Owner', { type: 'ManyToOne', column: { name: 'Owner' } }],
      ['RemoteId', { type: 'ManyToOneRef', column: { name: 'RemoteId' } }],
      ['Tags', { type: 'ManyToManyRef', column: { name: 'Tags' } }],
      ['Payload', { type: 'jsonobject', column: { name: 'Payload' } }],
      ['Amount', { type: 'decimal', name: 'Amount', column: { name: 'Amount', scaleField: 'Scale' } }],
      ['Scale', { type: 'int', column: { name: 'Scale' } }],
    ]),
  } as any;

  const encoded = encodeForDb(meta, {
    Owner: { Name: 'no-id-object' },
    RemoteId: { Id: 123 },
    Tags: null,
    Payload: null,
    Amount: '7.1234',
  } as any) as any;

  expect(encoded.Owner).toEqual({ Name: 'no-id-object' });
  expect(encoded.RemoteId).toBe('[object Object]');
  expect(encoded.Tags).toBeNull();
  expect(encoded.Payload).toBeNull();
  expect(encoded.Amount).toEqual({ $bigdecimal: '7.1234' });

  const decoded = decodeFromDb(
    {
      fields: new Map<string, any>([
        ['Tags', { type: 'ManyToManyRef', column: { name: 'Tags' } }],
        ['Payload', { type: 'jsonobject', column: { name: 'Payload' } }],
      ]),
    } as any,
    {
      Tags: '{"a":1}',
      Payload: 'not-json',
    } as any
  ) as any;

  expect(decoded.Tags).toEqual(['{"a":1}']);
  expect(decoded.Payload).toBe('not-json');

  const decodedScalar = decodeFromDb(
    {
      fields: new Map<string, any>([['Tags', { type: 'ManyToManyRef', column: { name: 'Tags' } }]]),
    } as any,
    { Tags: 9 } as any
  ) as any;
  expect(decodedScalar.Tags).toEqual(['9']);
});

test('repository row codec handles passthrough inputs and explicit relation ref branches', () => {
  const meta = {
    fields: new Map<string, any>([
      ['Owner', { type: 'ManyToOne', column: { name: 'Owner' } }],
      ['RemoteId', { type: 'ManyToOneRef', column: { name: 'RemoteId' } }],
      ['Tags', { type: 'ManyToManyRef', column: { name: 'Tags' } }],
    ]),
  } as any;

  expect(encodeForDb(meta, null as any)).toBeNull();
  expect(encodeForDb(meta, 'raw' as any)).toBe('raw' as any);

  const encoded = encodeForDb(meta, {
    Owner: { Id: null },
    RemoteId: 'remote-string',
    Tags: ['a', 2],
  } as any) as any;

  expect(encoded.Owner).toBeNull();
  expect(encoded.RemoteId).toBe('remote-string');
  expect(encoded.Tags).toEqual(['a', 2]);
});

test('repository row codec decodes early-return and many2many array/null branches', () => {
  const meta = {
    fields: new Map<string, any>([
      ['Tags', { type: 'ManyToManyRef', column: { name: 'Tags' } }],
      ['Amount', { type: 'decimal', column: { name: 'Amount', scaleField: 'AmountScale' } }],
    ]),
  } as any;

  expect(decodeFromDb(meta, null as any)).toBeNull();
  expect(decodeFromDb(meta, 123 as any)).toBe(123 as any);

  const decodedArray = decodeFromDb(meta, { Tags: [1, 'x'] } as any) as any;
  expect(decodedArray.Tags).toEqual(['1', 'x']);

  const decodedNull = decodeFromDb(meta, { Tags: null } as any) as any;
  expect(decodedNull.Tags).toEqual([]);
});

test('repository row codec skips owner binary/image writes but keeps storage blob carrier writes', () => {
  const ownerMeta = {
    application: 'auth',
    modelName: 'User',
    fields: new Map<string, any>([
      ['Name', { type: 'varchar', column: { name: 'Name' } }],
      ['Avatar', { type: 'image', column: { name: 'Avatar' } }],
      ['IdentityDoc', { type: 'binary', column: { name: 'IdentityDoc' } }],
    ]),
  } as any;

  const ownerEncoded = encodeForDb(ownerMeta, {
    Name: 'alice',
    Avatar: 'bind-avatar',
    IdentityDoc: 'bind-doc',
  } as any) as any;

  expect(ownerEncoded).toEqual({ Name: 'alice' });

  const storageMeta = {
    application: 'document',
    modelName: 'AttachmentObject',
    fullModelName: 'document.AttachmentObject',
    fields: new Map<string, any>([
      ['BlobData', { type: 'binary', column: { name: 'BlobData' } }],
      ['PreviewData', { type: 'image', column: { name: 'PreviewData' } }],
    ]),
  } as any;

  const storageEncoded = encodeForDb(storageMeta, {
    BlobData: 'raw-bytes',
    PreviewData: 'preview-bytes',
  } as any) as any;

  expect(storageEncoded).toEqual({
    BlobData: 'raw-bytes',
    PreviewData: 'preview-bytes',
  });
});

test('repository row codec encodeForDb translate field null, JSON passthrough, and rejection paths', () => {
  const meta = {
    fields: new Map<string, any>([['Name', { type: 'varchar', translate: true, column: { name: 'Name' } }]]),
  } as any;

  expect(encodeForDb(meta, { Name: null } as any)).toEqual({ Name: null });

  const json = '{"en_US":"Hi"}';
  expect(encodeForDb(meta, { Name: json } as any)).toEqual({ Name: json });

  expect(encodeForDb(meta, { Name: { en_US: 'Hello', zh_CN: '你好' } } as any)).toEqual({
    Name: JSON.stringify({ en_US: 'Hello', zh_CN: '你好' }),
  });

  expect(() => encodeForDb(meta, { Name: 'plain' } as any)).toThrow(/must be prepared as a lang map/);
  expect(() => encodeForDb(meta, { Name: 123 } as any)).toThrow(/expects a lang map object or null/);
});

test('repository row codec encodeForDb sanitize html and null blank markup', () => {
  const original = (globalThis as any).$choysum;
  try {
    (globalThis as any).$choysum = {
      html: {
        // Fixture map only — production sanitization lives in Go bluemonday.
        sanitize: (s: string) => {
          if (s === '<script>x</script><p>safe</p>') return '<p>safe</p>';
          return s;
        },
      },
    };
    const meta = {
      fields: new Map<string, any>([['Terms', { type: 'html', column: { name: 'Terms' } }]]),
    } as any;

    expect(encodeForDb(meta, { Terms: '<script>x</script><p>safe</p>' } as any)).toEqual({
      Terms: '<p>safe</p>',
    });
    expect(encodeForDb(meta, { Terms: null } as any)).toEqual({ Terms: null });
    expect(encodeForDb(meta, { Terms: '<p></p>' } as any)).toEqual({ Terms: null });
    expect(() => encodeForDb(meta, { Terms: 1 } as any)).toThrow(/html field value must be a string/);
  } finally {
    (globalThis as any).$choysum = original;
  }
});

test('repository row codec monetary quantize uses currency digits and E1 without currency', () => {
  const meta = {
    fields: new Map<string, any>([
      ['CurrencyId', { type: 'ManyToOneRef', column: { name: 'CurrencyId' } }],
      ['Amount', { type: 'monetary', name: 'Amount', column: { name: 'Amount', currencyField: 'CurrencyId' } }],
    ]),
  } as any;

  const encoded = encodeForDb(meta, {
    CurrencyId: { Id: 'CUR-1', DecimalDigits: 0 },
    Amount: '12.6',
  } as any);
  expect(encoded.Amount).toEqual({ $bigdecimal: '13' });

  expect(() => encodeForDb(meta, { Amount: '12.6' } as any)).toThrow(/currency required for monetary field Amount/);

  const stampedAlias = buildHiddenScaleAlias('Amount');
  const encodedFromStamp = encodeForDb(meta, {
    Amount: '1.234',
    [stampedAlias]: 2,
  } as any);
  expect(encodedFromStamp.Amount).toEqual({ $bigdecimal: '1.23' });

  const decoded = decodeFromDb(meta, {
    Amount: { $bigdecimal: '1.239' },
    CurrencyId: { Id: 'CUR-1', DecimalDigits: 2 },
  } as any);
  expect(String(decoded.Amount)).toBe('1.24');

  expect(resolveDecimalScaleForWrite(meta.fields.get('Amount'), { CurrencyId: { DecimalDigits: 0 } } as any)).toBe(0);
  expect(() => resolveDecimalScaleForWrite(meta.fields.get('Amount'), {} as any)).toThrow(/currency required/);
  expect(resolveDecimalScaleFromRow(meta, meta.fields.get('Amount'), 'Amount', { [stampedAlias]: 2 })).toBe(2);
  expect(resolveDecimalScaleFromRow(meta, meta.fields.get('Amount'), 'Amount', { CurrencyId: 'C1' })).toBeUndefined();

  const decodedNoDigits = decodeFromDb(meta, {
    Amount: { $bigdecimal: '1.239' },
    CurrencyId: 'C1',
  } as any);
  expect(decodedNoDigits.Amount != null).toBe(true);
});

test('repository row codec decimal soft-fallback and Decimal instance decode', () => {
  const decimalMeta = {
    fields: new Map<string, any>([
      ['Amount', { type: 'decimal', name: 'Amount', column: { name: 'Amount', scaleField: 'AmountScale' } }],
      ['AmountScale', { type: 'int', column: { name: 'AmountScale' } }],
    ]),
  } as any;

  // Missing scaleField: decimal encode soft-falls back instead of throwing.
  const soft = encodeForDb(decimalMeta, { Amount: '1.25' } as any) as any;
  expect(soft.Amount).toEqual({ $bigdecimal: '1.25' });

  const monetaryMeta = {
    fields: new Map<string, any>([
      ['CurrencyId', { type: 'ManyToOneRef', column: { name: 'CurrencyId' } }],
      ['Amount', { type: 'monetary', name: 'Amount', column: { name: 'Amount', currencyField: 'CurrencyId' } }],
    ]),
  } as any;

  const decoded = decodeFromDb(monetaryMeta, {
    Amount: new Decimal('1.239'),
    CurrencyId: { Id: 'C1', DecimalDigits: 2 },
  } as any) as any;
  expect(String(decoded.Amount)).toBe('1.24');

  // Soft catch on decode keeps the original value when normalize fails.
  const decodedBad = decodeFromDb(monetaryMeta, {
    Amount: { $bigdecimal: 'not-a-number' },
    CurrencyId: { Id: 'C1', DecimalDigits: 2 },
  } as any) as any;
  expect(decodedBad.Amount).toEqual({ $bigdecimal: 'not-a-number' });
});

test('repository row codec encode/decode companyDependent fields', async () => {
  const { withContext } = await import('../../../../runtime/context');
  const meta = {
    fields: new Map([
      ['Cost', { name: 'Cost', type: 'number', companyDependent: true, column: {} }],
      ['Amount', { name: 'Amount', type: 'decimal', companyDependent: true, column: { precision: 10, scale: 2 } }],
    ]),
  } as any;

  expect(encodeForDb(meta, { Cost: null } as any)).toEqual({ Cost: null });
  expect(encodeForDb(meta, { Cost: { C1: 1.5 } } as any)).toEqual({ Cost: JSON.stringify({ C1: 1.5 }) });
  expect(encodeForDb(meta, { Cost: '{"C1":1}' } as any)).toEqual({ Cost: '{"C1":1}' });
  expect(() => encodeForDb(meta, { Cost: '12.5' } as any)).toThrow(/must be prepared as a company map/);
  expect(() => encodeForDb(meta, { Cost: { Id: 'p1' } } as any)).toThrow(/expects a company map object or null/);

  await withContext({ activeCompanyId: 'C1' }, async () => {
    const decoded = decodeFromDb(meta, { Cost: JSON.stringify({ C1: 9, C2: 8 }) } as any) as any;
    expect(decoded.Cost).toBe(9);
  });

  await withContext({ activeCompanyId: 'C1', prefetch_companies: true }, async () => {
    const decoded = decodeFromDb(meta, { Cost: JSON.stringify({ C1: 9, C2: 8 }) } as any) as any;
    expect(decoded.Cost).toEqual({ C1: 9, C2: 8 });
  });

  await withContext({ activeCompanyId: 'C1' }, async () => {
    const decoded = decodeFromDb(meta, { Cost: null } as any) as any;
    expect(decoded.Cost).toBeNull();
  });

  await withContext({ activeCompanyId: 'C1', prefetch_companies: true }, async () => {
    const decoded = decodeFromDb(meta, {
      Amount: JSON.stringify({ C1: { $bigdecimal: '1.239' }, C2: null, C3: 'bad' }),
    } as any) as any;
    expect(decoded.Amount.C1.toString()).toBe('1.24');
    expect(decoded.Amount.C2).toBeNull();
    expect(decoded.Amount.C3).toBe('bad');
  });

  await withContext({ activeCompanyId: 'C1' }, async () => {
    const decoded = decodeFromDb(meta, { Amount: JSON.stringify({ C1: { $bigdecimal: '2.5' } }) } as any) as any;
    expect(decoded.Amount.toString()).toBe('2.5');
  });

  await withContext({ activeCompanyId: 'C1' }, async () => {
    const decoded = decodeFromDb(meta, { Amount: JSON.stringify({ C1: 'not-a-number' }) } as any) as any;
    expect(decoded.Amount).toBe('not-a-number');
  });
});
