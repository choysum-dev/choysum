// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withContext } from '../../../../runtime/context';
import {
  TRANSLATED_BASE_LANG,
  applyFieldTranslationsPatch,
  applyTranslatedFieldsForWrite,
  decodeTranslatedFieldValue,
  deleteLangKey,
  mergeTranslatedWrite,
  unwrapTranslatedValue,
} from '../translated_field_codec';
import { decodeFromDb, encodeForDb } from '../row_codec';

test('unwrapTranslatedValue falls back to en_US but keeps explicit empty string', () => {
  expect(unwrapTranslatedValue({ en_US: 'Hello', zh_CN: '你好' }, 'zh_CN')).toBe('你好');
  expect(unwrapTranslatedValue({ en_US: 'Hello' }, 'zh_CN')).toBe('Hello');
  expect(unwrapTranslatedValue({ en_US: 'Hello', zh_CN: '' }, 'zh_CN')).toBe('');
  expect(unwrapTranslatedValue({ zh_CN: '你好' }, 'fr_FR')).toBe(null);
  expect(unwrapTranslatedValue(null, 'en_US')).toBe(null);
});

test('mergeTranslatedWrite dual-writes en_US on create and merges on update', () => {
  expect(
    mergeTranslatedWrite({
      fieldName: 'Name',
      value: 'Hello',
      lang: 'en_US',
      currentMap: null,
      mode: 'create',
    })
  ).toEqual({ en_US: 'Hello' });

  expect(
    mergeTranslatedWrite({
      fieldName: 'Name',
      value: '你好',
      lang: 'zh_CN',
      currentMap: null,
      mode: 'create',
    })
  ).toEqual({ zh_CN: '你好', en_US: '你好' });

  expect(
    mergeTranslatedWrite({
      fieldName: 'Name',
      value: '你好',
      lang: 'zh_CN',
      currentMap: { en_US: 'Hello' },
      mode: 'update',
    })
  ).toEqual({ en_US: 'Hello', zh_CN: '你好' });

  expect(
    mergeTranslatedWrite({
      fieldName: 'Name',
      value: '',
      lang: 'zh_CN',
      currentMap: { en_US: 'Hello', zh_CN: '旧' },
      mode: 'update',
    })
  ).toEqual({ en_US: 'Hello', zh_CN: '' });

  expect(
    mergeTranslatedWrite({
      fieldName: 'Name',
      value: null,
      lang: 'zh_CN',
      currentMap: { en_US: 'Hello' },
      mode: 'update',
    })
  ).toBeNull();

  expect(
    mergeTranslatedWrite({
      fieldName: 'Name',
      value: { zh_CN: '中文', fr_FR: 'Bonjour' },
      lang: 'en_US',
      currentMap: { en_US: 'Hello' },
      mode: 'update',
    })
  ).toEqual({ en_US: 'Hello', zh_CN: '中文', fr_FR: 'Bonjour' });
});

test('mergeTranslatedWrite rejects UI locale keys and oversized values', () => {
  expect(() =>
    mergeTranslatedWrite({
      fieldName: 'Name',
      value: 'x',
      lang: 'zh-CN',
      currentMap: null,
      mode: 'create',
    })
  ).toThrow(/terminology code/);

  expect(() =>
    mergeTranslatedWrite({
      fieldName: 'Name',
      value: 'too-long',
      lang: 'en_US',
      currentMap: null,
      mode: 'create',
      size: 3,
    })
  ).toThrow(/exceeds size/);
});

test('row codec encode/decode translated fields with unwrap and prefetch_langs', () => {
  const meta = {
    fields: new Map<string, any>([
      ['Name', { type: 'varchar', translate: true, column: { name: 'Name' }, storageHints: { size: 100 } }],
    ]),
  } as any;

  const prepared = applyTranslatedFieldsForWrite(
    meta,
    { Name: '你好' } as any,
    { mode: 'create', lang: 'zh_CN' }
  );
  expect(prepared).toEqual({ Name: { zh_CN: '你好', en_US: '你好' } });

  const encoded = encodeForDb(meta, prepared as any);
  expect(JSON.parse(String((encoded as any).Name))).toEqual({ zh_CN: '你好', en_US: '你好' });

  const decodedZh = withContext({ lang: 'zh_CN' }, () => decodeFromDb(meta, encoded as any));
  expect((decodedZh as any).Name).toBe('你好');

  const decodedLegacy = withContext({ lang: 'zh_CN' }, () =>
    decodeFromDb(meta, {
      Name: 'Legacy name',
    } as any)
  );
  expect((decodedLegacy as any).Name).toBe('Legacy name');

  const decodedEn = withContext({ lang: 'en_US' }, () =>
    decodeFromDb(meta, {
      Name: JSON.stringify({ en_US: 'Hello', zh_CN: '' }),
    } as any)
  );
  expect((decodedEn as any).Name).toBe('Hello');

  const decodedEmpty = withContext({ lang: 'zh_CN' }, () =>
    decodeFromDb(meta, {
      Name: JSON.stringify({ en_US: 'Hello', zh_CN: '' }),
    } as any)
  );
  expect((decodedEmpty as any).Name).toBe('');

  const prefetched = withContext({ prefetch_langs: true }, () =>
    decodeTranslatedFieldValue(JSON.stringify({ en_US: 'Hello', zh_CN: '你好' }))
  );
  expect(prefetched).toEqual({ en_US: 'Hello', zh_CN: '你好' });
  expect(TRANSLATED_BASE_LANG).toBe('en_US');
});

test('deleteLangKey and applyFieldTranslationsPatch honor D12', () => {
  expect(deleteLangKey({ en_US: 'A', zh_CN: '甲' }, 'zh_CN', 'Name')).toEqual({ en_US: 'A' });
  expect(() => deleteLangKey({ en_US: 'A' }, 'en_US', 'Name')).toThrow(/cannot delete base language/);

  expect(
    applyFieldTranslationsPatch({
      fieldName: 'Name',
      currentMap: { en_US: 'Hello', zh_CN: '你好', fr_FR: 'Bonjour' },
      translations: { zh_CN: '您好', fr_FR: false, de_DE: '' },
    })
  ).toEqual({ en_US: 'Hello', zh_CN: '您好', de_DE: '' });

  expect(() =>
    applyFieldTranslationsPatch({
      fieldName: 'Name',
      currentMap: { en_US: 'Hello' },
      translations: { en_US: false },
    })
  ).toThrow(/cannot delete base language/);
});

test('mergeTranslatedWrite replace mode overwrites the whole map', () => {
  expect(
    mergeTranslatedWrite({
      fieldName: 'Name',
      value: { en_US: 'Only' },
      lang: 'en_US',
      currentMap: { en_US: 'Old', zh_CN: '旧' },
      mode: 'update',
      replace: true,
    })
  ).toEqual({ en_US: 'Only' });

  const meta = {
    fields: new Map<string, any>([['Name', { type: 'varchar', translate: true, storageHints: { size: 100 } }]]),
  } as any;
  const replaced = withContext({ translated_write_replace: true }, () =>
    applyTranslatedFieldsForWrite(meta, { Name: { en_US: 'New' } } as any, {
      mode: 'update',
      current: { Name: JSON.stringify({ en_US: 'Old', zh_CN: '旧' }) },
    })
  );
  expect(replaced).toEqual({ Name: { en_US: 'New' } });
});
