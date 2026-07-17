// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Locale from '@/base/service/models/locale';
import { ChoysumError } from '@/core/service/error';
import { resolveValidationSummary } from '@/core/service/api/validation';
import { MetadataStorage } from '@/core/service/api/metadata';
import { createTextDescriptor } from '@/core/service/i18n';

import { companyCode8, uid } from './_helpers';

test('base.locale: CurrencySymbolPosition selection exposes localized labels without changing values', () => {
  const field = MetadataStorage.instance.getModelMetadata(Locale).fields.get('CurrencySymbolPosition');

  expect(field?.selection).toEqual([
    {
      value: 'before',
      label: 'Before amount',
      labelText: createTextDescriptor('base', 'Before amount', { scope: 'base.Locale.CurrencySymbolPosition.before' }),
    },
    {
      value: 'after',
      label: 'After amount',
      labelText: createTextDescriptor('base', 'After amount', { scope: 'base.Locale.CurrencySymbolPosition.after' }),
    },
  ]);
});

test('base.locale: CurrencySymbolPosition invalid is rejected', async () => {
  let error: unknown;
  try {
    await Locale.Create(
      {
        Name: uid('Locale'),
        Code: companyCode8(),
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

test('base.locale: CurrencySymbolPosition defaults + spacing defaults', async () => {
  const created = await Locale.Create(
    {
      Name: uid('Locale'),
      Code: companyCode8(),
      CurrencySymbolPosition: null as any,
      CurrencySymbolSpacing: null as any,
      IsActive: true,
    } as any,
    ['Id', 'CurrencySymbolPosition', 'CurrencySymbolSpacing'] as any
  );

  expect(String((created as any).CurrencySymbolPosition)).toBe('before');
  expect(Boolean((created as any).CurrencySymbolSpacing)).toBe(false);
});

test('base.locale: CurrencySymbolPosition blank is rejected', async () => {
  let error: unknown;
  try {
    await Locale.Create(
      {
        Name: uid('LocaleBlank'),
        Code: companyCode8(),
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
