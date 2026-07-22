// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Language from '@/base/service/models/language';
import { ChoysumError } from '@/core/service/error';
import { resolveValidationSummary } from '@/core/service/api/validation';
import { MetadataStorage } from '@/core/service/api/metadata';

import { companyCode8, uid } from './_helpers';

test('base.language: Direction selection uses _lt msgid labels', () => {
  const field = MetadataStorage.instance.getModelMetadata(Language).fields.get('Direction');
  const selection = field?.selection as Array<{ value: string; label: string; labelText?: { src?: string; scope?: string } }> | undefined;

  expect(selection?.map(item => item.value)).toEqual(['ltr', 'rtl']);
  expect(selection?.map(item => item.label)).toEqual(['Left to right', 'Right to left']);
  expect(selection?.every(item => item.labelText?.src === item.label)).toBe(true);
  expect(selection?.every(item => item.labelText?.scope === 'base.model.Language.fields')).toBe(true);
});

test('base.language: Direction invalid is rejected', async () => {
  let error: unknown;
  try {
    await Language.Create(
      {
        Name: uid('Language'),
        Code: companyCode8(),
        Direction: 'sideways' as any,
      } as any,
      ['Id'] as any
    );
  } catch (err) {
    error = err;
  }

  expect(error instanceof ChoysumError).toBe(true);
  const oe = error as ChoysumError;
  expect(oe.domain).toBe('core.repository');
  expect(oe.code).toBe('validation_failed');
  expect(oe.metadata?.mode).toBe('create');

  const summary = resolveValidationSummary(oe);
  const codes = summary.issues.map(item => String(item?.code || ''));
  expect(codes.some(code => code === 'constraint_execution_failed' || code.startsWith('kernel_') || code.startsWith('sql_'))).toBe(true);
});

test('base.language: Direction ltr is accepted', async () => {
  const created = await Language.Create(
    {
      Name: uid('Language'),
      Code: companyCode8(),
      Direction: 'ltr' as any,
      IsActive: true,
    } as any,
    ['Id', 'Direction'] as any
  );

  expect(String((created as any).Direction)).toBe('ltr');
});

test('base.language: CurrencySymbolPosition selection uses _lt msgid labels', () => {
  const field = MetadataStorage.instance.getModelMetadata(Language).fields.get('CurrencySymbolPosition');
  const selection = field?.selection as Array<{ value: string; label: string; labelText?: { src?: string; scope?: string } }> | undefined;

  expect(selection?.map(item => item.value)).toEqual(['before', 'after']);
  expect(selection?.map(item => item.label)).toEqual(['Before amount', 'After amount']);
  expect(selection?.every(item => item.labelText?.src === item.label)).toBe(true);
  expect(selection?.every(item => item.labelText?.scope === 'base.model.Language.fields')).toBe(true);
});

test('base.language: CurrencySymbolPosition invalid is rejected', async () => {
  let error: unknown;
  try {
    await Language.Create(
      {
        Name: uid('Language'),
        Code: companyCode8(),
        Direction: 'ltr' as any,
        CurrencySymbolPosition: 'middle' as any,
      } as any,
      ['Id'] as any
    );
  } catch (err) {
    error = err;
  }

  expect(error instanceof ChoysumError).toBe(true);
  const oe = error as ChoysumError;
  expect(oe.domain).toBe('core.repository');
  expect(oe.code).toBe('validation_failed');
  expect(oe.metadata?.mode).toBe('create');
  const summary = resolveValidationSummary(oe);
  const codes = summary.issues.map(item => String(item?.code || ''));
  expect(codes.some(code => code === 'constraint_execution_failed' || code.startsWith('kernel_') || code.startsWith('sql_'))).toBe(true);
});

test('base.language: CurrencySymbolPosition defaults + spacing defaults', async () => {
  const created = await Language.Create(
    {
      Name: uid('Language'),
      Code: companyCode8(),
      Direction: 'ltr' as any,
      CurrencySymbolPosition: null as any,
      CurrencySymbolSpacing: null as any,
      IsActive: true,
    } as any,
    ['Id', 'CurrencySymbolPosition', 'CurrencySymbolSpacing', 'Grouping'] as any
  );

  expect(String((created as any).CurrencySymbolPosition)).toBe('before');
  expect(Boolean((created as any).CurrencySymbolSpacing)).toBe(false);
  expect(String((created as any).Grouping)).toBe('[3,0]');
});

test('base.language: CurrencySymbolPosition blank is rejected', async () => {
  let error: unknown;
  try {
    await Language.Create(
      {
        Name: uid('LanguageBlank'),
        Code: companyCode8(),
        Direction: 'ltr' as any,
        CurrencySymbolPosition: '' as any,
        IsActive: true,
      } as any,
      ['Id', 'CurrencySymbolPosition'] as any
    );
  } catch (err) {
    error = err;
  }

  expect(error instanceof ChoysumError).toBe(true);
  const oe = error as ChoysumError;
  expect(oe.domain).toBe('core.repository');
  expect(oe.code).toBe('validation_failed');
  expect(oe.metadata?.mode).toBe('create');
  const summary = resolveValidationSummary(oe);
  const codes = summary.issues.map(item => String(item?.code || ''));
  expect(codes.some(code => code === 'constraint_execution_failed' || code.startsWith('kernel_') || code.startsWith('sql_'))).toBe(true);
});

test('base.language: Create with format fields succeeds', async () => {
  const created = await Language.Create(
    {
      Name: uid('LanguageFmt'),
      Code: companyCode8(),
      Direction: 'ltr' as any,
      IsActive: true,
      DecimalSeparator: '.',
      ThousandSeparator: ',',
      Grouping: '[3,0]',
      DateFormat: 'YYYY-MM-DD',
      TimeFormat: 'HH:mm:ss',
      FirstDayOfWeek: 1,
      CurrencySymbolPosition: 'before' as any,
      CurrencySymbolSpacing: false,
    } as any,
    ['Id', 'Code', 'DecimalSeparator', 'ThousandSeparator', 'Grouping', 'FirstDayOfWeek'] as any
  );

  expect(String((created as any).DecimalSeparator)).toBe('.');
  expect(String((created as any).ThousandSeparator)).toBe(',');
  expect(String((created as any).Grouping)).toBe('[3,0]');
  expect(Number((created as any).FirstDayOfWeek)).toBe(1);
});
