// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Language from '@/base/service/models/language';
import { applyGrouping, formatNumberWithLanguage, parseGrouping } from '@/base/service/models/_language_format';
import { companyCode8, uid } from './_helpers';

test('base.language.format: parseGrouping accepts string and array forms', () => {
  expect(parseGrouping('[3,0]')).toEqual([3, 0]);
  expect(parseGrouping([3, 2, 0])).toEqual([3, 2, 0]);
  expect(parseGrouping('')).toEqual([3, 0]);
});

test('base.language.format: applyGrouping uses thousands separator', () => {
  expect(applyGrouping('1234567', [3, 0], ',')).toBe('1,234,567');
  expect(applyGrouping('1234567', [3, 0], ' ')).toBe('1 234 567');
});

test('base.language.format: formatNumberWithLanguage respects separators and grouping', () => {
  expect(
    formatNumberWithLanguage(1234567.89, {
      DecimalSeparator: ',',
      ThousandSeparator: '.',
      Grouping: '[3,0]',
    }, { digits: 2 })
  ).toBe('1.234.567,89');
});

test('base.language: Format reads Language separators after update', async () => {
  const code = `f_${companyCode8()}`.slice(0, 16);
  const created = await Language.Create(
    {
      Name: uid('LangFmt'),
      Code: code,
      Direction: 'ltr' as any,
      IsActive: true,
      DecimalSeparator: '.',
      ThousandSeparator: ',',
      Grouping: '[3,0]',
      DateFormat: 'YYYY-MM-DD',
      TimeFormat: 'HH:mm:ss',
    } as any,
    ['Id', 'Code'] as any
  );

  const before = await Language.Format({ Code: code, Value: 1234.5, Kind: 'number', Digits: 1 });
  expect(before).toBe('1,234.5');

  await Language.UpdateById(
    (created as any).Id,
    {
      DecimalSeparator: ',',
      ThousandSeparator: '.',
      Grouping: '[3,0]',
    } as any,
    ['Id'] as any
  );

  const after = await Language.Format({ Code: code, Value: 1234.5, Kind: 'number', Digits: 1 });
  expect(after).toBe('1.234,5');
});

test('base.language: Format respects Grouping changes (T2.1)', async () => {
  const code = `g_${companyCode8()}`.slice(0, 16);
  const created = await Language.Create(
    {
      Name: uid('LangGrp'),
      Code: code,
      Direction: 'ltr' as any,
      IsActive: true,
      DecimalSeparator: '.',
      ThousandSeparator: ',',
      Grouping: '[3,0]',
    } as any,
    ['Id'] as any
  );

  expect(await Language.Format({ Code: code, Value: 1234567, Kind: 'number', Digits: 0 })).toBe('1,234,567');

  await Language.UpdateById(
    (created as any).Id,
    { Grouping: '[3,2,0]', ThousandSeparator: ',' } as any,
    ['Id'] as any
  );

  // First group 3 from right, then groups of 2: 12,34,567
  expect(await Language.Format({ Code: code, Value: 1234567, Kind: 'number', Digits: 0 })).toBe('12,34,567');
});

test('base.language: Format date uses DateFormat / TimeFormat', async () => {
  const code = `d_${companyCode8()}`.slice(0, 16);
  await Language.Create(
    {
      Name: uid('LangDate'),
      Code: code,
      Direction: 'ltr' as any,
      IsActive: true,
      DateFormat: 'DD/MM/YYYY',
      TimeFormat: 'HH:mm',
    } as any,
    ['Id'] as any
  );

  const formatted = await Language.Format({
    Code: code,
    Value: new Date(2026, 0, 2, 15, 4, 0),
    Kind: 'datetime',
  });
  expect(formatted).toBe('02/01/2026 15:04');
});
